import path from "path";
import fs from "fs";
import { Output, Resource, all, output } from "@pulumi/pulumi";
import * as sst from "sst-plugin";

export interface BaseSsrSiteArgs {
  /**
   * Path to the directory where your site is located. This path is relative to your `sst.config.ts`.
   * @default `"."`
   */
  path?: sst.Input<string>;
  /**
   * The command used internally to build your site.
   * @default Dynamically determined based on package manager
   */
  buildCommand?: sst.Input<string>;
  /**
   * Set environment variables for your site.
   */
  environment?: sst.Input<Record<string, sst.Input<string>>>;
  /**
   * Link resources to your site.
   */
  link?: sst.Input<sst.Linkable[]>;
  /**
   * Configure how the site's assets are uploaded.
   */
  assets?: sst.Input<{
    /**
     * Character encoding for text based assets.
     * @default `"utf-8"`
     */
    textEncoding?: sst.Input<"utf-8" | "iso-8859-1" | "windows-1252" | "ascii" | "none">;
    /**
     * Cache header for versioned files.
     * @default `"public,max-age=31536000,immutable"`
     */
    versionedFilesCacheHeader?: sst.Input<string>;
    /**
     * Cache header for non-versioned files.
     * @default `"public,max-age=0,s-maxage=86400,stale-while-revalidate=8640"`
     */
    nonVersionedFilesCacheHeader?: sst.Input<string>;
  }>;
}

export function buildApp(
  parent: Resource,
  name: string,
  args: BaseSsrSiteArgs,
  sitePath: Output<string>,
  buildCommand?: Output<string>,
): Output<string> {
  return all([
    sitePath,
    buildCommand ?? args.buildCommand,
    args.environment,
  ]).apply(([sitePath, userCommand, environment]) => {
    const cmd = resolveBuildCommand();
    
    // For now, just return the site path
    // In a full implementation, this would run the build command
    return sitePath;

    function resolveBuildCommand() {
      if (userCommand) return userCommand;

      // Auto-detect build command based on lock files
      if (fs.existsSync(path.join(sitePath, "bun.lockb"))) {
        return "bun run build";
      }
      if (fs.existsSync(path.join(sitePath, "yarn.lock"))) {
        return "yarn run build";
      }
      if (fs.existsSync(path.join(sitePath, "pnpm-lock.yaml"))) {
        return "pnpm run build";
      }
      if (fs.existsSync(path.join(sitePath, "package-lock.json"))) {
        return "npm run build";
      }
      
      return "npm run build";
    }
  });
}