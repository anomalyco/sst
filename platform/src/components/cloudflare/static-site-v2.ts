import path from "path";
import { ComponentResourceOptions, output } from "@pulumi/pulumi";
import { Component, transform, type Transform } from "../component.js";
import { Link } from "../link.js";
import { Input } from "../input.js";
import { URL_UNAVAILABLE } from "../aws/linkable.js";
import { Worker, WorkerArgs } from "./worker.js";
import {
  BaseStaticSiteArgs,
  buildApp,
  prepare,
} from "../base/base-static-site.js";
import type { BaseSiteDev } from "../base/base-site.js";
import type { Prettify } from "../component.js";

export interface StaticSiteV2Args extends Omit<BaseStaticSiteArgs, "vite"> {
  /**
   * Configure how this component works in `sst dev`.
   *
   * :::note
   * In `sst dev` your static site is run in dev mode; it's not deployed.
   * :::
   *
   * Instead of deploying your static site, this starts it in dev mode. It's run
   * as a separate process in the `sst dev` multiplexer. Read more about
   * [`sst dev`](/docs/reference/cli/#dev).
   *
   * To disable dev mode, pass in `false`.
   *
   * @example
   * Use a custom dev command.
   * ```js
   * {
   *   dev: {
   *     command: "yarn dev"
   *   }
   * }
   * ```
   */
  dev?: false | Prettify<BaseSiteDev>;
  /**
   * Path to the directory where your static site is located. By default this assumes your static site is in the root of your SST app.
   *
   * This directory will be uploaded as static assets. The path is relative to your `sst.config.ts`.
   *
   * :::note
   * If the `build` options are specified, `build.output` will be uploaded as static assets instead.
   * :::
   *
   * If you are using a static site generator, like Vite, you'll need to configure the `build` options. When these are set, the `build.output` directory will be uploaded as static assets instead.
   *
   * @default `"."`
   *
   * @example
   *
   * Change where your static site is located.
   *
   * ```js
   * {
   *   path: "packages/web"
   * }
   * ```
   */
  path?: BaseStaticSiteArgs["path"];
  /**
   * Configure if your static site needs to be built. This is useful if you are using a static site generator.
   *
   * The `build.output` directory will be uploaded as static assets.
   *
   * @example
   * For a Vite project using npm this might look like this.
   *
   * ```js
   * {
   *   build: {
   *     command: "npm run build",
   *     output: "dist"
   *   }
   * }
   * ```
   */
  build?: BaseStaticSiteArgs["build"];
  /**
   * Set a custom domain for your static site. Supports domains hosted on Cloudflare.
   *
   * :::tip
   * You can migrate an externally hosted domain to Cloudflare by
   * [following this guide](https://developers.cloudflare.com/dns/zone-setups/full-setup/setup/).
   * :::
   *
   * @example
   *
   * ```js
   * {
   *   domain: "domain.com"
   * }
   * ```
   *
   * Redirect alternate domains to the main domain.
   *
   * ```js
   * {
   *   domain: {
   *     name: "domain.com",
   *     redirects: ["www.domain.com"]
   *   }
   * }
   * ```
   *
   * Or keep visitors on alternate domains with aliases.
   *
   * ```js
   * {
   *   domain: {
   *     name: "app1.domain.com",
   *     aliases: ["app2.domain.com"]
   *   }
   * }
   * ```
   */
  domain?: WorkerArgs["domain"];
  /**
   * Configure trailing slash behavior for HTML pages.
   *
   * - `"auto"`: Individual files served without slash, folder indexes with slash
   * - `"force"`: All HTML pages served with trailing slash
   * - `"drop"`: All HTML pages served without trailing slash
   *
   * @default `"auto"`
   *
   * @example
   *
   * #### Force trailing slashes
   *
   * ```js
   * {
   *   trailingSlash: "force"
   * }
   * ```
   */
  trailingSlash?: "auto" | "force" | "drop";
  /**
   * Configure the response when a request does not match a static asset.
   *
   * - `"single-page-application"`: Serve `index.html` for unmatched routes (SPA mode)
   * - `"404"`: Serve the nearest `404.html` file with a `404` status
   *
   * @default `"single-page-application"`
   */
  notFound?: Input<"single-page-application" | "404">;
  /** @deprecated */
  indexPage?: string;
  /** @deprecated */
  errorPage?: Input<string>;
  /**
   * [Transform](/docs/components#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the Worker component used for serving the static site.
     */
    server?: Transform<WorkerArgs>;
  };
}

