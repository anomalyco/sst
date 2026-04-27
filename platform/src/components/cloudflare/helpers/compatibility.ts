import { all, output } from "@pulumi/pulumi";
import type { Input } from "../../input.js";

export const DEFAULT_COMPATIBILITY_DATE = "2025-05-05";
export const DEFAULT_COMPATIBILITY_FLAGS = ["nodejs_compat"];

type CompatibilityArgs = {
  compatibility?: Input<{
    date?: Input<string>;
    flags?: Input<Input<string>[]>;
  }>;
  transform?: {
    worker?:
      | {
          compatibilityDate?: Input<string>;
          compatibilityFlags?: Input<Input<string>[]>;
        }
      | ((...args: any[]) => undefined);
  };
};

export function normalizeCompatibility(args?: CompatibilityArgs) {
  const compatibility = output(args?.compatibility);
  const workerTransform =
    typeof args?.transform?.worker === "function"
      ? undefined
      : args?.transform?.worker;
  return output({
    date: all([
      compatibility.apply((value) => value?.date),
      workerTransform?.compatibilityDate,
    ]).apply(
      ([argValue, transformValue]) =>
        transformValue ?? argValue ?? DEFAULT_COMPATIBILITY_DATE,
    ),
    flags: all([
      compatibility.apply((value) => value?.flags),
      workerTransform?.compatibilityFlags,
    ]).apply(
      ([argValue, transformValue]) =>
        transformValue ?? argValue ?? DEFAULT_COMPATIBILITY_FLAGS,
    ),
  });
}
