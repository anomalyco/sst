import { pgTable, serial, text, varchar } from "drizzle-orm/pg-core";

export const todosTable = pgTable("todo", {
  id: serial("id").primaryKey(),
  title: varchar("title", { length: 255 }).notNull(),
  description: text("description"),
});
