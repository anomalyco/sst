import { describe, it, expect, beforeEach } from "vitest";
import { execFileSync } from "child_process";
import fs from "fs";
import path from "path";
import os from "os";

const ADD_SCRIPT = path.resolve(__dirname, "../../src/ast/add.mjs");
const PKG = "@pulumiverse/grafana";
const VERSION = "0.0.1";

function run(config: string) {
  const tmp = path.join(os.tmpdir(), `sst-add-test-${Date.now()}.ts`);
  fs.writeFileSync(tmp, config);
  execFileSync("node", [ADD_SCRIPT, tmp, PKG, VERSION]);
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
    expect(result).toContain(`"${PKG}": "${VERSION}"`);
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
    expect(result).toContain(`"${PKG}": "${VERSION}"`);
  });

  it("arrow function with concise body", () => {
    const result = run(`export default $config({
  app: (input) => ({
    name: "my-app",
    providers: {},
  }),
});`);
    expect(result).toContain(`"${PKG}": "${VERSION}"`);
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
    expect(result).toContain(`"${PKG}": "${VERSION}"`);
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
    expect(result).toContain(`"${PKG}": "${VERSION}"`);
  });

  it("skips if provider already exists", () => {
    const config = `export default $config({
  app(input) {
    return {
      name: "my-app",
      providers: { "${PKG}": "0.0.0" },
    };
  },
});`;
    const result = run(config);
    expect(result).toContain(`"${PKG}": "0.0.0"`);
    expect(result).not.toContain(VERSION);
  });
});
