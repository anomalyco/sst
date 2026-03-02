import esbuild from "esbuild";
import fs from "fs/promises";

// Require transpile
await Promise.all(
  ["ssr-warmer", "vector-handler", "oac-edge-signer"].map((file) =>
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

// Minify CloudFront handler code
await fs.mkdir("./dist/cf-injection/", { recursive: true });
await Promise.all(
  ["router-handler", "site-handler"].map(async (file) => {
    const code = await fs.readFile(`./functions/cf-injection/${file}.js`, "utf8");
    const result = await esbuild.transform(code, {
      minifyWhitespace: true,
    });
    await fs.writeFile(`./dist/cf-injection/${file}.js`, result.code);
  }),
);
