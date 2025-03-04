// @ts-check
import { defineConfig } from "astro/config";
import aws from "astro-sst";
//import aws from "../../../../../../astro-sst/packages/astro-sst/dist/adapter";

// https://astro.build/config
export default defineConfig({
  image: {
    domains: ["sst.dev"],
  },
  output: "server",
  adapter: aws(),
});
