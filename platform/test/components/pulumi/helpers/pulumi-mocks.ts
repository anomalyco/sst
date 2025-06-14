import * as pulumi from "@pulumi/pulumi";

/**
 * Standardized Pulumi mocks for all SST components
 * Provides realistic AWS/Cloudflare provider responses for testing
 */

export interface MockResourceOptions {
  /** Custom ID generation function */
  idGenerator?: (args: pulumi.runtime.MockResourceArgs) => string;
  /** Custom state generation function */
  stateGenerator?: (args: pulumi.runtime.MockResourceArgs) => any;
  /** Mock specific resource types differently */
  resourceTypeHandlers?: Record<string, (args: pulumi.runtime.MockResourceArgs) => { id: string; state: any }>;
}

export interface MockCallOptions {
  /** Mock specific function calls */
  callHandlers?: Record<string, (args: pulumi.runtime.MockCallArgs) => any>;
}

/**
 * Creates standardized Pulumi mocks for SST testing
 */
export function createSSTPulumiMocks(
  resourceOptions: MockResourceOptions = {},
  callOptions: MockCallOptions = {}
): pulumi.runtime.Mocks {
  const {
    idGenerator = defaultIdGenerator,
    stateGenerator = defaultStateGenerator,
    resourceTypeHandlers = {},
  } = resourceOptions;

  const { callHandlers = {} } = callOptions;

  return {
    newResource: function (args: pulumi.runtime.MockResourceArgs): {
      id: string;
      state: any;
    } {
      // Check for custom resource type handlers
      if (resourceTypeHandlers[args.type]) {
        return resourceTypeHandlers[args.type](args);
      }

      // Generate realistic mock responses based on resource type
      const id = idGenerator(args);
      const state = stateGenerator(args);

      return { id, state };
    },
    call: function (args: pulumi.runtime.MockCallArgs) {
      // Check for custom call handlers
      if (callHandlers[args.token]) {
        return callHandlers[args.token](args);
      }

      // Default call behavior
      return defaultCallHandler(args);
    },
  };
}

/**
 * Default ID generator that creates realistic AWS-style IDs
 */
function defaultIdGenerator(args: pulumi.runtime.MockResourceArgs): string {
  const resourceType = args.type.split(":").pop() || "resource";
  const name = args.inputs.name || args.name;
  
  // Generate realistic IDs based on resource type
  switch (args.type) {
    case "aws:s3/bucket:Bucket":
      return `${name}-${generateRandomString(8)}`;
    case "aws:lambda/function:Function":
      return `arn:aws:lambda:us-east-1:123456789012:function:${name}`;
    case "aws:apigateway/restApi:RestApi":
      return generateRandomString(10);
    case "aws:apigatewayv2/api:Api":
      return generateRandomString(10);
    case "aws:cloudfront/distribution:Distribution":
      return `E${generateRandomString(13).toUpperCase()}`;
    case "aws:route53/hostedZone:HostedZone":
      return `Z${generateRandomString(13).toUpperCase()}`;
    case "aws:iam/role:Role":
      return `arn:aws:iam::123456789012:role/${name}`;
    case "cloudflare:index/worker:Worker":
      return `${name}-worker`;
    case "cloudflare:index/zone:Zone":
      return generateRandomString(32);
    default:
      return `${name}_${generateRandomString(8)}`;
  }
}

/**
 * Default state generator that creates realistic resource properties
 */
