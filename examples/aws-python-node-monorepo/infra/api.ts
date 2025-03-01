import { bucket } from "./storage";

const linkableValue = new sst.Linkable("MyLinkableValue", {
  properties: {
    foo: "Hello World",
  },
});

export const myNodeApi = new sst.aws.Function("MyNodeApi", {
	url: true,
  runtime: "nodejs22.x",
	link: [bucket, linkableValue],
	handler: "apps/node-backend/src/api.handler",
});

export const myPythonApi = new sst.aws.Function("MyPythonApi", {
  runtime: "python3.11",
  python: {
    monorepoPath: "apps/python-backend/src",
  },
  handler: "apps/python-backend/src/functions/api.handler",
  url: true,
  link: [bucket, linkableValue],
});

export const myPythonContainerApi = new sst.aws.Function("MyPythonContainerApi", {
  runtime: "python3.11",
  python: {
    container: true,
    monorepoPath: "apps/python-backend/src",
  },
  handler: "apps/python-backend/src/functions/api.handler",
  url: true,
  link: [bucket, linkableValue],
});


