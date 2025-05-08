"use strict";
/**
 * The Cloudflare Binding Linkable helper is used to define the Cloudflare bindings included
 * with the [`sst.Linkable`](/docs/component/linkable/) component.
 *
 * @example
 *
 * ```ts
 * sst.cloudflare.binding({
 *   type: "r2BucketBindings",
 *   properties: {
 *     bucketName: "my-bucket"
 *   }
 * })
 * ```
 *
 * @packageDocumentation
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.binding = binding;
function binding(input) {
    return {
        type: "cloudflare.binding",
        binding: input.type,
        properties: input.properties,
    };
}
