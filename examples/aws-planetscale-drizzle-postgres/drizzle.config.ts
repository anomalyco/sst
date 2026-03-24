import { defineConfig } from "drizzle-kit";
import { Resource } from "sst";

export default defineConfig({
  dialect: "postgresql",
  dbCredentials: {
    url: `postgresql://${Resource.Database.username}:${Resource.Database.password}@${Resource.Database.host}/${Resource.Database.database}?sslmode=require`,
  },
  schema: ["./src/schema.ts"],
});