function defaultStateGenerator(args: pulumi.runtime.MockResourceArgs): any {
  const baseState = { ...args.inputs };

  // Add realistic properties based on resource type
  switch (args.type) {
    case "aws:s3/bucket:Bucket":
      return {
        ...baseState,
        arn: `arn:aws:s3:::${baseState.bucket || baseState.name}`,
        bucketDomainName: `${baseState.bucket || baseState.name}.s3.amazonaws.com`,
        bucketRegionalDomainName: `${baseState.bucket || baseState.name}.s3.us-east-1.amazonaws.com`,
        region: "us-east-1",
        websiteEndpoint: `${baseState.bucket || baseState.name}.s3-website-us-east-1.amazonaws.com`,
      };

    case "aws:lambda/function:Function":
      return {
        ...baseState,
        arn: `arn:aws:lambda:us-east-1:123456789012:function:${baseState.functionName || baseState.name}`,
        invokeArn: `arn:aws:apigateway:us-east-1:lambda:path/2015-03-31/functions/arn:aws:lambda:us-east-1:123456789012:function:${baseState.functionName || baseState.name}/invocations`,
        lastModified: new Date().toISOString(),
        version: "$LATEST",
        qualifiedArn: `arn:aws:lambda:us-east-1:123456789012:function:${baseState.functionName || baseState.name}:$LATEST`,
      };

    case "aws:apigateway/restApi:RestApi":
      return {
        ...baseState,
        executionArn: `arn:aws:execute-api:us-east-1:123456789012:${generateRandomString(10)}`,
        createdDate: new Date().toISOString(),
        rootResourceId: generateRandomString(6),
      };

    case "aws:apigatewayv2/api:Api":
      return {
        ...baseState,
        apiEndpoint: `https://${generateRandomString(10)}.execute-api.us-east-1.amazonaws.com`,
        executionArn: `arn:aws:execute-api:us-east-1:123456789012:${generateRandomString(10)}`,
        createdDate: new Date().toISOString(),
      };

    case "aws:cloudfront/distribution:Distribution":
      return {
        ...baseState,
        arn: `arn:aws:cloudfront::123456789012:distribution/E${generateRandomString(13).toUpperCase()}`,
        domainName: `d${generateRandomString(13).toLowerCase()}.cloudfront.net`,
        etag: generateRandomString(32),
        status: "Deployed",
        lastModifiedTime: new Date().toISOString(),
      };

    case "aws:route53/hostedZone:HostedZone":
      return {
        ...baseState,
        arn: `arn:aws:route53:::hostedzone/Z${generateRandomString(13).toUpperCase()}`,
        nameServers: [
          `ns-${Math.floor(Math.random() * 2000)}.awsdns-${Math.floor(Math.random() * 100)}.com`,
          `ns-${Math.floor(Math.random() * 2000)}.awsdns-${Math.floor(Math.random() * 100)}.co.uk`,
          `ns-${Math.floor(Math.random() * 2000)}.awsdns-${Math.floor(Math.random() * 100)}.net`,
          `ns-${Math.floor(Math.random() * 2000)}.awsdns-${Math.floor(Math.random() * 100)}.org`,
        ],
      };

    case "aws:iam/role:Role":
      return {
        ...baseState,
        arn: `arn:aws:iam::123456789012:role/${baseState.name}`,
        createDate: new Date().toISOString(),
        uniqueId: `AROA${generateRandomString(16).toUpperCase()}`,
      };

    case "cloudflare:index/worker:Worker":
      return {
        ...baseState,
        id: `${baseState.name}-worker`,
        createdOn: new Date().toISOString(),
        modifiedOn: new Date().toISOString(),
      };

    case "cloudflare:index/zone:Zone":
      return {
        ...baseState,
        id: generateRandomString(32),
        nameServers: [
          `${generateRandomString(8)}.ns.cloudflare.com`,
          `${generateRandomString(8)}.ns.cloudflare.com`,
        ],
        status: "active",
      };

    default:
      return baseState;
  }
}

/**
 * Default call handler for Pulumi function calls
 */
function defaultCallHandler(args: pulumi.runtime.MockCallArgs): any {
  switch (args.token) {
    case "aws:index/getCallerIdentity:getCallerIdentity":
      return {
        accountId: "123456789012",
        arn: "arn:aws:iam::123456789012:user/test-user",
        userId: "AIDA" + generateRandomString(16).toUpperCase(),
      };

    case "aws:index/getRegion:getRegion":
      return {
        name: "us-east-1",
        description: "US East (N. Virginia)",
      };

    case "aws:s3/getBucket:getBucket":
      return {
        arn: `arn:aws:s3:::${args.inputs.bucket}`,
        bucketDomainName: `${args.inputs.bucket}.s3.amazonaws.com`,
        region: "us-east-1",
      };

    case "cloudflare:index/getZone:getZone":
      return {
        id: generateRandomString(32),
        name: args.inputs.name,
        nameServers: [
          `${generateRandomString(8)}.ns.cloudflare.com`,
          `${generateRandomString(8)}.ns.cloudflare.com`,
        ],
        status: "active",
      };

    default:
      return args.inputs;
  }
}

/**
 * Generates a random string of specified length
 */
