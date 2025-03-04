import path from "path";
import fs from "fs";
import { globSync } from "glob";
import crypto from "crypto";
import type { Loader } from "esbuild";
import {
  Output,
  Unwrap,
  output,
  all,
  interpolate,
  ComponentResource,
} from "@pulumi/pulumi";
import { Cdn, CdnArgs } from "./cdn.js";
import { Function, FunctionArgs } from "./function.js";
import { Bucket, BucketArgs } from "./bucket.js";
import { BucketFile, BucketFiles } from "./providers/bucket-files.js";
import { logicalName, physicalName } from "../naming.js";
import { Input } from "../input.js";
import { transform, type Transform } from "../component.js";
import { VisibleError } from "../error.js";
import { Cron } from "./cron.js";
import { BaseSiteFileOptions, getContentType } from "../base/base-site.js";
import { BaseSsrSiteArgs } from "../base/base-ssr-site.js";
import {
  cloudfront,
  getRegionOutput,
  lambda,
  Region,
  types,
} from "@pulumi/aws";
import { OriginAccessControl } from "./providers/origin-access-control.js";
import { readDirRecursivelySync } from "../../util/fs.js";
import { KvKeys } from "./providers/kv-keys.js";
import { useProvider } from "./helpers/provider.js";
import { Link } from "../link.js";
import { URL_UNAVAILABLE } from "./linkable.js";

const supportedRegions = {
  "af-south-1": { lat: -33.9249, lon: 18.4241 }, // Cape Town, South Africa
  "ap-east-1": { lat: 22.3193, lon: 114.1694 }, // Hong Kong
  "ap-northeast-1": { lat: 35.6895, lon: 139.6917 }, // Tokyo, Japan
  "ap-northeast-2": { lat: 37.5665, lon: 126.978 }, // Seoul, South Korea
  "ap-northeast-3": { lat: 34.6937, lon: 135.5023 }, // Osaka, Japan
  "ap-southeast-1": { lat: 1.3521, lon: 103.8198 }, // Singapore
  "ap-southeast-2": { lat: -33.8688, lon: 151.2093 }, // Sydney, Australia
  "ap-southeast-3": { lat: -6.2088, lon: 106.8456 }, // Jakarta, Indonesia
  "ap-southeast-4": { lat: -37.8136, lon: 144.9631 }, // Melbourne, Australia
  "ap-southeast-5": { lat: 3.139, lon: 101.6869 }, // Kuala Lumpur, Malaysia
  "ap-southeast-7": { lat: 13.7563, lon: 100.5018 }, // Bangkok, Thailand
  "ap-south-1": { lat: 19.076, lon: 72.8777 }, // Mumbai, India
  "ap-south-2": { lat: 17.385, lon: 78.4867 }, // Hyderabad, India
  "ca-central-1": { lat: 45.5017, lon: -73.5673 }, // Montreal, Canada
  "ca-west-1": { lat: 51.0447, lon: -114.0719 }, // Calgary, Canada
  "cn-north-1": { lat: 39.9042, lon: 116.4074 }, // Beijing, China
  "cn-northwest-1": { lat: 38.4872, lon: 106.2309 }, // Yinchuan, Ningxia
  "eu-central-1": { lat: 50.1109, lon: 8.6821 }, // Frankfurt, Germany
  "eu-central-2": { lat: 47.3769, lon: 8.5417 }, // Zurich, Switzerland
  "eu-north-1": { lat: 59.3293, lon: 18.0686 }, // Stockholm, Sweden
  "eu-south-1": { lat: 45.4642, lon: 9.19 }, // Milan, Italy
  "eu-south-2": { lat: 40.4168, lon: -3.7038 }, // Madrid, Spain
  "eu-west-1": { lat: 53.3498, lon: -6.2603 }, // Dublin, Ireland
  "eu-west-2": { lat: 51.5074, lon: -0.1278 }, // London, UK
  "eu-west-3": { lat: 48.8566, lon: 2.3522 }, // Paris, France
  "il-central-1": { lat: 32.0853, lon: 34.7818 }, // Tel Aviv, Israel
  "me-central-1": { lat: 25.2048, lon: 55.2708 }, // Dubai, UAE
  "me-south-1": { lat: 26.0667, lon: 50.5577 }, // Manama, Bahrain
  "mx-central-1": { lat: 19.4326, lon: -99.1332 }, // Mexico City, Mexico
  "sa-east-1": { lat: -23.5505, lon: -46.6333 }, // São Paulo, Brazil
  "us-east-1": { lat: 39.0438, lon: -77.4874 }, // Ashburn, VA
  "us-east-2": { lat: 39.9612, lon: -82.9988 }, // Columbus, OH
  "us-gov-east-1": { lat: 38.9696, lon: -77.3861 }, // Herndon, VA
  "us-gov-west-1": { lat: 34.0522, lon: -118.2437 }, // Los Angeles, CA
  "us-west-1": { lat: 37.7749, lon: -122.4194 }, // San Francisco, CA
  "us-west-2": { lat: 45.5122, lon: -122.6587 }, // Portland, OR
};

