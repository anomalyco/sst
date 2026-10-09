/**
 * The Cloudflare Binding Linkable helper is used to define the Cloudflare bindings included
 * with the [`sst.Linkable`](/docs/component/linkable/) component.
 *
 * The input is the provider-shaped [`CloudflareBinding`](#cloudflarebinding) object accepted by
 * `@pulumi/cloudflare`. Do not provide `name`; SST assigns the final binding name from the link
 * name when the link is attached to a Worker. SST does not maintain a closed list of binding
 * kinds, so new provider fields can pass through this helper. Cloudflare's provider or API may
 * still reject an unsupported kind or field.
 *
 * Binding objects from earlier SST releases are also accepted and normalized to the provider
 * shape, so existing `Linkable` definitions continue to work.
 *
 * A production binding only describes the Worker binding. It does not generate Wrangler
 * configuration or a runtime TypeScript type. Native Cloudflare resources should extend
 * `CloudflareComponent` from `sst.cloudflare` so those projections can be declared independently.
 *
 * @example
 *
 * ```ts
 * sst.cloudflare.binding({
 *   type: "r2_bucket",
 *   bucketName: "my-bucket"
 * })
 * ```
 *
 * For a custom native resource, set `binding` and `devConfig` in the constructor, declare its
 * runtime `type`, and return only runtime `properties` and any additional non-reserved includes
 * from `getLinkDefinition()`. The base class adds `cloudflare.binding`, `typescript.type`, and
 * `cloudflare.dev`; subclasses must not add those includes themselves.
 *
 * ```ts
 * class Search extends CloudflareComponent {
 *   protected readonly binding: CloudflareBinding;
 *   protected readonly type =
 *     `import("@cloudflare/workers-types").AiSearchInstance`;
 *   private readonly instanceName: Input<string>;
 *
 *   constructor(
 *     name: string,
 *     args: { instanceName: Input<string> },
 *     opts?: ComponentResourceOptions,
 *   ) {
 *     super("sst:cloudflare:Search", name, args, opts);
 *     this.instanceName = args.instanceName;
 *     this.binding = {
 *       type: "ai_search",
 *       instanceName: this.instanceName,
 *     };
 *     this.devConfig = {
 *       ai_search: [{
 *         binding: this.linkNamePlaceholder,
 *         instance_name: this.instanceName,
 *         remote: true,
 *       }],
 *     };
 *   }
 *
 *   protected getLinkDefinition() {
 *     return {
 *       properties: { instanceName: this.instanceName },
 *     };
 *   }
 * }
 * ```
 *
 * @packageDocumentation
 */

import type { types as cloudflare } from "@pulumi/cloudflare";

export type CloudflareBinding = Omit<
  cloudflare.input.WorkersScriptBinding,
  "name"
> & {
  readonly name?: never;
  readonly [field: string]: unknown;
};

const legacyBindingTypes = {
  aiBindings: "ai",
  plainTextBindings: "plain_text",
  secretTextBindings: "secret_text",
  queueBindings: "queue",
  serviceBindings: "service",
  durableObjectNamespaceBindings: "durable_object_namespace",
  kvNamespaceBindings: "kv_namespace",
  d1DatabaseBindings: "d1",
  r2BucketBindings: "r2_bucket",
  hyperdriveBindings: "hyperdrive",
  versionMetadataBindings: "version_metadata",
  workflowBindings: "workflow",
  rateLimitBindings: "ratelimit",
} as const;

export function binding(input: CloudflareBinding) {
  const providerType =
    typeof input.type === "string" &&
    Object.prototype.hasOwnProperty.call(legacyBindingTypes, input.type)
      ? legacyBindingTypes[input.type as keyof typeof legacyBindingTypes]
      : undefined;
  const workerBinding = providerType
    ? {
        type: providerType,
        ...(input.properties as Record<string, unknown>),
      }
    : input;

  return {
    type: "cloudflare.binding" as const,
    binding: workerBinding,
  };
}
