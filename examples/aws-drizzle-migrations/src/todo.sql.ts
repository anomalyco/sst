import { text, serial, pgTable } from "drizzle-orm/pg-core";

export const todo = pgTable("todo", {
  id: serial("id").primaryKey(),
  title: text("title").notNull(),
  description: text("description"),
});
