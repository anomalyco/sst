import { AuroraDSQLClient } from "@aws/aurora-dsql-node-postgres-connector";
import { Resource } from "sst";

export const handler = async () => {
  try {
    const client = new AuroraDSQLClient({
      host: Resource.MyCluster.endpoint,
      user: "admin",
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
