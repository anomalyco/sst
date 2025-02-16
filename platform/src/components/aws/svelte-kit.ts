import fs from "fs";
import path from "path";
import { ComponentResourceOptions, Output, all, output } from "@pulumi/pulumi";
import { Function } from "./function.js";
import {
  SsrSiteArgs,
  createBucket,
  createDevServer,
  createServersAndDistribution,
  prepare,
  useCloudFrontFunctionHostHeaderInjection,
  validatePlan,
} from "./ssr-site.js";
import { Cdn } from "./cdn.js";
import { Bucket } from "./bucket.js";
import { Component } from "../component.js";
import { Link } from "../link.js";
import type { Input } from "../input.js";
import { buildApp } from "../base/base-ssr-site.js";
import { URL_UNAVAILABLE } from "./linkable.js";

export interface SvelteKitArgs extends SsrSiteArgs {
  /**
   * Configure how this component works in `sst dev`.
   *
   * :::note
   * In `sst dev` your SvelteKit app is run in dev mode; it's not deployed.
   * :::
   *
   * Instead of deploying your SvelteKit app, this starts it in dev mode. It's run
   * as a separate process in the `sst dev` multiplexer. Read more about
   * [`sst dev`](/docs/reference/cli/#dev).
   *
   * To disable dev mode, pass in `false`.
   */
  dev?: SsrSiteArgs["dev"];
  /**
   * Permissions and the resources that the [server function](#nodes-server) in your SvelteKit app needs to access. These permissions are used to create the function's IAM role.
   *
   * :::tip
   * If you `link` the function to a resource, the permissions to access it are
   * automatically added.
   * :::
   *
   * @example
   * Allow reading and writing to an S3 bucket called `my-bucket`.
   * ```js
   * {
   *   permissions: [
   *     {
   *       actions: ["s3:GetObject", "s3:PutObject"],
   *       resources: ["arn:aws:s3:::my-bucket/*"]
   *     },
   *   ]
   * }
   * ```
   *
   * Perform all actions on an S3 bucket called `my-bucket`.
   *
   * ```js
   * {
   *   permissions: [
   *     {
   *       actions: ["s3:*"],
   *       resources: ["arn:aws:s3:::my-bucket/*"]
   *     },
   *   ]
   * }
   * ```
   *
   * Grant permissions to access all resources.
   *
   * ```js
   * {
   *   permissions: [
   *     {
   *       actions: ["*"],
   *       resources: ["*"]
   *     },
   *   ]
   * }
   * ```
   */
  permissions?: SsrSiteArgs["permissions"];
  /**
   * Path to the directory where your SvelteKit app is located.  This path is relative to your `sst.config.ts`.
   *
   * By default it assumes your SvelteKit app is in the root of your SST app.
   * @default `"."`
   *
   * @example
   *
   * If your SvelteKit app is in a package in your monorepo.
   *
   * ```js
   * {
   *   path: "packages/web"
   * }
   * ```
   */
  path?: SsrSiteArgs["path"];
  /**
   * [Link resources](/docs/linking/) to your SvelteKit app. This will:
   *
   * 1. Grant the permissions needed to access the resources.
   * 2. Allow you to access it in your site using the [SDK](/docs/reference/sdk/).
   *
   * @example
   *
   * Takes a list of resources to link to the function.
   *
   * ```js
   * {
   *   link: [bucket, stripeKey]
   * }
   * ```
   */
  link?: SsrSiteArgs["link"];
  /**
   * Configure how the CloudFront cache invalidations are handled. This is run after your SvelteKit app has been deployed.
   * :::tip
   * You get 1000 free invalidations per month. After that you pay $0.005 per invalidation path. [Read more here](https://aws.amazon.com/cloudfront/pricing/).
   * :::
   * @default `{paths: "all", wait: false}`
   * @example
   * Wait for all paths to be invalidated.
   * ```js
   * {
   *   invalidation: {
   *     paths: "all",
   *     wait: true
   *   }
   * }
   * ```
   */
  invalidation?: SsrSiteArgs["invalidation"];
  /**
   * Set [environment variables](https://vitejs.dev/guide/env-and-mode.html#env-files) in your SvelteKit app. These are made available:
   *
   * 1. In `vite build`, they are loaded into `process.env`.
   * 2. Locally while running through `sst dev`.
   *
   * :::tip
   * You can also `link` resources to your SvelteKit app and access them in a type-safe way with the [SDK](/docs/reference/sdk/). We recommend linking since it's more secure.
   * :::
   *
   * @example
   * ```js
   * {
   *   environment: {
   *     API_URL: api.url,
   *     STRIPE_PUBLISHABLE_KEY: "pk_test_123"
   *   }
   * }
   * ```
   */
  environment?: SsrSiteArgs["environment"];
  /**
   * Set a custom domain for your SvelteKit app.
   *
   * Automatically manages domains hosted on AWS Route 53, Cloudflare, and Vercel. For other
   * providers, you'll need to pass in a `cert` that validates domain ownership and add the
   * DNS records.
   *
   * :::tip
   * Built-in support for AWS Route 53, Cloudflare, and Vercel. And manual setup for other
   * providers.
   * :::
   *
   * @example
   *
   * By default this assumes the domain is hosted on Route 53.
   *
   * ```js
   * {
   *   domain: "example.com"
   * }
   * ```
   *
   * For domains hosted on Cloudflare.
   *
   * ```js
   * {
   *   domain: {
   *     name: "example.com",
   *     dns: sst.cloudflare.dns()
   *   }
   * }
   * ```
   *
   * Specify a `www.` version of the custom domain.
   *
   * ```js
   * {
   *   domain: {
   *     name: "domain.com",
   *     redirects: ["www.domain.com"]
   *   }
   * }
   * ```
   */
  domain?: SsrSiteArgs["domain"];
  /**
   * The command used internally to build your SvelteKit app.
   *
   * @default `"npm run build"`
   *
   * @example
   *
   * If you want to use a different build command.
   * ```js
   * {
   *   buildCommand: "yarn build"
   * }
   * ```
   */
  buildCommand?: SsrSiteArgs["buildCommand"];
  /**
   * Configure how the SvelteKit app assets are uploaded to S3.
   *
   * By default, this is set to the following. Read more about these options below.
   * ```js
   * {
   *   assets: {
   *     textEncoding: "utf-8",
   *     versionedFilesCacheHeader: "public,max-age=31536000,immutable",
   *     nonVersionedFilesCacheHeader: "public,max-age=0,s-maxage=86400,stale-while-revalidate=8640"
   *   }
   * }
   * ```
   */
  assets?: SsrSiteArgs["assets"];
  /**
   * Configure where the [server function](#nodes-server) is deployed.
   *
   * By default, it's deployed to AWS Lambda in a single region. Enable this option if you want to instead deploy it to Lambda@Edge.
   * @default `false`
   * @internal
   */
  edge?: Input<boolean>;
  /**
   * Configure the [server function](#nodes-server) in your SvelteKit app to connect
   * to private subnets in a virtual private cloud or VPC. This allows your app to
   * access private resources.
   *
   * @example
   * ```js
   * {
   *   vpc: {
   *     securityGroups: ["sg-0399348378a4c256c"],
   *     subnets: ["subnet-0b6a2b73896dc8c4c", "subnet-021389ebee680c2f0"]
   *   }
   * }
   * ```
   */
  vpc?: SsrSiteArgs["vpc"];
  /**
   * Configure the SvelteKit app to use an existing CloudFront cache policy.
   *
   * :::note
   * CloudFront has a limit of 20 cache policies per account, though you can request a limit
   * increase.
   * :::
   *
   * By default, a new cache policy is created for it. This allows you to reuse an existing
   * policy instead of creating a new one.
   *
   * @default A new cache policy is created
   * @example
   * ```js
   * {
   *   cachePolicy: "658327ea-f89d-4fab-a63d-7e88639e58f6"
   * }
   * ```
   */
  cachePolicy?: SsrSiteArgs["cachePolicy"];
}

