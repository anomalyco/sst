import path from "path";
import { ComponentResourceOptions, Output } from "@pulumi/pulumi";
import { VisibleError } from "../error.js";
import { Plan, SsrSite, SsrSiteArgs } from "./ssr-site.js";
import { existsAsync } from "../../util/fs.js";
import {
  validateFrameworkConfig,
  validateNoWranglerFile,
} from "./helpers/validation.js";

export interface ReactRouterArgs extends SsrSiteArgs {
  /**
   * Configure how this component works in `sst dev`.
   *
   * :::note
   * In `sst dev` your React Router app is run in dev mode; it's not deployed.
   * :::
   *
   * Instead of deploying your React Router app, this starts it in dev mode. It's run
   * as a separate process in the `sst dev` multiplexer. Read more about
   * [`sst dev`](/docs/reference/cli/#dev).
   *
   * To disable dev mode, pass in `false`.
   */
  dev?: SsrSiteArgs["dev"];
  /**
   * Path to the directory where your React Router app is located. This path is relative to your `sst.config.ts`.
   *
   * By default it assumes your React Router app is in the root of your SST app.
   * @default `"."`
   *
   * @example
   *
   * If your React Router app is in a package in your monorepo.
   *
   * ```js
   * {
   *   path: "packages/web"
   * }
   * ```
   */
  path?: SsrSiteArgs["path"];
  /**
   * [Link resources](/docs/linking/) to your React Router app. This will:
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
   * You can access the linked resources in your React Router app.
   *
   * ```ts
   * import { Resource } from "sst";
   *
   * console.log(Resource.MyBucket.name);
   * ```
   */
  link?: SsrSiteArgs["link"];
  /**
   * Set environment variables in your React Router app. These are made available:
   *
   * 1. In `vite build`, they are loaded into the build.
   * 2. At runtime as Worker bindings.
   * 3. Locally while running `vite dev` through `sst dev`.
   *
   * :::tip
   * You can also `link` resources to your React Router app and access them in a type-safe way with the [SDK](/docs/reference/sdk/). We recommend linking since it's more secure.
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
   * You can access the environment variables in your React Router app as follows:
   *
   * ```ts
   * export function loader({ context }: Route.LoaderArgs) {
   *   return { apiUrl: context.cloudflare.env.API_URL };
   * }
   * ```
   */
  environment?: SsrSiteArgs["environment"];
  /**
   * Set a custom domain for your React Router app.
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
   * The command used internally to build your React Router app.
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
 * The `ReactRouter` component lets you deploy a [React Router v7](https://reactrouter.com) app to Cloudflare.
 *
 * :::note
 * Create a Cloudflare-compatible app with `npm create cloudflare@latest -- my-react-router-app --framework=react-router`.
 * :::
 *
 * @example
 *
 * #### Minimal example
 *
 * Deploy the React Router app that's in the project root.
 *
 * ```js title="sst.config.ts"
 * new sst.cloudflare.ReactRouter("MyWeb");
 * ```
 *
 * #### Change the path
 *
 * Deploys the React Router app in the `my-app/` directory.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.cloudflare.ReactRouter("MyWeb", {
 *   path: "my-app/"
 * });
 * ```
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your React Router app.
 *
 * ```js {2} title="sst.config.ts"
 * new sst.cloudflare.ReactRouter("MyWeb", {
 *   domain: "my-app.com"
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your React Router app. This will grant permissions
 * to the resources and allow you to access it in your app.
 *
 * ```ts {4} title="sst.config.ts"
 * const bucket = new sst.cloudflare.Bucket("MyBucket");
 *
 * new sst.cloudflare.ReactRouter("MyWeb", {
 *   link: [bucket]
 * });
 * ```
 *
 * Add this to your `vite.config.ts` for SST to work correctly.
 *
 * ```ts title="vite.config.ts"
 * import { reactRouter } from "@react-router/dev/vite";
 * import { cloudflare } from "@cloudflare/vite-plugin";
 * import { defineConfig } from "vite";
 *
 * export default defineConfig({
 *   plugins: [
 *     cloudflare({
 *       viteEnvironment: { name: "ssr" },
 *       configPath: process.env.SST_WRANGLER_PATH,
 *     }),
 *     reactRouter(),
 *   ],
 * });
 * ```
 *
 * Use `sst/resource` for linked resources.
 *
 * ```ts title="app/routes/home.tsx"
 * import { Resource } from "sst/resource";
 *
 * export async function loader() {
 *   const files = await Resource.MyBucket.list();
 *   return { files };
 * }
 * ```
 */
export class ReactRouter extends SsrSite {
  constructor(
    name: string,
    args: ReactRouterArgs = {},
    opts: ComponentResourceOptions = {},
  ) {
    super(__pulumiType, name, args, opts);
  }

  protected validate(sitePath: string): void {
    validateFrameworkConfig({
      sitePath,
      configName: "vite.config",
      componentName: "ReactRouter",
    });
    validateNoWranglerFile(sitePath, "ReactRouter");
  }

  protected buildPlan(outputPath: Output<string>): Output<Plan> {
    return outputPath.apply(async (outputPath) => {
      const serverPath = path.join(outputPath, "build", "server", "index.js");
      const assetsPath = path.join(outputPath, "build", "client");

      if (!(await existsAsync(serverPath))) {
        throw new VisibleError(
          [
            `React Router server bundle not found at:\n  "${serverPath}".`,
            "",
            "Make sure the Cloudflare Vite plugin is configured in your `vite.config.ts`.",
            "If your React Router project is entirely pre-rendered, use `sst.cloudflare.StaticSite` instead.",
          ].join("\n"),
        );
      }

      if (!(await existsAsync(assetsPath))) {
        throw new VisibleError(
          `React Router assets directory not found at:\n  "${assetsPath}".`,
        );
      }

      return {
        server: "./build/server/index.js",
        assets: "./build/client",
      };
    });
  }

  protected buildWrangler(sitePath: string) {
    return {
      main: path.resolve(sitePath, "workers/app.ts"),
    };
  }

  /**
   * The URL of the React Router app.
   *
   * If the `domain` is set, this is the URL with the custom domain.
   * Otherwise, it's the auto-generated Worker URL.
   */
  public get url() {
    return super.url;
  }
}

const __pulumiType = "sst:cloudflare:ReactRouter";
// @ts-expect-error
ReactRouter.__pulumiType = __pulumiType;
