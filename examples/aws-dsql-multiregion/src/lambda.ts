import { DsqlSigner } from "@aws-sdk/dsql-signer";
import { Client } from "pg";
import { Resource } from "sst";

async function connectToCluster(region: string, endpoint: string) {
  const signer = new DsqlSigner({
    region,
    hostname: endpoint,
  });
  const token = await signer.getDbConnectAdminAuthToken();
  const client = new Client({
    host: endpoint,
    port: 5432,
    user: "admin",
    password: token,
    database: "postgres",
    ssl: true,
  });

  await client.connect();
  const result = await client.query("SELECT NOW() as now");
  await client.end();

  return result.rows[0].now;
}

export const handler = async (event: any) => {
  try {
    // Connect to the primary cluster
    const primaryTime = await connectToCluster(
      Resource.MyCluster.region,
      Resource.MyCluster.publicEndpoint,
    );

    // Connect to the peer cluster
    const peerTime = await connectToCluster(
      Resource.MyCluster.peerRegion,
      Resource.MyCluster.peerPublicEndpoint,
    );

    return {
      statusCode: 200,
      body: JSON.stringify({
        message: "Successfully connected to both DSQL clusters.",
        primaryTime,
        peerTime,
      }),
    };
  } catch (error) {
    console.error("Error accessing DSQL clusters:", error);
    return {
      statusCode: 500,
      body: JSON.stringify({
        error: "Failed to access DSQL clusters",
        details: error instanceof Error ? error.message : String(error),
      }),
    };
  }
};
