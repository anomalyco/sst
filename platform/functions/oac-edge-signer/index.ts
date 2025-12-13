// Lambda@Edge function for automatic SHA256 header signing
// This function adds the required x-amz-content-sha256 header for POST/PUT/PATCH requests
// going to Lambda function URLs with Origin Access Control enabled.

import crypto from "crypto";

interface CloudFrontRequest {
  method: string;
  uri: string;
  body?: {
    data: string;
    encoding?: "base64" | "text";
  };
  headers: Record<string, Array<{ key: string; value: string }>>;
}

interface CloudFrontEvent {
  Records: Array<{
    cf: {
      request: CloudFrontRequest;
    };
  }>;
}

export async function handler(event: CloudFrontEvent) {
  const request = event.Records[0].cf.request;

  // Only process requests that need SHA256 signing (methods with body)
  if (!["POST", "PUT", "PATCH", "DELETE"].includes(request.method)) {
    return request;
  }

  try {
    // Get the request body
    let bodyString = "";

    if (request.body && request.body.data) {
      // Lambda@Edge provides body as base64-encoded string
      if (request.body.encoding === "base64") {
        // Decode base64 to get the actual body content
        bodyString = Buffer.from(request.body.data, "base64").toString("utf8");
      } else {
        // If not base64 encoded, use as-is
        bodyString = request.body.data;
      }
    }

    // Compute SHA256 hash of the body
    const hash = crypto
      .createHash("sha256")
      .update(bodyString, "utf8")
      .digest("hex");

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
}
