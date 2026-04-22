import fs from "fs/promises";
import path from "path";
import { ComponentResourceOptions, Output } from "@pulumi/pulumi";
import { VisibleError } from "../error.js";
import { Plan, SsrSite, SsrSiteArgs } from "./ssr-site.js";
import { existsAsync } from "../../util/fs.js";
import { isALteB } from "../../util/compare-semver.js";
import { getPackageVersion } from "../../util/package.js";
import {
  validateFrameworkConfig,
  validateNoWranglerFile,
} from "./helpers/validation.js";

export interface AstroArgs extends SsrSiteArgs {
  /**
   * Configure how this component works in `sst dev`.
   *
   * :::note
   * In `sst dev` your Astro site is run in dev mode; it's not deployed.
   * :::
   *
   * Instead of deploying your Astro site, this starts it in dev mode. It's run
   * as a separate process in the `sst dev` multiplexer. Read more about
   * [`sst dev`](/docs/reference/cli/#dev).
   *
   * To disable dev mode, pass in `false`.
   */
  dev?: SsrSiteArgs["dev"];
  /**
   * Path to the directory where your Astro site is located.  This path is relative to your `sst.config.ts`.
   *
   * By default it assumes your Astro site is in the root of your SST app.
   * @default `"."`
   *
   * @example
   *
   * If your Astro site is in a package in your monorepo.
   *
   * ```js
   * {
   *   path: "packages/web"
   * }
   * ```
   */
  path?: SsrSiteArgs["path"];
  /**
   * [Link resources](/docs/linking/) to your Astro site. This will:
   *
   * 1. Grant the permissions needed to access the resources.
   * 2. Allow you to access them in your site using `sst/resource`.
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
   *
   * Access linked resources in your site with
   * [`sst/resource`](/docs/reference/sdk/#sstresource). This works in both
   * `sst dev` and after deploy.
   *
   * ```astro
   * ---
   * import { Resource } from "sst/resource";
   *
   * const files = await Resource.MyBucket.list();
   * ---
   * ```
   *
   */
  link?: SsrSiteArgs["link"];
  /**
   * Set [environment variables](https://docs.astro.build/en/guides/environment-variables/) in your Astro site.
   *
   * :::tip
   * You can also `link` resources to your Astro site and access them in a type-safe way with the [SDK](/docs/reference/sdk/). We recommend linking since it's more secure.
   * :::
   *
   * Recall that in Astro, you need to prefix your environment variables with `PUBLIC_` to access them on the client-side. [Read more here](https://docs.astro.build/en/guides/environment-variables/).
   *
   * @example
   * ```js
   * {
   *   environment: {
   *     API_URL: api.url,
   *     // Accessible on the client-side
   *     PUBLIC_STRIPE_PUBLISHABLE_KEY: "pk_test_123"
   *   }
   * }
   * ```
   */
  environment?: SsrSiteArgs["environment"];
  /**
   * Set a custom domain for your Astro site.
   *
   * @example
   *
   * ```js
   * {
   *   domain: "my-app.com"
   * }
   * ```
   */
  domain?: SsrSiteArgs["domain"];
  /**
   * The command used internally to build your Astro site.
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
}

/**
 * The `Astro` component lets you deploy an [Astro](https://astro.build) site to Cloudflare.
 *
 * :::caution
 * Features like `sst dev` support and `sst/resource` bindings require Astro v6 or newer.
 * :::
 *
 * @example
 *
 * #### Minimal example
 *
 * Deploy the Astro site that's in the project root.
 *
 * ```js title="sst.config.ts"
 * new sst.cloudflare.Astro("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Deploys the Astro site in the `my-astro-app/` directory.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.cloudflare.Astro("MyWeb", {
 *   path: "my-astro-app/"
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your Astro site.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.cloudflare.Astro("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your Astro site. This will grant permissions
 * to the resources and allow you to access it in your site.
 *
 * ```ts {4} title="sst.config.ts"
 * const bucket = new sst.cloudflare.Bucket("MyBucket");
 *
 * new sst.cloudflare.Astro("MyWeb", {
 *   link: [bucket]
 * });
 * ```
 *
 * Add this to your `astro.config.mjs` for SST to work correctly.
 *
 * ```js title="astro.config.mjs"
 * import { defineConfig } from "astro/config";
 * import cloudflare from "@astrojs/cloudflare";
 *
 * export default defineConfig({
 *   adapter: cloudflare({
 *     configPath: process.env.SST_WRANGLER_PATH,
 *   }),
 * });
 * ```
 *
 * Use `sst/resource` for linked resources.
 *
 * ```astro title="src/pages/index.astro"
 * ---
 * import { Resource } from "sst/resource";
 *
 * const files = await Resource.MyBucket.list();
 * ---
 * ```
 *
 * [Check out the full example](/docs/examples/#cloudflare-astro).
 */
export class Astro extends SsrSite {
  constructor(
    name: string,
    args: AstroArgs = {},
    opts: ComponentResourceOptions = {},
  ) {
    super(__pulumiType, name, args, opts);
  }