export const validatePlan = (input: {
  server?: Unwrap<FunctionArgs>;
  s3: {
    originPath?: string;
    copy: {
      from: string;
      to: string;
      cached: boolean;
      versionedSubDir?: string;
    }[];
  };
  cloudFrontFunction: {
    injection: string;
  };
  errorResponses?: types.input.cloudfront.DistributionCustomErrorResponse[];
  serverCachePolicy?: {
    allowedHeaders?: string[];
  };
  buildId?: string;
}) => input;
export type Plan = ReturnType<typeof validatePlan>;

export interface SsrSiteArgs extends BaseSsrSiteArgs {
  domain?: CdnArgs["domain"];
  permissions?: FunctionArgs["permissions"];
  regions?: Input<string[]>;
  cachePolicy?: Input<string>;
  invalidation?: Input<
    | false
    | {
        /**
         * Configure if `sst deploy` should wait for the CloudFront cache invalidation to finish.
         *
         * :::tip
         * For non-prod environments it might make sense to pass in `false`.
         * :::
         *
         * Waiting for this process to finish ensures that new content will be available after the deploy finishes. However, this process can sometimes take more than 5 mins.
         * @default `false`
         * @example
         * ```js
         * {
         *   invalidation: {
         *     wait: true
         *   }
         * }
         * ```
         */
        wait?: Input<boolean>;
        /**
         * The paths to invalidate.
         *
         * You can either pass in an array of glob patterns to invalidate specific files. Or you can use one of these built-in options:
         * - `all`: All files will be invalidated when any file changes
         * - `versioned`: Only versioned files will be invalidated when versioned files change
         *
         * :::note
         * Each glob pattern counts as a single invalidation. However, invalidating `all` counts as a single invalidation as well.
         * :::
         * @default `"all"`
         * @example
         * Invalidate the `index.html` and all files under the `products/` route. This counts as two invalidations.
         * ```js
         * {
         *   invalidation: {
         *     paths: ["/index.html", "/products/*"]
         *   }
         * }
         * ```
         */
        paths?: Input<"all" | "versioned" | string[]>;
      }
  >;
  /**
   * The number of instances of the [server function](#nodes-server) to keep warm. This is useful for cases where you are experiencing long cold starts. The default is to not keep any instances warm.
   *
   * This works by starting a serverless cron job to make _n_ concurrent requests to the server function every few minutes. Where _n_ is the number of instances to keep warm.
   *
   * @default `0`
   */
  warm?: Input<number>;
  /**
   * Configure the Lambda function used for server.
   * @default `{architecture: "x86_64", memory: "1024 MB"}`
   */
  server?: {
    /**
     * The amount of memory allocated to the server function.
     * Takes values between 128 MB and 10240 MB in 1 MB increments.
     *
     * @default `"1024 MB"`
     * @example
     * ```js
     * {
     *   server: {
     *     memory: "2048 MB"
     *   }
     * }
     * ```
     */
    memory?: FunctionArgs["memory"];
    /**
     * The runtime environment for the server function.
     *
     * @default `"nodejs20.x"`
     * @example
     * ```js
     * {
     *   server: {
     *     runtime: "nodejs22.x"
     *   }
     * }
     * ```
     */
    runtime?: Input<"nodejs18.x" | "nodejs20.x" | "nodejs22.x">;
    /**
     * The [architecture](https://docs.aws.amazon.com/lambda/latest/dg/foundation-arch.html)
     * of the server function.
     *
     * @default `"x86_64"`
     * @example
     * ```js
     * {
     *   server: {
     *     architecture: "arm64"
     *   }
     * }
     * ```
     */
    architecture?: FunctionArgs["architecture"];
    /**
     * Dependencies that need to be excluded from the server function package.
     *
     * Certain npm packages cannot be bundled using esbuild. This allows you to exclude them
     * from the bundle. Instead they'll be moved into a `node_modules/` directory in the
     * function package.
     *
     * :::tip
     * If esbuild is giving you an error about a package, try adding it to the `install` list.
     * :::
     *
     * This will allow your functions to be able to use these dependencies when deployed. They
     * just won't be tree shaken. You however still need to have them in your `package.json`.
     *
     * :::caution
     * Packages listed here still need to be in your `package.json`.
     * :::
     *
     * Esbuild will ignore them while traversing the imports in your code. So these are the
     * **package names as seen in the imports**. It also works on packages that are not directly
     * imported by your code.
     *
     * @example
     * ```js
     * {
     *   server: {
     *     install: ["sharp"]
     *   }
     * }
     * ```
     */
    install?: Input<string[]>;
    /**
     * Configure additional esbuild loaders for other file extensions. This is useful
     * when your code is importing non-JS files like `.png`, `.css`, etc.
     *
     * @example
     * ```js
     * {
     *   server: {
     *     loader: {
     *      ".png": "file"
     *     }
     *   }
     * }
     * ```
     */
    loader?: Input<Record<string, Loader>>;
    /**
     * A list of Lambda layer ARNs to add to the server function.
     *
     * @example
     * ```js
     * {
     *   server: {
     *     layers: ["arn:aws:lambda:us-east-1:123456789012:layer:my-layer:1"]
     *   }
     * }
     * ```
     */
    layers?: Input<Input<string>[]>;
    /**
     * Configure CloudFront Functions to customize the behavior of HTTP requests and responses at the edge.
     */
    edge?: Input<{
      /**
       * Configure the viewer request function.
       *
       * The viewer request function can be used to modify incoming requests before they
       * reach your origin server. For example, you can redirect users, rewrite URLs,
       * or add headers.
       */
      viewerRequest?: Input<{
        /**
         * The code to inject into the viewer request function.
         *
         * By default, a viewer request function is created to:
         * - Disable CloudFront default URL if custom domain is set.
         * - Add the `x-forwarded-host` header.
         *
         * The given code will be injected at the end of this function.
         *
         * ```js
         * async function handler(event) {
         *   // Default behavior code
         *
         *   // User injected code
         *
         *   return event.request;
         * }
         * ```
         *
         * @example
         * To add a custom header to all requests.
         *
         * ```js
         * {
         *   server: {
         *     edge: {
         *       viewerRequest: {
         *         injection: `event.request.headers["x-foo"] = "bar";`
         *       }
         *     }
         *   }
         * }
         * ```
         *
         * You can use this add basic auth, [check out an example](/docs/examples/#aws-nextjs-basic-auth).
         */
        injection: Input<string>;
        /**
         * The KV store to associate with the viewer request function.
         *
         * @example
         * ```js
         * {
         *   server: {
         *     edge: {
         *       viewerRequest: {
         *         kvStore: "arn:aws:cloudfront::123456789012:key-value-store/my-store"
         *       }
         *     }
         *   }
         * }
         * ```
         */
        kvStore?: Input<string>;
        /**
         * @deprecated Use `kvStore` instead because CloudFront Functions only support one KV store.
         */
        kvStores?: Input<Input<string>[]>;
      }>;
      /**
       * Configure the viewer response function.
       *
       * The viewer response function can be used to modify outgoing responses before they are
       * sent to the client. For example, you can add security headers or change the response
       * status code.
       */
      viewerResponse?: Input<{
        /**
         * The code to inject into the viewer response function.
         *
         * By default, no viewer response function is set. A new function will be created with
         * the provided code.
         *
         * ```js
         * async function handler(event) {
         *   // User injected code
         *
         *   return event.response;
         * }
         * ```
         *
         * @example
         * To add a custom header to all responses.
         *
         * ```js
         * {
         *   server: {
         *     edge: {
         *       viewerResponse: {
         *         injection: `event.response.headers["x-foo"] = {value: "bar"};`
         *       }
         *     }
         *   }
         * }
         * ```
         */
        injection: Input<string>;
        /**
         * The KV store to associate with the viewer response function.
         *
         * @example
         * ```js
         * {
         *   server: {
         *     edge: {
         *       viewerResponse: {
         *         kvStore: "arn:aws:cloudfront::123456789012:key-value-store/my-store"
         *       }
         *     }
         *   }
         * }
         * ```
         */
        kvStore?: Input<string>;
        /**
         * @deprecated Use `kvStore` instead because CloudFront Functions only support one KV store.
         */
        kvStores?: Input<Input<string>[]>;
      }>;
    }>;
  };
  vpc?: FunctionArgs["vpc"];
  /**
   * [Transform](/docs/components#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the Bucket resource used for uploading the assets.
     */
    assets?: Transform<BucketArgs>;
    /**
     * Transform the server Function resource.
     */
    server?: Transform<FunctionArgs>;
    /**
     * Transform the CloudFront CDN resource.
     */
    cdn?: Transform<CdnArgs>;
  };
}

