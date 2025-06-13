import path from "path";
import fs from "fs";
import { Output, Resource, all, output } from "@pulumi/pulumi";
import * as sst from "sst-plugin";

export interface BaseStaticSiteArgs {
  /**
   * Path to the directory where your static site is located.
   * @default `"."`
   */
  path?: sst.Input<string>;
  /**
   * Configure if your static site needs to be built.
   */
  build?: sst.Input<{
    /**
     * The command to build your site.
     */
    command?: sst.Input<string>;
    /**
     * The directory where the build output is located.
     */
    output?: sst.Input<string>;
  }>;
  /**
   * Set environment variables for your site.
   */
  environment?: sst.Input<Record<string, sst.Input<string>>>;
  /**
   * The error page to show when a page is not found.
   * @default `"404.html"`
   */
  errorPage?: sst.Input<string>;
  /**
   * The index page to show when the root path is accessed.
   * @default `"index.html"`
   */
  indexPage?: sst.Input<string>;
}

export interface BaseStaticSiteAssets {
  /**
   * Character encoding for text based assets.
   * @default `"utf-8"`
   */
  textEncoding?: sst.Input<"utf-8" | "iso-8859-1" | "windows-1252" | "ascii" | "none">;
  /**
   * Configure how files are cached.
   */
  fileOptions?: sst.Input<{
    files: sst.Input<string | string[]>;
    ignore?: sst.Input<string | string[]>;
    cacheControl?: sst.Input<string>;
    contentType?: sst.Input<string>;
  }[]>;
}

export function prepare(args: BaseStaticSiteArgs) {
  const sitePath = normalizeSitePath();
  const environment = normalizeEnvironment();
  const indexPage = args.indexPage || "index.html";

  return {
    sitePath,
    environment,
    indexPage,
  };

  function normalizeSitePath() {
    return output(args.path).apply((sitePath) => {
      if (!sitePath) return ".";

      if (!fs.existsSync(sitePath)) {
        throw new Error(`No site found at "${path.resolve(sitePath)}"`);
      }
      return sitePath;
    });
  }

  function normalizeEnvironment() {
    return output(args.environment).apply((environment) => environment || {});
  }
}

export function buildApp(
  parent: Resource,
  name: string,
  build: BaseStaticSiteArgs["build"],
  sitePath: Output<string>,
  environment: Output<Record<string, string>>,
): Output<string> {
  return all([sitePath, build, environment]).apply(([sitePath, build, environment]) => {
    if (!build) {
      return sitePath;
    }

    const buildCommand = build.command || "npm run build";
    const buildOutput = build.output || "dist";
    
    // For now, just return the build output path
    // In a full implementation, this would run the build command
    return path.join(sitePath, buildOutput);
  });
}