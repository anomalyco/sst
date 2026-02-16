import type { BuildOptions } from "esbuild";

export type EsbuildOptions = Pick<
  BuildOptions,
  | "target"
  | "sourcemap"
  | "keepNames"
  | "define"
  | "banner"
  | "external"
  | "nodePaths"
  | "mainFields"
  | "conditions"
>;
