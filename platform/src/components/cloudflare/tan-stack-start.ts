import fs from "fs/promises";
import path from "path";
import { ComponentResourceOptions, Output } from "@pulumi/pulumi";
import { VisibleError } from "../error.js";
import { Plan, SsrSite, SsrSiteArgs } from "./ssr-site.js";
import { existsAsync } from "../../util/fs.js";
import {
  validateFrameworkConfig,
  validateNoWranglerFile,
} from "./helpers/validation.js";

export interface TanStackStartArgs extends SsrSiteArgs {
  /**
   * Configure how this component works in `sst dev`.
   *
   * :::note
   * In `sst dev` your TanStack Start app is run in dev mode; it's not deployed.
   * :::
   *
   * Instead of deploying your TanStack Start app, this starts it in dev mode. It's run
   * as a separate process in the `sst dev` multiplexer. Read more about
   * [`sst dev`](/docs/reference/cli/#dev).
   *
   * To disable dev mode, pass in `false`.
   */
  dev?: SsrSiteArgs["dev"];
  /**
   * Path to the directory where your TanStack Start app is located. This path is relative to your `sst.config.ts`.
   *
   * By default it assumes your TanStack Start app is in the root of your SST app.
   * @default `"."`
   *
   * @example
   *
   * If your TanStack Start app is in a package in your monorepo.
   *
   * ```js
   * {
   *   path: "packages/web"
   * }
   * ```
   */
  path?: SsrSiteArgs["path"];
  /**
   * [Link resources](/docs/linking/) to your TanStack Start app. This will:
   *
   * 1. Grant the permissions needed to access the resources.
   * 2. Allow you to access it in your app using the [SDK](/docs/reference/sdk/).
   *
   * @example
   *
   * Takes a list of resources to link to the app.
   *
   * ```js
   * {
   *   link: [bucket, stripeKey]
   * }
   * ```
   *
   * You can access the linked resources in your TanStack Start app.
   *
   * ```ts
   * import { Resource } from "sst";
   *
   * console.log(Resource.MyBucket.name);
   * ```
   */
  link?: SsrSiteArgs["link"];
  /**
   * Set environment variables in your TanStack Start app. These are made available:
   *
   * 1. In `vite build`, they are loaded into the build.
   * 2. At runtime as Worker bindings.
   * 3. Locally while running `vite dev` through `sst dev`.
   *
   * :::tip
   * You can also `link` resources to your TanStack Start app and access them in a type-safe way with the [SDK](/docs/reference/sdk/). We recommend linking since it's more secure.
   * :::
   *
   * @example
   * ```js
   * {
   *   environment: {
   *     API_URL: api.url,
   *     PUBLIC_STRIPE_PUBLISHABLE_KEY: "pk_test_123"
   *   }
   * }
   * ```
   *
   * You can access the environment variables in your TanStack Start app as follows:
   *
   * ```ts
   * import { env } from "cloudflare:workers";
   *
   * const apiUrl = env.API_URL;
   * ```
   */
  environment?: SsrSiteArgs["environment"];
  /**
   * Set a custom domain for your TanStack Start app.
   *
   * @example
   *
   * ```js
   * {
   *   domain: "my-app.com"
   * }
   * ```
   *
   * Redirect alternate domains to the main domain.
   *
   * ```js
   * {
   *   domain: {
   *     name: "my-app.com",
   *     redirects: ["www.my-app.com"]
   *   }
   * }
   * ```
   *
   * Or keep visitors on alternate domains with aliases.
   *
   * ```js
   * {
   *   domain: {
   *     name: "app1.my-app.com",
   *     aliases: ["app2.my-app.com"]
   *   }
   * }
   * ```
   */
  domain?: SsrSiteArgs["domain"];
  /**
   * The command used internally to build your TanStack Start app.
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
 * The `TanStackStart` component lets you deploy a [TanStack Start](https://tanstack.com/start/latest) app to Cloudflare.
 *
 * :::note
 * Create a Cloudflare-compatible app with `bunx @tanstack/cli@latest create --deployment cloudflare`.
 * :::
 *
 * @example
 *
 * #### Minimal example
 *
 * Deploy the TanStack Start app that's in the project root.
 *
 * ```js title="sst.config.ts"
 * new sst.cloudflare.TanStackStart("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Deploys the TanStack Start app in the `my-app/` directory.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.cloudflare.TanStackStart("MyWeb", {
 *   path: "my-app/"
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your TanStack Start app.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.cloudflare.TanStackStart("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Redirect www to apex domain
 *
 * Redirect `www.my-app.com` to `my-app.com`.
 *
 * ```js {4} title="sst.config.ts"
 * new sst.cloudflare.TanStackStart("MyWeb", {
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
 * ```js {4} title="sst.config.ts"
 * new sst.cloudflare.TanStackStart("MyWeb", {
 *   domain: {
 *     name: "app1.my-app.com",
 *     aliases: ["app2.my-app.com"]
 *   }
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your TanStack Start app. This will grant permissions
 * to the resources and allow you to access it in your app.
 *
 * ```ts {4} title="sst.config.ts"
 * const bucket = new sst.cloudflare.Bucket("MyBucket");
 *
 * new sst.cloudflare.TanStackStart("MyWeb", {
 *   link: [bucket]
 * });
 * ```
 *
 * Add this to your `vite.config.ts` for SST to work correctly.
 *
 * ```ts title="vite.config.ts"
 * import { defineConfig } from 'vite'
 * import { tanstackStart } from '@tanstack/react-start/plugin/vite'
 * import { cloudflare } from '@cloudflare/vite-plugin'
 *
 * export default defineConfig({
 *   plugins: [
 *     cloudflare({
 *       viteEnvironment: { name: 'ssr' },
 *       configPath: process.env.SST_WRANGLER_PATH,
 *     }),
 *     tanstackStart(),
 *   ],
 * })
 * ```
 *
 * Use `sst/resource` for linked resources.
 *
 * ```ts title="src/routes/api.ts"
 * import { Resource } from "sst/resource";
 *
 * const files = await Resource.MyBucket.list();
 * ```
 *
 * [Check out the full example](/docs/examples/#cloudflare-tanstack-start).
 */
export class TanStackStart extends SsrSite {
  constructor(
    name: string,
    args: TanStackStartArgs = {},
    opts: ComponentResourceOptions = {},
  ) {
    super(__pulumiType, name, args, opts);
  }

