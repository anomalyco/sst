import postgres from "postgres";
import { Resource } from "sst";

export default {
  async fetch(_request: Request, _env: unknown, ctx: ExecutionContext) {
    const sql = postgres(Resource.Database.connectionString);

    try {
      const now = Date.now();
      const result = await sql`select * from pg_tables limit 10`;
      const delay = Date.now() - now;

      ctx.waitUntil(sql.end());

      return Response.json({ delay, result });
    } catch (e: any) {
      console.log(e);
      return Response.json({ error: e.message }, { status: 500 });
    }
  },
};