/**
 * The `StaticSiteV2` component lets you deploy a static website to Cloudflare. It uses [Cloudflare Workers](https://developers.cloudflare.com/workers/) with [static assets](https://developers.cloudflare.com/workers/static-assets/) to serve your files.
 *
 * It can also `build` your site by running your static site generator, like [Vite](https://vitejs.dev) and uploading the build output as static assets.
 *
 * @example
 *
 * #### Minimal example
 *
 * Simply uploads the current directory as a static site.
 *
 * ```js
 * new sst.cloudflare.StaticSiteV2("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Change the `path` that should be uploaded.
 *
 * ```js
 * new sst.cloudflare.StaticSiteV2("MyWeb", {
 *   path: "path/to/site"
 * });
 * ```
 *
 * #### Deploy a Vite SPA
 *
 * Use [Vite](https://vitejs.dev) to deploy a React/Vue/Svelte/etc. SPA by specifying the `build` config.
 *
 * ```js
 * new sst.cloudflare.StaticSiteV2("MyWeb", {
 *   build: {
 *     command: "npm run build",
 *     output: "dist"
 *   },
 *   notFound: "single-page-application"
 * });
 * ```
 *
 * #### Deploy a Jekyll site
 *
 * Use [Jekyll](https://jekyllrb.com) to deploy a static site.
 *
 * ```js
 * new sst.cloudflare.StaticSiteV2("MyWeb", {
 *   build: {
 *     command: "bundle exec jekyll build",
 *     output: "_site"
 *   }
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your site.
 *
 * ```js {2}
 * new sst.cloudflare.StaticSiteV2("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Redirect www to apex domain
 *
 * Redirect `www.my-app.com` to `my-app.com`.
 *
 * ```js {4}
 * new sst.cloudflare.StaticSiteV2("MyWeb", {
 *   domain: {
 *     name: "my-app.com",
 *     redirects: ["www.my-app.com"]
 *   }
 * });
 * ```
 *
 * #### Add domain aliases
 *
 * Allow visitors to use alternate domains without redirecting.
 *
 * ```js {4}
 * new sst.cloudflare.StaticSiteV2("MyWeb", {
 *   domain: {
 *     name: "app1.my-app.com",
 *     aliases: ["app2.my-app.com"]
 *   }
 * });
 * ```
 *
 * #### Set environment variables
 *
 * Set `environment` variables for the build process of your static site. These will be used locally and on deploy.
 *
 * For some static site generators like Vite, [environment variables](https://vitejs.dev/guide/env-and-mode) prefixed with `VITE_` can be accessed in the browser.
 *
 * ```ts {5-7}
 * const bucket = new sst.cloudflare.Bucket("MyBucket");
 *
 * new sst.cloudflare.StaticSiteV2("MyWeb", {
 *   environment: {
 *     BUCKET_NAME: bucket.name,
 *     // Accessible in the browser
 *     VITE_STRIPE_PUBLISHABLE_KEY: "pk_test_123"
 *   },
 *   build: {
 *     command: "npm run build",
 *     output: "dist"
 *   }
 * });
 * ```
 */
export class StaticSiteV2 extends Component implements Link.Linkable {
  private server?: Worker;
  private devUrl = output(URL_UNAVAILABLE);

  constructor(
    name: string,
    args: StaticSiteV2Args = {},
    opts: ComponentResourceOptions = {},
  ) {
    super(__pulumiType, name, args, opts);

    const self = this;
    const { sitePath, environment, indexPage } = prepare(args);
    const dev = normalizeDev();

    if (dev.enabled) {
      this.devUrl = dev.url;
      this.registerOutputs({
        _hint: undefined,
        _dev: dev.outputs,
        _metadata: {
          mode: "placeholder",
          path: sitePath,
          environment,
          url: this.url,
        },
      });
      return;
    }

    function normalizeDev() {
      const enabled = $dev && args.dev !== false;
      const devArgs = args.dev || {};

      return {
        enabled,
        url: output(devArgs.url ?? URL_UNAVAILABLE),
        outputs: {
          title: devArgs.title,
          environment,
          command: output(devArgs.command ?? "npm run dev"),
          autostart: output(devArgs.autostart ?? true),
          directory: output(devArgs.directory ?? sitePath),
        },
      };
    }

    const htmlHandling = normalizeHtmlHandling();
    const notFound = normalizeNotFound();
    const outputPath = buildApp(self, name, args.build, sitePath, environment);
    const worker = createRouter();

    this.server = worker;

    this.registerOutputs({
      _hint: this.url,
      _dev: dev.outputs,
      _metadata: {
        mode: "deployed",
        path: sitePath,
        environment,
        url: this.url,
      },
    });

    function createRouter() {
      return new Worker(
        ...transform(
          args.transform?.server,
          `${name}Router`,
          {
            handler: path.join(
              $cli.paths.platform,
              "functions",
              "cf-static-site-router-worker-experimental",
            ),
            environment: environment.apply((e) => ({
              ...e,
              ...(args.indexPage || args.errorPage
                ? { INDEX_PAGE: indexPage }
                : {}),
              ...(args.errorPage ? { ERROR_PAGE: args.errorPage } : {}),
            })),
            url: true,
            dev: false,
            domain: args.domain,
            assets: {
              directory: outputPath,
              htmlHandling,
              notFoundHandling: notFound,
            },
          },
          { parent: self },
        ),
      );
    }

    function normalizeHtmlHandling() {
      return args.trailingSlash === "force"
        ? "force-trailing-slash"
        : args.trailingSlash === "drop"
          ? "drop-trailing-slash"
          : "auto-trailing-slash";
    }

    function normalizeNotFound() {
      return args.notFound === "404" ? "404-page" : "single-page-application";
    }
  }

  /**
   * The URL of the website.
   *
   * If the `domain` is set, this is the URL with the custom domain.
   * Otherwise, it's the auto-generated worker URL.
   */
  public get url() {
    return this.server?.url ?? this.devUrl;
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The worker that serves the requests.
       */
      server: this.server,
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

const __pulumiType = "sst:cloudflare:StaticSite";
// @ts-expect-error
StaticSiteV2.__pulumiType = __pulumiType;
