import { CustomResourceOptions, Output, dynamic } from "@pulumi/pulumi";

// Timeout configuration for peering readiness
const PEERING_TIMEOUT = {
  MAX_ATTEMPTS: 60, // 5 minutes with 5 second intervals
  DELAY_MS: 5000, // 5 seconds
} as const;

export interface DsqlPeeringInputs {
  primaryClusterArn: string;
  peerClusterArn: string;
  primaryClusterIdentifier: string;
  peerClusterIdentifier: string;
  primaryRegion: string;
  peerRegion: string;
  witnessRegion: string;
  tags?: Record<string, string>;
}

export interface DsqlPeeringOutputs {
  primaryClusterArn: string;
  peerClusterArn: string;
  primaryClusterIdentifier: string;
  peerClusterIdentifier: string;
  primaryRegion: string;
  peerRegion: string;
  witnessRegion: string;
  peeringId: string;
}

export class DsqlPeeringProvider extends dynamic.Resource {
  public readonly primaryClusterArn!: Output<string>;
  public readonly peerClusterArn!: Output<string>;
  public readonly witnessRegion!: Output<string>;
  public readonly peeringId!: Output<string>;

  constructor(
    name: string,
    args: DsqlPeeringInputs,
    opts?: CustomResourceOptions,
  ) {
    super(new Provider(), `${name}.sst.aws.DsqlPeering`, args, opts);
  }
}

class Provider implements dynamic.ResourceProvider {
  async create(
    inputs: DsqlPeeringInputs,
  ): Promise<dynamic.CreateResult<DsqlPeeringOutputs>> {
    try {
      await this.establishPeering(inputs);

      const peeringId = `${inputs.primaryClusterIdentifier}-${inputs.peerClusterIdentifier}-peering`;

      return {
        id: peeringId,
        outs: {
          primaryClusterArn: inputs.primaryClusterArn,
          peerClusterArn: inputs.peerClusterArn,
          primaryClusterIdentifier: inputs.primaryClusterIdentifier,
          peerClusterIdentifier: inputs.peerClusterIdentifier,
          primaryRegion: inputs.primaryRegion,
          peerRegion: inputs.peerRegion,
          witnessRegion: inputs.witnessRegion,
          peeringId,
        },
      };
    } catch (error: any) {
      throw new Error(`Failed to create DSQL peering: ${error.message}`);
    }
  }

  private async establishPeering(inputs: DsqlPeeringInputs): Promise<void> {
    // Import AWS SDK inside the method to avoid serialization issues
    const { DSQLClient, UpdateClusterCommand, GetClusterCommand } =
      await import("@aws-sdk/client-dsql");

    // Setup AWS SDK clients for both regions
    const primaryClient = new DSQLClient({ region: inputs.primaryRegion });
    const peerClient = new DSQLClient({ region: inputs.peerRegion });

    // Check current cluster states
    const primaryStatus = await this.checkClusterPeering(
      primaryClient,
      inputs.primaryClusterIdentifier,
      inputs.peerClusterArn,
    );
    const peerStatus = await this.checkClusterPeering(
      peerClient,
      inputs.peerClusterIdentifier,
      inputs.primaryClusterArn,
    );

    // If already peered and active, return success (idempotency)
    if (primaryStatus.isReady && peerStatus.isReady) {
      return;
    }

    // Check if we can update the clusters (must be in PENDING_SETUP state)
    if (
      primaryStatus.status !== "PENDING_SETUP" ||
      peerStatus.status !== "PENDING_SETUP"
    ) {
      throw new Error(
        "Cannot establish peering. Clusters must be in PENDING_SETUP state. " +
          `Primary cluster status: ${primaryStatus.status}, Peer cluster status: ${peerStatus.status}. ` +
          "Aurora DSQL clusters must be created with witnessRegion to remain in PENDING_SETUP state for peering configuration.",
      );
    }

    // Establish peering by updating both clusters
    await Promise.all([
      primaryClient.send(
        new UpdateClusterCommand({
          identifier: inputs.primaryClusterIdentifier,
          multiRegionProperties: {
            witnessRegion: inputs.witnessRegion,
            clusters: [inputs.peerClusterArn],
          },
        }),
      ),
      peerClient.send(
        new UpdateClusterCommand({
          identifier: inputs.peerClusterIdentifier,
          multiRegionProperties: {
            witnessRegion: inputs.witnessRegion,
            clusters: [inputs.primaryClusterArn],
          },
        }),
      ),
    ]);

    // Wait for both clusters to reach ACTIVE status with peering
    await Promise.all([
      this.waitForPeeringReady(
        primaryClient,
        inputs.primaryClusterIdentifier,
        inputs.peerClusterArn,
      ),
      this.waitForPeeringReady(
        peerClient,
        inputs.peerClusterIdentifier,
        inputs.primaryClusterArn,
      ),
    ]);
  }

