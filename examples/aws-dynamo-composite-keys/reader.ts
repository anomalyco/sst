import { Resource } from "sst";
import { DynamoDBClient, QueryCommand } from "@aws-sdk/client-dynamodb";
const client = new DynamoDBClient();

export const handler = async () => {
  const result = await client.send(
    new QueryCommand({
      TableName: Resource.MyTable.name,
      IndexName: "RegionCategoryIndex",
      KeyConditionExpression: "#r = :region AND #c = :category",
      ExpressionAttributeNames: {
        "#r": "region",
        "#c": "category",
      },
      ExpressionAttributeValues: {
        ":region": { S: "us-east" },
        ":category": { S: "notes" },
      },
    })
  );

  return {
    statusCode: 200,
    body: JSON.stringify(result.Items, null, 2),
  };
};
