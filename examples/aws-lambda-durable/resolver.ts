import { APIGatewayProxyEventV2 } from "aws-lambda";
import {
  LambdaClient,
  SendDurableExecutionCallbackSuccessCommand,
} from "@aws-sdk/client-lambda";

type Event = {};

const lambdaClient = new LambdaClient();

export const handler = async (event: APIGatewayProxyEventV2) => {
  const { callbackId } = event.queryStringParameters || {};

  const command = new SendDurableExecutionCallbackSuccessCommand({
    CallbackId: callbackId!,
    Result: JSON.stringify({ message: "Callback success!" }),
  });

  await lambdaClient.send(command);

  return {
    statusCode: 200,
    body: JSON.stringify({ message: "Callback sent successfully!" }),
  };
};
