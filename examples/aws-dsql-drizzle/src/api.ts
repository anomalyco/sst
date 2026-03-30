import { db } from "./drizzle";
import { todo } from "./todo.sql";
import { APIGatewayProxyEventV2 } from "aws-lambda";

export const handler = async (evt: APIGatewayProxyEventV2) => {
  if (evt.requestContext.http.method === "GET") {
    const result = await db.select().from(todo).execute();

    return {
      statusCode: 200,
      body: JSON.stringify(result, null, 2),
    };
  }

  if (evt.requestContext.http.method === "POST") {
    const result = await db
      .insert(todo)
      .values({ title: "Todo", description: crypto.randomUUID() })
      .returning()
      .execute();

    return {
      statusCode: 200,
      body: JSON.stringify(result),
    };
  }
};
