import { defineConfig } from "astro/config";
import cloudflare from "@astrojs/cloudflare";

export default defineConfig({
  adapter: cloudflare({
    configPath: process.env.SST_WRANGLER_PATH,
  }),
});
