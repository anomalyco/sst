/**
 * The AWS Permission Linkable helper is used to define the AWS permissions included with the
 * [`sst.Linkable`](/docs/component/linkable/) component.
 *
 * @example
 *
 * ```ts
 * sst.aws.permission({
 *   actions: ["lambda:InvokeFunction"],
 *   resources: ["*"]
 * })
 * ```
 *
 * @packageDocumentation
 */

import { Prettify } from "sst-plugin/internal/prettify";
import type { FunctionPermissionArgs } from "./function.js";

export interface InputArgs extends Prettify<FunctionPermissionArgs> {}

export function permission(input: InputArgs) {
  return {
    type: "aws.permission" as const,
    ...input,
  };
}

export type Permission = ReturnType<typeof permission>;
