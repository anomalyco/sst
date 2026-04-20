import { text, bigint, pgTable } from "drizzle-orm/pg-core";

export const todo = pgTable("todo", {
  id: bigint("id", { mode: "number" }).primaryKey().generatedAlwaysAsIdentity(),
  title: text("title").notNull(),
  description: text("description"),
});
