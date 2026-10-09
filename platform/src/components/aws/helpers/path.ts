import { Input, output } from "@pulumi/pulumi";
import { VisibleError } from "../../error";
import path from "node:path";
import fs from "node:fs";

export function normalizePath(
  appPath: Input<string | undefined>,
  getErrorMessage = throwDefaultSiteDirectoryError,
) {
  return output(appPath).apply((resolvedAppOutput) => {
    if (!resolvedAppOutput) return ".";

    if (!fs.existsSync(resolvedAppOutput)) {
      throw new VisibleError(getErrorMessage(path.resolve(resolvedAppOutput)));
    }
    return resolvedAppOutput;
  });
}

export function throwDefaultSiteDirectoryError(path: string) {
  return `Site directory not found at "${path}". Please check the path setting in your configuration.`;
}
