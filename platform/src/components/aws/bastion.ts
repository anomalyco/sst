import {
  all,
  ComponentResourceOptions,
  interpolate,
  Output,
  output,
} from "@pulumi/pulumi";
import { Component, Transform, transform } from "../component.js";
import { Input } from "../input.js";
import {
  ec2,
  getPartitionOutput,
  getRegionOutput,
  iam,
  ssm,
} from "@pulumi/aws";
import { Vpc } from "./vpc.js";
import { VisibleError } from "../error.js";
import { PrivateKey } from "@pulumi/tls";

export interface BastionArgs {
  /**
   * The VPC to launch the bastion host in.
   *
   * @example
   * Create a VPC component.
   *
   * ```js
   * const myVpc = new sst.aws.Vpc("MyVpc");
   * ```
   *
   * And pass it in.
   *
   * ```js
   * {
   *   vpc: myVpc
   * }
   * ```
   *
   * Or pass in a custom VPC configuration.
   *
   * ```js
   * {
   *   vpc: {
   *     id: "vpc-0d19d2b8ca2b268a1",
   *     routeSubnets: ["subnet-0b6a2b73896dc8c4c", "subnet-021f7e8f975b2b9c2"],
   *     subnet: "subnet-0b6a2b73896dc8c4c"
   *   }
   * }
   * ```
   *
   * When using SSH mode, you need to provide a public subnet for the `subnet` arg. For SSM, this can be public or nat enabled (required for ssm agent to function).
   *
   * :::note
   * A security group ingress rule will be created which allows internet access over port 22. It is recommended you use the ssm mode which does not require opening port 22 or having a public instance
   * :::
   *
   */
  vpc:
    | Vpc
    | Input<{
        /**
         * The ID of the VPC.
         */
        id: Input<string>;
        /**
         * A list of subnet IDs in the VPC. Traffic to these subnets will be routed
         * through the bastion.
         */
        routeSubnets: Input<Input<string>[]>;
        /**
         * The subnet to launch the bastion host in. When in SSH mode, this must be a public subnet.
         * In SSM mode, this needs to be subnet with an egress route to the internet; either a public subnet or a private subnet with a nat gateway.
         * This is required for the SSM agent to function.
         */
        subnet: Input<string>;
      }>;
  /**
   * Enable SSM mode for the bastion host.
   *
   * Use SSM manager to create a tunneled ssm session.
   * This is more secure than SSH as it doesn't require any open ports or a public instance.
   *
   * @default false
   * @example
   * ```ts
   * {
   *   ssm: true
   * }
   * ```
   */
  ssm?: Input<boolean>;
  /**
   * Provide an existing IAM instance profile to use for the bastion host.
   *
   * By default, the component creates a new instance profile with the
   * `AmazonSSMManagedInstanceCore` managed policy attached.
   *
   * @example
   * ```ts
   * {
   *   instanceProfile: "my-instance-profile-name"
   * }
   * ```
   */
  instanceProfile?: Input<string>;
  /**
   * [Transform](/docs/components#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the EC2 security group for the bastion host.
     */
    securityGroup?: Transform<ec2.SecurityGroupArgs>;
    /**
     * Transform the EC2 instance for the bastion host.
     */
    instance?: Transform<ec2.InstanceArgs>;
  };
}

interface BastionRef {
  ref: boolean;
  instanceId: Input<string>;
}

