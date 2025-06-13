/**
 * SST plugin mocking utilities for testing component behavior
 * Provides mocks for SST-specific functionality and component creation
 */

import { mockOutput, type MockOutput } from './mock-pulumi';

export interface MockSST {
  name: string;
  stage: string;
  version: Record<string, string>;
  app: {
    name: string;
    stage: string;
  };
}

export interface MockComponentArgs {
  [key: string]: any;
}

export interface MockComponent {
  name: string;
  type: string;
  args: MockComponentArgs;
  registerVersion: (options: {
    new: string;
    old?: string;
    message: string;
    forceUpgrade?: boolean;
  }) => void;
}

/**
 * Creates a mock SST environment for testing
 */
export function createMockSST(overrides: Partial<MockSST> = {}): MockSST {
  return {
    name: 'test-app',
    stage: 'test',
    version: {},
    app: {
      name: 'test-app',
      stage: 'test'
    },
    ...overrides
  };
}

/**
 * Sets up global SST mocks for testing
 */
export function setupSSTMocks(sst: MockSST = createMockSST()) {
  // Mock global $app
  global.$app = sst.app;
  
  // Mock global sst object
  global.sst = sst;
  
  // Mock environment variables
  process.env.SST_APP = sst.app.name;
  process.env.SST_STAGE = sst.app.stage;
}

/**
 * Creates a mock component with version registration capability
 */
export function createMockComponent(
  name: string,
  type: string,
  args: MockComponentArgs = {}
): MockComponent {
  const versionHistory: Array<{
    new: string;
    old?: string;
    message: string;
    forceUpgrade?: boolean;
  }> = [];

  return {
    name,
    type,
    args,
    registerVersion: (options) => {
      versionHistory.push(options);
      
      // Simulate version registration behavior
      if (options.old && options.new !== options.old && !options.forceUpgrade) {
        console.warn(`Migration required for ${name}: ${options.message}`);
      }
    }
  };
}

/**
 * Mock component factory for creating test components
 */
export class MockComponentFactory {
  private components: Map<string, MockComponent> = new Map();

  create<T extends MockComponent>(
    name: string,
    type: string,
    args: MockComponentArgs = {}
  ): T {
    const component = createMockComponent(name, type, args);
    this.components.set(name, component);
    return component as T;
  }

  get(name: string): MockComponent | undefined {
    return this.components.get(name);
  }

  getAll(): MockComponent[] {
    return Array.from(this.components.values());
  }

  clear(): void {
    this.components.clear();
  }
}

/**
 * Mock AWS component base class for testing
 */
export class MockAWSComponent {
  public name: string;
  public type: string;
  public args: MockComponentArgs;
  private versionHistory: Array<{
    new: string;
    old?: string;
    message: string;
    forceUpgrade?: boolean;
  }> = [];

  constructor(name: string, type: string, args: MockComponentArgs = {}) {
    this.name = name;
    this.type = type;
    this.args = args;
  }

  registerVersion(options: {
    new: string;
    old?: string;
    message: string;
    forceUpgrade?: boolean;
  }): void {
    this.versionHistory.push(options);
    
    // Simulate version registration behavior
    if (options.old && options.new !== options.old && !options.forceUpgrade) {
      throw new Error(`Migration required for ${this.name}: ${options.message}`);
    }
  }

  getVersionHistory() {
    return [...this.versionHistory];
  }

  // Mock physical name generation
  generatePhysicalName(suffix: string = ''): MockOutput<string> {
    const app = global.$app?.name || 'test-app';
    const stage = global.$app?.stage || 'test';
    const normalizedName = this.name.toLowerCase().replace(/[^a-z0-9]/g, '-');
    const hash = 'abcd1234'; // Mock hash
    const physicalName = `${app}-${stage}-${normalizedName}${suffix ? '-' + suffix : ''}-${hash}`;
    return mockOutput(physicalName);
  }
}

/**
 * Cleans up SST mocks after testing
 */
export function cleanupSSTMocks() {
  delete (global as any).$app;
  delete (global as any).sst;
  delete process.env.SST_APP;
  delete process.env.SST_STAGE;
}