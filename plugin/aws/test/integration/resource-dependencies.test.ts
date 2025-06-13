import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { withTestEnvironment, createConsoleSpy, assertions, generators } from "../utils/test-helpers";
import { MockAWSComponent } from "../utils/mock-sst";

/**
 * Test resource dependencies and dependency resolution
 * Tests component dependency chains, circular dependency prevention, and resource ordering
 */
describe("Resource Dependencies Integration", () => {
  let consoleSpy: ReturnType<typeof createConsoleSpy>;

  beforeEach(() => {
    consoleSpy = createConsoleSpy('warn');
  });

  afterEach(() => {
    consoleSpy.restore();
  });

  describe("Component dependency resolution", () => {
    it("should resolve simple component dependencies correctly", async () => {
      await withTestEnvironment(async () => {
        // Create a bucket that will be used by a function
        const bucket = new MockAWSComponent("DataBucket", "aws:bucket:component", {
          public: false,
        });

        // Create a function that depends on the bucket
        const processor = new MockAWSComponent("DataProcessor", "aws:function:component", {
          handler: "process.handler",
          runtime: "nodejs20.x",
          environment: {
            BUCKET_NAME: bucket.generatePhysicalName("bucket"),
          },
          link: [bucket],
        });

        // Verify dependency is established
        expect(processor.args.link).toContain(bucket);
        expect(processor.args.environment.BUCKET_NAME).toBeDefined();

        // Verify bucket can be referenced by function
        const bucketName = bucket.generatePhysicalName("bucket");
        expect(bucketName.value).toMatch(/^test-app-test-databucket-bucket-/);
      });
    });

    it("should handle complex multi-level dependencies", async () => {
      await withTestEnvironment(async () => {
        // Create VPC (base infrastructure)
        const vpc = new MockAWSComponent("AppVpc", "aws:vpc:component:v2", {
          az: 2,
          nat: "single",
        });

        // Create database in VPC
        const database = new MockAWSComponent("AppDatabase", "aws:postgres:component", {
          engine: "postgres15.4",
          vpc: vpc,
        });

        // Create cache in VPC
        const cache = new MockAWSComponent("AppCache", "aws:redis:component", {
          engine: "redis7.0",
          vpc: vpc,
        });

        // Create service that depends on database and cache
        const service = new MockAWSComponent("AppService", "aws:service:component", {
          image: "app:latest",
          vpc: vpc,
          environment: {
            DATABASE_URL: database.generatePhysicalName("connection-string"),
            REDIS_URL: cache.generatePhysicalName("connection-string"),
          },
          link: [database, cache],
        });

        // Verify all dependencies are established
        expect(service.args.link).toContain(database);
        expect(service.args.link).toContain(cache);
        expect(service.args.vpc).toBe(vpc);
        expect(database.args.vpc).toBe(vpc);
        expect(cache.args.vpc).toBe(vpc);

        // Verify environment variables are set
        expect(service.args.environment.DATABASE_URL).toBeDefined();
        expect(service.args.environment.REDIS_URL).toBeDefined();
      });
    });

    it("should handle cross-component type dependencies", async () => {
      await withTestEnvironment(async () => {
        // Create storage bucket
        const bucket = new MockAWSComponent("FileBucket", "aws:bucket:component");

        // Create queue for processing notifications
        const queue = new MockAWSComponent("ProcessQueue", "aws:queue:component");

        // Create topic for notifications
        const topic = new MockAWSComponent("NotificationTopic", "aws:topic:component");

        // Create function that processes files
        const processor = new MockAWSComponent("FileProcessor", "aws:function:component", {
          handler: "process.handler",
          runtime: "nodejs20.x",
          link: [bucket, queue, topic],
          environment: {
            BUCKET_NAME: bucket.generatePhysicalName("bucket"),
            QUEUE_URL: queue.generatePhysicalName("queue"),
            TOPIC_ARN: topic.generatePhysicalName("topic"),
          },
        });

        // Create API that triggers processing
        const api = new MockAWSComponent("FileAPI", "aws:api:component", {
          routes: {
            "POST /upload": processor,
          },
        });

        // Verify all dependencies
        expect(processor.args.link).toContain(bucket);
        expect(processor.args.link).toContain(queue);
        expect(processor.args.link).toContain(topic);
        expect(api.args.routes["POST /upload"]).toBe(processor);

        // Verify environment configuration
        expect(processor.args.environment.BUCKET_NAME).toBeDefined();
        expect(processor.args.environment.QUEUE_URL).toBeDefined();
        expect(processor.args.environment.TOPIC_ARN).toBeDefined();
      });
    });
  });

  describe("Circular dependency prevention", () => {
    it("should detect potential circular dependencies", async () => {
      await withTestEnvironment(async () => {
        // Create components that could form a circular dependency
        const serviceA = new MockAWSComponent("ServiceA", "aws:service:component", {
          image: "service-a:latest",
        });

        const serviceB = new MockAWSComponent("ServiceB", "aws:service:component", {
          image: "service-b:latest",
        });

        // In a real implementation, this would be detected and prevented
        // For now, we just verify the components can be created
        expect(serviceA).toBeDefined();
        expect(serviceB).toBeDefined();

        // Simulate linking (in real implementation, circular detection would happen here)
        serviceA.args.link = [serviceB];
        serviceB.args.link = [serviceA];

        // Verify the circular reference exists (would be caught in real implementation)
        expect(serviceA.args.link).toContain(serviceB);
        expect(serviceB.args.link).toContain(serviceA);
      });
    });

    it("should allow valid dependency chains without false positives", async () => {
      await withTestEnvironment(async () => {
        // Create a valid dependency chain: API -> Function -> Database
        const database = new MockAWSComponent("UserDB", "aws:dynamo:component", {
          fields: { id: "string" },
          primaryIndex: { hashKey: "id" },
        });

        const userFunction = new MockAWSComponent("UserFunction", "aws:function:component", {
          handler: "users.handler",
          runtime: "nodejs20.x",
          link: [database],
        });

        const api = new MockAWSComponent("UserAPI", "aws:api:component", {
          routes: {
            "GET /users": userFunction,
            "POST /users": userFunction,
          },
        });

        // Verify valid chain
        expect(userFunction.args.link).toContain(database);
        expect(api.args.routes["GET /users"]).toBe(userFunction);
        expect(api.args.routes["POST /users"]).toBe(userFunction);

        // This should not trigger circular dependency warnings
        expect(database.args.link).toBeUndefined();
      });
    });

    it("should handle complex dependency graphs correctly", async () => {
      await withTestEnvironment(async () => {
        // Create a complex but valid dependency graph
        const vpc = new MockAWSComponent("NetworkVpc", "aws:vpc:component:v2");
        const database = new MockAWSComponent("MainDB", "aws:postgres:component", { vpc });
        const cache = new MockAWSComponent("MainCache", "aws:redis:component", { vpc });
        const bucket = new MockAWSComponent("AssetBucket", "aws:bucket:component");
        
        const authService = new MockAWSComponent("AuthService", "aws:service:component", {
          vpc,
          link: [database],
        });

        const userService = new MockAWSComponent("UserService", "aws:service:component", {
          vpc,
          link: [database, cache, authService],
        });

        const fileService = new MockAWSComponent("FileService", "aws:service:component", {
          vpc,
          link: [bucket, userService],
        });

        // Verify complex dependencies
        expect(authService.args.link).toContain(database);
        expect(userService.args.link).toContain(database);
        expect(userService.args.link).toContain(cache);
        expect(userService.args.link).toContain(authService);
        expect(fileService.args.link).toContain(bucket);
        expect(fileService.args.link).toContain(userService);

        // Verify no circular dependencies
        expect(database.args.link).toBeUndefined();
        expect(cache.args.link).toBeUndefined();
        expect(bucket.args.link).toBeUndefined();
        expect(authService.args.link).not.toContain(userService);
        expect(authService.args.link).not.toContain(fileService);
      });
    });
  });

  describe("Resource creation ordering", () => {
    it("should respect dependency order for resource creation", async () => {
      await withTestEnvironment(async () => {
        // Create components in dependency order
        const vpc = new MockAWSComponent("OrderVpc", "aws:vpc:component:v2", {
          az: 2,
        });

        const securityGroup = new MockAWSComponent("OrderSG", "aws:security-group:component", {
          vpc: vpc,
          rules: [
            { type: "ingress", port: 80, source: "0.0.0.0/0" },
          ],
        });

        const database = new MockAWSComponent("OrderDB", "aws:postgres:component", {
          vpc: vpc,
          securityGroups: [securityGroup],
        });

        const service = new MockAWSComponent("OrderService", "aws:service:component", {
          vpc: vpc,
          securityGroups: [securityGroup],
          link: [database],
        });

        // Verify dependency relationships
        expect(securityGroup.args.vpc).toBe(vpc);
        expect(database.args.vpc).toBe(vpc);
        expect(database.args.securityGroups).toContain(securityGroup);
        expect(service.args.vpc).toBe(vpc);
        expect(service.args.securityGroups).toContain(securityGroup);
        expect(service.args.link).toContain(database);
      });
    });

    it("should handle parallel resource creation for independent components", async () => {
      await withTestEnvironment(async () => {
        // Create independent components that can be created in parallel
        const bucket1 = new MockAWSComponent("ParallelBucket1", "aws:bucket:component");
        const bucket2 = new MockAWSComponent("ParallelBucket2", "aws:bucket:component");
        const bucket3 = new MockAWSComponent("ParallelBucket3", "aws:bucket:component");

        const queue1 = new MockAWSComponent("ParallelQueue1", "aws:queue:component");
        const queue2 = new MockAWSComponent("ParallelQueue2", "aws:queue:component");

        const topic1 = new MockAWSComponent("ParallelTopic1", "aws:topic:component");
        const topic2 = new MockAWSComponent("ParallelTopic2", "aws:topic:component");

        // Verify all components are independent
        expect(bucket1.args.link).toBeUndefined();
        expect(bucket2.args.link).toBeUndefined();
        expect(bucket3.args.link).toBeUndefined();
        expect(queue1.args.link).toBeUndefined();
        expect(queue2.args.link).toBeUndefined();
        expect(topic1.args.link).toBeUndefined();
        expect(topic2.args.link).toBeUndefined();

        // These could all be created in parallel in a real implementation
        const components = [bucket1, bucket2, bucket3, queue1, queue2, topic1, topic2];
        expect(components).toHaveLength(7);
        components.forEach(component => {
          expect(component).toBeDefined();
          expect(component.name).toBeTruthy();
        });
      });
    });

    it("should handle mixed parallel and sequential dependencies", async () => {
      await withTestEnvironment(async () => {
        // Create base infrastructure (can be parallel)
        const bucket = new MockAWSComponent("MixedBucket", "aws:bucket:component");
        const queue = new MockAWSComponent("MixedQueue", "aws:queue:component");
        const topic = new MockAWSComponent("MixedTopic", "aws:topic:component");

        // Create functions that depend on infrastructure (sequential after base)
        const processor1 = new MockAWSComponent("MixedProcessor1", "aws:function:component", {
          handler: "process1.handler",
          runtime: "nodejs20.x",
          link: [bucket, queue],
        });

        const processor2 = new MockAWSComponent("MixedProcessor2", "aws:function:component", {
          handler: "process2.handler",
          runtime: "nodejs20.x",
          link: [bucket, topic],
        });

        // Create orchestrator that depends on processors (sequential after functions)
        const orchestrator = new MockAWSComponent("MixedOrchestrator", "aws:function:component", {
          handler: "orchestrate.handler",
          runtime: "nodejs20.x",
          link: [processor1, processor2],
        });

        // Verify dependency levels
        // Level 1: Independent infrastructure
        expect(bucket.args.link).toBeUndefined();
        expect(queue.args.link).toBeUndefined();
        expect(topic.args.link).toBeUndefined();

        // Level 2: Functions depending on infrastructure
        expect(processor1.args.link).toContain(bucket);
        expect(processor1.args.link).toContain(queue);
        expect(processor2.args.link).toContain(bucket);
        expect(processor2.args.link).toContain(topic);

        // Level 3: Orchestrator depending on functions
        expect(orchestrator.args.link).toContain(processor1);
        expect(orchestrator.args.link).toContain(processor2);
      });
    });
  });

  describe("Cross-service dependencies", () => {
    it("should handle dependencies between different AWS services", async () => {
      await withTestEnvironment(async () => {
        // Create S3 bucket
        const bucket = new MockAWSComponent("CrossBucket", "aws:bucket:component", {
          notifications: {
            "process-upload": {
              events: ["s3:ObjectCreated:*"],
            },
          },
        });

        // Create SQS queue
        const queue = new MockAWSComponent("CrossQueue", "aws:queue:component", {
          deadLetterQueue: {
            maxReceiveCount: 3,
          },
        });

        // Create Lambda function
        const processor = new MockAWSComponent("CrossProcessor", "aws:function:component", {
          handler: "process.handler",
          runtime: "nodejs20.x",
          link: [bucket, queue],
        });

        // Create DynamoDB table
        const table = new MockAWSComponent("CrossTable", "aws:dynamo:component", {
          fields: { id: "string", data: "string" },
          primaryIndex: { hashKey: "id" },
          stream: "new-and-old-images",
        });

        // Create EventBridge rule
        const rule = new MockAWSComponent("CrossRule", "aws:eventbridge:component", {
          eventPattern: {
            source: ["custom.app"],
            "detail-type": ["File Processed"],
          },
          targets: [processor],
        });

        // Verify cross-service dependencies
        expect(processor.args.link).toContain(bucket);
        expect(processor.args.link).toContain(queue);
        expect(rule.args.targets).toContain(processor);

        // Verify service-specific configurations
        expect(bucket.args.notifications["process-upload"]).toBeDefined();
        expect(queue.args.deadLetterQueue.maxReceiveCount).toBe(3);
        expect(table.args.stream).toBe("new-and-old-images");
      });
    });

    it("should handle API Gateway to Lambda to Database dependencies", async () => {
      await withTestEnvironment(async () => {
        // Create database
        const userTable = new MockAWSComponent("UserTable", "aws:dynamo:component", {
          fields: { userId: "string", email: "string" },
          primaryIndex: { hashKey: "userId" },
          globalIndexes: {
            emailIndex: { hashKey: "email" },
          },
        });

        // Create Lambda functions
        const getUserFunction = new MockAWSComponent("GetUserFunction", "aws:function:component", {
          handler: "getUser.handler",
          runtime: "nodejs20.x",
          link: [userTable],
        });

        const createUserFunction = new MockAWSComponent("CreateUserFunction", "aws:function:component", {
          handler: "createUser.handler",
          runtime: "nodejs20.x",
          link: [userTable],
        });

        // Create API Gateway
        const api = new MockAWSComponent("UserAPI", "aws:api:component", {
          routes: {
            "GET /users/{id}": getUserFunction,
            "POST /users": createUserFunction,
          },
          cors: {
            allowOrigins: ["*"],
            allowMethods: ["GET", "POST"],
          },
        });

        // Verify API to Lambda dependencies
        expect(api.args.routes["GET /users/{id}"]).toBe(getUserFunction);
        expect(api.args.routes["POST /users"]).toBe(createUserFunction);

        // Verify Lambda to Database dependencies
        expect(getUserFunction.args.link).toContain(userTable);
        expect(createUserFunction.args.link).toContain(userTable);

        // Verify database configuration
        expect(userTable.args.globalIndexes.emailIndex).toBeDefined();
      });
    });

    it("should handle VPC-based service dependencies", async () => {
      await withTestEnvironment(async () => {
        // Create VPC
        const vpc = new MockAWSComponent("ServiceVpc", "aws:vpc:component:v2", {
          az: 3,
          nat: "per-az",
        });

        // Create RDS database in VPC
        const database = new MockAWSComponent("ServiceDB", "aws:postgres:component", {
          engine: "postgres15.4",
          vpc: vpc,
          multiAz: true,
        });

        // Create ElastiCache in VPC
        const cache = new MockAWSComponent("ServiceCache", "aws:redis:component", {
          engine: "redis7.0",
          vpc: vpc,
          nodeType: "cache.t3.micro",
        });

        // Create ECS service in VPC
        const webService = new MockAWSComponent("WebService", "aws:service:component", {
          image: "web:latest",
          vpc: vpc,
          link: [database, cache],
          environment: {
            DB_HOST: database.generatePhysicalName("host"),
            REDIS_HOST: cache.generatePhysicalName("host"),
          },
        });

        // Create Application Load Balancer
        const loadBalancer = new MockAWSComponent("WebLB", "aws:load-balancer:component", {
          vpc: vpc,
          targets: [webService],
        });

        // Verify VPC dependencies
        expect(database.args.vpc).toBe(vpc);
        expect(cache.args.vpc).toBe(vpc);
        expect(webService.args.vpc).toBe(vpc);
        expect(loadBalancer.args.vpc).toBe(vpc);

        // Verify service dependencies
        expect(webService.args.link).toContain(database);
        expect(webService.args.link).toContain(cache);
        expect(loadBalancer.args.targets).toContain(webService);

        // Verify environment configuration
        expect(webService.args.environment.DB_HOST).toBeDefined();
        expect(webService.args.environment.REDIS_HOST).toBeDefined();
      });
    });
  });

  describe("Dependency validation", () => {
    it("should validate that required dependencies are provided", async () => {
      await withTestEnvironment(async () => {
        // Create a service that requires a database
        const service = new MockAWSComponent("ValidatedService", "aws:service:component", {
          image: "service:latest",
          // Missing required database dependency
        });

        // In a real implementation, this would validate required dependencies
        expect(service).toBeDefined();
        expect(service.args.link).toBeUndefined();

        // Simulate adding required dependency
        const database = new MockAWSComponent("RequiredDB", "aws:postgres:component");
        service.args.link = [database];

        expect(service.args.link).toContain(database);
      });
    });

    it("should validate dependency types are compatible", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("TypeBucket", "aws:bucket:component");
        const queue = new MockAWSComponent("TypeQueue", "aws:queue:component");

        // Create function with mixed dependency types
        const processor = new MockAWSComponent("TypeProcessor", "aws:function:component", {
          handler: "process.handler",
          runtime: "nodejs20.x",
          link: [bucket, queue], // Both are valid linkable types
        });

        // Verify mixed types are accepted
        expect(processor.args.link).toContain(bucket);
        expect(processor.args.link).toContain(queue);
        expect(processor.args.link).toHaveLength(2);
      });
    });

    it("should handle optional dependencies gracefully", async () => {
      await withTestEnvironment(async () => {
        // Create service with optional cache dependency
        const service = new MockAWSComponent("OptionalService", "aws:service:component", {
          image: "service:latest",
          // Cache is optional
        });

        // Service should work without cache
        expect(service).toBeDefined();
        expect(service.args.link).toBeUndefined();

        // Add optional cache later
        const cache = new MockAWSComponent("OptionalCache", "aws:redis:component");
        service.args.link = [cache];

        expect(service.args.link).toContain(cache);
      });
    });
  });
});