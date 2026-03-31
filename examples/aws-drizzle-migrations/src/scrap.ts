import { eq } from "drizzle-orm";
import { db } from "./drizzle";
import { user } from "./todo.sql";

const uuid = "d997d46d-5769-4c78-9a35-93acadbe6076";
await db.query.user.findMany({
  where: eq(user.id, uuid),
  with: {
    todos: {
      with: {
        todo: true,
      },
    },
  },
});
