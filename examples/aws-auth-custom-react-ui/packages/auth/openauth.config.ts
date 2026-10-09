import { defineConfig } from "@openauthjs/vite";
import { react } from "@openauthjs/react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  integrations: [react()],
  plugins: [tailwindcss()],
});
