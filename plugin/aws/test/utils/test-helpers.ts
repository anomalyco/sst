/**
 * Common test utilities and helpers for AWS plugin testing
 * Provides assertion helpers, test data generators, and common test patterns
 */

import { expect } from 'bun:test';
import { setupPulumiMocks, cleanupPulumiMocks, type MockOutput } from './mock-pulumi';
import { setupSSTMocks, cleanupSSTMocks, createMockSST, type MockSST } from './mock-sst';

export interface TestEnvironment {
  sst: MockSST;
  cleanup: () => void;
}

/**
 * Sets up a complete test environment with all necessary mocks
 */
export function setupTestEnvironment(overrides: Partial<MockSST> = {}): TestEnvironment {
  const sst = createMockSST(overrides);
  
  setupPulumiMocks();
  setupSSTMocks(sst);
  
  return {
    sst,
    cleanup: () => {
      cleanupPulumiMocks();
      cleanupSSTMocks();
    }
  };
}

/**
 * Assertion helpers for testing AWS components
 */
export const assertions = {
  /**
   * Asserts that a physical name follows AWS naming conventions
   */
  validAWSName: (name: string | MockOutput<string>, maxLength?: number) => {
    const actualName = typeof name === 'string' ? name : name.value;
    
    // Should contain app-stage prefix
    expect(actualName).toMatch(/^[a-z0-9-]+-[a-z0-9-]+-/);
    
    // Should not contain uppercase letters
    expect(actualName).not.toMatch(/[A-Z]/);
    
    // Should not start or end with hyphen
    expect(actualName).not.toMatch(/^-|-$/);
    
    // Should not contain consecutive hyphens
    expect(actualName).not.toMatch(/--/);
    
    // Check length if specified
    if (maxLength) {
      expect(actualName.length).toBeLessThanOrEqual(maxLength);
    }
  },

  /**
   * Asserts that a component has registered a version correctly
   */
  versionRegistered: (component: any, expectedVersion: string) => {
    expect(component.registerVersion).toBeDefined();
    // Additional version-specific assertions can be added here
  },

  /**
   * Asserts that a migration warning was triggered
   */
  migrationWarning: (consoleSpy: any, componentName: string) => {
    expect(consoleSpy).toHaveBeenCalledWith(
      expect.stringContaining(`Migration required for ${componentName}`)
    );
  },

  /**
   * Asserts that a component output is valid
   */
  validOutput: <T>(output: MockOutput<T>, expectedValue?: T) => {
    expect(output).toBeDefined();
    expect(output.apply).toBeDefined();
    expect(output.value).toBeDefined();
    
    if (expectedValue !== undefined) {
      expect(output.value).toBe(expectedValue);
    }
  }
};

/**
 * Test data generators for creating consistent test data
 */
export const generators = {
  /**
   * Generates a random component name
   */
  componentName: (prefix: string = 'Test'): string => {
    const suffix = Math.random().toString(36).substring(2, 8);
    return `${prefix}${suffix}`;
  },

  /**
   * Generates AWS resource ARN
   */
  arn: (service: string, resource: string, region: string = 'us-east-1'): string => {
    return `arn:aws:${service}:${region}:123456789012:${resource}`;
  },

  /**
   * Generates mock AWS tags
   */
  tags: (additional: Record<string, string> = {}): Record<string, string> => {
    return {
      'sst:app': 'test-app',
      'sst:stage': 'test',
      ...additional
    };
  },

  /**
   * Generates mock environment variables
   */
  envVars: (additional: Record<string, string> = {}): Record<string, string> => {
    return {
      NODE_ENV: 'test',
      SST_APP: 'test-app',
      SST_STAGE: 'test',
      ...additional
    };
  }
};

/**
 * Common test patterns and utilities
 */
export const patterns = {
  /**
   * Tests component creation with various argument combinations
   */
  testComponentCreation: async <T>(
    ComponentClass: new (name: string, args?: any) => T,
    testCases: Array<{
      name: string;
      args?: any;
      shouldThrow?: boolean;
      expectedError?: string;
    }>
  ) => {
    for (const testCase of testCases) {
      if (testCase.shouldThrow) {
        expect(() => new ComponentClass(testCase.name, testCase.args)).toThrow(
          testCase.expectedError
        );
      } else {
        const component = new ComponentClass(testCase.name, testCase.args);
        expect(component).toBeDefined();
      }
    }
  },

  /**
   * Tests version migration scenarios
   */
  testVersionMigration: (
    component: any,
    scenarios: Array<{
      oldVersion?: string;
      newVersion: string;
      forceUpgrade?: boolean;
      shouldWarn?: boolean;
      shouldThrow?: boolean;
    }>
  ) => {
    scenarios.forEach((scenario, index) => {
      const testName = `scenario-${index}`;
      
      if (scenario.shouldThrow) {
        expect(() => {
          component.registerVersion({
            old: scenario.oldVersion,
            new: scenario.newVersion,
            message: `Test migration for ${testName}`,
            forceUpgrade: scenario.forceUpgrade
          });
        }).toThrow();
      } else {
        expect(() => {
          component.registerVersion({
            old: scenario.oldVersion,
            new: scenario.newVersion,
            message: `Test migration for ${testName}`,
            forceUpgrade: scenario.forceUpgrade
          });
        }).not.toThrow();
      }
    });
  }
};

/**
 * Utility for testing async operations with proper cleanup
 */
export async function withTestEnvironment<T>(
  testFn: (env: TestEnvironment) => Promise<T> | T,
  overrides: Partial<MockSST> = {}
): Promise<T> {
  const env = setupTestEnvironment(overrides);
  
  try {
    return await testFn(env);
  } finally {
    env.cleanup();
  }
}

/**
 * Creates a spy for console methods to test warning/error output
 */
export function createConsoleSpy(method: 'log' | 'warn' | 'error' = 'warn') {
  const originalMethod = console[method];
  const calls: any[][] = [];
  
  console[method] = (...args: any[]) => {
    calls.push(args);
  };
  
  return {
    calls,
    restore: () => {
      console[method] = originalMethod;
    },
    toHaveBeenCalledWith: (expected: any) => {
      const found = calls.some(call => 
        call.some(arg => 
          typeof expected === 'string' 
            ? arg.includes(expected)
            : arg === expected
        )
      );
      expect(found).toBe(true);
    }
  };
}