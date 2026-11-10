import { transformSync } from "esbuild";

const options = {
  minifyWhitespace: true,
  minifySyntax: true,
  target: "es2018",
} as const;

export function minify(strings: TemplateStringsArray, ...values: unknown[]) {
  const code = String.raw(strings, ...values);
  return transformSync(code, options).code;
}

const WRAPPER = "async function __sst_minify__(){";

/**
 * Like `minify`, for statements that run inside a function, such as the code
 * spliced into a CloudFront Function's `handler`: they may `return` and `await`
 * at their top level, which a module may not. They are minified inside a
 * function that is then unwrapped.
 *
 * The template is read the way an ordinary template literal is, so code that
 * used to go through `interpolate` means the same after: `\\.` is `\.`, where
 * `String.raw` would keep both backslashes and change a regex.
 */
export function minifyBody(
  strings: TemplateStringsArray,
  ...values: unknown[]
) {
  const code = strings.reduce(
    (out, string, i) =>
      out + string + (i < values.length ? String(values[i]) : ""),
    "",
  );
  const minified = transformSync(`${WRAPPER}${code}}`, options).code.trim();
  if (!minified.startsWith(WRAPPER) || !minified.endsWith("}")) {
    throw new Error(`minifyBody: unexpected esbuild output:\n${minified}`);
  }
  return minified.slice(WRAPPER.length, -1);
}
