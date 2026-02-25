import { Resource } from "sst";
import {
  ApiGatewayManagementApiClient,
  PostToConnectionCommand,
} from "@aws-sdk/client-apigatewaymanagementapi";

export async function connect(event) {
  console.log("!!! connect");

  // If subprotocols are requested, return the allowed protocol. In this example, we only
  // allow "MY_ALLOWED_PROTOCOL".
  const protocolHeader = event.headers["Sec-WebSocket-Protocol"];
  if (protocolHeader) {
    const subprotocols = protocolHeader.split(",").map((p) => p.trim());
    return subprotocols.includes("MY_ALLOWED_PROTOCOL")
      ? {
          statusCode: 200,
          headers: { "Sec-WebSocket-Protocol": "MY_ALLOWED_PROTOCOL" },
        }
      : { statusCode: 400 };
  }

  return { statusCode: 200 };
}

export async function disconnect(event) {
  console.log("!!! disconnect");
  return { statusCode: 200 };
}

export async function sendMessage(event) {
  console.log("!!! sendMessage");
  return { statusCode: 200 };
}

export async function catchAll(event) {
  console.log("!!! default");

  // Send a message back to the client
  const client = new ApiGatewayManagementApiClient({
    endpoint: Resource.MyApi.managementEndpoint,
  });
  await client.send(
    new PostToConnectionCommand({
      ConnectionId: event.requestContext.connectionId,
      Data: "Hey! What is this?",
    })
  );

  return { statusCode: 200 };
}
