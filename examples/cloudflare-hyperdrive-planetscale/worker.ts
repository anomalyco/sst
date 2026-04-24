import { Client } from "pg";
import { Resource } from "sst";

export default {
  async fetch() {
    const client = new Client({
      connectionString: Resource.Database.connectionString,
    });

    try {
      await client.connect();

      const result = await client.query(`SELECT * FROM "user"`);

      return Response.json({
        success: true,
        result: result.rows,
      });
    } catch (error: any) {
      console.error("Database error:", error.message);

      return new Response("Internal error occurred", { status: 500 });
    }
  },
};
