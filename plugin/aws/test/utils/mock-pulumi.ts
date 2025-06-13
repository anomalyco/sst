/**
 * Enhanced Pulumi mocking utilities for AWS plugin testing
 * Provides comprehensive mocking for Pulumi resources and outputs
 */

export interface MockOutput<T> {
  apply<U>(fn: (value: T) => U): MockOutput<U>;
  value: T;
}

export interface MockResource {
  urn: MockOutput<string>;
  id: MockOutput<string>;
  [key: string]: any;
}

/**
 * Creates a mock Pulumi Output that simulates the behavior of real Pulumi outputs
 */
export function mockOutput<T>(value: T): MockOutput<T> {
  return {
    apply: <U>(fn: (val: T) => U) => {
      const result = fn(value);
      return mockOutput(result);
    },
    value: value
  };
}

/**
 * Creates a mock Pulumi resource with standard properties
 */
export function mockResource(type: string, name: string, props: Record<string, any> = {}): MockResource {
  return {
    urn: mockOutput(`urn:pulumi:test::test::${type}::${name}`),
    id: mockOutput(`${name}-id`),
    ...Object.fromEntries(
      Object.entries(props).map(([key, value]) => [key, mockOutput(value)])
    )
  };
}

/**
 * Mock Pulumi.all function for combining multiple outputs
 */
export function mockAll<T extends readonly unknown[]>(values: T): MockOutput<T> {
  return mockOutput(values);
}

/**
 * Mock Pulumi interpolate function for string templating
 */
export function mockInterpolate(template: TemplateStringsArray, ...values: any[]): MockOutput<string> {
  let result = template[0];
  for (let i = 0; i < values.length; i++) {
    const value = values[i];
    const actualValue = value && typeof value === 'object' && 'value' in value ? value.value : value;
    result += actualValue + template[i + 1];
  }
  return mockOutput(result);
}

/**
 * Sets up global Pulumi mocks for testing
 */
export function setupPulumiMocks() {
  // Mock global Pulumi functions
  global.pulumi = {
    output: mockOutput,
    all: mockAll,
    interpolate: mockInterpolate,
    Config: class MockConfig {
      constructor(private name: string) {}
      get(key: string): string | undefined {
        return `mock-${this.name}-${key}`;
      }
      require(key: string): string {
        return `mock-${this.name}-${key}`;
      }
    }
  };
}

/**
 * Cleans up Pulumi mocks after testing
 */
export function cleanupPulumiMocks() {
  delete (global as any).pulumi;
}