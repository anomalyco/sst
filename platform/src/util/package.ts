import fs from "fs";
import path from "path";

/**
 * Gets the version of a package from the site's package.json.
 * Checks both dependencies and devDependencies.
 * 
 * @param sitePath - Path to the site directory containing package.json
 * @param packageName - Name of the package to look up
 * @returns The version string (e.g., "^6.0.0", "~5.2.0", "latest") or undefined if not found
 */
export function getPackageVersion(
  sitePath: string,
  packageName: string,
): string | undefined {
  try {
    const pkgPath = path.join(sitePath, "package.json");
    if (!fs.existsSync(pkgPath)) {
      return undefined;
    }

    const pkg = JSON.parse(fs.readFileSync(pkgPath, "utf-8"));
    return (
      pkg.dependencies?.[packageName] ?? pkg.devDependencies?.[packageName]
    );
  } catch {
    return undefined;
  }
}