export function prepare(parent: ComponentResource, args: SsrSiteArgs) {
  const regions = normalizeRegions();
  const sitePath = regions.apply(() => normalizeSitePath());
  const dev = normalizeDev();
  return { dev, sitePath, regions };

  function normalizeDev() {
    const enabled = $dev && args.dev !== false;
    const devArgs = args.dev || {};

    return {
      enabled,
      url: output(devArgs.url ?? URL_UNAVAILABLE),
      outputs: {
        title: devArgs.title,
        command: output(devArgs.command ?? "npm run dev"),
        autostart: output(devArgs.autostart ?? true),
        directory: output(devArgs.directory ?? sitePath),
        environment: args.environment,
        links: output(args.link || [])
          .apply(Link.build)
          .apply((links) => links.map((link) => link.name)),
      },
    };
  }

  function normalizeSitePath() {
    return output(args.path).apply((sitePath) => {
      if (!sitePath) return ".";

      if (!fs.existsSync(sitePath)) {
        throw new VisibleError(`No site found at "${path.resolve(sitePath)}"`);
      }
      return sitePath;
    });
  }

  function normalizeRegions() {
    return output(
      args.regions ?? [getRegionOutput(undefined, { parent }).name],
    ).apply((regions) => {
      if (regions.length === 0)
        throw new VisibleError(
          "No regions provided. Please provide at least one region.",
        );

      return regions.map((region) => {
        if (
          [
            "ap-south-2",
            "ap-southeast-4",
            "ap-southeast-5",
            "ca-west-1",
            "eu-south-2",
            "eu-central-2",
            "il-central-1",
            "me-central-1",
          ].includes(region)
        )
          throw new VisibleError(
            `Region ${region} is not currently supported. Please use a different region.`,
          );

        if (!Object.values(Region).includes(region as Region))
          throw new VisibleError(
            `Region ${region} is not a valid AWS region. Please use a different region.`,
          );
        return region as Region;
      });
    });
  }
}

