import path from "path";
import fs from "fs";
import { asset as pulumiAsset } from "@pulumi/pulumi";
import { VisibleError } from "./error";

export function asset(assetPath: string) {
  // TODO check process.cwd() resolves to the root of the project
  const fullPath = path.isAbsolute(assetPath)
    ? assetPath
    : path.join(process.cwd(), assetPath);

  try {
    return fs.statSync(fullPath).isDirectory()
      ? new pulumiAsset.FileArchive(fullPath)
      : new pulumiAsset.FileAsset(fullPath);
  } catch (e) {
    throw new VisibleError(`Asset not found: ${fullPath}`);
  }
}
