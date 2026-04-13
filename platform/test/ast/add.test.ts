import { describe, it, expect } from "vitest";
import { execFileSync } from "child_process";
import fs from "fs";
import path from "path";
import os from "os";

const ADD_SCRIPT = path.resolve(__dirname, "../../src/ast/add.mjs");
const PROVIDER = "grafana";
const PKG = "@pulumiverse/grafana";
const VERSION = "0.0.1";

function run(config: string) {
  const tmp = path.join(os.tmpdir(), `sst-add-test-${Date.now()}.ts`);
  fs.writeFileSync(tmp, config);
  execFileSync("node", [ADD_SCRIPT, tmp, PROVIDER, VERSION, PKG]);
  const result = fs.readFileSync(tmp, "utf-8");
  fs.unlinkSync(tmp);
  return result;
}

describe("add provider", () => {
  it("method declaration", () => {
    const result = run(`export default $config({
  app(input) {
    return {
      name: "my-app",
      providers: {},
    };
  },
});`);
    expect(result).toContain(`${PROVIDER}: {`);
    expect(result).toContain(`package: "${PKG}"`);
    expect(result).toContain(`version: "${VERSION}"`);
  });

  it("arrow function with block body", () => {
    const result = run(`export default $config({
  app: (input) => {
    return {
      name: "my-app",
      providers: {},
    };
  },
});`);
    expect(result).toContain(`${PROVIDER}: {`);
    expect(result).toContain(`package: "${PKG}"`);
    expect(result).toContain(`version: "${VERSION}"`);
  });

  it("arrow function with concise body", () => {
    const result = run(`export default $config({
  app: (input) => ({
    name: "my-app",
    providers: {},
  }),
});`);
    expect(result).toContain(`${PROVIDER}: {`);
    expect(result).toContain(`package: "${PKG}"`);
    expect(result).toContain(`version: "${VERSION}"`);
  });

  it("function expression", () => {
    const result = run(`export default $config({
  app: function(input) {
    return {
      name: "my-app",
      providers: {},
    };
  },
});`);
    expect(result).toContain(`${PROVIDER}: {`);
    expect(result).toContain(`package: "${PKG}"`);
    expect(result).toContain(`version: "${VERSION}"`);
  });

  it("adds providers key when missing", () => {
    const result = run(`export default $config({
  app(input) {
    return {
      name: "my-app",
    };
  },
});`);
    expect(result).toContain("providers");
    expect(result).toContain(`${PROVIDER}: {`);
    expect(result).toContain(`package: "${PKG}"`);
    expect(result).toContain(`version: "${VERSION}"`);
  });

  it("adds package to existing string provider", () => {
    const config = `export default $config({
  app(input) {
    return {
      name: "my-app",
      providers: { "${PROVIDER}": "0.0.0" },
    };
  },
});`;
    const result = run(config);
    expect(result).toContain(`${PROVIDER}: {`);
    expect(result).toContain(`package: "${PKG}"`);
    expect(result).toContain(`version: "0.0.0"`);
  });

  it("adds package to existing object provider", () => {
    const config = `export default $config({
  app(input) {
    return {
      name: "my-app",
      providers: {
        "${PROVIDER}": {
          version: "0.0.0",
          region: "us-east-1",
        },
      },
    };
  },
});`;
    const result = run(config);
    expect(result).toContain(`${PROVIDER}: {`);
    expect(result).toContain(`package: "${PKG}"`);
    expect(result).toContain(`version: "0.0.0"`);
    expect(result).toContain(`region: "us-east-1"`);
  });
});
