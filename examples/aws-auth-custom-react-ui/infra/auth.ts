export const auth = new sst.aws.CustomAuth("MyAuth", {
  path: "packages/auth",
  dev: {
    url: "http://localhost:5174" // Unfortunately, SST cannot detect the dev host yet, so always remember to set it here
  },
});
