import { AuroraDSQLClient } from "@aws/aurora-dsql-node-postgres-connector";
import { Resource } from "sst";

async function connectToCluster(endpoint: string) {
  const client = new AuroraDSQLClient({
    host: endpoint,
    user: "admin",
  });

  await client.connect();
  const result = await client.query("SELECT NOW() as now");
  await client.end();

  return result.rows[0].now;
}

export const handler = async () => {
  try {
    const usEast1Time = await connectToCluster(Resource.MultiRegion.endpoint);

    const usEast2Time = await connectToCluster(Resource.MultiRegion.peer.endpoint);

    return {
      statusCode: 200,
      body: JSON.stringify({
        message: "Successfully connected to both DSQL clusters.",
        usEast1Time,
        usEast2Time,
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
