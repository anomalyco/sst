import { ComponentResourceOptions, Output, output } from "@pulumi/pulumi";
import { Component, Transform, transform } from "../component";
import { Input } from "../input";
import { Dns } from "../dns";
import { cognito } from "@pulumi/aws";
import { dns as awsDns } from "./dns.js";
import { DnsValidatedCertificate } from "./dns-validated-certificate.js";
import { useProvider } from "./helpers/provider.js";
import { VisibleError } from "../error";

export interface Args {
  /**
   * The Cognito user pool ID.
   */
  userPool: Input<string>;
  /**
   * The domain configuration - either a string for custom domain or an object.
   */
  domain: Input<
    | string
    | {
        prefix?: Input<string>;
        name?: Input<string>;
        dns?: Input<false | (Dns & {})>;
        cert?: Input<string>;
      }
  >;
  /**
   * Transform the Cognito User Pool domain resource.
   */
  transform?: Transform<cognito.UserPoolDomainArgs>;
}

/**
 * The `CognitoUserPoolDomain` component is internally used by the `CognitoUserPool`
 * component to add a domain to your [Amazon Cognito user pool](https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-identity-pools.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * Use the `domain` prop on `CognitoUserPool` instead.
 *
 * @todo Add `managedLoginVersion` prop when SST upgrades to Pulumi AWS v7.
 * This will allow users to choose between classic hosted UI (1) and managed login (2).
 * See: https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-pools-managed-login.html
 */
export class CognitoUserPoolDomain extends Component {
  private _domain: Output<cognito.UserPoolDomain>;
  private _domainUrl: Output<string>;

  constructor(name: string, args: Args, opts?: ComponentResourceOptions) {
    super(__pulumiType, name, args, opts);

    const parent = this;

    const normalized = normalizeDomain();
    const domain = createDomainWithSsl();

    this._domain = domain;
    this._domainUrl = normalized.apply((n) =>
      n.prefix
        ? `https://${n.prefix}.auth.${process.env.AWS_REGION || "us-east-1"}.amazoncognito.com`
        : `https://${n.name}`,
    );

    function normalizeDomain() {
      return output(args.domain).apply((domain) => {
        const norm = typeof domain === "string" ? { name: domain } : domain;

        // Validate
        if (norm.prefix && norm.name) {
          throw new VisibleError(
            `Cannot specify both "prefix" and "name". Use "prefix" for a Cognito-hosted domain or "name" for a custom domain.`,
          );
        }
        if (!norm.prefix && !norm.name) {
          throw new VisibleError(
            `Must specify either "prefix" or "name" in domain configuration.`,
          );
        }

        // For custom domains, validate DNS/cert requirements
        if (norm.name && norm.dns === false && !norm.cert) {
          throw new VisibleError(
            `Need to provide a validated certificate via "cert" when DNS is disabled.`,
          );
        }

        return {
          prefix: norm.prefix,
          name: norm.name,
          dns: norm.dns === false ? undefined : norm.dns ?? (norm.name ? awsDns() : undefined),
          cert: norm.cert,
        };
      });
    }

    function createDomainWithSsl() {
      return normalized.apply((norm) => {
        // For prefix domains, no SSL needed
        if (norm.prefix) {
          const [resourceName, domainArgs, resourceOpts] = transform(
            args.transform,
            `${name}Domain`,
            {
              userPoolId: args.userPool,
              domain: norm.prefix,
            },
            { parent, deleteBeforeReplace: true },
          );

          const domain = new cognito.UserPoolDomain(
            resourceName,
            domainArgs,
            resourceOpts,
          );

          return domain;
        }

        // For custom domains, create SSL cert if needed
        let certificateArn: Input<string>;
        if (norm.cert) {
          certificateArn = norm.cert;
        } else {
          const cert = new DnsValidatedCertificate(
            `${name}Ssl`,
            {
              domainName: norm.name!,
              dns: output(norm.dns!),
            },
            { parent, provider: useProvider("us-east-1") },
          );
          certificateArn = cert.arn;
        }

        const [resourceName, domainArgs, resourceOpts] = transform(
          args.transform,
          `${name}Domain`,
          {
            userPoolId: args.userPool,
            domain: norm.name!,
            certificateArn,
          },
          { parent, deleteBeforeReplace: true },
        );

        const domain = new cognito.UserPoolDomain(
          resourceName,
          domainArgs,
          resourceOpts,
        );

        // Create DNS records for custom domain
        if (norm.dns) {
          norm.dns.createAlias(
            name,
            {
              name: norm.name!,
              aliasName: domain.cloudfrontDistribution,
              aliasZone: domain.cloudfrontDistributionZoneId,
            },
            { parent },
          );
        }

        return domain;
      });
    }
  }

  /**
   * The Cognito User Pool domain string. For prefix domains, this is the prefix.
   * For custom domains, this is the full domain name.
   */
  public get domainName() {
    return this._domain.apply((d) => d.domain);
  }

  /**
   * The full URL of the hosted UI domain.
   */
  public get domainUrl() {
    return this._domainUrl;
  }

  /**
   * The CloudFront distribution domain name. Useful for custom domain DNS setup
   * when managing DNS outside of SST.
   */
  public get cloudfrontDistribution() {
    return this._domain.apply((d) => d.cloudfrontDistribution);
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The Cognito User Pool domain.
       */
      domain: this._domain,
    };
  }
}

const __pulumiType = "sst:aws:CognitoUserPoolDomain";
// @ts-expect-error
CognitoUserPoolDomain.__pulumiType = __pulumiType;
