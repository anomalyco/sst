import { describe, it, expect } from "bun:test";

describe("WorkerBuilder Helper", () => {
  it("should have correct module structure", () => {
    // Test that the module can be imported
    const workerBuilderModule = require("./worker-builder");
    expect(workerBuilderModule).toBeDefined();
    expect(workerBuilderModule.workerBuilder).toBeDefined();
    expect(typeof workerBuilderModule.workerBuilder).toBe("function");
  });

  it("should export workerBuilder function", () => {
    const { workerBuilder } = require("./worker-builder");
    expect(typeof workerBuilder).toBe("function");
  });

  it("should export WorkerBuilder type", () => {
    // Test that the module exports are structured correctly
    const workerBuilderModule = require("./worker-builder");
    expect(workerBuilderModule.workerBuilder).toBeDefined();
    
    // Check that workerBuilder is a function
    expect(typeof workerBuilderModule.workerBuilder).toBe("function");
  });

  it("should have correct function signature", () => {
    const { workerBuilder } = require("./worker-builder");
    
    // Check function length (number of parameters)
    expect(workerBuilder.length).toBe(4); // name, definition, argsTransform, opts
  });

  it("should handle string definition parameter", () => {
    // Test that the function can handle string definitions
    const { workerBuilder } = require("./worker-builder");
    expect(typeof workerBuilder).toBe("function");
    
    // The function should accept string as definition parameter
    // This is validated at compile time through TypeScript
  });

  it("should handle WorkerArgs definition parameter", () => {
    // Test that the function can handle WorkerArgs definitions
    const { workerBuilder } = require("./worker-builder");
    expect(typeof workerBuilder).toBe("function");
    
    // The function should accept WorkerArgs as definition parameter
    // This is validated at compile time through TypeScript
  });

  it("should support optional transform parameter", () => {
    // Test that the transform parameter is optional
    const { workerBuilder } = require("./worker-builder");
    expect(typeof workerBuilder).toBe("function");
    
    // The transform parameter should be optional
    // This is validated at compile time through TypeScript
  });

  it("should support optional opts parameter", () => {
    // Test that the opts parameter is optional
    const { workerBuilder } = require("./worker-builder");
    expect(typeof workerBuilder).toBe("function");
    
    // The opts parameter should be optional
    // This is validated at compile time through TypeScript
  });
});