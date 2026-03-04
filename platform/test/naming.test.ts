import { describe, it, expect } from "bun:test";
import { physicalName } from "../src/components/naming.js";

// @ts-ignore
global.$app = {
  name: "app",
  stage: "test",
};

describe("generateName", function () {
  it("length 10", () => expect(physicalName(10, "foo")).toMatch(/^f-[a-z]{8}$/));
  it("length 11", () => expect(physicalName(11, "foo")).toMatch(/^fo-[a-z]{8}$/));
  it("length 12", () => expect(physicalName(12, "foo")).toMatch(/^foo-[a-z]{8}$/));
  it("length 13", () => expect(physicalName(13, "foo")).toMatch(/^foo-[a-z]{8}$/));
  it("length 14", () => expect(physicalName(14, "foo")).toMatch(/^t-foo-[a-z]{8}$/));
  it("length 15", () => expect(physicalName(15, "foo")).toMatch(/^te-foo-[a-z]{8}$/));
  it("length 16", () => expect(physicalName(16, "foo")).toMatch(/^tes-foo-[a-z]{8}$/));
  it("length 17", () => expect(physicalName(17, "foo")).toMatch(/^test-foo-[a-z]{8}$/));
  it("length 18", () => expect(physicalName(18, "foo")).toMatch(/^test-foo-[a-z]{8}$/));
  it("length 19", () => expect(physicalName(19, "foo")).toMatch(/^a-test-foo-[a-z]{8}$/));
  it("length 20", () => expect(physicalName(20, "foo")).toMatch(/^ap-test-foo-[a-z]{8}$/));
  it("length 21", () => expect(physicalName(21, "foo")).toMatch(/^app-test-foo-[a-z]{8}$/));
  it("length 22", () => expect(physicalName(22, "foo")).toMatch(/^app-test-foo-[a-z]{8}$/));
  it("length 23", () => expect(physicalName(23, "foo")).toMatch(/^app-test-foo-[a-z]{8}$/));
  it("with suffix", () =>
    expect(physicalName(23, "foo", ".fifo")).toMatch(
      /^test-foo-[a-z]{8}\.fifo$/,
    ),
  );
});
