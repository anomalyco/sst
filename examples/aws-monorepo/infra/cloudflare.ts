import { api } from "./api";

const bucket = new sst.cloudflare.Bucket("CfBucket");
export const worker = new sst.cloudflare.Worker("MyWorker", {
  handler: "packages/cloudflare-worker/src/index.ts",
  link: [bucket],
  url: true,
});
