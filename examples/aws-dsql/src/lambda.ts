import { DsqlSigner } from "@aws-sdk/dsql-signer";
import { Client } from "pg";
import { Resource } from "sst";

export const handler = async (event: any) => {
  try {
    // Use the Resource object to get the region and endpoint
    const signer = new DsqlSigner({
      region: Resource.MyCluster.region,
      hostname: Resource.MyCluster.publicEndpoint,
    });

    // Generate the token
    const token = await signer.getDbConnectAdminAuthToken();

    // Connect using the token and endpoint from the Resource object
    const client = new Client({
      host: Resource.MyCluster.publicEndpoint,
      port: 5432,
      user: "admin",
      password: token,
      database: "postgres",
      ssl: true,
    });

    await client.connect();
    const now = await client.query("SELECT NOW() as now");
    await client.end();

    return {
      statusCode: 200,
      body: JSON.stringify({
        message: "Successfully connected to DSQL cluster.",
        now: now.rows[0].now,
      }),
    };
  } catch (error) {
    console.error("Error accessing DSQL cluster:", error);
    return {
      statusCode: 500,
      body: JSON.stringify({
        error: "Failed to access DSQL cluster",
        details: error instanceof Error ? error.message : String(error),
      }),
    };
  }
};
