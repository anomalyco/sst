import { reactRouter } from "@react-router/dev/vite";
import { cloudflare } from "@cloudflare/vite-plugin";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";
import tsconfigPaths from "vite-tsconfig-paths";

export default defineConfig({
  plugins: [
    cloudflare({
      viteEnvironment: { name: "ssr" },
      configPath: process.env.SST_WRANGLER_PATH,
    }),
    tailwindcss(),
    reactRouter(),
    tsconfigPaths({ projects: ["./tsconfig.json"] }),
  ],
});
