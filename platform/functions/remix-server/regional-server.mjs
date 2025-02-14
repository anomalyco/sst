// This is a custom Lambda URL handler which imports the Remix server
// build and performs the Remix server rendering.

import { createRequestHandler as createNodeRequestHandler } from "@remix-run/node";
import { pipeline } from "node:stream/promises";
import { Readable } from "node:stream";
import { createGzip, createDeflate, createBrotliCompress } from "node:zlib";

function convertApigRequestToNode(event) {
  if (event.headers["x-forwarded-host"]) {
    event.headers.host = event.headers["x-forwarded-host"];
  }

  const search = event.rawQueryString.length ? `?${event.rawQueryString}` : "";
  const url = new URL(event.rawPath + search, `https://${event.headers.host}`);
  const isFormData = event.headers["content-type"]?.includes(
    "multipart/form-data",
  );

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
        ? isFormData
          ? Buffer.from(event.body, "base64")
          : Buffer.from(event.body, "base64").toString()
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

    if (response.body) {
      const acceptEncodingHeader = event.headers["accept-encoding"] || "";
      const acceptEncodings = acceptEncodingHeader.split(",");

      // ordered by precedence
      const compressionMap = {
        br: createBrotliCompress,
        gzip: createGzip,
        deflate: createDeflate,
      };

      const contentEncoding = Object.keys(compressionMap).find((encoding) =>
        acceptEncodings.includes(encoding),
      );

      const readable = Readable.fromWeb(response.body);
      const pipelineComponents = [readable];

      // If the client accepts an encoding, we'll compress the response body
      // and add the encoding to the response headers.
      if (contentEncoding) {
        httpResponseMetadata.headers["content-encoding"] = contentEncoding;
        pipelineComponents.push(compressionMap[contentEncoding]());
      }

      const writer = awslambda.HttpResponseStream.from(
        responseStream,
        httpResponseMetadata,
      );
      pipelineComponents.push(writer);

      await pipeline(...pipelineComponents);
    } else {
      const writer = awslambda.HttpResponseStream.from(
        responseStream,
        httpResponseMetadata,
      );

      // without this, redirects will cause a server error
      writer.write(" ");
      writer.end();
    }
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

export const handler = createApigHandler(remixServerBuild);
