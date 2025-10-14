import fs from "fs";
import path from "path";
import {
  ComponentResourceOptions,
  Input,
  Output,
  all,
  interpolate,
  output,
  secret,
} from "@pulumi/pulumi";
import { Component, Prettify, Transform, transform } from "../component.js";
import { Link } from "../link.js";
import { DevCommand } from "../experimental/dev-command.js";
import { Input as PulumiInput } from "../input.js";
import { prefixName, physicalName } from "../naming.js";
import { VisibleError } from "../error.js";
import {
  ecr,
  getCallerIdentityOutput,
  getPartitionOutput,
  getRegionOutput,
  iam,
} from "@pulumi/aws";
import * as awsNative from "@pulumi/aws-native";
import { imageBuilder } from "./helpers/container-builder.js";
import { bootstrap } from "./helpers/bootstrap.js";
import { Permission } from "./permission.js";
import { Platform } from "@pulumi/docker-build";

export interface AgentRuntimeArgs {
  /**
   * Configure the Docker build command for building the image.
   *
   * Prior to building the image, SST will automatically add the `.sst` directory
   * to the `.dockerignore` if not already present.
   *
   * @default Build from Dockerfile in the root directory
   * @example
   *
   * ```js
   * {
   *   image: {
   *     context: "./agent",
   *     dockerfile: "Dockerfile",
   *     args: {
   *       MY_VAR: "value"
   *     }
   *   }
   * }
   * ```
   */
  image?: Input<{
    /**
     * The path to the [Docker build context](https://docs.docker.com/build/building/context/#local-context).
     * The path is relative to your project's `sst.config.ts`.
     * @default `"."`
     * @example
     *
     * To change where the Docker build context is located.
     *
     * ```js
     * {
     *   context: "./agent"
     * }
     * ```
     */
    context?: Input<string>;
    /**
     * The path to the [Dockerfile](https://docs.docker.com/reference/cli/docker/image/build/#file).
     * The path is relative to the build `context`.
     * @default `"Dockerfile"`
     * @example
     * To use a different Dockerfile.
     * ```js
     * {
     *   dockerfile: "Dockerfile.prod"
     * }
     * ```
     */
    dockerfile?: Input<string>;
    /**
     * Key-value pairs of [build args](https://docs.docker.com/build/guide/build-args/) to pass to the Docker build command.
     * @example
     * ```js
     * {
     *   args: {
     *     MY_VAR: "value"
     *   }
     * }
     * ```
     */
    args?: Input<Record<string, Input<string>>>;
  }>;
  /**
   * Configure how this component works in `sst dev`.
   *
   * :::note
   * In `sst dev` your agent runtime is not deployed.
   * :::
   *
   * By default, your agent runtime is not deployed in `sst dev`. Instead, you can set the
   * `dev.command` and it'll be started locally in a separate tab in the
   * `sst dev` multiplexer. Read more about [`sst dev`](/docs/reference/cli/#dev).
   *
   * This makes it so that the container doesn't have to be redeployed on every change.
   *
   * @example
   *
   * ```js
   * {
   *   dev: {
   *     command: "npm run dev",
   *     directory: "./agent"
   *   }
   * }
   * ```
   */
  dev?: {
    /**
     * The command that `sst dev` runs to start this in dev mode. This is the command you run
     * when you want to run your agent locally.
     * @example
     * ```js
     * {
     *   command: "npm run dev"
     * }
     * ```
     * For Python agents:
     * ```js
     * {
     *   command: "python agent.py"
     * }
     * ```
     */
    command?: Input<string>;
    /**
     * Configure if you want to automatically start this when `sst dev` starts. You can still
     * start it manually later.
     * @default `true`
     */
    autostart?: Input<boolean>;
    /**
     * Change the directory from where the `command` is run.
     * @default Uses the `image.context` path
     */
    directory?: Input<string>;
    /**
     * The title of the tab in the multiplexer.
     * @default The name of the component
     */
    title?: Input<string>;
  };
  /**
   * The name for the agent runtime.
   * @example
   * ```js
   * {
   *   agentRuntimeName: "customer-support-agent"
   * }
   * ```
   */
  agentRuntimeName?: Input<string>;
  /**
   * Description of the agent runtime.
   * @example
   * ```js
   * {
   *   description: "AI agent for customer support"
   * }
   * ```
   */
  description?: Input<string>;
  /**
   * Environment variables to pass to the agent runtime container.
   * @example
   * ```js
   * {
   *   environmentVariables: {
   *     MODEL_NAME: "claude-3-5-sonnet",
   *     TEMPERATURE: "0.7"
   *   }
   * }
   * ```
   */
  environmentVariables?: Input<Record<string, Input<string>>>;
  /**
   * The protocol configuration for the agent runtime.
   * @default `"MCP"`
   * @example
   * ```js
   * {
   *   protocolConfiguration: "HTTP"
   * }
   * ```
   */
  protocolConfiguration?: Input<"MCP" | "HTTP">;
  /**
   * Network access configuration for the agent runtime.
   * @default `{ networkMode: "PUBLIC" }`
   * @example
   * ```js
   * {
   *   networkConfiguration: {
   *     networkMode: "PUBLIC"
   *   }
   * }
   * ```
   */
  networkConfiguration?: Input<{
    /**
     * The network mode for the agent runtime.
     * @default `"PUBLIC"`
     */
    networkMode: Input<"PUBLIC">;
  }>;
  /**
   * Authorizer configuration for the agent runtime using Custom JWT.
   * @example
   * ```js
   * {
   *   authorizerConfiguration: {
   *     customJwtAuthorizer: {
   *       discoveryUrl: "https://cognito-idp.us-east-1.amazonaws.com/us-east-1_xxxxx/.well-known/openid-configuration",
   *       allowedAudience: ["my-app-client-id"],
   *       allowedClients: ["client1", "client2"]
   *     }
   *   }
   * }
   * ```
   */
  authorizerConfiguration?: Input<{
    /**
     * Custom JWT authorizer configuration.
     */
    customJwtAuthorizer?: Input<{
      /**
       * The OIDC discovery URL for JWT validation.
       */
      discoveryUrl: Input<string>;
      /**
       * List of allowed audience values in the JWT token.
       */
      allowedAudience?: Input<Input<string>[]>;
      /**
       * List of allowed client IDs in the JWT token.
       */
      allowedClients?: Input<Input<string>[]>;
    }>;
  }>;
  /**
   * The ARN of an IAM role for the agent runtime. If not provided, a role will be created.
   * @example
   * ```js
   * {
   *   role: "arn:aws:iam::123456789012:role/my-agent-role"
   * }
   * ```
   */
  role?: Input<string>;
  /**
   * Permissions and the resources that the agent runtime can access.
   * @example
   * Allow the agent runtime to access Bedrock models.
   * ```js
   * {
   *   permissions: [
   *     {
   *       actions: ["bedrock:InvokeModel"],
   *       resources: ["*"]
   *     }
   *   ]
   * }
   * ```
   */
  permissions?: {
    /**
     * The [IAM actions](https://docs.aws.amazon.com/service-authorization/latest/reference/reference_policies_actions-resources-contextkeys.html#actions_table) that can be performed.
     * @example
     * ```js
     * {
     *   actions: ["bedrock:*"]
     * }
     * ```
     */
    actions: string[];
    /**
     * The resources specified using the [IAM ARN format](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_identifiers.html).
     * @example
     * ```js
     * {
     *   resources: ["arn:aws:bedrock:us-east-1::foundation-model/*"]
     * }
     * ```
     */
    resources: Input<Input<string>[]>;
  }[];
  /**
   * [Link resources](/docs/linking/) to your agent runtime. This will:
   *
   * 1. Grant the permissions needed to access the resources.
   * 2. Allow you to access it in your agent using the [SDK](/docs/reference/sdk/).
   *
   * @example
   *
   * Takes a list of resources to link to the agent runtime.
   *
   * ```js
   * {
   *   link: [bucket, stripeKey]
   * }
   * ```
   */
  link?: PulumiInput<any[]>;
  /**
   * [Transform](/docs/components#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the Bedrock Agent Core Runtime resource.
     */
    runtime?: Transform<awsNative.bedrockagentcore.RuntimeArgs>;
    /**
     * Transform the IAM role used by the agent runtime.
     */
    role?: Transform<iam.RoleArgs>;
  };
}

