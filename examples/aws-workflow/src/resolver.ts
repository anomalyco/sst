import { APIGatewayProxyEventV2 } from "aws-lambda";
import { workflow } from "sst/aws/workflow";

export const handler = async (event: APIGatewayProxyEventV2) => {
  const { action, callbackId, message } = event.queryStringParameters || {};

  if (!callbackId) {
    return {
      statusCode: 400,
      body: JSON.stringify({
        message: "Missing callbackId in query parameters",
      }),
    };
  }

  if (action === "heartbeat") {
    await workflow.heartbeat(callbackId);
    return {
      statusCode: 200,
      body: JSON.stringify({ message: "Callback heartbeat sent successfully!" }),
    };
  }

  if (action === "fail") {
    await workflow.fail(callbackId, {
      error: {
        data: {
          message: message ?? "Callback failure!",
        },
        message: message ?? "An error occurred during the callback execution.",
        type: "CallbackError",
      },
    });
  }

  if (action === "succeed") {
    await workflow.succeed(callbackId, {
      payload: {
        message: message ?? "Callback success!",
        resolvedAt: new Date().toISOString(),
      },
    });
  }

  return {
    statusCode: 200,
    body: JSON.stringify({ message: "Callback sent successfully!" }),
  };
};
