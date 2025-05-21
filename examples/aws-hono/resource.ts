import { S3Client } from "@aws-sdk/client-s3";

export const MyResource = sst.resource({
  async create(inputs: { butt: number }) {
    console.log(S3Client);
    return {
      id: "123",
      outputs: {
        hello: "world",
        updated: Date.now(),
      },
    };
  },
});
