import { defineConfig } from "astro/config";
import aws from "astro-sst";

export default defineConfig({
  output: "server",
  adapter: aws({
    serverRoutes: ["/_actions/getGreeting"],
  }),
});
