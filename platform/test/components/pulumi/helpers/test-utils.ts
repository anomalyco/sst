import { describe, beforeAll, beforeEach, afterEach, it, expect } from "vitest";
import * as pulumi from "@pulumi/pulumi";
import { setupSSTTestEnvironment, ResourceValidator } from "./pulumi-mocks";

/**
 * Test utilities for SST component testing
 */

export interface TestComponentOptions {
  appName?: string;
  stage?: string;
  setupMocks?: boolean;
}

/**
 * Sets up a test environment for SST components
 */
export function setupComponentTest(options: TestComponentOptions = {}) {
  const { appName = "test-app", stage = "test", setupMocks = true } = options;

  beforeEach(() => {
    if (setupMocks) {
      setupSSTTestEnvironment(appName, stage);
    }
  });

  afterEach(() => {
    // Clean up any test state if needed
  });
}

/**
 * Helper to test component creation and basic properties
 */
export async function testComponentCreation<T>(
  componentFactory: () => T,
  expectedProperties: Record<string, any> = {}
): Promise<T> {
  const component = componentFactory();
  
  // Validate basic properties
  expect(component).toBeDefined();
  
  // Validate expected properties if provided
  if (Object.keys(expectedProperties).length > 0) {
    ResourceValidator.validateResourceProperties(component, expectedProperties);
  }
  
  return component;
}

/**
 * Helper to test component outputs
 */
export async function testComponentOutputs<T>(
  component: any,
  outputTests: Record<string, (value: T) => void>
): Promise<void> {
  for (const [outputName, testFn] of Object.entries(outputTests)) {
    if (component[outputName]) {
      await pulumi.all([component[outputName]]).apply(([value]) => {
        testFn(value);
      });
    }
  }
}

/**
 * Helper to test component naming conventions
 */
export function testSSTNaming(
  resourceName: string,
  expectedBaseName: string,
  appName: string = "test-app",
  stage: string = "test"
): void {
  const expectedPrefix = `${appName}-${stage}-${expectedBaseName.toLowerCase()}`;
  expect(resourceName.toLowerCase()).toMatch(new RegExp(`^${expectedPrefix}-[a-z0-9]{8}$`));
}

/**
 * Helper to test component linking functionality
 */
export function testComponentLinking(
  component: any,
  expectedLinkableProperties: string[] = []
): void {
  // Test that component has linkable properties
  for (const prop of expectedLinkableProperties) {
    expect(component).toHaveProperty(prop);
  }
  
  // Test that component can be used in links (if it's linkable)
  if (typeof component.getSSTLink === "function") {
    const link = component.getSSTLink();
    expect(link).toBeDefined();
    expect(link).toHaveProperty("properties");
  }
}

/**
 * Helper to test component environment variables
 */
export async function testComponentEnvironmentVariables(
  component: any,
  expectedEnvVars: Record<string, string | RegExp>
): Promise<void> {
  if (component.environment) {
    await pulumi.all([component.environment]).apply(([env]) => {
      for (const [key, expectedValue] of Object.entries(expectedEnvVars)) {
        expect(env).toHaveProperty(key);
        if (typeof expectedValue === "string") {
          expect(env[key]).toBe(expectedValue);
        } else {
          expect(env[key]).toMatch(expectedValue);
        }
      }
    });
  }
}

/**
 * Helper to test component IAM permissions
 */
export async function testComponentIAMPermissions(
  component: any,
  expectedPermissions: string[]
): Promise<void> {
  if (component.role && component.role.inlinePolicies) {
    await pulumi.all([component.role.inlinePolicies]).apply(([policies]) => {
      const policyDocument = JSON.parse(policies[0].policy);
      const statements = policyDocument.Statement;
      
      for (const permission of expectedPermissions) {
        const hasPermission = statements.some((statement: any) =>
          statement.Action.includes(permission) || statement.Action.includes("*")
        );
        expect(hasPermission).toBe(true);
      }
    });
  }
}

/**
 * Helper to test component VPC configuration
 */
