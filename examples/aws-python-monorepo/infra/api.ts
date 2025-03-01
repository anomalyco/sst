import { bucket } from "./storage";

const linkableValue = new sst.Linkable("MyLinkableValue", {
  properties: {
    foo: "Hello World",
  },
});

export const myPythonApi = new sst.aws.Function("MyPythonApi", {
  runtime: "python3.11",
  python: {
    container: true,
    monorepoPath: "apps/backend/src",
  },
  handler: "apps/backend/src/functions/api.handler",
  url: true,
  link: [bucket, linkableValue],
});
