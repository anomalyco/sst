import {
  ComponentResourceOptions,
  jsonStringify,
  Output,
  output,
  all,
} from "@pulumi/pulumi";
import { Component } from "../component";
import { Link } from "../link";
import { FunctionArgs, Function, Dynamo, CdnArgs, Router } from ".";
import { functionBuilder } from "./helpers/function-builder";
import { env } from "../linkable";
import { Auth as AuthV1 } from "./auth-v1";
import { Input } from "../input";
import { Permission } from "./permission";

export interface AuthArgs {
  /**
   * The issuer function.
   * @deprecated renamed to `issuer`
   * @example
   * ```js
   * {
   *   authorizer: "src/auth.handler"
   * }
   * ```
   *
   * You can also pass in the full `FunctionArgs`.
   *
   * ```js
   * {
   *   authorizer: {
   *     handler: "src/auth.handler",
   *     link: [table]
   *   }
   * }
   * ```
   */
  authorizer?: Input<string | FunctionArgs>;
  /**
   * The function that's running your OpenAuth server.
   *
   * @example
   * ```js
   * {
   *   issuer: "src/auth.handler"
   * }
   * ```
   *
   * You can also pass in the full `FunctionArgs`.
   *
   * ```js
   * {
   *   issuer: {
   *     handler: "src/auth.handler",
   *     link: [table]
   *   }
   * }
   * ```
   *
   * Since the `issuer` function is a Hono app, you want to export it with the Lambda adapter.
   *
   * ```ts title="src/auth.ts"
   * import { handle } from "hono/aws-lambda";
   * import { issuer } from "@openauthjs/openauth";
   *
   * const app = issuer({
   *   // ...
   * });
   *
   * export const handler = handle(app);
   * ```
   *
   * This `Auth` component will always use the
   * [`DynamoStorage`](https://openauth.js.org/docs/storage/dynamo/) storage provider.
   *
   * :::note
   * This will always use the `DynamoStorage` storage provider.
   * :::
   *
   * Learn more on the [OpenAuth docs](https://openauth.js.org/docs/issuer/) on how to configure
   * the `issuer` function.
   */
  issuer?: Input<string | FunctionArgs>;
  /**
   * [Link resources](/docs/linking/) to your Auth issuer function. This will:
   *
   * 1. Grant the permissions needed to access the resources.
   * 2. Allow you to access it in your function using the [SDK](/docs/reference/sdk/).
   *
   * @example
   *
   * Takes a list of components to link to the issuer function.
   *
   * ```js
   * {
   *   link: [bucket, stripeKey]
   * }
   * ```
   */
  link?: Input<any[]>;
  /**
   * Key-value pairs that are made available to the issuer function as environment
   * variables. The keys need to:
   * - Start with a letter
   * - Be at least 2 characters long
   * - Contain only letters, numbers, or underscores
   *
   * They can be accessed in your function using `process.env.<key>`.
   *
   * :::note
   * The total size of the environment variables cannot exceed 4 KB.
   * :::
   *
   * @example
   *
   * ```js
   * {
   *   environment: {
   *     DEBUG: "true"
   *   }
   * }
   * ```
   */
  environment?: Input<Record<string, Input<string>>>;
  /**
   * Set a custom domain for your Auth server.
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
   *   domain: "auth.example.com"
   * }
   * ```
   *
   * For domains hosted on Cloudflare.
   *
   * ```js
   * {
   *   domain: {
   *     name: "auth.example.com",
   *     dns: sst.cloudflare.dns()
   *   }
   * }
   * ```
   */
  domain?: CdnArgs["domain"];
  /**
   * Force upgrade from `Auth.v1` to the latest `Auth` version. The only valid value
   * is `v2`, which is the version of the new `Auth`.
   *
   * The latest `Auth` is powered by [OpenAuth](https://openauth.js.org). To
   * upgrade, add the prop.
   *
   * ```ts
   * {
   *   forceUpgrade: "v2"
   * }
   * ```
   *
   * Run `sst deploy`.
   *
   * :::tip
   * You can remove this prop after you upgrade.
   * :::
   *
   * This upgrades your component and the resources it created. You can now optionally
   * remove the prop.
   *
   * @internal
   */
  forceUpgrade?: "v2";
}

