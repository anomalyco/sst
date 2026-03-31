import { api } from "./api";
import { auth } from "./auth";

export const web = new sst.aws.StaticSite("MyWeb", {
  path: "packages/web",
  build: {
    output: "dist",
    command: "npm run build",
  },
  environment: {
    VITE_API_URL: api.url,
    VITE_AUTH_URL: auth.url,
  },
});

