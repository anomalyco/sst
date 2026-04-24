import { Client } from "pg";
import { Resource } from "sst";

export default {
  async fetch() {
    const client = new Client({
      connectionString: Resource.Database.connectionString,
    });

    await client.connect();

    const result = await client.query(`SELECT NOW()`);

    return Response.json({
      success: true,
      result: result.rows,
    });
  },
};
