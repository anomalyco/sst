import fs from "fs";
import path from "path";
import { VisibleError } from "../../error.js";

/**
 * Validates that the framework config file contains the required SST_WRANGLER_PATH configuration.
 * This ensures linked resources work correctly in Cloudflare SSR sites.
 */
export function validateViteConfig(input: {
  sitePath: string | undefined;
  configName: string;
  componentName: string;
}): void {
  const { sitePath, configName, componentName } = input;

  const extensions = [".ts", ".js", ".mjs"];
  const configDir = sitePath || ".";

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
        `Missing required configuration in "${path.resolve(configPath)}".`,
        "",
        `The Cloudflare adapter must be configured with:`,
        `  configPath: process.env.SST_WRANGLER_PATH,`,
        "",
        `This is required for linked resources to work correctly.`,
        `Refer to the SST documentation for ${componentName} component setup.`,
      ].join("\n")
    );
  }
}
