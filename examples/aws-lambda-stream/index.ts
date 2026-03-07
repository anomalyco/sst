import { APIGatewayProxyEventV2 } from "aws-lambda";

export const handler = awslambda.streamifyResponse(myHandler);

async function myHandler(
  _event: APIGatewayProxyEventV2,
  responseStream: awslambda.HttpResponseStream
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
