import { describe, it, expect } from "vitest";
import { physicalName } from "../../src/components/naming.js";

// @ts-ignore
global.$app = {
  name: "my-app",
  stage: "test",
};
describe("generateName with dashed app name", function () {
  it(() => expect(physicalName(10, "myapp")).toMatch(/^m-[a-z]{8}$/));
  it(() => expect(physicalName(11, "myapp")).toMatch(/^my-[a-z]{8}$/));
  it(() => expect(physicalName(12, "myapp")).toMatch(/^mya-[a-z]{8}$/));
  it(() => expect(physicalName(13, "myapp")).toMatch(/^myap-[a-z]{8}$/));
  it(() => expect(physicalName(14, "myapp")).toMatch(/^myapp-[a-z]{8}$/));
  it(() => expect(physicalName(15, "myapp")).toMatch(/^myapp-[a-z]{8}$/));
  it(() => expect(physicalName(16, "myapp")).toMatch(/^t-myapp-[a-z]{8}$/));
  it(() => expect(physicalName(17, "myapp")).toMatch(/^te-myapp-[a-z]{8}$/));
  it(() => expect(physicalName(18, "myapp")).toMatch(/^tes-myapp-[a-z]{8}$/));
  it(() => expect(physicalName(19, "myapp")).toMatch(/^test-myapp-[a-z]{8}$/));
  it(() => expect(physicalName(20, "myapp")).toMatch(/^test-myapp-[a-z]{8}$/));
  it(() =>
    expect(physicalName(21, "myapp")).toMatch(/^m-test-myapp-[a-z]{8}$/),
  );
  it(() =>
    expect(physicalName(22, "myapp")).toMatch(/^my-test-myapp-[a-z]{8}$/),
  );
  it(() =>
    expect(physicalName(23, "myapp")).toMatch(/^my-test-myapp-[a-z]{8}$/),
  );
});
