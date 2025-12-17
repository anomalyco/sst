// Lambda@Edge function for automatic SHA256 header signing
// This function adds the required x-amz-content-sha256 header for POST/PUT/PATCH requests
// going to Lambda function URLs with Origin Access Control enabled.

import { CloudFrontRequestHandler } from "aws-lambda";
import crypto from "node:crypto";

export const handler: CloudFrontRequestHandler = async (event) => {
  const request = event.Records[0].cf.request;

  // Only process requests that need SHA256 signing (methods with body)
  if (!["POST", "PUT", "PATCH", "DELETE"].includes(request.method)) {
    return request;
  }

  // Check if body was truncated (exceeds 1MB Lambda@Edge limit)
  if (request.body?.inputTruncated) {
    return {
      status: "413",
      statusDescription: "Payload Too Large",
      headers: {
        "content-type": [{ key: "Content-Type", value: "application/json" }],
      },
      body: JSON.stringify({
        error:
          "Request body exceeds 1MB Lambda@Edge limit. Use presigned S3 URLs for large uploads.",
      }),
    };
  }

  try {
    // Get the request body as raw bytes (never convert to UTF-8 string)
    const bodyBuffer = request.body?.data
      ? request.body.encoding === "base64"
        ? Buffer.from(request.body.data, "base64")
        : Buffer.from(request.body.data)
      : Buffer.alloc(0);

    // Compute SHA256 hash of the raw bytes
    const hash = crypto.createHash("sha256").update(bodyBuffer).digest("hex");

    // Add the x-amz-content-sha256 header in CloudFront format
    request.headers["x-amz-content-sha256"] = [
      {
        key: "x-amz-content-sha256",
        value: hash,
      },
    ];

    console.log(
      `Added SHA256 header for ${request.method} request to ${request.uri}: ${hash}`,
    );
  } catch (error) {
    console.error("Error computing SHA256 hash:", error);
    // Continue without the header rather than failing the request
  }

  return request;
};
