import fs from "fs/promises";
import path from "path";
import * as cf from "@pulumi/cloudflare";
import { all, interpolate } from "@pulumi/pulumi";
import type { ComponentResourceOptions } from "@pulumi/pulumi";
import { Component } from "../../component.js";
import { Link } from "../../link.js";
import type { Input } from "../../input.js";
import { WorkerUrl } from "../providers/worker-url.js";
import { ZoneLookup } from "../providers/zone-lookup.js";
import { DEFAULT_ACCOUNT_ID } from "../account-id.js";
import { physicalName } from "../../naming.js";
import {
  buildApp,
  prepare,
} from "../../base/base-static-site.js";
import type { BaseStaticSiteArgs } from "../../base/base-static-site.js";

type StaticSiteAssets = {
  /**
   * The SHA-256 hash of the asset manifest of files to upload.
   */
  assetManifestSha256?: Input<string>;
  /**
   * The directory containing assets. Defaults to this site's build output.
   */
  directory?: Input<string>;
  /**
   * Token provided upon successful upload of all files from a registered manifest.
   */
  jwt?: Input<string>;
  /**
   * The contents of a `_headers` file.
   */
  headers?: Input<string>;
  /**
   * The contents of a `_redirects` file.
   */
  redirects?: Input<string>;
  /**
   * Determines how HTML files are handled.
   */
  htmlHandling?: Input<
    | "auto-trailing-slash"
    | "force-trailing-slash"
    | "drop-trailing-slash"
    | "none"
  >;
  /**
   * Determines how requests that don't match a static asset are handled.
   */
  notFoundHandling?: Input<"single-page-application" | "404-page" | "none">;
  /**
   * Controls whether the Worker script runs before serving static assets.
   */
  runWorkerFirst?: Input<boolean | string[]>;
};

export interface StaticSiteArgs extends BaseStaticSiteArgs {
  /**
   * Path to the directory where your static site is located. By default this assumes your static site is in the root of your SST app.
   *
   * This directory will be uploaded as static assets. The path is relative to your `sst.config.ts`.
   *
   * :::note
   * If the `build` options are specified, `build.output` will be uploaded instead.
   * :::
   *
   * If you are using a static site generator, like Vite, you'll need to configure the `build` options. When these are set, the `build.output` directory will be uploaded instead.
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
   * The `build.output` directory will be uploaded instead.
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
  /**
   * Configure static assets uploaded with the Worker.
   *
   * Matches Cloudflare `workersScript.assets` options.
   */
  assets?: Input<StaticSiteAssets>;
}