/**
 * The `SvelteKit` component lets you deploy a [SvelteKit](https://kit.svelte.dev/) app to AWS.
 *
 * @example
 *
 * #### Minimal example
 *
 * Deploy a SvelteKit app that's in the project root.
 *
 * ```js title="sst.config.ts"
 * new sst.aws.SvelteKit("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Deploys the SvelteKit app in the `my-svelte-app/` directory.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.SvelteKit("MyWeb", {
 *   path: "my-svelte-app/"
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your SvelteKit app.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.aws.SvelteKit("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Redirect www to apex domain
 *
 * Redirect `www.my-app.com` to `my-app.com`.
 *
 * ```js {4} title="sst.config.ts"
 * new sst.aws.SvelteKit("MyWeb", {
 *   domain: {
 *     name: "my-app.com",
 *     redirects: ["www.my-app.com"]
 *   }
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your SvelteKit app. This will grant permissions
 * to the resources and allow you to access it in your app.
 *
 * ```ts {4} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.SvelteKit("MyWeb", {
 *   link: [bucket]
 * });
 * ```
 *
 * You can use the [SDK](/docs/reference/sdk/) to access the linked resources
 * in your SvelteKit app.
 *
 * ```ts title="src/routes/+page.server.ts"
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyBucket.name);
 * ```
 */