export async function testComponentVPCConfiguration(
  component: any,
  expectedVPCConfig: {
    hasVPC?: boolean;
    subnetCount?: number;
    securityGroupCount?: number;
  }
): Promise<void> {
  if (expectedVPCConfig.hasVPC && component.vpc) {
    expect(component.vpc).toBeDefined();
    
    if (expectedVPCConfig.subnetCount && component.vpc.subnets) {
      await pulumi.all([component.vpc.subnets]).apply(([subnets]) => {
        expect(subnets).toHaveLength(expectedVPCConfig.subnetCount!);
      });
    }
    
    if (expectedVPCConfig.securityGroupCount && component.securityGroups) {
      await pulumi.all([component.securityGroups]).apply(([securityGroups]) => {
        expect(securityGroups).toHaveLength(expectedVPCConfig.securityGroupCount!);
      });
    }
  }
}

/**
 * Helper to test component scaling configuration
 */
export async function testComponentScaling(
  component: any,
  expectedScaling: {
    minCapacity?: number;
    maxCapacity?: number;
    targetUtilization?: number;
  }
): Promise<void> {
  if (component.scaling) {
    await pulumi.all([component.scaling]).apply(([scaling]) => {
      if (expectedScaling.minCapacity !== undefined) {
        expect(scaling.minCapacity).toBe(expectedScaling.minCapacity);
      }
      if (expectedScaling.maxCapacity !== undefined) {
        expect(scaling.maxCapacity).toBe(expectedScaling.maxCapacity);
      }
      if (expectedScaling.targetUtilization !== undefined) {
        expect(scaling.targetUtilization).toBe(expectedScaling.targetUtilization);
      }
    });
  }
}

/**
 * Helper to test component monitoring configuration
 */
export async function testComponentMonitoring(
  component: any,
  expectedMonitoring: {
    hasCloudWatchLogs?: boolean;
    hasMetrics?: boolean;
    hasAlarms?: boolean;
    logRetentionDays?: number;
  }
): Promise<void> {
  if (expectedMonitoring.hasCloudWatchLogs && component.logGroup) {
    expect(component.logGroup).toBeDefined();
    
    if (expectedMonitoring.logRetentionDays && component.logGroup.retentionInDays) {
      await pulumi.all([component.logGroup.retentionInDays]).apply(([retention]) => {
        expect(retention).toBe(expectedMonitoring.logRetentionDays!);
      });
    }
  }
  
  if (expectedMonitoring.hasMetrics && component.metrics) {
    expect(component.metrics).toBeDefined();
  }
  
  if (expectedMonitoring.hasAlarms && component.alarms) {
    expect(component.alarms).toBeDefined();
  }
}

/**
 * Helper to test component tags
 */
export async function testComponentTags(
  component: any,
  expectedTags: Record<string, string>
): Promise<void> {
  if (component.tags) {
    await pulumi.all([component.tags]).apply(([tags]) => {
      for (const [key, expectedValue] of Object.entries(expectedTags)) {
        expect(tags).toHaveProperty(key);
        expect(tags[key]).toBe(expectedValue);
      }
    });
  }
}

/**
 * Helper to test component cost optimization
 */
export async function testComponentCostOptimization(
  component: any,
  expectedOptimizations: {
    hasReservedCapacity?: boolean;
    hasSpotInstances?: boolean;
    hasLifecyclePolicies?: boolean;
    hasCompressionEnabled?: boolean;
  }
): Promise<void> {
  if (expectedOptimizations.hasReservedCapacity && component.reservedCapacity) {
    expect(component.reservedCapacity).toBeDefined();
  }
  
  if (expectedOptimizations.hasSpotInstances && component.spotInstances) {
    expect(component.spotInstances).toBeDefined();
  }
  
  if (expectedOptimizations.hasLifecyclePolicies && component.lifecyclePolicies) {
    expect(component.lifecyclePolicies).toBeDefined();
  }
  
  if (expectedOptimizations.hasCompressionEnabled && component.compression) {
    await pulumi.all([component.compression]).apply(([compression]) => {
      expect(compression.enabled).toBe(true);
    });
  }
}

/**
 * Helper to test component security configuration
 */
