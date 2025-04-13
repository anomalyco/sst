import postgres from "postgres";
import { Resource } from "sst";

const sql = postgres(
	`postgres://${Resource.Postgres.username}:${Resource.Postgres.password}@${Resource.Postgres.host}:${Resource.Postgres.port}/${Resource.Postgres.database}`,
);

export async function handler() {
	try {
		// A very simple test query
		const now = Date.now();
		const result = await sql`select * from pg_tables limit 10`;
		const delay = Date.now() - now;
		return {
			statusCode: 200,
			body: JSON.stringify({
				result,
				delay,
			}),
		};
	} catch (e) {
		return {
			statusCode: 200,
			body: JSON.stringify({
				error: e.message,
			}),
		};
	}
}
