export const auth = new sst.aws.Auth("MyAuth", {
  issuer: "packages/functions/src/auth.handler",
});

