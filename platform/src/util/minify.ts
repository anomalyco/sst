import { transformSync } from "esbuild";

export function minify(strings: TemplateStringsArray, ...values: unknown[]) {
  const code = String.raw(strings, ...values);
  return transformSync(code, {
    minifyWhitespace: true,
    minifySyntax: true,
    target: "es2018",
  }).code;
}
