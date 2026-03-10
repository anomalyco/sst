import { Resource } from "sst";
import { DynamoDBClient, ScanCommand } from "@aws-sdk/client-dynamodb";
const client = new DynamoDBClient();

export const handler = async () => {
  const result = await client.send(
    new ScanCommand({
      TableName: Resource.MyTable.name,
    })
  );

  return {
    statusCode: 200,
    body: JSON.stringify(result.Items, null, 2),
  };
};
