import { defineConfig } from "vitest/config";

export default defineConfig({
  resolve: {
    alias: {
      // @types/node has empty "exports": {} which Vite can't resolve.
      // It's type-only, so alias to the .d.ts file at runtime.
      "@types/node": "@types/node/index.d.ts",
    },
  },
});
