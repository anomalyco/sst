// This is a custom Lambda URL handler which imports the Remix server
// build and performs the Remix server rendering.

import { createRequestHandler as createNodeRequestHandler } from "@remix-run/node";

function convertApigRequestToNode(event) {
  if (event.headers["x-forwarded-host"]) {
    event.headers.host = event.headers["x-forwarded-host"];
  }

  const search = event.rawQueryString.length ? `?${event.rawQueryString}` : "";
  const url = new URL(event.rawPath + search, `https://${event.headers.host}`);
  // Build headers
  const headers = new Headers();
  for (let [header, value] of Object.entries(event.headers)) {
    if (value) {
      headers.append(header, value);
    }
  }

  return new Request(url.href, {
    method: event.requestContext.http.method,
    headers,
    body:
      event.body && event.isBase64Encoded
        ? Buffer.from(event.body, "base64")
        : event.body,
  });
}

const createApigHandler = (build) => {
  const requestHandler = createNodeRequestHandler(build, process.env.NODE_ENV);

  return awslambda.streamifyResponse(async (event, responseStream, context) => {
    context.callbackWaitsForEmptyEventLoop = false;
    const request = convertApigRequestToNode(event);
    const response = await requestHandler(request);
    const httpResponseMetadata = {
      statusCode: response.status,
      headers: {
        ...Object.fromEntries(response.headers.entries()),
        "Transfer-Encoding": "chunked",
      },
      cookies: accumulateCookies(response.headers),
    };

    const writer = awslambda.HttpResponseStream.from(
      responseStream,
      httpResponseMetadata,
    );

    if (response.body) {
      await streamToNodeStream(response.body.getReader(), responseStream);
    } else {
      writer.write(" ");
    }
    writer.end();
  });
};

const accumulateCookies = (headers) => {
  // node >= 19.7.0 with no remix fetch polyfill
  if (typeof headers.getSetCookie === "function") {
    return headers.getSetCookie();
  }
  // node < 19.7.0 or with remix fetch polyfill
  const cookies = [];
  for (let [key, value] of headers.entries()) {
    if (key === "set-cookie") {
      cookies.push(value);
    }
  }
  return cookies;
};

const streamToNodeStream = async (reader, writer) => {
  let readResult = await reader.read();
  while (!readResult.done) {
    writer.write(readResult.value);
    readResult = await reader.read();
  }
  writer.end();
};

export const handler = createApigHandler(remixServerBuild);