export function createDevResources(
  parent: ComponentResource,
  name: string,
  args: SsrSiteArgs,
) {
  const server = new Function(
    ...transform(
      args.transform?.server,
      `${name}DevServer`,
      {
        description: `${name} dev server`,
        runtime: "nodejs20.x",
        timeout: "20 seconds",
        memory: "128 MB",
        bundle: path.join($cli.paths.platform, "functions", "empty-function"),
        handler: "index.handler",
        environment: args.environment,
        permissions: args.permissions,
        link: args.link,
        dev: false,
      },
      { parent },
    ),
  );

  return { server };
}

export function createResources(
  parent: ComponentResource,
  name: string,
  args: SsrSiteArgs,
  outputPath: Output<string>,
  plan: Output<Plan>,
  regions: Output<Region[]>,
) {
  const access = createCloudFrontOriginAccessControl();
  const bucket = createS3Bucket();
  const bucketFiles = uploadAssets();
  const servers = createServers();
  const requestFunction = createRequestFunction();
  const responseFunction = createResponseFunction();
  const cachePolicyId = args.cachePolicy ?? createCachePolicy().id;
  const distribution = createDistribution();

  return {
    distribution,
    bucket,
    servers: servers.apply((servers) => servers.map((s) => s.server)),
  };

  function createCloudFrontOriginAccessControl() {
    return new OriginAccessControl(
      `${name}S3AccessControl`,
      { name: physicalName(64, name) },
      { parent, ignoreChanges: ["name"] },
    );
  }

  function createS3Bucket() {
    return new Bucket(
      ...transform(
        args.transform?.assets,
        `${name}Assets`,
        { access: "cloudfront" },
        { parent, retainOnDelete: false },
      ),
    );
  }

  function uploadAssets() {
    return all([args.assets, plan, outputPath]).apply(
      async ([assets, plan, outputPath]) => {
        // Define content headers
        const versionedFilesTTL = 31536000; // 1 year
        const nonVersionedFilesTTL = 86400; // 1 day

        const bucketFiles: BucketFile[] = [];

        // Handle each copy source
        for (const copy of plan.s3.copy) {
          // Build fileOptions
          const fileOptions: BaseSiteFileOptions[] = [
            // unversioned files
            {
              files: "**",
              ignore: copy.versionedSubDir
                ? path.posix.join(copy.versionedSubDir, "**")
                : undefined,
              cacheControl:
                assets?.nonVersionedFilesCacheHeader ??
                `public,max-age=0,s-maxage=${nonVersionedFilesTTL},stale-while-revalidate=${nonVersionedFilesTTL}`,
            },
            // versioned files
            ...(copy.versionedSubDir
              ? [
                  {
                    files: path.posix.join(copy.versionedSubDir, "**"),
                    cacheControl:
                      assets?.versionedFilesCacheHeader ??
                      `public,max-age=${versionedFilesTTL},immutable`,
                  },
                ]
              : []),
            ...(assets?.fileOptions ?? []),
          ];

          // Upload files based on fileOptions
          const filesUploaded: string[] = [];
          for (const fileOption of fileOptions.reverse()) {
            const files = globSync(fileOption.files, {
              cwd: path.resolve(outputPath, copy.from),
              nodir: true,
              dot: true,
              ignore: fileOption.ignore,
            }).filter((file) => !filesUploaded.includes(file));

            bucketFiles.push(
              ...(await Promise.all(
                files.map(async (file) => {
                  const source = path.resolve(outputPath, copy.from, file);
                  const content = await fs.promises.readFile(source, "utf-8");
                  const hash = crypto
                    .createHash("sha256")
                    .update(content)
                    .digest("hex");
                  return {
                    source,
                    key: path.posix.join(copy.to, file),
                    hash,
                    cacheControl: fileOption.cacheControl,
                    contentType:
                      fileOption.contentType ?? getContentType(file, "UTF-8"),
                  };
                }),
              )),
            );
            filesUploaded.push(...files);
          }
        }

        return new BucketFiles(
          `${name}AssetFiles`,
          {
            bucketName: bucket.name,
            files: bucketFiles,
            purge: false,
            region: getRegionOutput(undefined, { parent }).name,
          },
          { parent },
        );
      },
    );
  }

  function createServers() {
    return all([regions, plan.server]).apply(([regions, planServer]) => {
      if (!planServer) return [];

      return regions.map((region) => {
        const provider = useProvider(region);
        const server = new Function(
          ...transform(
            args.transform?.server,
            `${name}Server${logicalName(region)}`,
            {
              ...planServer,
              description: planServer.description ?? `${name} server`,
              runtime: output(args.server?.runtime).apply(
                (v) => v ?? planServer.runtime ?? "nodejs20.x",
              ),
              timeout: planServer.timeout ?? "20 seconds",
              memory: output(args.server?.memory).apply(
                (v) => v ?? planServer.memory ?? "1024 MB",
              ),
              architecture: output(args.server?.architecture).apply(
                (v) => v ?? planServer.architecture ?? "x86_64",
              ),
              vpc: args.vpc,
              nodejs: {
                format: "esm" as const,
                install: args.server?.install,
                loader: args.server?.loader,
                ...planServer.nodejs,
              },
              environment: output(args.environment).apply((environment) => ({
                ...environment,
                ...planServer.environment,
              })),
              permissions: output(args.permissions).apply((permissions) => [
                {
                  actions: ["cloudfront:CreateInvalidation"],
                  resources: ["*"],
                },
                ...(permissions ?? []),
                ...(planServer.permissions ?? []),
              ]),
              injections: [
                ...(args.warm
                  ? [useServerWarmingInjection(planServer.streaming)]
                  : []),
                ...(planServer.injections || []),
              ],
              link: output(args.link).apply((link) => [
                ...(planServer.link ?? []),
                ...(link ?? []),
              ]),
              layers: output(args.server?.layers).apply((layers) => [
                ...(planServer.layers ?? []),
                ...(layers ?? []),
              ]),
              url: true,
              dev: false,
              _skipHint: true,
            },
            { provider, parent },
          ),
        );

        if (args.warm) {
          // Create cron job
          const cron = new Cron(
            `${name}Warmer${logicalName(region)}`,
            {
              schedule: "rate(5 minutes)",
              job: {
                description: `${name} warmer`,
                bundle: path.join($cli.paths.platform, "dist", "ssr-warmer"),
                runtime: "nodejs20.x",
                handler: "index.handler",
                timeout: "900 seconds",
                memory: "128 MB",
                dev: false,
                environment: {
                  FUNCTION_NAME: server.nodes.function.name,
                  CONCURRENCY: output(args.warm).apply((warm) =>
                    warm.toString(),
                  ),
                },
                link: [server],
                _skipMetadata: true,
              },
              transform: {
                target: (args) => {
                  args.retryPolicy = {
                    maximumRetryAttempts: 0,
                    maximumEventAgeInSeconds: 60,
                  };
                },
              },
            },
            { provider, parent },
          );

          // Prewarm on deploy
          new lambda.Invocation(
            `${name}Prewarm${logicalName(region)}`,
            {
              functionName: cron.nodes.job.name,
              triggers: {
                version: Date.now().toString(),
              },
              input: JSON.stringify({}),
            },
            { provider, parent },
          );
        }

        return { region, server };
      });
    });
  }

  function createCachePolicy() {
    return output(plan.serverCachePolicy).apply(
      (serverCachePolicy) =>
        new cloudfront.CachePolicy(
          `${name}ServerCachePolicy`,
          {
            comment: "SST server response cache policy",
            defaultTtl: 0,
            maxTtl: 31536000, // 1 year
            minTtl: 0,
            parametersInCacheKeyAndForwardedToOrigin: {
              cookiesConfig: {
                cookieBehavior: "none",
              },
              headersConfig:
                (serverCachePolicy?.allowedHeaders ?? []).length > 0
                  ? {
                      headerBehavior: "whitelist",
                      headers: {
                        items: serverCachePolicy?.allowedHeaders,
                      },
                    }
                  : {
                      headerBehavior: "none",
                    },
              queryStringsConfig: {
                queryStringBehavior: "all",
              },
              enableAcceptEncodingBrotli: true,
              enableAcceptEncodingGzip: true,
            },
          },
          { parent },
        ),
    );
  }

  function createRequestFunction() {
    return all([args.server, servers, outputPath, plan]).apply(
      ([argsServer, servers, outputPath, plan]) => {
        const userConfig = argsServer?.edge?.viewerRequest;
        const userInjection = userConfig?.injection;
        const userKvStore = userConfig?.kvStore ?? userConfig?.kvStores?.[0];
        const frameworkInjection = plan.cloudFrontFunction.injection;
        const blockCloudfrontUrlInjection = args.domain
          ? useCfBlockCloudFrontUrlInjection()
          : "";
        const kvStoreArn = (() => {
          if (userKvStore) return userKvStore;
          if (!plan.server) return;
          return new cloudfront.KeyValueStore(`${name}KvStore`, {}, { parent })
            .arn;
        })();
        const routerInjection = (() => {
          if (!plan.server) return;

          // In the case multiple sites use the same kv store, we need to namespace the keys
          const kvNamespace = crypto
            .createHash("md5")
            .update(`${$app.name}-${$app.stage}-${name}`)
            .digest("hex")
            .substring(0, 4);
          new KvKeys(
            `${name}KvKeys`,
            {
              store: kvStoreArn!,
              namespace: kvNamespace,
              entries: Object.fromEntries(
                plan.s3.copy
                  .flatMap((copy) =>
                    readDirRecursivelySync(path.join(outputPath, copy.from)),
                  )
                  .map((file) => [`/${file}`, "s3"]),
              ),
              purge: false,
            },
            { parent },
          );

          const domainNameInjection = all(
            servers.map((s) => ({ region: s.region, url: s.server!.url })),
          ).apply((servers) =>
            servers.length === 1
              ? `"${new URL(servers[0].url).host}"`
              : `(() => {
  const lat = event.request.headers["cloudfront-viewer-latitude"] && event.request.headers["cloudfront-viewer-latitude"].value;
  const lon = event.request.headers["cloudfront-viewer-longitude"] && event.request.headers["cloudfront-viewer-longitude"].value;
  if (!lat || !lon) return "${new URL(servers[0].url).host}";
  return ${JSON.stringify(
    servers.map((s) => ({
      host: new URL(s.url).host,
      lat: supportedRegions[s.region as keyof typeof supportedRegions].lat,
      lon: supportedRegions[s.region as keyof typeof supportedRegions].lon,
    })),
  )}
    .map((s) => ({
      distance: haversineDistance(lat, lon, s.lat, s.lon),
      host: s.host,
    }))
    .sort((a, b) => a.distance - b.distance)[0]
    .host;

  function haversineDistance(lat1, lon1, lat2, lon2) {
    const toRad = angle => angle * Math.PI / 180;
    const radLat1 = toRad(lat1);
    const radLat2 = toRad(lat2);
    const dLat = toRad(lat2 - lat1);
    const dLon = toRad(lon2 - lon1);
    const a = Math.sin(dLat / 2) ** 2 + Math.cos(radLat1) * Math.cos(radLat2) * Math.sin(dLon / 2) ** 2;
    return 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
  }
})()`,
          );
          return interpolate`
  try {
    const v = await cf.kvs().get("${kvNamespace}:" + decodeURIComponent(event.request.uri));
    if (v === "s3") return event.request;
  } catch (e) {}

  try {
    const v = await cf.kvs().get("${kvNamespace}:" + decodeURIComponent(event.request.uri) + "/index.html");
    if (v === "s3") {
      event.request.uri = event.request.uri + "/index.html";
      return event.request;
    }
  } catch (e) {}

  event.request.headers["x-forwarded-host"] = event.request.headers.host;
  cf.updateRequestOrigin({
    domainName: ${domainNameInjection},
    customOriginConfig: {
      port: 443,
      protocol: "https",
      sslProtocols: ["TLSv1.2"],
    },
    originAccessControlConfig: {
      enabled: false,
    }
  });`;
        })();

        return new cloudfront.Function(
          `${name}CloudfrontFunctionRequest`,
          {
            runtime: "cloudfront-js-2.0",
            keyValueStoreAssociations: kvStoreArn ? [kvStoreArn] : [],
            code: interpolate`
import cf from "cloudfront";
async function handler(event) {
  ${userInjection ?? ""}
  ${blockCloudfrontUrlInjection}
  ${frameworkInjection ?? ""}
  ${routerInjection ?? ""}
  return event.request;
}`,
          },
          { parent },
        );
      },
    );
  }

  function createResponseFunction() {
    return all([args.server]).apply(([server]) => {
      const userConfig = server?.edge?.viewerResponse;
      const userInjection = userConfig?.injection;
      const kvStoreArn = userConfig?.kvStore ?? userConfig?.kvStores?.[0];

      if (!userConfig && !kvStoreArn) return;

      return new cloudfront.Function(
        `${name}CloudfrontFunctionResponse`,
        {
          runtime: "cloudfront-js-2.0",
          keyValueStoreAssociations: kvStoreArn ? [kvStoreArn] : [],
          code: interpolate`
import cf from "cloudfront";
async function handler(event) {
  ${userInjection ?? ""}
  return event.response;
}`,
        },
        { parent },
      );
    });
  }

  function createDistribution() {
    return output(plan).apply((plan) => {
      const errorRoutes = (plan.errorResponses ?? [])
        .filter((e) => !e.responsePagePath?.endsWith(".html"))
        .map((e) => e.responsePagePath!);

      return new Cdn(
        ...transform(
          args.transform?.cdn,
          `${name}Cdn`,
          {
            comment: `${name} app`,
            origins: [
              {
                originId: "default",
                domainName: bucket.nodes.bucket.bucketRegionalDomainName,
                originPath: plan.s3.originPath ? `/${plan.s3.originPath}` : "",
                originAccessControlId: access.id,
              },
              ...(errorRoutes.length > 0
                ? [
                    {
                      originId: "error",
                      domainName: servers.apply((servers) =>
                        servers[0].server.url.apply((url) => new URL(url).host),
                      ),
                      customOriginConfig: {
                        httpPort: 80,
                        httpsPort: 443,
                        originProtocolPolicy: "https-only",
                        originReadTimeout: 20,
                        originSslProtocols: ["TLSv1.2"],
                      },
                    },
                  ]
                : []),
            ],
            defaultCacheBehavior: {
              targetOriginId: "default",
              viewerProtocolPolicy: "redirect-to-https",
              allowedMethods: [
                "DELETE",
                "GET",
                "HEAD",
                "OPTIONS",
                "PATCH",
                "POST",
                "PUT",
              ],
              cachedMethods: ["GET", "HEAD"],
              compress: true,
              cachePolicyId,
              // CloudFront's Managed-AllViewerExceptHostHeader policy
              originRequestPolicyId: "b689b0a8-53d0-40ab-baf2-68738e2966ac",
              functionAssociations: all([
                requestFunction,
                responseFunction,
              ]).apply(([reqFn, resFn]) => [
                ...(reqFn
                  ? [{ eventType: "viewer-request", functionArn: reqFn.arn }]
                  : []),
                ...(resFn
                  ? [{ eventType: "viewer-response", functionArn: resFn.arn }]
                  : []),
              ]),
            },
            orderedCacheBehaviors: [...new Set(errorRoutes)].map((route) => ({
              pathPattern: route,
              targetOriginId: "error",
              viewerProtocolPolicy: "redirect-to-https",
              allowedMethods: ["GET", "HEAD", "OPTIONS"],
              cachedMethods: ["GET", "HEAD"],
              compress: true,
              cachePolicyId,
            })),
            customErrorResponses: plan.errorResponses,
            domain: args.domain,
            invalidation: buildInvalidation(),
          },
          // create distribution after assets are uploaded
          { dependsOn: bucketFiles, parent },
        ),
      );
    });
  }

  function buildInvalidation() {
    return all([args.invalidation, outputPath, plan]).apply(
      ([invalidationRaw, outputPath, plan]) => {
        // Normalize invalidation
        if (invalidationRaw === false) return false;
        const invalidation = {
          wait: false,
          paths: "all",
          ...invalidationRaw,
        };

        // We will generate a hash based on the contents of the S3 files with cache enabled.
        // This will be used to determine if we need to invalidate our CloudFront cache.
        const s3Origin = plan.s3;
        const cachedS3Files = s3Origin.copy.filter((file) => file.cached);
        if (cachedS3Files.length === 0) return false;

        // Build invalidation paths
        const invalidationPaths: string[] = [];
        if (invalidation.paths === "all") {
          invalidationPaths.push("/*");
        } else if (invalidation.paths === "versioned") {
          cachedS3Files.forEach((item) => {
            if (!item.versionedSubDir) return false;
            invalidationPaths.push(
              path.posix.join("/", item.to, item.versionedSubDir, "*"),
            );
          });
        } else {
          invalidationPaths.push(...(invalidation?.paths || []));
        }
        if (invalidationPaths.length === 0) return false;

        // Build build ID
        let invalidationBuildId: string;
        if (plan.buildId) {
          invalidationBuildId = plan.buildId;
        } else {
          const hash = crypto.createHash("md5");

          cachedS3Files.forEach((item) => {
            // The below options are needed to support following symlinks when building zip files:
            // - nodir: This will prevent symlinks themselves from being copied into the zip.
            // - follow: This will follow symlinks and copy the files within.

            // For versioned files, use file path for digest since file version in name should change on content change
            if (item.versionedSubDir) {
              globSync("**", {
                dot: true,
                nodir: true,
                follow: true,
                cwd: path.resolve(outputPath, item.from, item.versionedSubDir),
              }).forEach((filePath) => hash.update(filePath));
            }

            // For non-versioned files, use file content for digest
            if (invalidation.paths !== "versioned") {
              globSync("**", {
                ignore: item.versionedSubDir
                  ? [path.posix.join(item.versionedSubDir, "**")]
                  : undefined,
                dot: true,
                nodir: true,
                follow: true,
                cwd: path.resolve(outputPath, item.from),
              }).forEach((filePath) =>
                hash.update(
                  fs.readFileSync(
                    path.resolve(outputPath, item.from, filePath),
                    "utf-8",
                  ),
                ),
              );
            }
          });
          invalidationBuildId = hash.digest("hex");
        }

        return {
          paths: invalidationPaths,
          token: invalidationBuildId,
          wait: invalidation.wait,
        };
      },
    );
  }

  function useServerWarmingInjection(streaming?: boolean) {
    return [
      `if (event.type === "warmer") {`,
      `  const p = new Promise((resolve) => {`,
      `    setTimeout(() => {`,
      `      resolve({ serverId: "server-" + Math.random().toString(36).slice(2, 8) });`,
      `    }, event.delay);`,
      `  });`,
      ...(streaming
        ? [
            `  const response = await p;`,
            `  responseStream.write(JSON.stringify(response));`,
            `  responseStream.end();`,
            `  return;`,
          ]
        : [`  return p;`]),
      `}`,
    ].join("\n");
  }

  function useCfBlockCloudFrontUrlInjection() {
    return `
if (event.request.headers.host.value.includes('cloudfront.net')) {
  return {
    statusCode: 403,
    statusDescription: 'Forbidden',
    body: {
      encoding: "text",
      data: '<html><head><title>403 Forbidden</title></head><body><center><h1>403 Forbidden</h1></center></body></html>'
    }
  };
}`;
  }
}
