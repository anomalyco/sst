import fs from "fs";
import { all, ComponentResourceOptions, Input, interpolate, Output, secret } from "@pulumi/pulumi";
import { Component, Transform, transform } from "../component.js";
import { Link } from "../link.js";
import { getRegionOutput } from "@pulumi/aws";
import { ecr } from "@pulumi/aws";
import { Semaphore } from "../../util/semaphore.js";
import { bootstrap } from "./helpers/bootstrap.js";
import {
  Image as PulumiDockerImage,
  ImageArgs as PulumiDockerImageArgs,
} from "@pulumi/docker-build";
import path from "path";

// Extracted from `@pulumi/docker-build` `Platform` - unhandled by doc generator.
export type Platform = "darwin/386" | "darwin/amd64" | "darwin/arm" | "darwin/arm64" | "dragonfly/amd64" | "freebsd/386" | "freebsd/amd64" | "freebsd/arm" | "linux/386" | "linux/amd64" | "linux/arm" | "linux/arm64" | "linux/mips64" | "linux/mips64le" | "linux/ppc64le" | "linux/riscv64" | "linux/s390x" | "netbsd/386" | "netbsd/amd64" | "netbsd/arm" | "openbsd/386" | "openbsd/amd64" | "openbsd/arm" | "plan9/386" | "plan9/amd64" | "solaris/amd64" | "windows/386" | "windows/amd64"

// Extracted from `@pulumi/docker-build` `CompressionType` - unhandled by doc generator.
export type CompressionType = "zstd" | "gzip" | "estargz"

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
   * Specify the folder of the Docker build context:
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
   * Specify different Dockerfile:
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
  secrets?: Input<Record<string, string>>;
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
   * Specify compression type.
   */
  compression?: {
    /**
     * Specify compression algorithm used:
     * - `zstd` - smallest compression.
     * - `gzip` - best compatibility.
     * - `estargz` - fastest pull.
     * 
     * @default `"zstd"`
     */
    type: Input<CompressionType>
  }
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

/**
 * The `Image` component builds docker images and uploads them to [AWS ECR (Elastic Container Registry)](https://aws.amazon.com/ecr/).
 *
 * @example
 * #### Minimal example
 * Create `Dockerfile` and `sst.config.ts` in root directory.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Image("MyImage", {});
 * ```
 * 
 * @example
 * #### Different dockerfile and context
 * [Minimal example](#minimal-example) setup with `./app/Dockerfile.function`.
 *
 * Note:
 * - By default, context is the root directory where `sst.config.ts` is located.
 * - By default, dockerfile is `Dockerfile`.
 * 
 * ```ts {2,3} title="sst.config.ts"
 * new sst.aws.Image("MyImage", {
 *   context: './app',
 *   dockerfile: 'Dockerfile.function'
 * });
 * ```
 * 
 * @example
 * #### Smaller compression example
 * [Minimal example](#minimal-example) setup, optimised for smaller image.
 * This setup reduces both storage and pull speed.
 * 
 * ```ts {2-5} title="sst.config.ts"
 * new sst.aws.Image("MyImage", {
 *   compression: {
 *     type: 'zstd',
 *     level: 9
 *   }
 * });
 * ```
 * 
 * @example
 * #### Faster pull speed example
 * [Minimal example](#minimal-example) setup, optimised for faster pulls.
 * This setup reduces pull speed by indexing image in chunks.
 * 
 * Alternatively, consider `"gzip"` compression with [SOCI indexing](https://github.com/awslabs/soci-snapshotter) - which can be [automatically created on AWS](https://github.com/awslabs/cfn-ecr-aws-soci-index-builder).
 *
 * ```ts {2-4} title="sst.config.ts"
 * new sst.aws.Image("MyImage", {
 *   compression: {
 *     type: 'estargz',
 *   }
 * });
 * ```
 */
export class Image extends Component implements Link.Linkable {
  private _uri: Output<string>;

  constructor(name: string, args: ImageArgs, opts?: ComponentResourceOptions) {
    const componentName = `${name}Image`
    super(__pulumiType, componentName, args, opts);

    const region = getRegionOutput({}, opts).region
    const bootstrapData = region.apply((region) => bootstrap.forRegion(region));

    const wrappedUri = all([args, bootstrapData]).apply(
      async ([args, bootstrapData]) => {
        await limiter.acquire(componentName);
        // Wait for the all args values to be resolved before acquiring the semaphore

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

        function normalizeCompressionType() {
          if (!args.compression?.type) return 'zstd'
          return args.compression.type
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
                    opts,
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
                    compression: normalizeCompressionType(),
                  },
                },
              ],
              push: true,
              ...(process.env.BUILDX_BUILDER
                ? { builder: { name: process.env.BUILDX_BUILDER } }
                : {}),
            },
            opts,
          ),
        );

        const  uri = interpolate`${bootstrapData.assetEcrUrl}@${image.digest}`
        uri.apply(() => limiter.release());
        return uri
      },
    );

    this._uri = wrappedUri.apply(uri => uri)
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
        uri: this.uri,
      },
    };
  }
}

const __pulumiType = "sst:aws:Image";
// @ts-expect-error
Image.__pulumiType = __pulumiType;
