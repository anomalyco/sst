import fs from "fs";
import path from "path";
import { VisibleError } from "../../error.js";
import { isALteB } from "../../util/compare-semver.js";
import { getPackageVersion } from "../../util/package.js";

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
 *
 * Starting with the specified minimum version, the plugin automatically detects the SST-managed
 * Wrangler config, so this validation is only needed for older versions.
 */
export function validateFrameworkConfig(input: {
  sitePath: string;
  configName: string;
  componentName: string;
  packageName: string;
  minVersion: string;
}): void {
  const { sitePath, configName, componentName, packageName, minVersion } = input;

  // Check the package version first
  const packageVersion = getPackageVersion(sitePath, packageName);

  // If version is the minimum or higher, no validation needed (plugin auto-detects SST config)
  if (packageVersion && isALteB(minVersion, packageVersion)) {
    return;
  }

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

  if (hasWranglerPath) {
    // Show warning for backwards compatibility - user has old config but env var is present
    console.warn(
      `Starting with ${packageName} v${minVersion}, you no longer need to set configPath: process.env.SST_WRANGLER_PATH in your config file for ${componentName}. The plugin now automatically detects the SST-managed Wrangler configuration. You can safely remove this configuration.`
    );
    return;
  }

  // No env var set and version is old or unknown - require upgrade
  throw new VisibleError(
    [
      `Please upgrade ${packageName} to v${minVersion} or higher for SST to work correctly with ${componentName}.`,
      "",
      packageVersion
        ? `Detected ${packageName} v${packageVersion}.`
        : `Could not detect ${packageName} version in package.json.`,
      "",
      `Starting with v${minVersion}, the plugin automatically detects the SST-managed Wrangler configuration.`,
      "Alternatively, you can add the following to your config file for older versions:",
      `  configPath: process.env.SST_WRANGLER_PATH,`,
      "",
      `https://sst.dev/docs/cloudflare/#cloudflare-vite-plugin`,
    ].join("\n")
  );
}
