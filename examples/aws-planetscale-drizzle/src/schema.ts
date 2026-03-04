import { mysqlTable, int, text, varchar } from "drizzle-orm/mysql-core";

export const todosTable = mysqlTable("todo", {
  id: int("id").primaryKey().autoincrement(),
  title: varchar("title", { length: 255 }).notNull(),
  description: text("description"),
});