function generateRandomString(length: number): string {
  const chars = "abcdefghijklmnopqrstuvwxyz0123456789";
  let result = "";
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

/**
 * Sets up the global SST test environment with Pulumi mocks
 */
export function setupSSTTestEnvironment(
  appName: string = "test-app",
  stage: string = "test",
  resourceOptions?: MockResourceOptions,
  callOptions?: MockCallOptions
): void {
  // Set up global SST variables
  // @ts-ignore
  global.$app = {
    name: appName,
    stage: stage,
  };
  global.$util = pulumi;
  
  // Set up $dev global for development mode detection
  // @ts-ignore
  global.$dev = false;
  
  // Set up $output global for output handling
  // @ts-ignore
  global.$output = pulumi.output;
  
  // Set up $resolve global for resolving outputs
  // @ts-ignore
  global.$resolve = (value: any) => Promise.resolve(value);
  
  // Set up server URL for RPC calls (mock server)
  // @ts-ignore
  global.$server = {
    url: "http://localhost:13557",
  };

  // Set up Pulumi mocks
  pulumi.runtime.setMocks(
    createSSTPulumiMocks(resourceOptions, callOptions),
    "project",
    "stack",
    false // dryRun
  );
}

/**
 * Helper function to create AWS-specific mocks
 */
export function createAWSMocks(): pulumi.runtime.Mocks {
  return createSSTPulumiMocks(
    {
      resourceTypeHandlers: {
        "aws:s3/bucket:Bucket": (args) => ({
          id: `${args.inputs.bucket || args.inputs.name}-${generateRandomString(8)}`,
          state: {
            ...args.inputs,
            arn: `arn:aws:s3:::${args.inputs.bucket || args.inputs.name}`,
            bucketDomainName: `${args.inputs.bucket || args.inputs.name}.s3.amazonaws.com`,
            region: "us-east-1",
          },
        }),
        "aws:lambda/function:Function": (args) => ({
          id: `arn:aws:lambda:us-east-1:123456789012:function:${args.inputs.functionName || args.inputs.name}`,
          state: {
            ...args.inputs,
            arn: `arn:aws:lambda:us-east-1:123456789012:function:${args.inputs.functionName || args.inputs.name}`,
            invokeArn: `arn:aws:apigateway:us-east-1:lambda:path/2015-03-31/functions/arn:aws:lambda:us-east-1:123456789012:function:${args.inputs.functionName || args.inputs.name}/invocations`,
            version: "$LATEST",
          },
        }),
      },
    },
    {
      callHandlers: {
        "aws:index/getCallerIdentity:getCallerIdentity": () => ({
          accountId: "123456789012",
          arn: "arn:aws:iam::123456789012:user/test-user",
          userId: "AIDA" + generateRandomString(16).toUpperCase(),
        }),
      },
    }
  );
}

/**
 * Helper function to create Cloudflare-specific mocks
 */
export function createCloudflareMocks(): pulumi.runtime.Mocks {
  return createSSTPulumiMocks(
    {
      resourceTypeHandlers: {
        "cloudflare:index/worker:Worker": (args) => ({
          id: `${args.inputs.name}-worker`,
          state: {
            ...args.inputs,
            id: `${args.inputs.name}-worker`,
            createdOn: new Date().toISOString(),
            modifiedOn: new Date().toISOString(),
          },
        }),
        "cloudflare:index/zone:Zone": (args) => ({
          id: generateRandomString(32),
          state: {
            ...args.inputs,
            id: generateRandomString(32),
            nameServers: [
              `${generateRandomString(8)}.ns.cloudflare.com`,
              `${generateRandomString(8)}.ns.cloudflare.com`,
            ],
            status: "active",
          },
        }),
      },
    },
    {
      callHandlers: {
        "cloudflare:index/getZone:getZone": (args) => ({
          id: generateRandomString(32),
          name: args.inputs.name,
          nameServers: [
            `${generateRandomString(8)}.ns.cloudflare.com`,
            `${generateRandomString(8)}.ns.cloudflare.com`,
          ],
          status: "active",
        }),
      },
    }
  );
}

/**
 * Resource validation utilities
 */
export class ResourceValidator {
  /**
   * Validates that a resource has the expected properties
   */
  static validateResourceProperties(
    resource: any,
    expectedProperties: Record<string, any>
  ): void {
    for (const [key, expectedValue] of Object.entries(expectedProperties)) {
      if (resource[key] === undefined) {
        throw new Error(`Resource missing expected property: ${key}`);
      }
      if (expectedValue !== undefined && resource[key] !== expectedValue) {
        throw new Error(
          `Resource property ${key} has value ${resource[key]}, expected ${expectedValue}`
        );
      }
    }
  }

  /**
   * Validates that a resource has the expected type
   */
  static validateResourceType(resource: any, expectedType: string): void {
    if (!resource.__pulumiType || resource.__pulumiType !== expectedType) {
      throw new Error(
        `Resource has type ${resource.__pulumiType}, expected ${expectedType}`
      );
    }
  }

  /**
   * Validates that a resource follows SST naming conventions
   */
  static validateSSTNaming(
    resourceName: string,
    appName: string,
    stage: string
  ): void {
    const expectedPrefix = `${appName}-${stage}-`;
    if (!resourceName.toLowerCase().startsWith(expectedPrefix.toLowerCase())) {
      throw new Error(
        `Resource name ${resourceName} does not follow SST naming convention (expected to start with ${expectedPrefix})`
      );
    }
  }
}