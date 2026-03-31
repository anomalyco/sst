import { expect, test } from "vitest";
import { Example } from "../";

test("Hello test", () => {
  const expected = "Hello, world!";

  expect(Example.hello()).toEqual(expected);
});
