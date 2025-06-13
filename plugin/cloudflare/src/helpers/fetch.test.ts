import { describe, it, expect } from "bun:test";
import { cfFetch, FetchError, FetchResult } from "./fetch";

describe("cfFetch", () => {
  it("should be a function", () => {
    expect(typeof cfFetch).toBe("function");
  });

  it("should have correct FetchError interface", () => {
    const error: FetchError = {
      code: 1000,
      message: "Test error"
    };
    expect(error.code).toBe(1000);
    expect(error.message).toBe("Test error");
  });

  it("should have correct FetchResult interface", () => {
    const result: FetchResult<{ test: string }> = {
      success: true,
      result: { test: "value" },
      errors: []
    };
    expect(result.success).toBe(true);
    expect(result.result.test).toBe("value");
    expect(result.errors).toEqual([]);
  });

  it("should handle FetchResult with result_info", () => {
    const result: FetchResult<any> = {
      success: true,
      result: {},
      errors: [],
      result_info: {
        page: 1,
        per_page: 20,
        count: 5,
        total_count: 5
      }
    };
    expect(result.result_info?.page).toBe(1);
    expect(result.result_info?.per_page).toBe(20);
  });

  it("should handle FetchError with error_chain", () => {
    const chainedError: FetchError = {
      code: 1001,
      message: "Chained error"
    };
    const error: FetchError = {
      code: 1000,
      message: "Main error",
      error_chain: [chainedError]
    };
    expect(error.error_chain).toHaveLength(1);
    expect(error.error_chain?.[0].code).toBe(1001);
  });

  it("should handle messages in FetchResult", () => {
    const result: FetchResult<any> = {
      success: true,
      result: {},
      errors: [],
      messages: ["Info message", "Warning message"]
    };
    expect(result.messages).toHaveLength(2);
    expect(result.messages?.[0]).toBe("Info message");
  });
});