  protected validate(sitePath: string): void {
    // Only validate configPath requirement for Astro v6+
    // If version cannot be determined, default to v6+ (validate)
    const astroVersion = getPackageVersion(sitePath, "astro");
    const isV6Plus = !astroVersion || isALteB("6.0.0", astroVersion);

    if (isV6Plus) {
      validateFrameworkConfig({
        sitePath,
        configName: "astro.config",
        componentName: "Astro",
      });
    }
    validateNoWranglerFile(sitePath, "Astro");
  }

  protected buildPlan(outputPath: Output<string>): Output<Plan> {
    return outputPath.apply(async (outputPath) => {
      const distPath = path.join(outputPath, "dist");
      const legacyServer = path.join(distPath, "_worker.js", "index.js");
      if (await existsAsync(legacyServer)) {
        // Astro v5 writes both the Worker bundle and assets into `dist/`.
        const ignorePath = path.join(distPath, ".assetsignore");
        const ignorePatterns = (await existsAsync(ignorePath))
          ? (await fs.readFile(ignorePath, "utf-8")).split("\n")
          : [];
        let dirty = false;
        ["_worker.js", "_routes.json"].forEach((pattern) => {
          if (ignorePatterns.includes(pattern)) return;
          ignorePatterns.push(pattern);
          dirty = true;
        });

        if (dirty) {
          await fs.appendFile(ignorePath, "\n_worker.js\n_routes.json");
        }

        return {
          server: "./dist/_worker.js/index.js",
          assets: "./dist",
        };
      }

      const wranglerPath = path.join(distPath, "server", "wrangler.json");
      if (await existsAsync(wranglerPath)) {
        type WranglerConfig = {
          main?: string;
          assets?: {
            directory?: string;
          };
        };

        const serverPath = path.dirname(wranglerPath);
        const wrangler = JSON.parse(
          await fs.readFile(wranglerPath, "utf-8"),
        ) as WranglerConfig;
        const main = wrangler.main ?? "entry.mjs";
        const serverEntry = path.resolve(serverPath, main);
        const assetsDirectory = wrangler.assets?.directory;
        const assetsPath = assetsDirectory
          ? path.resolve(serverPath, assetsDirectory)
          : path.join(distPath, "client");

        if (await existsAsync(serverEntry)) {
          return {
            server: toRelativePath(serverEntry),
            assets: toRelativePath(assetsPath),
          };
        }
      }

      throw new VisibleError(
        `SSR server bundle not found in the build output at:\n` +
          `  "${path.resolve(distPath)}".\n\n` +
          `Expected either Astro v5 output in \`dist/_worker.js/index.js\` or Astro v6+ output in \`dist/server/wrangler.json\`.\n` +
          `If your Astro project is entirely pre-rendered, use the \`sst.cloudflare.StaticSite\` component instead of \`sst.cloudflare.Astro\`.`,
      );

      function toRelativePath(filePath: string) {
        const relative = path.relative(outputPath, filePath);
        return relative.startsWith(".") ? relative : `./${relative}`;
      }
    });
  }

  /**
   * The URL of the Astro site.
   *
   * If the `domain` is set, this is the URL with the custom domain.
   * Otherwise, it's the auto-generated Worker URL.
   */
  public get url() {
    return super.url;
  }
}
const __pulumiType = "sst:cloudflare:Astro";
// @ts-expect-error
Astro.__pulumiType = __pulumiType;
