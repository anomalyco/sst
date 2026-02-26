import { Resource } from "sst";
import { DynamoDBClient, PutItemCommand } from "@aws-sdk/client-dynamodb";
const client = new DynamoDBClient();

export const handler = async () => {
  await client.send(
    new PutItemCommand({
      TableName: Resource.MyTable.name,
      Item: {
        userId: { S: "user1" },
        noteId: { S: Date.now().toString() },
        region: { S: "us-east" },
        category: { S: "notes" },
        createdAt: { N: Date.now().toString() },
      },
    })
  );

  return {
    statusCode: 200,
    body: JSON.stringify({ status: "sent" }, null, 2),
  };
};
