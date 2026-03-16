import {
  createTRPCClient,
  httpBatchStreamLink,
  TRPCClientError,
} from "@trpc/client";
import { ApiRouter } from "./trpc-server";

const client = createTRPCClient<ApiRouter>({
  links: [
    httpBatchStreamLink({
      url: process.env.TRPC_SERVER_URL,
      methodOverride: "POST",
    }),
  ],
});

export const main = async () => {
  await Promise.all(
    Array.from({ length: 10 }, async (_, idx) => {
      const delay = Math.random() * 8_000;
      console.log("sending request", idx);
      try {
        const res = await client.getById.query({ delay, idx });
        console.log("got response", res.idx);
      } catch (err) {
        if (isTrpcError(err)) {
          console.error("received error", idx, err.shape.message);
        } else {
          console.error("received unexpecte error", idx);
        }
      }
    }),
  );
};

const isTrpcError = (err: unknown): err is TRPCClientError<ApiRouter> => {
  return !!err && err instanceof TRPCClientError;
};

main().catch(console.error);