/**
 * The `Bastion` component lets you add a bastion host to a VPC for secure access to
 * private resources. This standalone component is similar to the VPC component bastion, but allows you to tunnel to a non-sst vpc and also has a more secure SSM mode.
 *
 * By default, the bastion uses SSH for tunneling. Which is the same behaviour as the VPC component bastion.
 * This new component however also has an opt-in SSM mode, this doesn't require opening
 * port 22, having a public IP, or managing SSH keys. This mode is recommended for teams which have strict network security rules. You can read more about SSM port forwarding [here](https://aws.amazon.com/blogs/aws/new-port-forwarding-using-aws-system-manager-sessions-manager/)
 *
 * :::note
 * SSM mode requires the [AWS Session Manager Plugin](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html) to be installed on your local machine.
 * :::
 *
 * SSH mode doesn't require the Session Manager Plugin, but it opens port 22 to the
 * internet and requires a public subnet.
 *
 * @example
 *
 * #### Create a bastion with an SST VPC
 *
 * ```ts title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 * const bastion = new sst.aws.Bastion("MyBastion", { vpc });
 * ```
 *
 * #### Create a bastion with SSM mode
 *
 * ```ts title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 * const bastion = new sst.aws.Bastion("MyBastion", {
 *   vpc,
 *   ssm: true
 * });
 * ```
 *
 * #### Create a bastion with a custom VPC
 *
 * If you have an existing VPC that was not created with SST, you can still use the Bastion
 * component by providing the VPC ID and subnet IDs.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Bastion("MyBastion", {
 *   vpc: {
 *     id: "vpc-0d19d2b8ca2b268a1",
 *     routeSubnets: ["subnet-0b6a2b73896dc8c4c", "subnet-021f7e8f975b2b9c2"],
 *     subnet: "subnet-0b6a2b73896dc8c4c" // must have route to the internet
 *   },
 *   ssm: true
 * });
 * ```
 *
 * For SSH mode with a custom VPC, you also need to specify a public subnet:
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Bastion("MyBastion", {
 *   vpc: {
 *     id: "vpc-0d19d2b8ca2b268a1",
 *     routeSubnets: ["subnet-0b6a2b73896dc8c4c", "subnet-021f7e8f975b2b9c2"],
 *     subnet: "subnet-0b6a2b73896dc8c4c" // must be public, i.e. have route from/to the internet
 *   }
 * });
 * ```
 *
 * ---
 *
 * ### Cost
 *
 * The bastion host uses a `t4g.nano` instance which costs about $3/month.
 *
 */
export class Bastion extends Component {
  private _instance: Output<ec2.Instance>;
  private _mode: Output<string>;
  private _cidrRange: Output<string[]>;
  private _privateKeyValue: Output<string | undefined>;

