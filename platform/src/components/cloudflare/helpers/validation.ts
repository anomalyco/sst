import fs from "fs";
import path from "path";
import { VisibleError } from "../../error.js";

/**
 * Validates that no wrangler configuration file exists in the site directory.
 * SST manages wrangler configuration automatically and user-provided files can cause conflicts.
 */
export function validateNoWranglerFile(sitePath: string, componentName: string): void {
  const wranglerFiles = ["wrangler.toml", "wrangler.json", "wrangler.jsonc"];

  for (const file of wranglerFiles) {
    const filePath = path.join(sitePath, file);
    if (fs.existsSync(filePath)) {
      throw new VisibleError(
        [
          `Found ${file} in "${path.resolve(sitePath)}" for ${componentName}.`,
          "",
          "Remove it to avoid interfering with SST managed wrangler configurations:",
          `https://sst.dev/docs/cloudflare/#cloudflare-vite-plugin`,
        ].join("\n"),
      );
    }
  }
}

/**
 * Validates that the framework config file contains the required SST_WRANGLER_PATH configuration.
 * This ensures linked resources work correctly in Cloudflare SSR sites.
 */
export function validateFrameworkConfig(input: {
  sitePath: string;
  configName: string;
  componentName: string;
}): void {
  const { sitePath, configName, componentName } = input;

  const extensions = [".ts", ".js", ".mjs"];
  const configDir = sitePath;

  // Find the config file
  let configPath: string | undefined;
  for (const ext of extensions) {
    const candidate = path.join(configDir, `${configName}${ext}`);
    if (fs.existsSync(candidate)) {
      configPath = candidate;
      break;
    }
  }

  if (!configPath) {
    throw new VisibleError(
      `Could not find config file for ${componentName} in "${path.resolve(configDir)}".\nExpected one of: ${extensions.map(e => `${configName}${e}`).join(", ")}.`
    );
  }

  // Read and check for SST_WRANGLER_PATH pattern
  const content = fs.readFileSync(configPath, "utf-8");
  const hasWranglerPath = /configPath\s*[:=]\s*process\.env\.SST_WRANGLER_PATH/.test(content);

  if (!hasWranglerPath) {
    throw new VisibleError(
      [
        `Missing required configuration for ${componentName}.`,
        "",
        `The Cloudflare adapter must be configured with:`,
        `  configPath: process.env.SST_WRANGLER_PATH,`,
        "",
        `This is required for linked resources to work correctly:`,
        `https://sst.dev/docs/cloudflare/#cloudflare-vite-plugin`,
      ].join("\n")
    );
  }
}