export class SvelteKit extends Component implements Link.Linkable {
  private cdn?: Output<Cdn>;
  private assets?: Bucket;
  private server?: Output<Function>;
  private devUrl?: Output<string>;

  constructor(
    name: string,
    args: SvelteKitArgs = {},
    opts: ComponentResourceOptions = {},
  ) {
    super(__pulumiType, name, args, opts);

    const parent = this;
    const edge = normalizeEdge();
    const { sitePath, partition } = prepare(parent, args);
    const dev = normalizeDev();

    if (dev.enabled) {
      const server = createDevServer(parent, name, args);
      this.devUrl = dev.url;
      this.registerOutputs({
        _metadata: {
          mode: "placeholder",
          path: sitePath,
          edge,
          server: server.arn,
        },
        _dev: {
          ...dev.outputs,
          aws: { role: server.nodes.role.arn },
        },
      });
      return;
    }

    const { access, bucket } = createBucket(parent, name, partition, args);
    const outputPath = buildApp(parent, name, args, sitePath);
    const buildMeta = loadBuildMetadata();
    const plan = buildPlan();
    const { distribution, ssrFunctions, edgeFunctions } =
      createServersAndDistribution(
        parent,
        name,
        args,
        outputPath,
        access,
        bucket,
        plan,
      );
    const serverFunction = ssrFunctions[0] ?? Object.values(edgeFunctions)[0];

    this.assets = bucket;
    this.cdn = distribution;
    this.server = serverFunction;
    this.registerOutputs({
      _hint: all([this.cdn.domainUrl, this.cdn.url]).apply(
        ([domainUrl, url]) => domainUrl ?? url,
      ),
      _metadata: {
        mode: "deployed",
        path: sitePath,
        url: distribution.apply((d) => d.domainUrl ?? d.url),
        edge,
        server: serverFunction.arn,
      },
      _dev: {
        ...dev.outputs,
        aws: { role: serverFunction.nodes.role.arn },
      },
    });

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

    function normalizeEdge() {
      return output(args?.edge).apply((edge) => edge ?? false);
    }

    function loadBuildMetadata() {
      const serverPath = ".svelte-kit/svelte-kit-sst/server";
      const assetsPath = ".svelte-kit/svelte-kit-sst/client";

      return outputPath.apply((outputPath) => {
        let basePath = "";
        try {
          const manifest = fs
            .readFileSync(path.join(serverPath, "manifest.js"))
            .toString();
          const appDir = manifest.match(/appDir: "(.+?)"/)?.[1];
          const appPath = manifest.match(/appPath: "(.+?)"/)?.[1];
          if (appDir && appPath && appPath.endsWith(appDir)) {
            basePath = appPath.substring(0, appPath.length - appDir.length);
          }
        } catch (e) {}

        return {
          basePath,
          serverPath,
          serverFiles: ".svelte-kit/svelte-kit-sst/prerendered",
          prerenderedPath: ".svelte-kit/svelte-kit-sst/prerendered",
          assetsPath,
          assetsVersionedSubDir: "_app",
          // create 1 behaviour for each top level asset file/folder
          staticRoutes: fs
            .readdirSync(path.join(outputPath, assetsPath), {
              withFileTypes: true,
            })
            .map((item) =>
              item.isDirectory()
                ? `${basePath}${item.name}/*`
                : `${basePath}${item.name}`,
            ),
        };
      });
    }

    function buildPlan() {
      return all([outputPath, edge, buildMeta]).apply(
        ([outputPath, edge, buildMeta]) => {
          const serverConfig = {
            handler: path.join(
              outputPath,
              buildMeta.serverPath,
              "lambda-handler",
              "index.handler",
            ),
            nodejs: {
              esbuild: {
                minify: process.env.SST_DEBUG ? false : true,
                sourcemap: process.env.SST_DEBUG ? ("inline" as const) : false,
                define: {
                  "process.env.SST_DEBUG": process.env.SST_DEBUG
                    ? "true"
                    : "false",
                },
              },
            },
            copyFiles: buildMeta.serverFiles
              ? [
                  {
                    from: path.join(outputPath, buildMeta.serverFiles),
                    to: "prerendered",
                  },
                ]
              : undefined,
          };

          return validatePlan({
            edge,
            cloudFrontFunctions: {
              serverCfFunction: {
                injections: [
                  useCloudFrontFunctionHostHeaderInjection(),
                  useCloudFrontFormActionInjection(),
                ],
              },
            },
            edgeFunctions: edge
              ? {
                  server: { function: serverConfig },
                }
              : undefined,
            origins: {
              ...(edge
                ? {}
                : {
                    server: {
                      server: { function: serverConfig },
                    },
                  }),
              s3: {
                s3: {
                  copy: [
                    {
                      from: buildMeta.assetsPath,
                      to: buildMeta.basePath,
                      cached: true,
                      versionedSubDir: buildMeta.assetsVersionedSubDir,
                    },
                    {
                      from: buildMeta.prerenderedPath,
                      to: buildMeta.basePath,
                      cached: false,
                    },
                  ],
                },
              },
            },
            behaviors: [
              edge
                ? {
                    cacheType: "server",
                    cfFunction: "serverCfFunction",
                    edgeFunction: "server",
                    origin: "s3",
                  }
                : {
                    cacheType: "server",
                    cfFunction: "serverCfFunction",
                    origin: "server",
                  },
              ...buildMeta.staticRoutes.map(
                (route) =>
                  ({
                    cacheType: "static",
                    pattern: route,
                    origin: "s3",
                  }) as const,
              ),
            ],
          });
        },
      );
    }
  }

