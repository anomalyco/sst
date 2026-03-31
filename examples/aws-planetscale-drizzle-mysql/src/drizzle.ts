import { drizzle } from "drizzle-orm/planetscale-serverless";
import { Resource } from "sst";

export const db = drizzle({
  connection: {
    host: Resource.Database.host,
    username: Resource.Database.username,
    password: Resource.Database.password,
  },
});
