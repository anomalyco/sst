import fs from "fs";
import { all, Input, interpolate, Output, secret } from "@pulumi/pulumi";
import { Component, Transform, transform } from "../component.js";
import { Link } from "../link.js";
import { getRegionOutput } from "@pulumi/aws";
import { ecr } from "@pulumi/aws";
import { Semaphore } from "../../util/semaphore.js";
import { bootstrap } from "./helpers/bootstrap.js";
import {
  Platform,
  Image as PulumiDockerImage,
  ImageArgs as PulumiDockerImageArgs,
} from "@pulumi/docker-build";
import path from "path";

export interface ImageArgs {
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
  /**
   * The path to the [Docker build context](https://docs.docker.com/build/building/context/#local-context). The path is relative to your project's `sst.config.ts`.
   * @default `"."`
   * @example
   *
   * To change where the Docker build context is located.
   *
   * ```js
   * {
   *   context: "./app"
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
   * Set target platform(s) for the build. Defaults to the host's platform.
   *
   * Equivalent to Docker's `--platform` flag.
   */
  platforms?: Input<Platform[]>;
  /**
   * A mapping of secret names to their corresponding values.
   *
   * Unlike the Docker CLI, these can be passed by value and do not need to
   * exist on-disk or in environment variables.
   *
   * Build arguments and environment variables are persistent in the final
   * image, so you should use this for sensitive values.
   *
   * Similar to Docker's `--secret` flag.
   */
  secrets?: Input<{
    [k: string]: string;
  }>;
  /**
   * Tags to apply to the Docker image.
   * @example
   * ```js
   * {
   *   tags: ["v1.0.0", "commit-613c1b2"]
   * }
   * ```
   */
  tags?: Input<string[]>;
  /**
   * The stage to build up to in a [multi-stage Dockerfile](https://docs.docker.com/build/building/multi-stage/#stop-at-a-specific-build-stage).
   * @example
   * ```js
   * {
   *   target: "stage1"
   * }
   * ```
   */
  target?: Input<string>;
  /**
   * [Transform](/docs/components#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the Docker Image resource.
     */
    image?: Transform<PulumiDockerImageArgs>;
  };
}

const limiter = new Semaphore(
  parseInt(process.env.SST_BUILD_CONCURRENCY_CONTAINER || "1"),
);

export class Image extends Component implements Link.Linkable {
  private constructorName: string;
  private image: Output<PulumiDockerImage>;
  private _uri: Output<string>;

  constructor(name: string, args: ImageArgs, opts?: any) {
    const componentName = `${name}Image`
    super(__pulumiType, componentName, args, opts);
    this.constructorName = componentName;

    const parent = this;
    const region = getRegionOutput({}, opts).name;
    const bootstrapData = region.apply((region) => bootstrap.forRegion(region));
    // Empty uri should fail deployment if not set
    this._uri = interpolate``

    this.image = all([args, bootstrapData]).apply(
      async ([args, bootstrapData]) => {
        // Wait for the all args values to be resolved before acquiring the semaphore
        await limiter.acquire(componentName);

        const contextPath = path.join($cli.paths.root, args.context ?? ".");
        const dockerfile = args.dockerfile ?? "Dockerfile";
        const dockerfilePath = path.join(contextPath, dockerfile);

        // add .sst to .dockerignore if not exist
        const dockerIgnorePath = fs.existsSync(
          path.join(contextPath, `${dockerfile}.dockerignore`),
        )
          ? path.join(contextPath, `${dockerfile}.dockerignore`)
          : path.join(contextPath, ".dockerignore");
        const lines = fs.existsSync(dockerIgnorePath)
          ? fs.readFileSync(dockerIgnorePath).toString().split("\n")
          : [];
        if (!lines.find((line) => line === ".sst")) {
          fs.writeFileSync(
            dockerIgnorePath,
            [...lines, "", "# sst", ".sst"].join("\n"),
          );
        }

        const image = new PulumiDockerImage(
          ...transform(
            args.transform?.image,
            componentName,
            {
              context: { location: contextPath },
              dockerfile: { location: dockerfilePath },
              buildArgs: args.args,
              secrets: args.secrets,
              target: args.target,
              platforms: args.platforms,
              tags: [name, ...(args.tags ?? [])].map(
                (tag) => interpolate`${bootstrapData.assetEcrUrl}:${tag}`,
              ),
              registries: [
                ecr
                  .getAuthorizationTokenOutput(
                    {
                      registryId: bootstrapData.assetEcrRegistryId,
                    },
                    { parent },
                  )
                  .apply((authToken) => ({
                    address: authToken.proxyEndpoint,
                    password: secret(authToken.password),
                    username: authToken.userName,
                  })),
              ],
              cacheFrom: [
                {
                  registry: {
                    ref: $interpolate`${bootstrapData.assetEcrUrl}:${name}-cache`,
                  },
                },
              ],
              cacheTo: [
                {
                  registry: {
                    ref: $interpolate`${bootstrapData.assetEcrUrl}:${name}-cache`,
                    imageManifest: true,
                    ociMediaTypes: true,
                    mode: "max" as const,
                  },
                },
              ],
              push: true,
              ...(process.env.BUILDX_BUILDER
                ? { builder: { name: process.env.BUILDX_BUILDER } }
                : {}),
            },
            { parent },
          ),
        );
        this._uri = interpolate`${bootstrapData.assetEcrUrl}@${image.digest}`

        image.urn.apply(() => {
          limiter.release();
        });
        return image;
      },
    );
  }

  /**
   * SHA256 digest of the ECR image
   */
  public get digest() {
    return this.image.digest;
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The AWS ECR image.
       */
      image: this.image,
    };
  }

  /**
   * The uri of the ECR container image.
   */
  public get uri() {
    return this._uri
  }

  /** @internal */
  public getSSTLink() {
    return {
      properties: {
        name: this.constructorName,
        uri: this.uri,
      },
    };
  }
}

const __pulumiType = "sst:aws:Image";
// @ts-expect-error
Image.__pulumiType = __pulumiType;
