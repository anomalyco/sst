import {
  ComponentResourceOptions,
  Output,
  output,
} from "@pulumi/pulumi";
import { Component, Transform, transform } from "../component.js";
import { dns as awsDns } from "./dns.js";
import { VisibleError } from "../error.js";
import { DnsValidatedCertificate } from "./dns-validated-certificate.js";
import { Link } from "../link.js";
import { ec2, lb } from "@pulumi/aws";
import { Vpc } from "./vpc.js";
import { Input } from "../input.js";
import { Dns } from "../dns.js";
import { listenerKey } from "./helpers/load-balancer.js";

export interface AlbListenerArgs {
  /**
   * The port to listen on.
   *
   * @example
   * ```js
   * {
   *   port: 443
   * }
   * ```
   */
  port: number;
  /**
   * The protocol to listen on. Only `http` and `https` are supported (ALB-only).
   *
   * @example
   * ```js
   * {
   *   protocol: "https"
   * }
   * ```
   */
  protocol: "http" | "https";
}

export interface AlbArgs {
  /**
   * The VPC to deploy the ALB in. Can be an SST `Vpc` component or a custom VPC configuration.
   *
   * @example
   *
   * Using an SST Vpc component:
   * ```js
   * {
   *   vpc: myVpc
   * }
   * ```
   *
   * Using a custom VPC:
   * ```js
   * {
   *   vpc: {
   *     id: "vpc-0123456789abcdef0",
   *     publicSubnets: ["subnet-abc", "subnet-def"],
   *     privateSubnets: ["subnet-ghi", "subnet-jkl"]
   *   }
   * }
   * ```
   */
  vpc: Vpc | Input<{
    /**
     * The VPC ID.
     */
    id: Input<string>;
    /**
     * The public subnet IDs.
     */
    publicSubnets: Input<Input<string>[]>;
    /**
     * The private subnet IDs.
     */
    privateSubnets: Input<Input<string>[]>;
  }>;
  /**
   * Configure if the load balancer should be public (internet-facing) or private (internal).
   *
   * When set to `false`, the load balancer endpoint will only be accessible within the VPC.
   *
   * @default `true`
   */
  public?: Input<boolean>;
  /**
   * Set a custom domain for the load balancer.
   *
   * Automatically manages domains hosted on AWS Route 53, Cloudflare, and Vercel. For other
   * providers, you'll need to pass in a `cert` that validates domain ownership and add the
   * DNS records.
   *
   * @example
   *
   * ```js
   * {
   *   domain: "example.com"
   * }
   * ```
   *
   * For domains on Cloudflare:
   * ```js
   * {
   *   domain: {
   *     name: "example.com",
   *     dns: sst.cloudflare.dns()
   *   }
   * }
   * ```
   */
  domain?:
    | string
    | {
        /**
         * The custom domain name.
         */
        name: string;
        /**
         * Alias domains that should also point to this load balancer.
         */
        aliases?: string[];
        /**
         * The ARN of an ACM certificate that proves ownership of the domain.
         * By default, a certificate is created and validated automatically.
         */
        cert?: Input<string>;
        /**
         * The DNS provider to use. Defaults to AWS Route 53.
         * Set to `false` for manual DNS setup.
         *
         * @default `sst.aws.dns`
         */
        dns?: false | (Dns & {});
      };
  /**
   * The listeners for the load balancer. Each entry creates a listener on the specified
   * port and protocol.
   *
   * @example
   * ```js
   * {
   *   listeners: [
 *     { port: 80, protocol: "http" },
 *     { port: 443, protocol: "https" }
 *   ]
   * }
   * ```
   */
  listeners: AlbListenerArgs[];
  /**
   * [Transform](/docs/components#transform) how this component creates its underlying resources.
   */
  transform?: {
    /**
     * Transform the AWS Load Balancer resource.
     */
    loadBalancer?: Transform<lb.LoadBalancerArgs>;
    /**
     * Transform the AWS Security Group resource for the Load Balancer.
     */
    securityGroup?: Transform<ec2.SecurityGroupArgs>;
    /**
     * Transform the AWS Load Balancer listener resource.
     */
    listener?: Transform<lb.ListenerArgs>;
  };
}

interface AlbRef {
  ref: true;
  loadBalancerArn: Input<string>;
}