  constructor(
    name: string,
    args: BastionArgs,
    opts?: ComponentResourceOptions,
  ) {
    super(__pulumiType, name, args, opts);

    const parent = this;
    const self = this;

    if (args && "ref" in args) {
      const ref = reference();
      this._instance = ref.instance;
      this._mode = ref.mode;
      this._privateKeyValue = ref.privateKeyValue;
      this._cidrRange = ref.cidrRange;
      registerOutputs();
      return;
    }

    const partition = getPartitionOutput({}, { parent }).partition;
    const mode = output(args.ssm ? "ssm" : "ssh");

    const vpc = normalizeVpc();
    const keyPairResult = createKeyPair();
    const securityGroup = createSecurityGroup();
    const instanceProfile = createInstanceProfile();
    const instance = createInstance();

    this._instance = instance;
    this._mode = mode;
    this._privateKeyValue = output(keyPairResult?.privateKeyValue);
    this._cidrRange = vpc.apply((vpc) =>
      all(
        vpc.routeSubnets.map((id) =>
          ec2.getSubnetOutput({ id }, { parent }).apply((s) => s.cidrBlock),
        ),
      ),
    );

    registerOutputs();

    function reference() {
      const ref = args as unknown as BastionRef;
      const instanceId = output(ref.instanceId);

      const instance = ec2.Instance.get(
        `${name}Instance`,
        instanceId,
        undefined,
        { parent },
      );

      const mode = instance.tags.apply((tags) => {
        if (!tags) return "ssh";
        return tags["sst:bastion-mode"] ?? "ssh";
      });

      return mode.apply((mode) => {
        const subnetData = ec2.getSubnetOutput(
          { id: instance.subnetId },
          { parent },
        );

        const vpcId = subnetData.apply((s) => s.vpcId);
        const vpcSubnets = vpcId.apply((vpcId) =>
          ec2.getSubnetsOutput(
            { filters: [{ name: "vpc-id", values: [vpcId] }] },
            { parent },
          ),
        );

        const subnets = vpcSubnets.apply((s) =>
          all(
            s.ids.map((id) =>
              ec2.getSubnetOutput({ id }, { parent }).apply((s) => s.cidrBlock),
            ),
          ),
        );

        if (mode === "ssm") {
          return {
            instance: output(instance),
            cidrRange: subnets,
            privateKeyValue: undefined,
            mode,
          };
        }

        const privateKeyValue = vpcId.apply((vpcId) => {
          const param = ssm.Parameter.get(
            `${name}PrivateKeyValue`,
            `/sst/bastionv2/${vpcId}/private-key-value`,
            undefined,
            { parent },
          );
          return param.value;
        });
        return {
          instance: output(instance),
          cidrRange: subnets,
          privateKeyValue,
          mode,
        };
      });
    }

    function normalizeVpc() {
      // "vpc" is a Vpc component
      if (args.vpc instanceof Vpc) {
        return all([
          args.vpc.id,
          args.vpc.privateSubnets,
          args.vpc.publicSubnets,
        ]).apply(([id, privateSubnets, publicSubnets]) => ({
          id,
          routeSubnets: [...publicSubnets, ...privateSubnets],
          subnet: publicSubnets[0],
        }));
      }

      // "vpc" is object
      return output(args.vpc).apply((vpc) => {
        if (!vpc.id) {
          throw new VisibleError(
            `Missing "vpc.id" for the "${name}" Bastion component.`,
          );
        }
        if (!vpc.routeSubnets?.length) {
          throw new VisibleError(
            `Missing "vpc.subnets" for the "${name}" Bastion component. At least one subnet is required.`,
          );
        }
        return {
          id: vpc.id,
          routeSubnets: vpc.routeSubnets,
          subnet: vpc.subnet,
        };
      });
    }

    function createKeyPair():
      | {
          keyPair: ec2.KeyPair;
          privateKeyValue: Output<string>;
        }
      | undefined {
      if (args.ssm === true) {
        return undefined;
      }

      const tlsPrivateKey = new PrivateKey(
        `${name}TlsPrivateKey`,
        {
          algorithm: "RSA",
          rsaBits: 4096,
        },
        { parent },
      );

      new ssm.Parameter(
        `${name}PrivateKeyValue`,
        {
          name: vpc.apply((v) => `/sst/bastionv2/${v.id}/private-key-value`),
          description: "Bastion host private key",
          type: ssm.ParameterType.SecureString,
          value: tlsPrivateKey.privateKeyOpenssh,
        },
        { parent },
      );

      const keyPair = new ec2.KeyPair(
        `${name}KeyPair`,
        {
          publicKey: tlsPrivateKey.publicKeyOpenssh,
        },
        { parent },
      );

      return { keyPair, privateKeyValue: tlsPrivateKey.privateKeyOpenssh };
    }

    function createSecurityGroup() {
      const ingress = !args.ssm
        ? [
            {
              protocol: "tcp",
              fromPort: 22,
              toPort: 22,
              cidrBlocks: ["0.0.0.0/0"],
            },
          ]
        : [];

      return new ec2.SecurityGroup(
        ...transform(
          args.transform?.securityGroup,
          `${name}SecurityGroup`,
          {
            vpcId: vpc.apply((v) => v.id),
            ingress,
            egress: [
              {
                protocol: "-1",
                fromPort: 0,
                toPort: 0,
                cidrBlocks: ["0.0.0.0/0"],
              },
            ],
          },
          { parent },
        ),
      );
    }

    function createInstanceProfile() {
      return output(args.instanceProfile).apply((instanceProfileName) => {
        if (instanceProfileName) {
          if (instanceProfileName.startsWith("arn:")) {
            throw new VisibleError(
              "Bastion instance profile must be a name, not an ARN.",
            );
          }

          return iam.InstanceProfile.get(
            `${name}InstanceProfile`,
            instanceProfileName,
            {},
            { parent },
          );
        }

        const role = new iam.Role(
          `${name}Role`,
          {
            assumeRolePolicy: iam.getPolicyDocumentOutput({
              statements: [
                {
                  actions: ["sts:AssumeRole"],
                  principals: [
                    {
                      type: "Service",
                      identifiers: ["ec2.amazonaws.com"],
                    },
                  ],
                },
              ],
            }).json,
            managedPolicyArns: [
              interpolate`arn:${partition}:iam::aws:policy/AmazonSSMManagedInstanceCore`,
            ],
          },
          { parent },
        );

        return new iam.InstanceProfile(
          `${name}InstanceProfile`,
          { role: role.name },
          { parent },
        );
      });
    }

    function createInstance() {
      const ami = ec2.getAmiOutput(
        {
          owners: ["amazon"],
          filters: [
            {
              name: "name",
              // The AMI has the SSM agent pre-installed
              values: ["al2023-ami-20*"],
            },
            {
              name: "architecture",
              values: ["arm64"],
            },
          ],
          mostRecent: true,
        },
        { parent },
      );

      return all([vpc, instanceProfile]).apply(([vpc, instanceProfile]) => {
        return new ec2.Instance(
          ...transform(
            args.transform?.instance,
            `${name}Instance`,
            {
              instanceType: "t4g.nano",
              ami: ami.id,
              subnetId: vpc.subnet,
              vpcSecurityGroupIds: [securityGroup.id],
              iamInstanceProfile: instanceProfile.name,
              keyName: keyPairResult?.keyPair.keyName,
              associatePublicIpAddress: true,
              tags: {
                "sst:bastion-mode": args.ssm ? "ssm" : "ssh",
              },
            },
            { parent },
          ),
        );
      });
    }

    function registerOutputs() {
      const region = getRegionOutput({}, { parent }).region;
      self.registerOutputs({
        _tunnel: all([
          self._instance,
          self._cidrRange,
          self._mode,
          region,
          self._privateKeyValue,
        ]).apply(([instance, cidrRange, mode, region, privateKeyValue]) => ({
          ip: mode === "ssh" ? instance.publicIp : undefined,
          username: "ec2-user",
          privateKey: privateKeyValue,
          instanceId: instance.id,
          region: region,
          subnets: cidrRange,
          mode,
        })),
      });
    }
  }