/**
 * The `AgentRuntime` component lets you deploy AI agents to AWS using Bedrock Agent Core Runtime.
 *
 * @example
 *
 * #### Minimal example
 *
 * Deploy an agent with a Dockerfile.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.AgentRuntime("MyAgent", {
 *   agentRuntimeName: "my-agent",
 *   image: {
 *     context: "./agent"
 *   },
 *   dev: {
 *     command: "npm run dev"
 *   }
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to the agent. This will grant permissions
 * to access the resources and allow you to access them in your agent.
 *
 * ```ts {5} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.AgentRuntime("MyAgent", {
 *   agentRuntimeName: "my-agent",
 *   link: [bucket]
 * });
 * ```
 *
 * #### Add permissions
 *
 * Add permissions to access AWS resources.
 *
 * ```ts {3-7} title="sst.config.ts"
 * new sst.aws.AgentRuntime("MyAgent", {
 *   agentRuntimeName: "my-agent",
 *   permissions: [
 *     {
 *       actions: ["bedrock:InvokeModel"],
 *       resources: ["*"]
 *     }
 *   ]
 * });
 * ```
 */
export class AgentRuntime extends Component implements Link.Linkable {
  private runtime?: awsNative.bedrockagentcore.Runtime;
  private role!: iam.Role;
  private _workloadIdentityArn?: Output<string>;

