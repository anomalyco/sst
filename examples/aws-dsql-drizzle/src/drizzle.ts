import { drizzle } from "drizzle-orm/node-postgres";
import { AuroraDSQLPool } from "@aws/aurora-dsql-node-postgres-connector";
import { Resource } from "sst";
import * as schema from "./todo.sql";

const pool = new AuroraDSQLPool({
  host: Resource.MyCluster.endpoint,
  user: "admin",
});

export const db = drizzle(pool, { schema });
