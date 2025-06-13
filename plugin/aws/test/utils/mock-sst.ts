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
  public originalName: string;
  public type: string;
  public args: MockComponentArgs;
  public urn: MockOutput<string>;
  private versionHistory: Array<{
    new: string;
    old?: string;
    message: string;
    forceUpgrade?: boolean;
  }> = [];

  constructor(name: string, type: string, args: MockComponentArgs = {}) {
    this.originalName = name;
    this.type = type;
    this.args = args;
    
    // Apply environment context to name
    const app = global.$app?.name || 'test-app';
    const stage = global.$app?.stage || 'test';
    
    this.name = this.generatePhysicalNameInternal(name, app, stage);
    this.urn = mockOutput(`urn:pulumi:${stage}::${app}::${type}::${this.name}`);
  }

  registerVersion(options: {
    new: string;
    old?: string;
    message: string;
    forceUpgrade?: boolean;
  }): void {
    this.versionHistory.push(options);
    
    // Simulate version registration behavior
    // For incremental migration tests, we want to throw errors to simulate migration requirements
    // For force upgrade tests, we want to bypass the error when forceUpgrade is true
    if (options.old !== undefined && options.new !== options.old && !options.forceUpgrade) {
      // Check if this is a downgrade (should always throw)
      const oldParts = options.old.split('.').map(Number);
      const newParts = options.new.split('.').map(Number);
      
      for (let i = 0; i < Math.max(oldParts.length, newParts.length); i++) {
        const oldPart = oldParts[i] || 0;
        const newPart = newParts[i] || 0;
        
        if (newPart < oldPart) {
          throw new Error(`Cannot downgrade from ${options.old} to ${options.new}`);
        } else if (newPart > oldPart) {
          break; // This is an upgrade
        }
      }
      
      // For any version changes, throw migration error (unless it's the same version)
      if (options.old !== options.new) {
        throw new Error(`Migration required for ${this.name}: ${options.message}`);
      }
    }
    
    // Handle rollback prevention even with force upgrade
    if (options.forceUpgrade && options.old !== undefined && options.new !== options.old) {
      const oldParts = options.old.split('.').map(Number);
      const newParts = options.new.split('.').map(Number);
      
      for (let i = 0; i < Math.max(oldParts.length, newParts.length); i++) {
        const oldPart = oldParts[i] || 0;
        const newPart = newParts[i] || 0;
        
        if (newPart < oldPart) {
          throw new Error(`Cannot downgrade from ${options.old} to ${options.new}, even with force upgrade`);
        } else if (newPart > oldPart) {
          break; // This is an upgrade
        }
      }
    }
  }

  getVersionHistory() {
    return [...this.versionHistory];
  }

  // Public method to generate physical names for testing
  public generatePhysicalName(name: string): MockOutput<string> {
    const app = global.$app?.name || 'test-app';
    const stage = global.$app?.stage || 'test';
    const normalizedComponentName = this.originalName.toLowerCase().replace(/[^a-z0-9]/g, '');
    const normalizedResourceName = name.toLowerCase().replace(/[^a-z0-9]/g, '');
    const hash = 'abcd1234'; // Mock hash
    
    // Apply length limits based on AWS service requirements
    const maxLength = this.getMaxNameLength();
    let physicalName = `${app}-${stage}-${normalizedComponentName}-${normalizedResourceName}-${hash}`;
    
    if (physicalName.length > maxLength) {
      const availableLength = maxLength - app.length - stage.length - hash.length - 4; // 4 hyphens
      const totalNameLength = normalizedComponentName.length + normalizedResourceName.length + 1; // +1 for hyphen
      const componentLength = Math.floor((availableLength * normalizedComponentName.length) / totalNameLength);
      const resourceLength = availableLength - componentLength - 1;
      
      const truncatedComponent = normalizedComponentName.substring(0, Math.max(1, componentLength));
      const truncatedResource = normalizedResourceName.substring(0, Math.max(1, resourceLength));
      physicalName = `${app}-${stage}-${truncatedComponent}-${truncatedResource}-${hash}`;
    }
    
    return mockOutput(physicalName);
  }

  // Mock physical name generation with proper AWS naming conventions
  private generatePhysicalNameInternal(name: string, app: string, stage: string): string {
    const normalizedName = name.toLowerCase().replace(/[^a-z0-9]/g, '');
    const hash = 'abcd1234'; // Mock hash
    
    // Apply length limits based on AWS service requirements
    const maxLength = this.getMaxNameLength();
    let physicalName = `${app}-${stage}-${normalizedName}-${hash}`;
    
    if (physicalName.length > maxLength) {
      const availableLength = maxLength - app.length - stage.length - hash.length - 3; // 3 hyphens
      const truncatedName = normalizedName.substring(0, Math.max(1, availableLength));
      physicalName = `${app}-${stage}-${truncatedName}-${hash}`;
    }
    
    return physicalName;
  }

  private getMaxNameLength(): number {
    // Return appropriate length limits based on component type
    if (this.type.includes('bucket')) return 63;
    if (this.type.includes('function')) return 64;
    return 255; // Default for most AWS resources
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