/**
 * The `Auth` component lets you create centralized auth servers on AWS. It deploys
 * [OpenAuth](https://openauth.js.org) to [AWS Lambda](https://aws.amazon.com/lambda/)
 * and uses [Amazon DynamoDB](https://aws.amazon.com/dynamodb/) for storage.
 *
 * :::note
 * `Auth` and OpenAuth are currently in beta.
 * :::
 *
 * @example
 *
 * #### Create an OpenAuth server
 *
 * ```ts title="sst.config.ts"
 * const auth = new sst.aws.Auth("MyAuth", {
 *   issuer: "src/auth.handler"
 * });
 * ```
 *
 * Where the `issuer` function might look like this.
 *
 * ```ts title="src/auth.ts"
 * import { handle } from "hono/aws-lambda";
 * import { issuer } from "@openauthjs/openauth";
 * import { CodeProvider } from "@openauthjs/openauth/provider/code";
 * import { subjects } from "./subjects";
 *
 * const app = issuer({
 *   subjects,
 *   providers: {
 *     code: CodeProvider()
 *   },
 *   success: async (ctx, value) => {}
 * });
 *
 * export const handler = handle(app);
 * ```
 *
 * This `Auth` component will always use the
 * [`DynamoStorage`](https://openauth.js.org/docs/storage/dynamo/) storage provider.
 *
 * Learn more on the [OpenAuth docs](https://openauth.js.org/docs/issuer/) on how to configure
 * the `issuer` function.
 *
 * #### Add a custom domain
 *
 * Set a custom domain for your auth server.
 *
 * ```js {3} title="sst.config.ts"
 * new sst.aws.Auth("MyAuth", {
 *   issuer: "src/auth.handler",
 *   domain: "auth.example.com"
 * });
 * ```
 * 
 * #### Customize theme with assets
 * 
 * You can link a bucket to store and serve assets for your auth UI theme, like logos.
 * 
 * ```ts title="sst.config.ts"
 * // Create a bucket for assets
 * const bucket = new sst.aws.Bucket("Assets", {
 *   cors: true,
 *   cdk: {
 *     bucket: {
 *       publicReadAccess: true
 *     }
 *   }
 * });
 * 
 * // Link the bucket to Auth
 * new sst.aws.Auth("MyAuth", {
 *   issuer: "src/auth.handler",
 *   link: [bucket]
 * });
 * ```
 * 
 * Then in your issuer function, you can use the bucket URL for assets in your theme:
 * 
 * ```ts title="src/auth.ts"
 * import { Resource } from "sst";
 * 
 * const app = issuer({
 *   subjects,
 *   theme: {
 *     title: "My App",
 *     // Use the bucket URL for the logo
 *     logo: {
 *       light: `${Resource.Assets.url}/logo-light.svg`,
 *       dark: `${Resource.Assets.url}/logo-dark.svg`
 *     },
 *     // ... other theme options
 *   },
 *   // ... other options
 * });
 * ```
 *
 * #### Link resources to Auth
 *
 * You can link resources to your Auth component. This allows the issuer function
 * to access these resources.
 *
 * ```ts title="sst.config.ts" {3}
 * new sst.aws.Auth("MyAuth", {
 *   issuer: "src/auth.handler",
 *   link: [bucket, stripeKey]
 * });
 * ```
 *
 * #### Set environment variables
 *
 * You can set environment variables for the issuer function.
 *
 * ```ts title="sst.config.ts" {3-5}
 * new sst.aws.Auth("MyAuth", {
 *   issuer: "src/auth.handler",
 *   environment: {
 *     STRIPE_SECRET_KEY: "sk_test_123"
 *   }
 * });
 * ```
 *
 * #### Link to a resource
 *
 * You can link the auth server to other resources, like a function or your Next.js app,
 * that needs authentication.
 *
 * ```ts title="sst.config.ts" {2}
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
export class Auth extends Component implements Link.Linkable {
  private readonly _table: Dynamo;
  private readonly _issuer: Output<Function>;
  private readonly _router?: Router;
  public static v1 = AuthV1;

  constructor(name: string, args: AuthArgs, opts?: ComponentResourceOptions) {
    super(__pulumiType, name, args, opts);
    const _version = 2;
    const self = this;

    self.registerVersion({
      new: _version,
      old: $cli.state.version[name],
      message: [
        `There is a new version of "Auth" that has breaking changes.`,
        ``,
        `What changed:`,
        `  - The latest version is now powered by OpenAuth - https://openauth.js.org`,
        ``,
        `To upgrade:`,
        `  - Set \`forceUpgrade: "v${_version}"\` on the "Auth" component. Learn more https://sst.dev/docs/component/aws/auth#forceupgrade`,
        ``,
        `To continue using v${$cli.state.version[name]}:`,
        `  - Rename "Auth" to "Auth.v${$cli.state.version[name]}". Learn more about versioning - https://sst.dev/docs/components/#versioning`,
      ].join("\n"),
      forceUpgrade: args.forceUpgrade,
    });

    const table = createTable();
    const issuer = createIssuer();
    const router = createRouter();

    this._table = table;
    this._issuer = issuer;
    this._router = router;
    registerOutputs();

    function registerOutputs() {
      self.registerOutputs({
        _hint: self.url,
      });
    }

    function createTable() {
      return new Dynamo(
        `${name}Storage`,
        {
          fields: { pk: "string", sk: "string" },
          primaryIndex: { hashKey: "pk", rangeKey: "sk" },
          ttl: "expiry",
        },
        { parent: self },
      );
    }

    function createIssuer() {
      const fn = args.authorizer || args.issuer;
      if (!fn) throw new Error("Auth: issuer field must be set");

      const linkData = output(args.link || []).apply((links: any[]) => Link.build(links));
      const linkPermissions = Link.getInclude<Permission>("aws.permission", args.link);

      return functionBuilder(
        `${name}Issuer`,
        fn,
        {
          link: all([args.link, table]).apply(([link, table]) => [table, ...(link || [])]),
          environment: all([args.environment]).apply(([environment]) => ({
            OPENAUTH_STORAGE: jsonStringify({
              type: "dynamo",
              options: { table: table.name },
            }),
            ...(environment ?? {}),
          })),
          permissions: linkPermissions,
          _skipHint: true,
        },
        (args) => {
          args.url = {
            cors: false,
          };
        },
        { parent: self },
      ).apply(v => v.getFunction());
    }

    function createRouter() {
      if (!args.domain) return;

      const router = new Router(
        `${name}Router`,
        {
          domain: args.domain,
          _skipHint: true,
        },
        { parent: self },
      );
      router.route("/", issuer.url);

      return router;
    }
  }

  /**
   * The URL of the Auth component.
   *
   * If the `domain` is set, this is the URL with the custom domain.
   * Otherwise, it's the auto-generated function URL for the issuer.
   */
  public get url() {
    return this._router?.url ?? this._issuer.url.apply((v) => v.slice(0, -1));
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The DynamoDB component.
       */
      table: this._table,
      /**
       * The Function component for the issuer.
       */
      issuer: this._issuer,
      /**
       * @deprecated Use `issuer` instead.
       * The Function component for the issuer.
       */
      authorizer: this._issuer,
      /**
       * The Router component for the custom domain.
       */
      router: this._router,
    };
  }

  /** @internal */
  public getSSTLink() {
    return {
      properties: {
        url: this.url,
      },
      include: [
        env({
          OPENAUTH_ISSUER: this.url,
        }),
      ],
    };
  }
}

const __pulumiType = "sst:aws:Auth";
// @ts-expect-error
Auth.__pulumiType = __pulumiType;
