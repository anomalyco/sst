import type { ExecutionContext, Hyperdrive } from "@cloudflare/workers-types";
import postgres from "postgres";

export interface Env {
	// If you set another name in wrangler.toml as the value for 'binding',
	// replace "HYPERDRIVE" with the variable name you defined.
	HYPERDRIVE: Hyperdrive;
}

export default {
	async fetch(request: Request, env: Env, ctx: ExecutionContext) {
		// NOTE: if `prepare: false` is passed when connecting, performance will
		// be slower but still correctly supported.
		const sql = postgres(env.HYPERDRIVE.connectionString);

		try {
			// A very simple test query
			const now = Date.now();
			const result = await sql`select * from pg_tables limit 10`;
			const delay = Date.now() - now;

			// Clean up the client, ensuring we don't kill the worker before that is
			// completed.
			ctx.waitUntil(sql.end());

			// Return result rows as JSON
			return Response.json({ delay, result });
		} catch (e) {
			console.log(e);
			return Response.json({ error: e.message }, { status: 500 });
		}
	},
};
