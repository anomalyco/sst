import { ComponentResourceOptions, all, output } from "@pulumi/pulumi";
import { DnsValidatedCertificate } from "./dns-validated-certificate.js";
import { Bucket } from "./bucket.js";
import { Component } from "../component.js";
import { logicalName, physicalName } from "../naming.js";
import { useProvider } from "./helpers/provider.js";
import { Input } from "../input.js";
import { Dns } from "../dns.js";
import { cloudfront, s3 } from "@pulumi/aws";
import { CF_BLOCK_CLOUDFRONT_URL_INJECTION } from "./router.js";
import { OriginAccessControl } from "./providers/origin-access-control.js";

/**
 * Properties to configure an HTTPS Redirect
 */
export interface HttpsRedirectArgs {
  /**
   * The redirect target fully qualified domain name (FQDN). An alias record
   * will be created that points to your CloudFront distribution. Root domain
   * or sub-domain can be supplied.
   */
  targetDomain: Input<string>;
  /**
   * The domain names that will redirect to `targetDomain`
   *
   * @default Domain name of the hosted zone
   */
  sourceDomains: Input<string[]>;
  /**
   * The ARN of an ACM (AWS Certificate Manager) certificate that proves ownership of the
   * domain. By default, a certificate is created and validated automatically.
   */
  cert?: Input<string>;
  /**
   * The DNS adapter you want to use for managing DNS records.
   */
  dns?: Input<Dns & {}>;
}

/**
 * Allows creating a domainA -> domainB redirect at the CloudFront edge.
 * You can specify multiple domains to be redirected. Paths and query strings
 * are preserved.
 */
export class HttpsRedirect extends Component {
  constructor(
    name: string,
    args: HttpsRedirectArgs,
    opts?: ComponentResourceOptions,
  ) {
    super(__pulumiType, name, args, opts);

    const parent = this;

    validateArgs();
    const certificateArn = createSsl();
    const bucket = createBucket();
    // Retain the website configuration so existing edge locations can still use
    // the old origin while the distribution update is deploying.
    createBucketWebsite();
    const originAccessControl = createOriginAccessControl();
    const distribution = createDistribution();
    createDnsRecords();

    function validateArgs() {
      if (!args.dns && !args.cert)
        throw new Error(
          `Need to provide a validated certificate via "cert" when DNS is disabled`,
        );
    }

    function createSsl() {
      if (args.cert) return args.cert;

      return new DnsValidatedCertificate(
        `${name}Ssl`,
        {
          domainName: output(args.sourceDomains).apply((domains) => domains[0]),
          alternativeNames: output(args.sourceDomains).apply((domains) =>
            domains.slice(1),
          ),
          dns: args.dns!,
        },
        { parent, provider: useProvider("us-east-1") },
      ).arn;
    }

    function createBucket() {
      return new Bucket(`${name}Bucket`, {}, { parent });
    }

    function createBucketWebsite() {
      return new s3.BucketWebsiteConfiguration(
        `${name}BucketWebsite`,
        {
          bucket: bucket.name,
          redirectAllRequestsTo: {
            hostName: args.targetDomain,
            protocol: "https",
          },
        },
        { parent },
      );
    }

    function createOriginAccessControl() {
      return new OriginAccessControl(
        `${name}S3AccessControl`,
        { name: physicalName(64, name) },
        { parent, ignoreChanges: ["name"] },
      );
    }

    function createDistribution() {
      return new cloudfront.Distribution(
        `${name}Distribution`,
        {
          enabled: true,
          waitForDeployment: false,
          aliases: args.sourceDomains,
          restrictions: {
            geoRestriction: {
              restrictionType: "none",
            },
          },
          comment: all([args.targetDomain, args.sourceDomains]).apply(
            ([targetDomain, sourceDomains]) => {
              const comment = `Redirect to ${targetDomain} from ${sourceDomains.join(
                ", ",
              )}`;
              return comment.length > 128
                ? comment.slice(0, 125) + "..."
                : comment;
            },
          ),
          priceClass: "PriceClass_All",
          viewerCertificate: {
            acmCertificateArn: certificateArn,
            sslSupportMethod: "sni-only",
          },
          defaultCacheBehavior: {
            allowedMethods: ["GET", "HEAD", "OPTIONS"],
            targetOriginId: "s3Origin",
            viewerProtocolPolicy: "redirect-to-https",
            cachedMethods: ["GET", "HEAD"],
            forwardedValues: {
              cookies: { forward: "none" },
              queryString: false,
            },
            functionAssociations: [
              {
                eventType: "viewer-request",
                functionArn: new cloudfront.Function(
                  `${name}CloudfrontFunctionRequest`,
                  {
                    name: physicalName(64, `${name}CloudfrontFunctionRequest`),
                    runtime: "cloudfront-js-2.0",
                    code: output(args.targetDomain).apply(
                      (targetDomain) => `
async function handler(event) {
  ${CF_BLOCK_CLOUDFRONT_URL_INJECTION}
  var request = event.request;
  var query = request.rawQueryString();
  return {
    statusCode: 301,
    statusDescription: "Moved Permanently",
    headers: {
      location: {
        value: ${JSON.stringify("https://" + targetDomain)} + request.uri
          + (query === undefined ? "" : "?" + query)
      }
    }
  };
}`,
                    ),
                  },
                  { ignoreChanges: ["name"] },
                ).arn,
              },
            ],
          },
          origins: [
            {
              originId: "s3Origin",
              // CloudFront requires an origin even though the function responds
              // before contacting it. Keep this fallback private and HTTPS-only.
              domainName: bucket.nodes.bucket.bucketRegionalDomainName,
              originAccessControlId: originAccessControl.id,
              s3OriginConfig: {
                originAccessIdentity: "",
              },
            },
          ],
        },
        { parent },
      );
    }

    function createDnsRecords() {
      if (!args.dns) return;

      all([args.dns, args.sourceDomains]).apply(([dns, sourceDomains]) => {
        const existing: string[] = [];
        for (const [i, recordName] of sourceDomains.entries()) {
          // Note: The way `dns` is implemented, the logical name for the DNS record is
          // based on the sanitized version of the record name (ie. logicalName()). This
          // means the logical name for `*.sst.sh` and `sst.sh` will trash b/c `*.` is
          // stripped out.
          // ```
          // domain: {
          //   name: "*.sst.sh",
          //   aliases: ['sst.sh'],
          // },
          // ```
          //
          // Ideally, we don't sanitize the logical name. But that's a breaking change.
          //
          // As a workaround, starting v3.19.2, we prefix the logical name with a unique
          // index for records with logical names that will trash.
          const key = logicalName(recordName);
          const namePrefix = existing.includes(key) ? `${name}${i}` : name;
          existing.push(key);

          dns.createAlias(
            namePrefix,
            {
              name: recordName,
              aliasName: distribution.domainName,
              aliasZone: distribution.hostedZoneId,
            },
            { parent },
          );
        }
      });
    }
  }
}

const __pulumiType = "sst:aws:HttpsRedirect";
// @ts-expect-error
HttpsRedirect.__pulumiType = __pulumiType;
