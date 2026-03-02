import {
  ComponentResourceOptions,
  Output,
  interpolate,
  output,
} from "@pulumi/pulumi";
import { Component, Transform, transform } from "../component";
import { Input } from "../input";
import { Dns } from "../dns";
import { cognito, getRegionOutput } from "@pulumi/aws";
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
        prefix: Input<string>;
      }
    | {
        name: Input<string>;
        dns?: Input<false | (Dns & {})>;
        cert?: Input<string>;
      }
  >;
  /**
   * The managed login version. Mapped from the `login` prop on `CognitoUserPool`.
   */
  managedLoginVersion?: Input<number | undefined>;
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
 */
export class CognitoUserPoolDomain extends Component {
  private _domain: cognito.UserPoolDomain;
  private _domainUrl: Output<string>;

  constructor(name: string, args: Args, opts?: ComponentResourceOptions) {
    super(__pulumiType, name, args, opts);

    const parent = this;
    const region = getRegionOutput(undefined, { parent }).region;

    const normalized = normalizeDomain();
    const certificateArn = createSsl();
    const domain = createDomain();
    createDnsRecords();

    this._domain = domain;
    this._domainUrl = normalized.apply((n) =>
      n.prefix
        ? interpolate`https://${n.prefix}.auth.${region}.amazoncognito.com`
        : interpolate`https://${n.name}`,
    );

    function normalizeDomain() {
      return output(args.domain).apply((domain) => {
        if (typeof domain === "string") domain = { name: domain };

        if ("prefix" in domain) {
          return {
            prefix: domain.prefix,
            name: undefined,
            dns: undefined,
            cert: undefined,
          };
        }

        if (domain.dns === false && !domain.cert) {
          throw new VisibleError(
            `Need to provide a validated certificate via "cert" when DNS is disabled.`,
          );
        }

        return {
          prefix: undefined,
          name: domain.name,
          dns:
            domain.dns === false
              ? undefined
              : domain.dns ?? awsDns(),
          cert: domain.cert,
        };
      });
    }

    function createSsl() {
      return normalized.apply((norm) => {
        if (norm.prefix) return output(undefined);
        if (norm.cert) return output(norm.cert);

        return new DnsValidatedCertificate(
          `${name}Ssl`,
          {
            domainName: norm.name!,
            dns: output(norm.dns!),
          },
          { parent, provider: useProvider("us-east-1") },
        ).arn;
      });
    }

    function createDomain() {
      return new cognito.UserPoolDomain(
        ...transform(
          args.transform,
          `${name}Domain`,
          {
            userPoolId: args.userPool,
            domain: normalized.apply((n) => (n.prefix ?? n.name)!),
            certificateArn: certificateArn as Output<string>,
            managedLoginVersion: args.managedLoginVersion,
          },
          { parent, deleteBeforeReplace: true },
        ),
      );
    }

    function createDnsRecords() {
      normalized.apply((norm) => {
        if (!norm.name || !norm.dns) return;

        norm.dns.createAlias(
          name,
          {
            name: norm.name,
            aliasName: domain.cloudfrontDistribution,
            aliasZone: domain.cloudfrontDistributionZoneId,
          },
          { parent },
        );
      });
    }
  }

  /**
   * The Cognito User Pool domain string. For prefix domains, this is the prefix.
   * For custom domains, this is the full domain name.
   */
  public get domainName() {
    return this._domain.domain;
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
    return this._domain.cloudfrontDistribution;
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