  protected validate(sitePath: string): void {
    validateFrameworkConfig({
      sitePath,
      configName: "vite.config",
      componentName: "TanStackStart",
    });
    validateNoWranglerFile(sitePath, "TanStackStart");
  }

  protected buildPlan(outputPath: Output<string>): Output<Plan> {
    return outputPath.apply(async (outputPath) => {
      const distPlan = await resolveDistPlan();
      if (distPlan) return distPlan;

      const legacyPlan = await resolveLegacyPlan();
      if (legacyPlan) return legacyPlan;

      throw new VisibleError(
        [
          `TanStack Start build output not found in "${path.resolve(outputPath)}".`,
          "",
          "Expected one of:",
          "  - dist/server/wrangler.json and dist/client",
          "  - dist/server/index.js and dist/client",
          "  - .output/server/index.mjs and .output/public",
          "",
          "Use the Cloudflare deployment adapter in TanStack Start, or if your project is entirely pre-rendered, use `sst.cloudflare.StaticSite` instead.",
        ].join("\n"),
      );

      async function resolveDistPlan() {
        const wranglerPath = path.join(
          outputPath,
          "dist",
          "server",
          "wrangler.json",
        );
        if (await existsAsync(wranglerPath)) {
          const wrangler = JSON.parse(await fs.readFile(wranglerPath, "utf-8")) as {
            main?: string;
            assets?: {
              directory?: string;
            };
          };
          if (!wrangler.main) {
            throw new VisibleError(
              `TanStack Start build output at "${path.resolve(wranglerPath)}" is missing the Worker entry in \`main\`.`,
            );
          }

          const serverPath = path.resolve(outputPath, "dist", "server", wrangler.main);
          const assetsPath = path.resolve(
            outputPath,
            "dist",
            "server",
            wrangler.assets?.directory ?? "../client",
          );

          if (!(await existsAsync(serverPath))) {
            throw new VisibleError(
              `TanStack Start server bundle not found at:\n  "${serverPath}".`,
            );
          }
          if (!(await existsAsync(assetsPath))) {
            throw new VisibleError(
              `TanStack Start assets directory not found at:\n  "${assetsPath}".`,
            );
          }

          return {
            server: toPlanPath(serverPath),
            assets: toPlanPath(assetsPath),
          };
        }

        for (const serverFile of ["index.js", "index.mjs"]) {
          const serverPath = path.join(
            outputPath,
            "dist",
            "server",
            serverFile,
          );
          const assetsPath = path.join(outputPath, "dist", "client");
          if (
            (await existsAsync(serverPath)) &&
            (await existsAsync(assetsPath))
          ) {
            return {
              server: toPlanPath(serverPath),
              assets: toPlanPath(assetsPath),
            };
          }
        }
      }

      async function resolveLegacyPlan() {
        const serverPath = path.join(
          outputPath,
          ".output",
          "server",
          "index.mjs",
        );
        const assetsPath = path.join(outputPath, ".output", "public");

        if (!(await existsAsync(serverPath))) return;
        if (!(await existsAsync(assetsPath))) {
          throw new VisibleError(
            `TanStack Start assets directory not found at:\n  "${path.resolve(
              assetsPath,
            )}".`,
          );
        }

        return {
          server: toPlanPath(serverPath),
          assets: toPlanPath(assetsPath),
        };
      }

      function toPlanPath(filePath: string) {
        const relativePath = path.relative(outputPath, filePath);
        return `./${relativePath.split(path.sep).join("/")}`;
      }
    });
  }

  protected buildWrangler() {
    return {
      main: "@tanstack/react-start/server-entry",
    };
  }

  /**
   * The URL of the TanStack Start app.
   *
   * If the `domain` is set, this is the URL with the custom domain.
   * Otherwise, it's the auto-generated Worker URL.
   */
  public get url() {
    return super.url;
  }
}

const __pulumiType = "sst:cloudflare:TanStackStart";
// @ts-expect-error
TanStackStart.__pulumiType = __pulumiType;