  private async checkClusterPeering(
    client: any,
    identifier: string,
    expectedPeerArn: string,
  ): Promise<{ isReady: boolean; status: string; peerCount: number }> {
    const { GetClusterCommand } = await import("@aws-sdk/client-dsql");

    const result = await client.send(
      new GetClusterCommand({
        identifier: identifier,
      }),
    );

    const status = result?.status || "unknown";
    const multiRegionClusters = result?.multiRegionProperties?.clusters || [];
    const peerCount = multiRegionClusters.length;
    const hasPeer = multiRegionClusters.includes(expectedPeerArn);

    return {
      isReady: status === "ACTIVE" && hasPeer,
      status,
      peerCount,
    };
  }

  private async waitForPeeringReady(
    client: any,
    identifier: string,
    expectedPeerArn: string,
  ): Promise<void> {
    const maxAttempts = PEERING_TIMEOUT.MAX_ATTEMPTS;
    const delayMs = PEERING_TIMEOUT.DELAY_MS;

    let lastStatus = "unknown";
    let lastPeerCount = 0;
    let lastError: string | null = null;

    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      try {
        const result = await this.checkClusterPeering(
          client,
          identifier,
          expectedPeerArn,
        );

        lastStatus = result.status;
        lastPeerCount = result.peerCount;

        if (result.isReady) {
          return; // Cluster is ready with peering
        }

        if (
          result.status === "FAILED" ||
          result.status === "DELETING" ||
          result.status === "DELETED"
        ) {
          throw new Error(
            `Cluster ${identifier} is in unexpected state: ${result.status}`,
          );
        }

        // Continue waiting
        if (attempt < maxAttempts - 1) {
          await new Promise((resolve) => setTimeout(resolve, delayMs));
        }
      } catch (error: any) {
        lastError = error.message;
        if (attempt === maxAttempts - 1) {
          throw new Error(
            `Timeout waiting for cluster ${identifier} to establish peering: ${error.message}`,
          );
        }
        // Continue waiting on transient errors
        await new Promise((resolve) => setTimeout(resolve, delayMs));
      }
    }

    throw new Error(
      `Timeout waiting for cluster ${identifier} to establish peering after ${
        (maxAttempts * delayMs) / 1000
      } seconds. ` +
        `Last status: ${lastStatus}, peer clusters: ${lastPeerCount}${
          lastError ? `, last error: ${lastError}` : ""
        }`,
    );
  }

  async update(
    id: string,
    olds: DsqlPeeringInputs,
    news: DsqlPeeringInputs,
  ): Promise<dynamic.UpdateResult<DsqlPeeringOutputs>> {
    // For simplicity, require replacement for any changes
    // This ensures clean peering setup without complex update logic
    throw new Error("DSQL peering updates require replacement.");
  }

  async delete(id: string, props: DsqlPeeringOutputs): Promise<void> {
    // Import AWS SDK inside the method to avoid serialization issues
    const { DSQLClient, DeleteClusterCommand } = await import(
      "@aws-sdk/client-dsql"
    );

    // CRITICAL: Coordinated deletion to solve deadlock
    const primaryClient = new DSQLClient({ region: props.primaryRegion });
    const peerClient = new DSQLClient({ region: props.peerRegion });

    try {
      // Simultaneously initiate deletion - THIS SOLVES THE DEADLOCK
      // Note: Clusters must have deletion protection disabled by the user beforehand
      await Promise.all([
        primaryClient.send(
          new DeleteClusterCommand({
            identifier: props.primaryClusterIdentifier,
          }),
        ),
        peerClient.send(
          new DeleteClusterCommand({
            identifier: props.peerClusterIdentifier,
          }),
        ),
      ]);

      // No waiting needed - individual dsql.Cluster resources handle deletion completion
      // Pulumi automatically waits for all resources to finish before considering deletion complete
    } catch (error: any) {
      // Check for deletion protection error and provide helpful message
      if (
        error.message?.includes("deletion protection") ||
        error.message?.includes("DeletionProtectionEnabled")
      ) {
        throw new Error(
          "Cannot delete DSQL clusters with deletion protection enabled. " +
            "Please set deletionProtection: false on both clusters before destroying.",
        );
      }

      // Log but don't fail deletion if clusters are already being deleted
      if (
        !error.message?.includes("does not exist") &&
        !error.message?.includes("PENDING_DELETE")
      ) {
        throw new Error(
          `Failed to coordinate DSQL cluster deletion: ${error.message}`,
        );
      }
    }
  }
}