/**
 * The `StaticSite` component lets you deploy a static website to Cloudflare. It uses [Cloudflare Workers static assets](https://developers.cloudflare.com/workers/static-assets/) to store and serve your files.
 *
 * It can also `build` your site by running your static site generator, like [Vite](https://vitejs.dev) and uploading the build output to Cloudflare.
 *
 * @example
 *
 * #### Minimal example
 *
 * Simply uploads the current directory as a static site.
 *
 * ```js
 * new sst.cloudflare.StaticSite("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Change the `path` that should be uploaded.
 *
 * ```js
 * new sst.cloudflare.StaticSite("MyWeb", {
 *   path: "path/to/site"
 * });
 * ```
 *
 * #### Deploy a Vite SPA
 *
 * Use [Vite](https://vitejs.dev) to deploy a React/Vue/Svelte/etc. SPA by specifying the `build` config.
 *
 * ```js
 * new sst.cloudflare.StaticSite("MyWeb", {
 *   build: {
 *     command: "npm run build",
 *     output: "dist"
 *   },
 *   assets: {
 *     notFoundHandling: "single-page-application"
 *   }
 * });
 * ```
 *
 * #### Deploy a Jekyll site
 *
 * Use [Jekyll](https://jekyllrb.com) to deploy a static site.
 *
 * ```js
 * new sst.cloudflare.StaticSite("MyWeb", {
 *   build: {
 *     command: "bundle exec jekyll build",
 *     output: "_site"
 *   },
 *   assets: {
 *     notFoundHandling: "404-page"
 *   }
 * });
 * ```
 *
 * #### Deploy a Gatsby site
 *
 * Use [Gatsby](https://www.gatsbyjs.com) to deploy a static site.
 *
 * ```js
 * new sst.cloudflare.StaticSite("MyWeb", {
 *   build: {
 *     command: "npm run build",
 *     output: "public"
 *   },
 *   assets: {
 *     notFoundHandling: "404-page"
 *   }
 * });
 * ```
 *
 * #### Deploy an Angular SPA
 *
 * Use [Angular](https://angular.dev) to deploy a SPA.
 *
 * ```js
 * new sst.cloudflare.StaticSite("MyWeb", {
 *   build: {
 *     command: "ng build --output-path dist",
 *     output: "dist"
 *   },
 *   assets: {
 *     notFoundHandling: "single-page-application"
 *   }
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your site.
 *
 * ```js {2}
 * new sst.cloudflare.StaticSite("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Redirect www to apex domain
 *
 * Redirect `www.my-app.com` to `my-app.com`.
 *
 * ```js {4}
 * new sst.cloudflare.StaticSite("MyWeb", {
 *   domain: {
 *     name: "my-app.com",
 *     redirects: ["www.my-app.com"]
 *   }
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
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.cloudflare.StaticSite("MyWeb", {
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
  private script: cf.WorkersScript;
  private workerUrl: WorkerUrl;
  private workerDomain?: cf.WorkersCustomDomain;

  constructor(
    name: string,
    args: StaticSiteArgs = {},
    opts: ComponentResourceOptions = {},
  ) {
    super(__pulumiType, name, args, opts);

    const parent = this;
    const { sitePath, environment } = prepare(args);
    const outputPath = $dev
      ? path.join($cli.paths.platform, "functions", "empty-site")
      : buildApp(parent, name, args.build, sitePath, environment);
    const script = createScript();
    const workerUrl = createWorkersUrl();
    const workerDomain = createWorkersDomain();

    this.script = script;
    this.workerUrl = workerUrl;
    this.workerDomain = workerDomain;

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

    function createScript() {
      return new cf.WorkersScript(
        `${name}Script`,
        {
          scriptName: physicalName(64, `${name}Script`).toLowerCase(),
          accountId: DEFAULT_ACCOUNT_ID,
          compatibilityDate: "2025-05-05",
          content: "export default { fetch: (request, env) => env.ASSETS.fetch(request) };",
          mainModule: "worker.js",
          assets: all([outputPath, args.assets]).apply(
            async ([dir, assets]) => {
              const directory = path.join(
                $cli.paths.root,
                assets?.directory ?? dir,
              );
              const htmlHandling = assets?.htmlHandling;
              const notFoundHandling = assets?.notFoundHandling;
              const runWorkerFirst = assets?.runWorkerFirst;

              let headers = assets?.headers;
              if (!headers) {
                try {
                  headers = await fs.readFile(
                    path.join(directory, "_headers"),
                    "utf-8",
                  );
                } catch (e) {}
              }

              return {
                directory,
                ...(assets?.assetManifestSha256
                  ? { assetManifestSha256: assets.assetManifestSha256 }
                  : {}),
                ...(assets?.jwt ? { jwt: assets.jwt } : {}),
                config: {
                  ...(headers ? { headers } : {}),
                  ...(assets?.redirects ? { redirects: assets.redirects } : {}),
                  ...(htmlHandling
                    ? { htmlHandling }
                    : {}),
                  ...(notFoundHandling
                    ? { notFoundHandling }
                    : {}),
                  ...(runWorkerFirst !== undefined
                    ? { runWorkerFirst }
                    : {}),
                },
              };
            },
          ),
          bindings: environment.apply((env) => [
            {
              type: "assets" as const,
              name: "ASSETS",
            },
            ...Object.entries(env).map(([key, value]) => ({
              type: "plain_text" as const,
              name: key,
              text: value,
            })),
          ]),
        },
        { parent, ignoreChanges: ["scriptName"] },
      );
    }

    function createWorkersUrl() {
      return new WorkerUrl(
        `${name}Url`,
        {
          accountId: DEFAULT_ACCOUNT_ID,
          scriptName: script.scriptName,
          enabled: true,
        },
        { parent },
      );
    }

    function createWorkersDomain() {
      if (!args.domain) return;

      const zone = new ZoneLookup(
        `${name}ZoneLookup`,
        {
          accountId: DEFAULT_ACCOUNT_ID,
          domain: args.domain,
        },
        { parent },
      );

      return new cf.WorkersCustomDomain(
        `${name}Domain`,
        {
          accountId: DEFAULT_ACCOUNT_ID,
          service: script.scriptName,
          hostname: args.domain,
          zoneId: zone.id,
          environment: "production",
        },
        { parent },
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
    return this.workerDomain
      ? interpolate`https://${this.workerDomain.hostname}`
      : this.workerUrl.url.apply((url) => (url ? `https://${url}` : url));
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The Cloudflare Worker script.
       */
      script: this.script,
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
