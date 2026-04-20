import { ComponentResourceOptions, Output, output } from "@pulumi/pulumi";
import { Component } from "../component";
import { Link } from "../link";
import { Input } from "../input";
import { binding } from "./binding";

export interface AiSearchArgs {
  /**
   * The name of the AI Search instance to bind to.
   *
   * Use this when you know which instance you need at deploy time. The instance
   * must exist in the `default` namespace.
   *
   * You must specify either `instance` or `namespace`, but not both.
   *
   * @example
   * ```ts title="sst.config.ts"
   * const search = new sst.cloudflare.AiSearch("MySearch", {
   *   instance: "my-docs-index"
   * });
   * ```
   */
  instance?: Input<string>;
  /**
   * The name of the AI Search namespace to bind to.
   *
   * Use this when you need access to multiple instances at runtime. You can
   * get, create, list, and delete instances within the namespace.
   *
   * A default namespace is created automatically for every account. If the
   * namespace does not exist, it will be created on deploy.
   *
   * You must specify either `instance` or `namespace`, but not both.
   *
   * @example
   * ```ts title="sst.config.ts"
   * const search = new sst.cloudflare.AiSearch("MySearch", {
   *   namespace: "my-namespace"
   * });
   * ```
   */
  namespace?: Input<string>;
}

/**
 * The `AiSearch` component lets you add a [Cloudflare AI Search](https://developers.cloudflare.com/ai-search/)
 * binding to your app.
 *
 * AI Search is a managed search service. Connect a website, an R2 bucket, or upload
 * your own documents, and AI Search indexes your content for natural language queries.
 *
 * There are two types of bindings:
 *
 * - **Instance binding**: Binds directly to a single AI Search instance. Use this when
 *   you know which instance you need at deploy time.
 * - **Namespace binding**: Binds to a namespace that can contain multiple instances.
 *   Use this when you need to access or manage multiple instances at runtime.
 *
 * @example
 *
 * #### Instance binding
 *
 * Bind to a specific AI Search instance.
 *
 * ```ts title="sst.config.ts"
 * const search = new sst.cloudflare.AiSearch("MySearch", {
 *   instance: "my-docs-index"
 * });
 * ```
 *
 * #### Namespace binding
 *
 * Bind to a namespace to access multiple instances.
 *
 * ```ts title="sst.config.ts"
 * const search = new sst.cloudflare.AiSearch("MySearch", {
 *   namespace: "my-namespace"
 * });
 * ```
 *
 * #### Link to a worker
 *
 * You can link AI Search to a worker.
 *
 * ```ts {3} title="sst.config.ts"
 * new sst.cloudflare.Worker("MyWorker", {
 *   handler: "./index.ts",
 *   link: [search],
 *   url: true
 * });
 * ```
 *
 * Once linked, you can use the binding to search your indexed content.
 *
 * For an **instance binding**, call methods directly:
 *
 * ```ts title="index.ts"
 * const results = await env.MySearch.search({
 *   messages: [{ role: "user", content: "What is Cloudflare?" }]
 * });
 * ```
 *
 * For a **namespace binding**, get an instance handle first:
 *
 * ```ts title="index.ts"
 * const instance = env.MySearch.get("my-docs-index");
 * const results = await instance.search({
 *   messages: [{ role: "user", content: "What is Cloudflare?" }]
 * });
 * ```
 */
export class AiSearch extends Component implements Link.Linkable {
  private _instance?: Output<string>;
  private _namespace?: Output<string>;

  constructor(
    name: string,
    args: AiSearchArgs,
    opts?: ComponentResourceOptions,
  ) {
    super(__pulumiType, name, args, opts);

    if (args.instance && args.namespace) {
      throw new Error(
        `Cannot specify both "instance" and "namespace" for AiSearch "${name}". Choose one.`,
      );
    }
    if (!args.instance && !args.namespace) {
      throw new Error(
        `Must specify either "instance" or "namespace" for AiSearch "${name}".`,
      );
    }

    if (args.instance) {
      this._instance = output(args.instance);
    }
    if (args.namespace) {
      this._namespace = output(args.namespace);
    }
  }

  /**
   * The name of the AI Search instance, if using an instance binding.
   */
  public get instance() {
    return this._instance;
  }

  /**
   * The name of the AI Search namespace, if using a namespace binding.
   */
  public get namespace() {
    return this._namespace;
  }

  /**
   * When you link AI Search, it will be available to the worker and you can
   * interact with it using its binding methods.
   *
   * For an instance binding:
   * @example
   * ```ts title="index.ts"
   * const results = await env.MySearch.search({
   *   messages: [{ role: "user", content: "What is Cloudflare?" }]
   * });
   * ```
   *
   * For a namespace binding:
   * @example
   * ```ts title="index.ts"
   * const instance = env.MySearch.get("my-docs-index");
   * const results = await instance.search({
   *   messages: [{ role: "user", content: "What is Cloudflare?" }]
   * });
   * ```
   *
   * @internal
   */
  getSSTLink() {
    if (this._instance) {
      const instanceName = this._instance;
      return {
        properties: {
          instanceName,
        },
        include: [
          binding({
            type: "aiSearchBindings",
            properties: {
              instanceName,
            },
          }),
        ],
      };
    }

    const namespace = this._namespace!;
    return {
      properties: {
        namespace,
      },
      include: [
        binding({
          type: "aiSearchNamespaceBindings",
          properties: {
            namespace,
          },
        }),
      ],
    };
  }
}

const __pulumiType = "sst:cloudflare:AiSearch";
// @ts-expect-error
AiSearch.__pulumiType = __pulumiType;
