import { drizzle } from "drizzle-orm/postgres-js";
import { Resource } from "sst";
import postgres from "postgres";

const client = postgres({
  host: Resource.Database.host,
  username: Resource.Database.username,
  password: Resource.Database.password,
  database: Resource.Database.database,
});

export const db = drizzle(client);
