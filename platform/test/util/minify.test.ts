import { describe, expect, it } from "vitest";
import { minifyBody } from "../../src/util/minify";

describe("minifyBody", () => {
  it("minifies statements that return and await at their top level", () => {
    const code = minifyBody`
      // a comment
      const value = await read();
      if (!value) return null;
      return value;
    `;
    expect(code).toBe("const value=await read();return value||null");
  });

  it("splices values in", () => {
    const namespace = "ab12";
    expect(minifyBody`return "${namespace}:routes";`).toBe(
      'return"ab12:routes"',
    );
  });

  // Reads like the `interpolate` template the code used to live in: one
  // backslash in the output for two in the source. String.raw would keep two,
  // and /\\./ would stop matching a dot.
  it("reads escapes the way an ordinary template literal does", () => {
    const code = minifyBody`return "a.b".replace(/\\./g, "\\\\.");`;
    expect(new Function(code)()).toBe("a\\.b");
  });
});
