import { createMiddleware } from "@tanstack/react-start";
import { digestMessage } from "../utils/digestMessage";

export const signatureMiddleware = createMiddleware().client(
  async ({ next, data, context, method }) => {
    if (method !== "POST") {
      return next();
    }

    const payload = JSON.stringify({ data, context });
    const digestHex = await digestMessage(payload);

    return next({
      headers: {
        "x-amz-content-sha256": digestHex,
      },
    });
  }
);
