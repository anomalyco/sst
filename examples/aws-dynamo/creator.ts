import { Resource } from "sst";
import { DynamoDBClient, PutItemCommand } from "@aws-sdk/client-dynamodb";
const client = new DynamoDBClient();

export const handler = async (event) => {
  await client.send(
    new PutItemCommand({
      TableName: Resource.MyTable.name,
      Item: {
        id: { S: Date.now().toString() },
        message: { S: "Hello" },
      },
    })
  );

  return {
    statusCode: 200,
    body: JSON.stringify({ status: "sent" }, null, 2),
  };
};
