// Streaming Lambda handler for SvelteKit.
// Copied by SST at deploy time — replaces the buffered handler from svelte-kit-sst.

import fs from "node:fs";
import path from "node:path";
import { installPolyfills } from "@sveltejs/kit/node/polyfills";
import { Server } from "./index.js";
import { manifest } from "./manifest.js";
import prerenderedFiles from "./lambda-handler/prerendered-file-list.js";

installPolyfills();

const app = new Server(manifest);
await app.init({ env: process.env });

export function convertLambdaRequestToNode(event) {
  if (event.headers["x-forwarded-host"]) {
    event.headers.host = event.headers["x-forwarded-host"];
  }

  const search = event.rawQueryString.length ? `?${event.rawQueryString}` : "";
  const url = new URL(event.rawPath + search, `https://${event.headers.host}`);

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

export function isPrerenderedFile(uri, prerenderedFiles) {
  uri = uri.replace(/^\/|\/$/g, "");

  if (uri === "") {
    return prerenderedFiles.includes("index.html") ? "index.html" : undefined;
  }

  if (prerenderedFiles.includes(uri)) {
    return uri;
  }
  if (prerenderedFiles.includes(uri + "/index.html")) {
    return uri + "/index.html";
  }
  if (prerenderedFiles.includes(uri + ".html")) {
    return uri + ".html";
  }
}

export const handler = awslambda.streamifyResponse(
  async (event, responseStream, context) => {
    context.callbackWaitsForEmptyEventLoop = false;

    // Check for prerendered files on GET requests
    if (event.requestContext?.http?.method === "GET") {
      const filePath = isPrerenderedFile(event.rawPath, prerenderedFiles);
      if (filePath) {
        const body = fs.readFileSync(
          path.join("prerendered", filePath),
          "utf8",
        );
        const writer = awslambda.HttpResponseStream.from(responseStream, {
          statusCode: 200,
          headers: {
            "content-type": "text/html",
            "cache-control":
              "public, max-age=0, s-maxage=31536000, must-revalidate",
          },
        });
        writer.write(body);
        writer.end();
        return;
      }
    }

    // Handle dynamic requests through SvelteKit
    const request = convertLambdaRequestToNode(event);
    const response = await app.respond(request, {
      getClientAddress: () => event.requestContext.http.sourceIp,
    });

    const writer = awslambda.HttpResponseStream.from(responseStream, {
      statusCode: response.status,
      headers: {
        ...Object.fromEntries(response.headers.entries()),
        "Transfer-Encoding": "chunked",
      },
      cookies: response.headers.getSetCookie(),
    });

    if (response.body) {
      const reader = response.body.getReader();
      let readResult = await reader.read();
      while (!readResult.done) {
        writer.write(readResult.value);
        readResult = await reader.read();
      }
    } else {
      writer.write(" ");
    }
    writer.end();
  },
);
