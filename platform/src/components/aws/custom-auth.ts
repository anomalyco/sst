import path from "path";
import {
  all,
  ComponentResourceOptions,
  Output,
  output,
  jsonStringify,
} from "@pulumi/pulumi";
import { Plan, SsrSite, SsrSiteArgs } from "./ssr-site.js";
import { createAuthTable } from "./auth/shared.js";
import { transform } from "../component.js";
import type { Dynamo } from "./dynamo.js";

/**
 * The `CustomAuth` component lets you deploy a fully customizable [OpenAuth](https://openauth.js.org/) server to AWS.
 * It allows you to use any modern UI framework (e.g., React) with SSR and full HMR support in dev mode.
 *
 * @example
 *
 * #### Minimal example
 *
 * Deploy a custom auth server located in the `packages/auth/` directory. 
 * Note that you must specify the `dev.url` to proxy traffic during local development.
 *
 * ```js title="sst.config.ts"
 * new sst.aws.CustomAuth("MyAuth", {
 *   path: "packages/auth/",
 *   dev: {
 *     url: "http://localhost:5174"
 *   }
 * });
 * ```
 *
 * #### Link to a frontend
 *
 * [Link the auth component](/docs/linking/) to a frontend application, such as Next.js. This allows the frontend to access the issuer URL and other auth resources.
 *
 * ```ts {9} title="sst.config.ts"
 * const auth = new sst.aws.CustomAuth("MyAuth", {
 *   path: "packages/auth/",
 *   dev: {
 *     url: "http://localhost:5174"
 *   }
 * });
 *
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [auth]
 * });
 * ```
 *
 * Once linked, you can now use it to create an [OpenAuth
 * client](https://openauth.js.org/docs/client/).
 *
 * ```ts title="app/page.tsx" {1,6}
 * import { Resource } from "sst"
 * import { createClient } from "@openauthjs/openauth/client"
 *
 * export const client = createClient({
 *   clientID: "nextjs",
 *   issuer: Resource.MyAuth.url
 * });
 * ```
 */
export class CustomAuth extends SsrSite {
  constructor(
    name: string,
    args: SsrSiteArgs = {},
    opts: ComponentResourceOptions = {},
  ) {
    const { promise: tablePromise, resolve: resolveTable } =
      Promise.withResolvers<Dynamo>();

    super(
      __pulumiType,
      name,
      {
        ...args,
        environment: {
          ...args.environment,
          OPENAUTH_STORAGE: jsonStringify({
            type: "dynamo",
            options: { table: output(tablePromise).name },
          }),
        },
        link: all([args.link, tablePromise]).apply(
          ([prevLinks, tableResolved]) => {
            return [...(prevLinks ?? []), tableResolved];
          },
        ),
        transform: {
          ...args.transform,
          server(serverArgs, opts, fnName) {
            const [_, newArgs] = transform(
              args.transform?.server,
              fnName,
              {
                ...serverArgs,
                url: {
                  cors: false,
                },
              },
              opts,
            );

            Object.assign(serverArgs, newArgs);
          },
        },
      },
      opts,
    );

    const table = createAuthTable(name, { parent: this });
    resolveTable(table);
  }

  protected normalizeBuildCommand() {}

  protected buildPlan(outputPath: Output<string>): Output<Plan> {
    return output(outputPath).apply((outputPath) => {
      const serverPath = path.join(outputPath, ".openauth/build/server");

      return {
        server: {
          bundle: serverPath,
          handler: "index.handler",
          streaming: true,
        },
        assets: [
          {
            from: ".openauth/build/client",
            to: "",
            cached: true,
          },
        ],
      };
    });
  }

  public get url() {
    return super.url;
  }
}

const __pulumiType = "sst:aws:CustomAuth";
// @ts-expect-error
CustomAuth.__pulumiType = __pulumiType;
