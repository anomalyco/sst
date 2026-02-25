import { db } from "./drizzle";
import { migrate } from "drizzle-orm/postgres-js/migrator";

export const handler = async (event: any) => {
  await migrate(db, {
    migrationsFolder: "./migrations",
  });
};