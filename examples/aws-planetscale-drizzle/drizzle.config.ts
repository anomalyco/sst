import { defineConfig } from "drizzle-kit";
import { Resource } from "sst";

export default defineConfig({
  dialect: "mysql",
  dbCredentials: {
    host: Resource.Database.host,
    user: Resource.Database.username,
    password: Resource.Database.password,
    database: Resource.Database.database,
  },
  schema: ["./src/schema.ts"],
});
