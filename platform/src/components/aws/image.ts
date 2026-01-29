import fs from 'fs';
import { all, Input, interpolate, output, Output, secret } from "@pulumi/pulumi";
import { Component, Transform, transform } from "../component.js";
import { Link } from "../link.js";
import { getRegionOutput } from "@pulumi/aws";
import {
  ecr,
} from "@pulumi/aws";
import { Semaphore } from "../../util/semaphore.js";
import { bootstrap } from "./helpers/bootstrap.js";
import { Platform, Image as PulumiDockerImage, ImageArgs as PulumiDockerImageArgs, } from "@pulumi/docker-build";
import path from "path";

export interface ImageArgs {
  context?: Input<string>;
  dockerfile?: Input<string>
  platforms?: Input<Platform[]>
  tags?: Input<string[]>;
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

  constructor(
    name: string,
    args: ImageArgs,
    opts?: any,
  ) {
    super(__pulumiType, name, args, opts);
    this.constructorName = name;

    const parent = this;
    const region = getRegionOutput({}, opts).name;
    const bootstrapData = region.apply((region) => bootstrap.forRegion(region));

    this.image = all([args, bootstrapData]).apply(
      async ([args, bootstrapData]) => {
        // Wait for the all args values to be resolved before acquiring the semaphore
        await limiter.acquire(name);

        const contextPath = path.join($cli.paths.root, args.context ?? '.');
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
            `${name}Image`,
            {
              context: { location: contextPath },
              dockerfile: { location: dockerfilePath },
              // buildArgs: args.args,
              // secrets: args.linkEnvs,
              // target: args.target,
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
          )
        )

        image.urn.apply(() => {
          limiter.release();
        })
        return image
      }
    );
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The AWS Lambda function.
       */
      image: this.image
    };
  }

  /**
   * The uri of the ECR container image.
   */
  public get uri() {
    return this.image.ref.apply(
      (ref) => ref?.replace(":latest", ""),
    );
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
