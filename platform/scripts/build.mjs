import esbuild from "esbuild";
import fs from "fs/promises";

// Require transpile
await Promise.all(
  ["ssr-warmer", "vector-handler"].map((file) =>
    esbuild.build({
      bundle: true,
      minify: true,
      platform: "node",
      target: "esnext",
      format: "esm",
      entryPoints: [`./functions/${file}/index.ts`],
      banner: {
        js: [
          `import { createRequire as topLevelCreateRequire } from 'module';`,
          `const require = topLevelCreateRequire(import.meta.url);`,
        ].join(""),
      },
      outfile: `./dist/${file}/index.mjs`,
    }),
  ),
);
