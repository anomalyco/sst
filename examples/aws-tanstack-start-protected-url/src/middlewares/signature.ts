import { createMiddleware } from "@tanstack/react-start";

// See https://developer.mozilla.org/en-US/docs/Web/API/SubtleCrypto/digest
async function digestMessage(message: string) {
  const msgUint8 = new TextEncoder().encode(message);
  const hashBuffer = await crypto.subtle.digest("SHA-256", msgUint8);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  const hashHex = hashArray
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");

  return hashHex;
}

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
  },
);
