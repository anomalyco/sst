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
await Promise.all(
  ["cloudfront-router-handler", "cloudfront-site-handler"].map(async (dir) => {
    await fs.mkdir(`./dist/${dir}/`, { recursive: true });
    const code = await fs.readFile(`./functions/${dir}/index.js`, "utf8");
    const result = await esbuild.transform(code, {
      minifyWhitespace: true,
      minifySyntax: true,
      // CloudFront Functions don't support optional catch binding (`catch {}`)
      // despite claiming ES2021 support. Target es2018 to avoid it.
      target: "es2018",
    });
    await fs.writeFile(`./dist/${dir}/index.js`, result.code);
  }),
);
