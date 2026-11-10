import { output } from "@pulumi/pulumi";
import { getRegionOutput } from "@pulumi/aws";
import { bootstrap } from "./helpers/bootstrap";
import { VisibleError } from "../error";

export interface EcrRepositoryInfo {
  /**
   * The ECR repository name.
   */
  name: string;
  /**
   * The ECR repository URL.
   */
  url: string;
  /**
   * The ECR registry ID.
   */
  registryId: string;
}

/**
 * Get the bootstrap ECR repository used by SST for storing assets.
 */
export function getECRRepository() {
  const region = getRegionOutput().name;
  const bootstrapData = region.apply((value) => bootstrap.forRegion(value));

  return output(bootstrapData).apply((data) => {
    const name = data.assetEcrUrl.split("/").pop();
    if (!name) {
      throw new VisibleError(
        `Failed to resolve the SST asset ECR repository name from "${data.assetEcrUrl}".`,
      );
    }

    return {
      name,
      url: data.assetEcrUrl,
      registryId: data.assetEcrRegistryId,
    } satisfies EcrRepositoryInfo;
  });
}