  constructor(
    name: string,
    args: AgentRuntimeArgs = {},
    opts?: ComponentResourceOptions,
  ) {
    super(__pulumiType, name, args, opts);

    const parent = this;
    const region = getRegionOutput({}, opts).name;
    const partition = getPartitionOutput({}, opts).partition;
    const accountId = getCallerIdentityOutput({}, opts).accountId;
    const bootstrapData = region.apply((region) => bootstrap.forRegion(region));
    
    // Create aws-native provider with region configuration
    const awsNativeProvider = new awsNative.Provider(
      `${name}AwsNativeProvider`,
      {
        region: region as any,
      },
      { parent },
    );

    const imageArgs = normalizeImage();
    const dev = normalizeDev();
    const linkData = buildLinkData();
    const linkPermissions = buildLinkPermissions();

    if ($dev) {
      registerDevCommand();
      registerReceiver();
      return;
    }

    const role = createRole();
    this.role = role;

    const image = createImage();
    const runtime = createRuntime();
    this.runtime = runtime;
    this._workloadIdentityArn = runtime.workloadIdentityDetails.apply(
      (details) => details?.workloadIdentityArn,
    );

    registerReceiver();

    function normalizeImage() {
      return output(args.image).apply((image) => ({
        context: image?.context ?? ".",
        dockerfile: image?.dockerfile ?? "Dockerfile",
        args: image?.args ?? {},
      }));
    }

    function normalizeDev() {
      if (!args.dev) return;

      return output(args.dev).apply((dev) => ({
        command: dev?.command,
        autostart: dev?.autostart ?? true,
        directory: dev?.directory,
        title: dev?.title ?? name,
      }));
    }

    function buildLinkData() {
      return output(args.link || []).apply((links) => Link.build(links));
    }

    function buildLinkPermissions() {
      return Link.getInclude<Permission>("aws.permission", args.link);
    }

    function registerDevCommand() {
      if (!dev) return;

      all([dev, imageArgs, linkData]).apply(([dev, imageArgs, linkData]) => {
        const directory = dev.directory ?? imageArgs.context;

        new DevCommand(
          `${name}Dev`,
          {
            dev: {
              title: dev.title,
              autostart: dev.autostart,
              command: dev.command,
              directory,
            },
            link: args.link,
            environment: output(args.environmentVariables).apply(
              (env) => env ?? {},
            ),
          },
          { parent },
        );
      });
    }

    function createRole() {
      return new iam.Role(
        ...transform(
          args.transform?.role,
          `${name}Role`,
          {
            assumeRolePolicy: JSON.stringify({
              Version: "2012-10-17",
              Statement: [
                {
                  Effect: "Allow",
                  Action: "sts:AssumeRole",
                  Principal: {
                    Service: "bedrock-agentcore.amazonaws.com",
                  },
                },
              ],
            }),
            managedPolicyArns: [
              interpolate`arn:${partition}:iam::aws:policy/CloudWatchLogsFullAccess`,
            ],
            inlinePolicies: all([linkPermissions, args.permissions]).apply(
              ([linkPermissions, permissions]) => {
                const statements: any[] = [];

                // Add ECR permissions for Bedrock Agent Core to pull images
                statements.push({
                  actions: [
                    "ecr:GetAuthorizationToken",
                    "ecr:BatchCheckLayerAvailability",
                    "ecr:GetDownloadUrlForLayer",
                    "ecr:BatchGetImage",
                  ],
                  resources: ["*"],
                });

                // Add link permissions
                if (linkPermissions) {
                  statements.push(...linkPermissions);
                }

                // Add custom permissions
                if (permissions) {
                  statements.push(
                    ...permissions.map((p) => ({
                      actions: p.actions,
                      resources: output(p.resources),
                    })),
                  );
                }

                const policyJson = iam.getPolicyDocumentOutput({
                  statements,
                }).json;

                return [
                  {
                    name: "inline",
                    policy: policyJson,
                  },
                ];
              },
            ),
          },
          { parent },
        ),
      );
    }

    function createImage() {
      return all([imageArgs, bootstrapData]).apply(
        async ([imageArgs, bootstrapData]) => {
          const contextPath = path.join($cli.paths.root, imageArgs.context);
          const dockerfile = imageArgs.dockerfile;
          const dockerfilePath = path.join(contextPath, dockerfile);

          // Validate Dockerfile exists
          if (!fs.existsSync(dockerfilePath)) {
            throw new VisibleError(
              `Could not find Dockerfile at "${dockerfilePath}" for AgentRuntime "${name}". Please create a Dockerfile or specify the correct path using the "image.dockerfile" argument.`,
            );
          }

          const dockerIgnorePath = fs.existsSync(
            path.join(contextPath, `${dockerfile}.dockerignore`),
          )
            ? path.join(contextPath, `${dockerfile}.dockerignore`)
            : path.join(contextPath, ".dockerignore");

          // Add .sst to .dockerignore if not exist
          const lines = fs.existsSync(dockerIgnorePath)
            ? fs.readFileSync(dockerIgnorePath).toString().split("\n")
            : [];
          if (!lines.find((line) => line === ".sst")) {
            fs.writeFileSync(
              dockerIgnorePath,
              [...lines, "", "# sst", ".sst"].join("\n"),
            );
          }

          // Get ECR auth token
          const authToken = ecr.getAuthorizationTokenOutput({
            registryId: bootstrapData.assetEcrRegistryId,
          });

          // Build image
          const image = await imageBuilder(
            `${name}Image`,
            {
              context: {
                location: contextPath,
              },
              dockerfile: {
                location: dockerfilePath,
              },
              platforms: [Platform.Linux_arm64],
              buildArgs: imageArgs.args,
              push: true,
              registries: [
                authToken.apply((authToken) => ({
                  address: authToken.proxyEndpoint,
                  username: authToken.userName,
                  password: secret(authToken.password),
                })),
              ],
              tags: [interpolate`${bootstrapData.assetEcrUrl}:${name}`],
              cacheFrom: [
                {
                  registry: {
                    ref: interpolate`${bootstrapData.assetEcrUrl}:${name}-cache`,
                  },
                },
              ],
              cacheTo: [
                {
                  registry: {
                    ref: interpolate`${bootstrapData.assetEcrUrl}:${name}-cache`,
                    imageManifest: true,
                    ociMediaTypes: true,
                    mode: "max",
                  },
                },
              ],
            },
            { parent },
          );

          return image;
        },
      );
    }

    function createRuntime() {
      // Generate the runtime name using SST's naming convention
      // AWS Bedrock Agent Runtime names must match: [a-zA-Z][a-zA-Z0-9_]{0,47}
      // - Start with a letter
      // - Only letters, numbers, and underscores (no hyphens!)
      // - Max 48 characters
      const runtimeName =
        args.agentRuntimeName ??
        physicalName(48, name)
          .replace(/[^a-zA-Z0-9_]/g, "_") // Replace invalid chars with underscore
          .replace(/^[^a-zA-Z]+/, ""); // Ensure it starts with a letter

      return new awsNative.bedrockagentcore.Runtime(
        ...transform(
          args.transform?.runtime,
          `${name}Runtime`,
          {
            agentRuntimeName: runtimeName,
            description: args.description,
            agentRuntimeArtifact: {
              containerConfiguration: {
                containerUri: interpolate`${bootstrapData.assetEcrUrl}:${name}`,
              },
            },
            roleArn: args.role ?? role.arn,
            networkConfiguration: output(args.networkConfiguration).apply(
              (net) => ({
                networkMode: net?.networkMode ?? "PUBLIC",
              }),
            ),
            protocolConfiguration: args.protocolConfiguration ?? "MCP",
            environmentVariables: all([
              args.environmentVariables,
              linkData,
            ]).apply(([envVars, linkData]) => {
              const env: Record<string, string> = {};

              // Add user-defined environment variables
              if (envVars) {
                Object.entries(envVars).forEach(([key, value]) => {
                  // Since we're already inside an apply, the value is resolved
                  env[key] = String(value);
                });
              }

              // Add link data as environment variables
              env.SST_RESOURCE_App = JSON.stringify({
                name: $app.name,
                stage: $app.stage,
              });

              linkData.forEach((link) => {
                env[`SST_RESOURCE_${link.name}`] = JSON.stringify(
                  link.properties,
                );
              });

              return env;
            }),
            authorizerConfiguration: args.authorizerConfiguration
              ? output(args.authorizerConfiguration).apply((auth) => ({
                  customJwtAuthorizer: auth.customJwtAuthorizer
                    ? {
                        discoveryUrl: output(
                          auth.customJwtAuthorizer.discoveryUrl,
                        ),
                        allowedAudience:
                          auth.customJwtAuthorizer.allowedAudience,
                        allowedClients:
                          auth.customJwtAuthorizer.allowedClients,
                      }
                    : undefined,
                }))
              : undefined,
          },
          { parent, provider: awsNativeProvider, dependsOn: [role, image] },
        ),
      );
    }

    function registerReceiver() {
      parent.registerOutputs({
        _hint: all([parent._workloadIdentityArn]).apply(
          ([workloadIdentityArn]) => {
            return workloadIdentityArn ? workloadIdentityArn : undefined;
          },
        ),
        _receiver: {
          links: output(args.link || [])
            .apply(Link.build)
            .apply((links) => links.map((link) => link.name)),
          environment: output(args.environmentVariables || {}),
        },
      });
    }
  }

  /**
   * The ARN of the agent runtime.
   */
  public get agentRuntimeArn() {
    return this.runtime?.agentRuntimeArn;
  }

  /**
   * The ID of the agent runtime.
   */
  public get agentRuntimeId() {
    return this.runtime?.agentRuntimeId;
  }

  /**
   * The version of the agent runtime.
   */
  public get agentRuntimeVersion() {
    return this.runtime?.agentRuntimeVersion;
  }

  /**
   * The ARN of the workload identity for the agent runtime.
   */
  public get workloadIdentityArn() {
    return this._workloadIdentityArn;
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The AWS Bedrock Agent Core Runtime resource.
       */
      runtime: this.runtime,
      /**
       * The IAM role used by the agent runtime.
       */
      role: this.role,
    };
  }

  /** @internal */
  public getSSTLink() {
    return {
      properties: {
        arn: this.agentRuntimeArn,
        id: this.agentRuntimeId,
        workloadIdentityArn: this.workloadIdentityArn,
      },
    };
  }
}

const __pulumiType = "sst:aws:AgentRuntime";

