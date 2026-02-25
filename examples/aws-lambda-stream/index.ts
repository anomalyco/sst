import { APIGatewayProxyEventV2 } from "aws-lambda";
import { streamifyResponse, ResponseStream } from "lambda-stream";

export const handler = streamifyResponse(myHandler);

async function myHandler(
  _event: APIGatewayProxyEventV2,
  responseStream: ResponseStream
): Promise<void> {
  return new Promise((resolve, _reject) => {
    responseStream.setContentType('text/plain')
    responseStream.write('Hello')
    setTimeout(() => {
      responseStream.write(' World')
      responseStream.end()
      resolve()
    }, 3000)
  })
}