/**
 * The `Alb` component lets you create a standalone Application Load Balancer that can be
 * shared across multiple services.
 *
 * @example
 *
 * #### Create a shared ALB
 *
 * ```ts title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 *
 * const alb = new sst.aws.Alb("SharedAlb", {
 *   vpc,
 *   domain: "app.example.com",
 *   listeners: [
 *     { port: 80, protocol: "http" },
 *     { port: 443, protocol: "https" },
 *   ],
 * });
 * ```
 *
 * #### Attach services to the ALB
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Service("Api", {
 *   cluster,
 *   image: "api:latest",
 *   loadBalancer: {
 *     instance: alb,
 *     rules: [
 *       { listen: "443/https", forward: "8080/http", conditions: { path: "/api/*" }, priority: 100 },
 *     ],
 *   },
 * });
 * ```
 *
 * #### Reference an existing ALB
 *
 * ```ts title="sst.config.ts"
 * const alb = sst.aws.Alb.get("SharedAlb", "arn:aws:elasticloadbalancing:...");
 * ```
 */
export class Alb extends Component implements Link.Linkable {
  private loadBalancer: lb.LoadBalancer;
  private securityGroup: ec2.SecurityGroup;
  private listeners: Record<string, lb.Listener> = {};
  private certificateArn?: Output<string | undefined>;
  private vpcId: Output<string>;
  private _url: Output<string>;
  private constructorName: string;

  constructor(
    name: string,
    args: AlbArgs,
    opts: ComponentResourceOptions = {},
  ) {
    super(__pulumiType, name, args, opts);

    this.constructorName = name;

    if (args && "ref" in args) {
      const ref = args as unknown as AlbRef;
      const loadBalancer = lb.LoadBalancer.get(
        `${name}LoadBalancer`,
        output(ref.loadBalancerArn),
        {},
        { parent: this },
      );
      const securityGroup = ec2.SecurityGroup.get(
        `${name}SecurityGroup`,
        loadBalancer.securityGroups.apply((sgs) => {
          if (!sgs?.length) {
            throw new VisibleError(
              `No security groups found on the referenced ALB "${name}".`,
            );
          }
          return sgs[0];
        }),
        {},
        { parent: this },
      );
      this.loadBalancer = loadBalancer;
      this.securityGroup = securityGroup;
      this.vpcId = loadBalancer.vpcId;
      this._url = loadBalancer.dnsName.apply(
        (dnsName) => `http://${dnsName}`,
      );
      this.registerOutputs({ _hint: this._url });
      return;
    }

    const self = this;
    const pub = output(args.public ?? true);
    const { vpcId, subnets } = normalizeVpc();
    const domain = normalizeDomain();
    const securityGroup = createSecurityGroup();
    const certificateArn = createSsl();
    const loadBalancer = createLoadBalancer();
    createListeners();
    createDnsRecords();

    this.loadBalancer = loadBalancer;
    this.securityGroup = securityGroup;
    this.certificateArn = certificateArn;
    this.vpcId = vpcId;
    this._url = domain
      ? output(`https://${domain.name}/`)
      : loadBalancer.dnsName.apply((dnsName) => `http://${dnsName}`);

    this.registerOutputs({ _hint: this._url });

    function normalizeVpc() {
      if (args.vpc instanceof Vpc) {
        const vpc = args.vpc;
        return {
          vpcId: vpc.id,
          subnets: pub.apply((isPublic) =>
            isPublic ? vpc.publicSubnets : vpc.privateSubnets,
          ),
        };
      }

      return output(args.vpc).apply((vpc) => ({
        vpcId: output(vpc.id),
        subnets: pub.apply((isPublic) =>
          isPublic
            ? output(vpc.publicSubnets)
            : output(vpc.privateSubnets),
        ),
      }));
    }

    function normalizeDomain() {
      if (!args.domain) return undefined;
      const raw =
        typeof args.domain === "string" ? { name: args.domain } : args.domain;
      return {
        name: raw.name,
        aliases: raw.aliases ?? [],
        dns: raw.dns === false ? undefined : raw.dns ?? awsDns(),
        cert: raw.cert,
      };
    }

    function createSecurityGroup() {
      return new ec2.SecurityGroup(
        ...transform(
          args.transform?.securityGroup,
          `${name}SecurityGroup`,
          {
            description: "Managed by SST",
            vpcId,
            egress: [
              {
                fromPort: 0,
                toPort: 0,
                protocol: "-1",
                cidrBlocks: ["0.0.0.0/0"],
              },
            ],
            ingress: [
              {
                fromPort: 0,
                toPort: 0,
                protocol: "-1",
                cidrBlocks: ["0.0.0.0/0"],
              },
            ],
          },
          { parent: self },
        ),
      );
    }

    function createSsl(): Output<string | undefined> {
      if (!domain) return output(undefined);
      if (domain.cert) return output(domain.cert);

      return new DnsValidatedCertificate(
        `${name}Ssl`,
        {
          domainName: domain.name,
          alternativeNames: domain.aliases,
          dns: domain.dns!,
        },
        { parent: self },
      ).arn;
    }

    function createLoadBalancer() {
      return new lb.LoadBalancer(
        ...transform(
          args.transform?.loadBalancer,
          `${name}LoadBalancer`,
          {
            internal: pub.apply((v) => !v),
            loadBalancerType: "application",
            subnets,
            securityGroups: [securityGroup.id],
            enableCrossZoneLoadBalancing: true,

          },
          { parent: self },
        ),
      );
    }

    function createListeners() {
      for (const l of args.listeners) {
        const protocol = l.protocol.toUpperCase();
        const port = l.port;
        const key = listenerKey(protocol, port);

        const listener = new lb.Listener(
          ...transform(
            args.transform?.listener,
            `${name}Listener${protocol}${port}`,
            {
              loadBalancerArn: loadBalancer.arn,
              port,
              protocol,
              certificateArn: protocol === "HTTPS"
                ? certificateArn.apply((arn) => arn!) as Output<string>
                : undefined,
              defaultActions: [
                {
                  type: "fixed-response",
                  fixedResponse: {
                    statusCode: "403",
                    contentType: "text/plain",
                    messageBody: "Forbidden",
                  },
                },
              ],
            },
            { parent: self },
          ),
        );

        self.listeners[key] = listener;
      }
    }

    function createDnsRecords() {
      if (!domain?.dns) return;

      for (const recordName of [domain.name, ...domain.aliases]) {
        const namePrefix =
          recordName === domain.name ? name : `${name}${recordName}`;
        domain.dns.createAlias(
          namePrefix,
          {
            name: recordName,
            aliasName: loadBalancer.dnsName,
            aliasZone: loadBalancer.zoneId,
          },
          { parent: self },
        );
      }
    }
  }

