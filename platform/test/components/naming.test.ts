import { describe, it, expect } from "vitest";
import { physicalName } from "../../src/components/naming.js";

// @ts-ignore
global.$app = {
  name: "app",
  stage: "test",
};

describe("generateName", function () {
  it(() => expect(physicalName(10, "foo")).toMatch(/^f-[a-z]{8}$/));
  it(() => expect(physicalName(11, "foo")).toMatch(/^fo-[a-z]{8}$/));
  it(() => expect(physicalName(12, "foo")).toMatch(/^foo-[a-z]{8}$/));
  it(() => expect(physicalName(13, "foo")).toMatch(/^foo-[a-z]{8}$/));
  it(() => expect(physicalName(14, "foo")).toMatch(/^t-foo-[a-z]{8}$/));
  it(() => expect(physicalName(15, "foo")).toMatch(/^te-foo-[a-z]{8}$/));
  it(() => expect(physicalName(16, "foo")).toMatch(/^tes-foo-[a-z]{8}$/));
  it(() => expect(physicalName(17, "foo")).toMatch(/^test-foo-[a-z]{8}$/));
  it(() => expect(physicalName(18, "foo")).toMatch(/^test-foo-[a-z]{8}$/));
  it(() => expect(physicalName(19, "foo")).toMatch(/^a-test-foo-[a-z]{8}$/));
  it(() => expect(physicalName(20, "foo")).toMatch(/^ap-test-foo-[a-z]{8}$/));
  it(() => expect(physicalName(21, "foo")).toMatch(/^app-test-foo-[a-z]{8}$/));
  it(() => expect(physicalName(22, "foo")).toMatch(/^app-test-foo-[a-z]{8}$/));
  it(() => expect(physicalName(23, "foo")).toMatch(/^app-test-foo-[a-z]{8}$/));
  it(() =>
    expect(physicalName(23, "foo", ".fifo")).toMatch(
      /^test-foo-[a-z]{8}\.fifo$/,
    ),
  );
});