  /**
   * The URL of the SvelteKit app.
   *
   * If the `domain` is set, this is the URL with the custom domain.
   * Otherwise, it's the autogenerated CloudFront URL.
   */
  public get url() {
    return all([this.cdn?.domainUrl, this.cdn?.url, this.devUrl]).apply(
      ([domainUrl, url, dev]) => domainUrl ?? url ?? dev!,
    );
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The AWS Lambda server function that renders the app.
       */
      server: this.server,
      /**
       * The Amazon S3 Bucket that stores the assets.
       */
      assets: this.assets,
      /**
       * The Amazon CloudFront CDN that serves the app.
       */
      cdn: this.cdn,
    };
  }

  /** @internal */
  public getSSTLink() {
    return {
      properties: {
        url: this.url,
      },
    };
  }
}

function useCloudFrontFormActionInjection() {
  // Note: form action requests contain "/" in request query string
  //       ie. POST request with query string "?/action"
  //       CloudFront does not allow query string with "/". It needs to be encoded.
  return `
for (var key in event.request.querystring) {
  if (key.includes("/")) {
    event.request.querystring[encodeURIComponent(key)] = event.request.querystring[key];
    delete event.request.querystring[key];
  }
}`;
}

const __pulumiType = "sst:aws:SvelteKit";
// @ts-expect-error
SvelteKit.__pulumiType = __pulumiType;
