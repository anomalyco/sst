import { getCloudflarePreset } from "@cloudflare/unenv-preset";
import { defineEnv } from "unenv";
import { fileURLToPath } from "node:url";

export function getCloudflareUnenvConfig(input = {}) {
  const { env } = defineEnv({
    npmShims: true,
    presets: [
      getCloudflarePreset({
        compatibilityDate: input.date,
        compatibilityFlags: input.flags,
      }),
    ],
  });

  return {
    alias: env.alias,
    external: env.external,
    polyfill: env.polyfill,
  };
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const input = process.argv[2] ? JSON.parse(process.argv[2]) : {};
  process.stdout.write(JSON.stringify(getCloudflareUnenvConfig(input)));
}
