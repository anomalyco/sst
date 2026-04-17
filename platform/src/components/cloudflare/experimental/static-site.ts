import path from "path";
import { ComponentResourceOptions, output } from "@pulumi/pulumi";
import { Component, transform, type Transform } from "../../component.js";
import { Link } from "../../link.js";
import { Input } from "../../input.js";
import { Worker, WorkerArgs } from "../worker.js";
import {
  BaseStaticSiteArgs,
  buildApp,
  prepare,
} from "../../base/base-static-site.js";

export interface StaticSiteArgs
  extends Omit<BaseStaticSiteArgs, "vite"> {
  /**
   * Path to the directory where your static site is located. By default this assumes your static site is in the root of your SST app.
   *
   * This directory will be uploaded to KV. The path is relative to your `sst.config.ts`.
   *
   * :::note
   * If the `build` options are specified, `build.output` will be uploaded to KV instead.
   * :::
   *
   * If you are using a static site generator, like Vite, you'll need to configure the `build` options. When these are set, the `build.output` directory will be uploaded to KV instead.
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
   * The `build.output` directory will be uploaded to KV instead.
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
   */
  domain?: Input<string>;
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
   * - `"404-page"`: Serve the nearest `404.html` file with a `404` status
   * - `"single-page-application"`: Serve `index.html` with a `200` status for SPAs
   * - `"none"`: Return a `404` without a body
   *
   * @example
   *
   * #### Deploy a Single Page Application
   *
   * For SPAs like React, Vue, or Svelte apps, use `single-page-application` to
   * serve `index.html` for all navigation requests that don't match an asset.
   *
   * ```js
   * new sst.cloudflare.x.StaticSite("MyWeb", {
   *   build: {
   *     command: "npm run build",
   *     output: "dist"
   *   },
   *   notFoundHandling: "single-page-application"
   * });
   * ```
   *
   * #### Use a custom 404 page
   *
   * For static sites with a custom 404 page, use `404-page` to serve `404.html`
   * when a file is not found.
   *
   * ```js
   * new sst.cloudflare.x.StaticSite("MyWeb", {
   *   notFoundHandling: "404-page"
   * });
   * ```
   */
  notFoundHandling?: Input<"404-page" | "single-page-application" | "none">;
}

/**
 * The `StaticSite` component lets you deploy a static website to Cloudflare. It uses [Cloudflare KV storage](https://developers.cloudflare.com/kv/) to store your files and [Cloudflare Workers](https://developers.cloudflare.com/workers/) to serve them.
 *
 * It can also `build` your site by running your static site generator, like [Vite](https://vitejs.dev) and uploading the build output to Cloudflare KV.
 *
 * @example
 *
 * #### Minimal example
 *
 * Simply uploads the current directory as a static site.
 *
 * ```js
 * new sst.cloudflare.x.StaticSite("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Change the `path` that should be uploaded.
 *
 * ```js
 * new sst.cloudflare.x.StaticSite("MyWeb", {
 *   path: "path/to/site"
 * });
 * ```
 *
 * #### Deploy a Vite SPA
 *
 * Use [Vite](https://vitejs.dev) to deploy a React/Vue/Svelte/etc. SPA by specifying the `build` config.
 *
 * ```js
 * new sst.cloudflare.x.StaticSite("MyWeb", {
 *   build: {
 *     command: "npm run build",
 *     output: "dist"
 *   },
 *  notFoundHandling: "single-page-application"
 * });
 * ```
 *
 * #### Deploy a Jekyll site
 *
 * Use [Jekyll](https://jekyllrb.com) to deploy a static site.
 *
 * ```js
 * new sst.cloudflare.x.StaticSite("MyWeb", {
 *   build: {
 *     command: "bundle exec jekyll build",
 *     output: "_site"
 *   }
 * });
 * ```
 *
 * #### Deploy a Gatsby site
 *
 * Use [Gatsby](https://www.gatsbyjs.com) to deploy a static site.
 *
 * ```js
 * new sst.cloudflare.x.StaticSite("MyWeb", {
 *   build: {
 *     command: "npm run build",
 *     output: "public"
 *   }
 * });
 * ```
 *
 * #### Deploy an Angular SPA
 *
 * Use [Angular](https://angular.dev) to deploy a SPA.
 *
 * ```js
 * new sst.cloudflare.x.StaticSite("MyWeb", {
 *   build: {
 *     command: "ng build --output-path dist",
 *     output: "dist"
 *   }
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your site.
 *
 * ```js {2}
 * new sst.cloudflare.x.StaticSite("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Set environment variables
 *
 * Set `environment` variables for the build process of your static site. These will be used locally and on deploy.
 *
 * :::tip
 * For Vite, the types for the environment variables are also generated. This can be configured through the `vite` prop.
 * :::
 *
 * For some static site generators like Vite, [environment variables](https://vitejs.dev/guide/env-and-mode) prefixed with `VITE_` can be accessed in the browser.
 *
 * ```ts {5-7}
 * const bucket = new sst.cloudflare.Bucket("MyBucket");
 *
 * new sst.cloudflare.x.StaticSite("MyWeb", {
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
export class StaticSite extends Component implements Link.Linkable {
  private server: Worker;

  constructor(
    name: string,
    args: StaticSiteArgs = {},
    opts: ComponentResourceOptions = {},
  ) {
    super(__pulumiType, name, args, opts);

    const self = this;
    const { sitePath, environment, indexPage } = prepare(args);
    const htmlHandling = normalizeHtmlHandling();
    const outputPath = $dev
      ? path.join($cli.paths.platform, "functions", "empty-site")
      : buildApp(self, name, args.build, sitePath, environment);
    const worker = createRouter();

    this.server = worker;

    this.registerOutputs({
      _hint: $dev ? undefined : this.url,
      _dev: {
        environment,
        command: "npm run dev",
        directory: sitePath,
        autostart: true,
      },
      _metadata: {
        mode: $dev ? "placeholder" : "deployed",
        path: sitePath,
        environment,
        url: this.url,
      },
    });

    function normalizeHtmlHandling() {
      return args.trailingSlash === "force"
        ? "force-trailing-slash"
        : args.trailingSlash === "drop"
          ? "drop-trailing-slash"
          : "auto-trailing-slash";
    }

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
              htmlHandling: htmlHandling,
              notFoundHandling: args.notFoundHandling,
            },
          },
          { parent: self },
        ),
      );
    }
  }

  /**
   * The URL of the website.
   *
   * If the `domain` is set, this is the URL with the custom domain.
   * Otherwise, it's the auto-generated worker URL.
   */
  public get url() {
    return this.server.url;
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
StaticSite.__pulumiType = __pulumiType;
