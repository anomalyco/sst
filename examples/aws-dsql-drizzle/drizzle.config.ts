import { Resource } from "sst";
import { defineConfig } from "drizzle-kit";

export default defineConfig({
  dialect: "postgresql",
  schema: ["./src/**/*.sql.ts"],
  out: "./migrations",
  dbCredentials: {
    host: Resource.MyCluster.endpoint,
    port: 5432,
    user: "admin",
    password: process.env.DSQL_TOKEN!,
    database: "postgres",
    ssl: {
      rejectUnauthorized: false,
    },
  },
});
