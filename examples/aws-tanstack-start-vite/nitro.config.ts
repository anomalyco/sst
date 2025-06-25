import { defineNitroConfig } from "nitropack/config";
// https://github.com/TanStack/router/issues/4404#issuecomment-2971434717
export default defineNitroConfig({
  preset: "aws-lambda",
  awsLambda: { streaming: true },
});
