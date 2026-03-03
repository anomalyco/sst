import { InvokeCommand, LambdaClient } from "@aws-sdk/client-lambda";
import { Resource } from "sst";
const client = new LambdaClient();

export const handler = async () => {
  const command = new InvokeCommand({
    FunctionName: Resource.Durable.name,
    Qualifier: "$LATEST",
    InvocationType: "Event",
  });

  await client.send(command);

  return { message: "Durable function invoked successfully!" };
};