  /**
   * The public IP address of the bastion host. Only available in SSH mode.
   */
  public get publicIp() {
    return this._instance.publicIp;
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The Amazon EC2 instance for the bastion host.
       */
      instance: this._instance,
    };
  }

  /**
   * Reference an existing Bastion component with the given instance ID. This is useful when you
   * create a Bastion in one stage and want to share it in another stage.
   *
   * @param name The name of the component.
   * @param instanceId The ID of the EC2 instance.
   * @param opts Resource options.
   *
   * @example
   * Imagine you create a bastion in the `dev` stage. And in your personal stage, `frank`,
   * instead of creating a new bastion, you want to reuse the one from `dev`.
   *
   * ```ts title="sst.config.ts"
   * const bastion = $app.stage === "frank"
   *   ? sst.aws.Bastion.get("MyBastion", "i-1234567890abcdef0")
   *   : new sst.aws.Bastion("MyBastion", { vpc });
   * ```
   */
  public static get(
    name: string,
    instanceId: Input<string>,
    opts?: ComponentResourceOptions,
  ) {
    return new Bastion(
      name,
      {
        ref: true,
        instanceId,
      } as unknown as BastionArgs,
      opts,
    );
  }
}

const __pulumiType = "sst:aws:Bastion";
// @ts-expect-error
Bastion.__pulumiType = __pulumiType;
