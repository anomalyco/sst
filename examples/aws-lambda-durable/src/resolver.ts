import { APIGatewayProxyEventV2 } from "aws-lambda";
import {
  LambdaClient,
  SendDurableExecutionCallbackSuccessCommand,
  SendDurableExecutionCallbackFailureCommand,
  SendDurableExecutionCallbackHeartbeat$,
  SendDurableExecutionCallbackHeartbeatCommand,
} from "@aws-sdk/client-lambda";

type Event = {};

const lambdaClient = new LambdaClient();

export const handler = async (event: APIGatewayProxyEventV2) => {
  const { callbackId, action } = event.queryStringParameters || {};

  if (!callbackId) {
    return {
      statusCode: 400,
      body: JSON.stringify({ message: "Missing callbackId in query parameters" }),
    };
  }

  let command = new SendDurableExecutionCallbackSuccessCommand({
    CallbackId: callbackId!,
    Result: JSON.stringify({ message: "Callback success!" }),
  });

  if (action === "failure") {
    command = new SendDurableExecutionCallbackFailureCommand({
      CallbackId: callbackId!,
      Error: {
        ErrorData: JSON.stringify({ message: "Callback failure!" }),
        ErrorType: "CallbackError",
        ErrorMessage: "An error occurred during the callback execution.",
      },
    });
  }

  if(action === "heartbeat") {
    command = new SendDurableExecutionCallbackHeartbeatCommand({
      CallbackId: callbackId!,
    });
  }

  await lambdaClient.send(command);

  return {
    statusCode: 200,
    body: JSON.stringify({ message: "Callback sent successfully!" }),
  };
};
