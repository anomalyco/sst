import { defineConfig } from "drizzle-kit";
import { Resource } from "sst";

export default defineConfig({
  dialect: "mysql",
  dbCredentials: {
    url: `mysql://${Resource.Database.username}:${Resource.Database.password}@${Resource.Database.host}/${Resource.Database.database}?ssl={"rejectUnauthorized":true}`,
  },
  schema: ["./src/schema.ts"],
});
