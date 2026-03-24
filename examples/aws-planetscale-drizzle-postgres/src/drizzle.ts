import { Resource } from "sst";
import { neon, neonConfig } from "@neondatabase/serverless";
import { drizzle } from "drizzle-orm/neon-http";

// Required for PlanetScale Postgres connections
neonConfig.fetchEndpoint = (host) => `https://${host}/sql`;
const sql = neon(
  `postgresql://${Resource.Database.username}:${Resource.Database.password}@${Resource.Database.host}:6432/postgres?sslmode=verify-full`,
);

export const db = drizzle({ client: sql });