  /**
   * The URL of the load balancer. If a custom domain is set, this will be the custom
   * domain URL (eg. `https://app.example.com/`). Otherwise, it's the ALB's DNS name.
   */
  public get url(): Output<string> {
    return this._url;
  }

  /**
   * The ARN of the load balancer.
   */
  public get arn(): Output<string> {
    return this.loadBalancer.arn;
  }

  /**
   * The DNS name of the load balancer.
   */
  public get dnsName(): Output<string> {
    return this.loadBalancer.dnsName;
  }

  /**
   * The zone ID of the load balancer.
   */
  public get zoneId(): Output<string> {
    return this.loadBalancer.zoneId;
  }

  /**
   * The security group ID of the load balancer.
   */
  public get securityGroupId(): Output<string> {
    return this.securityGroup.id;
  }

  /**
   * The underlying resources this component creates.
   */
  public get nodes() {
    return {
      /**
       * The AWS Load Balancer resource.
       */
      loadBalancer: this.loadBalancer,
      /**
       * The AWS Security Group resource.
       */
      securityGroup: this.securityGroup,
      /**
       * The AWS Listener resources, keyed by "PROTOCOLPORT" (e.g. "HTTPS443").
       */
      listeners: this.listeners,
    };
  }

  /** @internal */
  public get _certArn(): Output<string | undefined> {
    return this.certificateArn ?? output(undefined);
  }

  /** @internal */
  public get _vpc(): Output<string> {
    return this.vpcId;
  }

  /**
   * Get a specific listener by protocol and port.
   *
   * @example
   * ```ts
   * const listener = alb.getListener("https", 443);
   * ```
   */
  public getListener(
    protocol: string,
    port: number,
  ): lb.Listener {
    const key = listenerKey(protocol, port);
    if (this.listeners[key]) return this.listeners[key];

    const discovered = lb.Listener.get(
      `${this.constructorName}Listener${protocol.toUpperCase()}${port}`,
      lb.getListenerOutput({
        loadBalancerArn: this.loadBalancer.arn,
        port,
      }).arn,
      {},
      { parent: this },
    );
    this.listeners[key] = discovered;
    return discovered;
  }

  /** @internal */
  public getSSTLink() {
    return {
      properties: {
        url: this._url,
      },
    };
  }

  /**
   * Reference an existing ALB by its ARN.
   *
   * @param name The name of the component.
   * @param loadBalancerArn The ARN of the existing ALB.
   * @param opts Component resource options.
   *
   * @example
   * ```ts
   * const alb = sst.aws.Alb.get("SharedAlb", "arn:aws:elasticloadbalancing:...");
   * ```
   */
  public static get(
    name: string,
    loadBalancerArn: Input<string>,
    opts?: ComponentResourceOptions,
  ): Alb {
    return new Alb(
      name,
      { ref: true, loadBalancerArn } as unknown as AlbArgs,
      opts,
    );
  }
}

const __pulumiType = "sst:aws:Alb";
// @ts-expect-error
Alb.__pulumiType = __pulumiType;
