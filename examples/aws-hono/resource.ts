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
  async delete(id, inputs) {
    console.log("remove");
  },
  async update(id, state, inputs) {
    return {
      ...state.outputs,
      updated: Date.now(),
    };
  },
});
