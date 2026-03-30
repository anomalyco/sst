import { db } from "./drizzle";
import { todosTable } from "./schema";
import { APIGatewayProxyHandlerV2 } from "aws-lambda";

export const handler: APIGatewayProxyHandlerV2 = async (evt) => {
  if (evt.requestContext.http.method === "GET") {
    const result = await db.select().from(todosTable).execute();
    return {
      statusCode: 200,
      body: JSON.stringify(result),
    };
  }

  if (evt.requestContext.http.method === "POST") {
    await db.insert(todosTable).values({ title: "new todosTable" }).execute();
    return {
      statusCode: 200,
      body: "created",
    };
  }

  return {
    statusCode: 404,
    body: "not found",
  };
};