export async function testComponentSecurity(
  component: any,
  expectedSecurity: {
    hasEncryption?: boolean;
    hasAccessLogging?: boolean;
    hasPublicAccess?: boolean;
    hasSSL?: boolean;
    encryptionType?: string;
  }
): Promise<void> {
  if (expectedSecurity.hasEncryption && component.encryption) {
    expect(component.encryption).toBeDefined();
    
    if (expectedSecurity.encryptionType && component.encryption.type) {
      await pulumi.all([component.encryption.type]).apply(([type]) => {
        expect(type).toBe(expectedSecurity.encryptionType!);
      });
    }
  }
  
  if (expectedSecurity.hasAccessLogging && component.accessLogging) {
    expect(component.accessLogging).toBeDefined();
  }
  
  if (expectedSecurity.hasPublicAccess !== undefined && component.publicAccess !== undefined) {
    await pulumi.all([component.publicAccess]).apply(([publicAccess]) => {
      expect(publicAccess).toBe(expectedSecurity.hasPublicAccess!);
    });
  }
  
  if (expectedSecurity.hasSSL && component.ssl) {
    expect(component.ssl).toBeDefined();
  }
}

/**
 * Helper to create a comprehensive test suite for a component
 */
export function createComponentTestSuite<T>(
  componentName: string,
  componentFactory: () => T,
  testConfig: {
    basicProperties?: Record<string, any>;
    outputs?: Record<string, (value: any) => void>;
    naming?: { baseName: string; appName?: string; stage?: string };
    linking?: { expectedProperties?: string[] };
    environment?: Record<string, string | RegExp>;
    iam?: string[];
    vpc?: { hasVPC?: boolean; subnetCount?: number; securityGroupCount?: number };
    scaling?: { minCapacity?: number; maxCapacity?: number; targetUtilization?: number };
    monitoring?: { hasCloudWatchLogs?: boolean; hasMetrics?: boolean; hasAlarms?: boolean; logRetentionDays?: number };
    tags?: Record<string, string>;
    costOptimization?: { hasReservedCapacity?: boolean; hasSpotInstances?: boolean; hasLifecyclePolicies?: boolean; hasCompressionEnabled?: boolean };
    security?: { hasEncryption?: boolean; hasAccessLogging?: boolean; hasPublicAccess?: boolean; hasSSL?: boolean; encryptionType?: string };
  }
) {
  return describe(componentName, () => {
    setupComponentTest();

    it("should create component successfully", async () => {
      await testComponentCreation(componentFactory, testConfig.basicProperties);
    });

    if (testConfig.outputs) {
      it("should have correct outputs", async () => {
        const component = componentFactory();
        await testComponentOutputs(component, testConfig.outputs);
      });
    }

    if (testConfig.naming) {
      it("should follow SST naming conventions", async () => {
        const component = componentFactory() as any;
        if (component.name) {
          await pulumi.all([component.name]).apply(([name]) => {
            testSSTNaming(
              name,
              testConfig.naming!.baseName,
              testConfig.naming!.appName,
              testConfig.naming!.stage
            );
          });
        }
      });
    }

    if (testConfig.linking) {
      it("should support component linking", () => {
        const component = componentFactory();
        testComponentLinking(component, testConfig.linking.expectedProperties);
      });
    }

    if (testConfig.environment) {
      it("should have correct environment variables", async () => {
        const component = componentFactory();
        await testComponentEnvironmentVariables(component, testConfig.environment);
      });
    }

    if (testConfig.iam) {
      it("should have correct IAM permissions", async () => {
        const component = componentFactory();
        await testComponentIAMPermissions(component, testConfig.iam);
      });
    }

    if (testConfig.vpc) {
      it("should have correct VPC configuration", async () => {
        const component = componentFactory();
        await testComponentVPCConfiguration(component, testConfig.vpc);
      });
    }

    if (testConfig.scaling) {
      it("should have correct scaling configuration", async () => {
        const component = componentFactory();
        await testComponentScaling(component, testConfig.scaling);
      });
    }

    if (testConfig.monitoring) {
      it("should have correct monitoring configuration", async () => {
        const component = componentFactory();
        await testComponentMonitoring(component, testConfig.monitoring);
      });
    }

    if (testConfig.tags) {
      it("should have correct tags", async () => {
        const component = componentFactory();
        await testComponentTags(component, testConfig.tags);
      });
    }

    if (testConfig.costOptimization) {
      it("should have cost optimization features", async () => {
        const component = componentFactory();
        await testComponentCostOptimization(component, testConfig.costOptimization);
      });
    }

    if (testConfig.security) {
      it("should have correct security configuration", async () => {
        const component = componentFactory();
        await testComponentSecurity(component, testConfig.security);
      });
    }
  });
}