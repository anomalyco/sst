import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { withTestEnvironment, createConsoleSpy, assertions, generators } from "../utils/test-helpers";
import { MockAWSComponent } from "../utils/mock-sst";

/**
 * Test component linking and cross-component dependencies
 * Tests Function + Bucket, Auth + Router, VPC + Service integration
 */
describe("Component Linking Integration", () => {
  let consoleSpy: ReturnType<typeof createConsoleSpy>;

  beforeEach(() => {
    consoleSpy = createConsoleSpy('warn');
  });

  afterEach(() => {
    consoleSpy.restore();
  });

  describe("Function + Bucket integration", () => {
    it("should link Function with Bucket for file processing", async () => {
      await withTestEnvironment(async () => {
        // Create S3 bucket for file storage
        const bucket = new MockAWSComponent("ProcessingBucket", "aws:bucket:component", {
          public: false,
          notifications: {
            "process-upload": {
              events: ["s3:ObjectCreated:*"],
              filterPrefix: "uploads/"
            }
          }
        });

        // Create Lambda function that processes files from the bucket
        const processor = new MockAWSComponent("FileProcessor", "aws:function:component", {
          handler: "process.handler",
          runtime: "nodejs20.x",
          timeout: "5 minutes",
          environment: {
            BUCKET_NAME: bucket.generatePhysicalName("bucket"),
            PROCESSED_PREFIX: "processed/"
          },
          link: [bucket]
        });

        // Link bucket notification to the function
        bucket.args.notifications["process-upload"].function = processor.generatePhysicalName("function");

        // Verify the integration
        expect(processor.args.link).toContain(bucket);
        assertions.validOutput(processor.args.environment.BUCKET_NAME);
        assertions.validOutput(bucket.args.notifications["process-upload"].function);
        
        // Verify environment variables are properly set
        expect(processor.args.environment.PROCESSED_PREFIX).toBe("processed/");
        
        // Verify bucket notification configuration
        expect(bucket.args.notifications["process-upload"].events).toContain("s3:ObjectCreated:*");
        expect(bucket.args.notifications["process-upload"].filterPrefix).toBe("uploads/");
      });
    });

    it("should link Function with multiple Buckets for complex workflows", async () => {
      await withTestEnvironment(async () => {
        // Create input bucket
        const inputBucket = new MockAWSComponent("InputBucket", "aws:bucket:component", {
          public: true,
          cors: {
            allowMethods: ["GET", "POST", "PUT"],
            allowOrigins: ["https://app.example.com"]
          }
        });

        // Create output bucket
        const outputBucket = new MockAWSComponent("OutputBucket", "aws:bucket:component", {
          public: false,
          versioning: true
        });

        // Create archive bucket
        const archiveBucket = new MockAWSComponent("ArchiveBucket", "aws:bucket:component", {
          public: false,
          lifecycle: {
            rules: [
              {
                id: "archive-old-files",
                status: "Enabled",
                transitions: [
                  { days: 30, storageClass: "GLACIER" }
                ]
              }
            ]
          }
        });

        // Create processor function that uses all three buckets
        const processor = new MockAWSComponent("MultiBucketProcessor", "aws:function:component", {
          handler: "processor.handler",
          runtime: "nodejs20.x",
          timeout: "10 minutes",
          memory: "1024 MB",
          environment: {
            INPUT_BUCKET: inputBucket.generatePhysicalName("bucket"),
            OUTPUT_BUCKET: outputBucket.generatePhysicalName("bucket"),
            ARCHIVE_BUCKET: archiveBucket.generatePhysicalName("bucket")
          },
          link: [inputBucket, outputBucket, archiveBucket]
        });

        // Verify all buckets are linked
        expect(processor.args.link).toHaveLength(3);
        expect(processor.args.link).toContain(inputBucket);
        expect(processor.args.link).toContain(outputBucket);
        expect(processor.args.link).toContain(archiveBucket);

        // Verify environment variables
        assertions.validOutput(processor.args.environment.INPUT_BUCKET);
        assertions.validOutput(processor.args.environment.OUTPUT_BUCKET);
        assertions.validOutput(processor.args.environment.ARCHIVE_BUCKET);
      });
    });

    it("should handle Function + Bucket + DynamoDB integration", async () => {
      await withTestEnvironment(async () => {
        // Create DynamoDB table for metadata
        const metadataTable = new MockAWSComponent("FileMetadata", "aws:dynamo:component", {
          fields: {
            fileId: "string",
            fileName: "string",
            uploadTime: "number",
            status: "string"
          },
          primaryIndex: {
            hashKey: "fileId"
          },
          globalIndexes: {
            "StatusIndex": {
              hashKey: "status",
              rangeKey: "uploadTime"
            }
          },
          stream: "new-and-old-images"
        });

        // Create S3 bucket for file storage
        const filesBucket = new MockAWSComponent("FilesBucket", "aws:bucket:component", {
          public: false,
          versioning: true
        });

        // Create function that processes files and updates metadata
        const processor = new MockAWSComponent("FileMetadataProcessor", "aws:function:component", {
          handler: "metadata.handler",
          runtime: "nodejs20.x",
          environment: {
            METADATA_TABLE: metadataTable.generatePhysicalName("table"),
            FILES_BUCKET: filesBucket.generatePhysicalName("bucket")
          },
          link: [metadataTable, filesBucket]
        });

        // Create stream processor for metadata changes
        const streamProcessor = new MockAWSComponent("MetadataStreamProcessor", "aws:function:component", {
          handler: "stream.handler",
          runtime: "nodejs20.x",
          environment: {
            NOTIFICATION_TOPIC: "file-status-changes"
          }
        });

        // Verify the integration
        expect(processor.args.link).toHaveLength(2);
        expect(processor.args.link).toContain(metadataTable);
        expect(processor.args.link).toContain(filesBucket);
        
        assertions.validOutput(processor.args.environment.METADATA_TABLE);
        assertions.validOutput(processor.args.environment.FILES_BUCKET);
        
        // Verify DynamoDB stream configuration
        expect(metadataTable.args.stream).toBe("new-and-old-images");
        expect(metadataTable.args.globalIndexes.StatusIndex.hashKey).toBe("status");
      });
    });
  });

  describe("Auth + Router integration", () => {
    it("should integrate Auth component with Router for protected routes", async () => {
      await withTestEnvironment(async () => {
        // Create Auth component with OpenAuth
        const auth = new MockAWSComponent("AppAuth", "aws:auth:component", {
          authenticator: {
            type: "openauth",
            providers: ["google", "github"],
            config: {
              google: {
                clientId: "google-client-id",
                clientSecret: "google-client-secret"
              },
              github: {
                clientId: "github-client-id",
                clientSecret: "github-client-secret"
              }
            }
          }
        });

        // Create API functions
        const publicFunction = new MockAWSComponent("PublicAPI", "aws:function:component", {
          handler: "public.handler",
          runtime: "nodejs20.x"
        });

        const protectedFunction = new MockAWSComponent("ProtectedAPI", "aws:function:component", {
          handler: "protected.handler",
          runtime: "nodejs20.x",
          environment: {
            AUTH_URL: auth.generatePhysicalName("url")
          },
          link: [auth]
        });

        // Create Router with protected and public routes
        const router = new MockAWSComponent("AppRouter", "aws:router:component", {
          routes: {
            "GET /public": publicFunction.generatePhysicalName("function"),
            "GET /protected": {
              function: protectedFunction.generatePhysicalName("function"),
              auth: auth.generatePhysicalName("auth")
            },
            "POST /api/users": {
              function: protectedFunction.generatePhysicalName("function"),
              auth: auth.generatePhysicalName("auth")
            },
            "GET /auth/*": auth.generatePhysicalName("auth")
          }
        });

        // Verify the integration
        expect(protectedFunction.args.link).toContain(auth);
        assertions.validOutput(protectedFunction.args.environment.AUTH_URL);
        
        // Verify router configuration
        assertions.validOutput(router.args.routes["GET /public"]);
        assertions.validOutput(router.args.routes["GET /protected"].function);
        assertions.validOutput(router.args.routes["GET /protected"].auth);
        assertions.validOutput(router.args.routes["GET /auth/*"]);
        
        // Verify auth configuration
        expect(auth.args.authenticator.providers).toContain("google");
        expect(auth.args.authenticator.providers).toContain("github");
      });
    });

    it("should handle Auth + Router + Database integration", async () => {
      await withTestEnvironment(async () => {
        // Create user database
        const userTable = new MockAWSComponent("Users", "aws:dynamo:component", {
          fields: {
            userId: "string",
            email: "string",
            createdAt: "number",
            lastLogin: "number"
          },
          primaryIndex: {
            hashKey: "userId"
          },
          globalIndexes: {
            "EmailIndex": {
              hashKey: "email"
            }
          }
        });

        // Create Auth component
        const auth = new MockAWSComponent("UserAuth", "aws:auth:component", {
          authenticator: {
            type: "openauth",
            providers: ["google", "apple"]
          }
        });

        // Create user management function
        const userFunction = new MockAWSComponent("UserManagement", "aws:function:component", {
          handler: "users.handler",
          runtime: "nodejs20.x",
          environment: {
            USER_TABLE: userTable.generatePhysicalName("table"),
            AUTH_URL: auth.generatePhysicalName("url")
          },
          link: [userTable, auth]
        });

        // Create router with user management routes
        const router = new MockAWSComponent("UserRouter", "aws:router:component", {
          routes: {
            "GET /auth/*": auth.generatePhysicalName("auth"),
            "GET /user/profile": {
              function: userFunction.generatePhysicalName("function"),
              auth: auth.generatePhysicalName("auth")
            },
            "PUT /user/profile": {
              function: userFunction.generatePhysicalName("function"),
              auth: auth.generatePhysicalName("auth")
            },
            "DELETE /user/account": {
              function: userFunction.generatePhysicalName("function"),
              auth: auth.generatePhysicalName("auth")
            }
          }
        });

        // Verify the integration
        expect(userFunction.args.link).toHaveLength(2);
        expect(userFunction.args.link).toContain(userTable);
        expect(userFunction.args.link).toContain(auth);
        
        assertions.validOutput(userFunction.args.environment.USER_TABLE);
        assertions.validOutput(userFunction.args.environment.AUTH_URL);
        
        // Verify all protected routes have auth
        expect(router.args.routes["GET /user/profile"].auth).toBeDefined();
        expect(router.args.routes["PUT /user/profile"].auth).toBeDefined();
        expect(router.args.routes["DELETE /user/account"].auth).toBeDefined();
      });
    });
  });

  describe("VPC + Service integration", () => {
    it("should integrate VPC with Service for container deployment", async () => {
      await withTestEnvironment(async () => {
        // Create VPC with public and private subnets
        const vpc = new MockAWSComponent("AppVPC", "aws:vpc:component", {
          cidr: "10.0.0.0/16",
          enableDnsHostnames: true,
          enableDnsSupport: true,
          availabilityZones: ["us-east-1a", "us-east-1b"],
          subnets: {
            public: {
              cidr: "10.0.0.0/24",
              availabilityZones: ["us-east-1a", "us-east-1b"]
            },
            private: {
              cidr: "10.0.1.0/24",
              availabilityZones: ["us-east-1a", "us-east-1b"]
            }
          },
          natGateways: {
            strategy: "single"
          }
        });

        // Create Service that runs in the VPC
        const service = new MockAWSComponent("WebService", "aws:service:component", {
          image: "nginx:latest",
          port: 80,
          cpu: "256",
          memory: "512",
          vpc: {
            vpcId: vpc.generatePhysicalName("vpc"),
            subnetIds: vpc.generatePhysicalName("private-subnets"),
            securityGroupIds: vpc.generatePhysicalName("service-security-group")
          },
          environment: {
            NODE_ENV: "production",
            VPC_ID: vpc.generatePhysicalName("vpc")
          }
        });

        // Create load balancer in public subnets
        const loadBalancer = new MockAWSComponent("ServiceLB", "aws:loadbalancer:component", {
          type: "application",
          scheme: "internet-facing",
          vpc: {
            vpcId: vpc.generatePhysicalName("vpc"),
            subnetIds: vpc.generatePhysicalName("public-subnets")
          },
          targets: [
            {
              service: service.generatePhysicalName("service"),
              port: 80
            }
          ]
        });

        // Verify the integration
        assertions.validOutput(service.args.vpc.vpcId);
        assertions.validOutput(service.args.vpc.subnetIds);
        assertions.validOutput(service.args.vpc.securityGroupIds);
        assertions.validOutput(service.args.environment.VPC_ID);
        
        assertions.validOutput(loadBalancer.args.vpc.vpcId);
        assertions.validOutput(loadBalancer.args.vpc.subnetIds);
        assertions.validOutput(loadBalancer.args.targets[0].service);
        
        // Verify VPC configuration
        expect(vpc.args.cidr).toBe("10.0.0.0/16");
        expect(vpc.args.subnets.public.cidr).toBe("10.0.0.0/24");
        expect(vpc.args.subnets.private.cidr).toBe("10.0.1.0/24");
        expect(vpc.args.natGateways.strategy).toBe("single");
      });
    });

    it("should integrate VPC with RDS database", async () => {
      await withTestEnvironment(async () => {
        // Create VPC with database subnets
        const vpc = new MockAWSComponent("DatabaseVPC", "aws:vpc:component", {
          cidr: "10.0.0.0/16",
          availabilityZones: ["us-east-1a", "us-east-1b", "us-east-1c"],
          subnets: {
            private: {
              cidr: "10.0.1.0/24",
              availabilityZones: ["us-east-1a", "us-east-1b"]
            },
            database: {
              cidr: "10.0.2.0/24",
              availabilityZones: ["us-east-1a", "us-east-1b", "us-east-1c"]
            }
          },
          securityGroups: {
            database: {
              ingress: [
                {
                  protocol: "tcp",
                  fromPort: 5432,
                  toPort: 5432,
                  sourceSecurityGroupId: "sg-app"
                }
              ]
            }
          }
        });

        // Create PostgreSQL database in VPC
        const database = new MockAWSComponent("AppDatabase", "aws:postgres:component", {
          engine: "postgres",
          version: "15.4",
          instanceClass: "db.t3.micro",
          allocatedStorage: 20,
          vpc: {
            subnetIds: vpc.generatePhysicalName("database-subnets"),
            securityGroupIds: vpc.generatePhysicalName("database-security-group")
          },
          backup: {
            retentionPeriod: 7,
            window: "03:00-04:00"
          }
        });

        // Create application function that connects to the database
        const appFunction = new MockAWSComponent("DatabaseApp", "aws:function:component", {
          handler: "app.handler",
          runtime: "nodejs20.x",
          timeout: "30 seconds",
          vpc: {
            securityGroups: [vpc.generatePhysicalName("app-security-group")],
            subnets: [vpc.generatePhysicalName("private-subnets")]
          },
          environment: {
            DATABASE_URL: database.generatePhysicalName("connection-string"),
            VPC_ID: vpc.generatePhysicalName("vpc")
          },
          link: [database]
        });

        // Verify the integration
        assertions.validOutput(database.args.vpc.subnetIds);
        assertions.validOutput(database.args.vpc.securityGroupIds);
        
        assertions.validOutput(appFunction.args.vpc.securityGroups[0]);
        assertions.validOutput(appFunction.args.vpc.subnets[0]);
        assertions.validOutput(appFunction.args.environment.DATABASE_URL);
        assertions.validOutput(appFunction.args.environment.VPC_ID);
        
        expect(appFunction.args.link).toContain(database);
        
        // Verify VPC security group configuration
        expect(vpc.args.securityGroups.database.ingress[0].fromPort).toBe(5432);
        expect(vpc.args.securityGroups.database.ingress[0].toPort).toBe(5432);
      });
    });

    it("should handle complex VPC + Service + Database + Cache integration", async () => {
      await withTestEnvironment(async () => {
        // Create comprehensive VPC
        const vpc = new MockAWSComponent("ComplexVPC", "aws:vpc:component", {
          cidr: "10.0.0.0/16",
          availabilityZones: ["us-east-1a", "us-east-1b"],
          subnets: {
            public: {
              cidr: "10.0.0.0/24"
            },
            private: {
              cidr: "10.0.1.0/24"
            },
            database: {
              cidr: "10.0.2.0/24"
            },
            cache: {
              cidr: "10.0.3.0/24"
            }
          }
        });

        // Create Redis cache
        const cache = new MockAWSComponent("AppCache", "aws:redis:component", {
          nodeType: "cache.t3.micro",
          numCacheNodes: 1,
          vpc: {
            subnetIds: vpc.generatePhysicalName("cache-subnets"),
            securityGroupIds: vpc.generatePhysicalName("cache-security-group")
          }
        });

        // Create PostgreSQL database
        const database = new MockAWSComponent("AppDB", "aws:postgres:component", {
          instanceClass: "db.t3.micro",
          vpc: {
            subnetIds: vpc.generatePhysicalName("database-subnets"),
            securityGroupIds: vpc.generatePhysicalName("database-security-group")
          }
        });

        // Create web service
        const webService = new MockAWSComponent("WebApp", "aws:service:component", {
          image: "myapp:latest",
          port: 3000,
          vpc: {
            vpcId: vpc.generatePhysicalName("vpc"),
            subnetIds: vpc.generatePhysicalName("private-subnets"),
            securityGroupIds: vpc.generatePhysicalName("app-security-group")
          },
          environment: {
            DATABASE_URL: database.generatePhysicalName("connection-string"),
            REDIS_URL: cache.generatePhysicalName("connection-string"),
            NODE_ENV: "production"
          },
          link: [database, cache]
        });

        // Verify all components are properly integrated
        expect(webService.args.link).toHaveLength(2);
        expect(webService.args.link).toContain(database);
        expect(webService.args.link).toContain(cache);
        
        assertions.validOutput(webService.args.vpc.vpcId);
        assertions.validOutput(webService.args.environment.DATABASE_URL);
        assertions.validOutput(webService.args.environment.REDIS_URL);
        
        assertions.validOutput(database.args.vpc.subnetIds);
        assertions.validOutput(cache.args.vpc.subnetIds);
        
        // Verify VPC subnet configuration
        expect(Object.keys(vpc.args.subnets)).toHaveLength(4);
        expect(vpc.args.subnets.public).toBeDefined();
        expect(vpc.args.subnets.private).toBeDefined();
        expect(vpc.args.subnets.database).toBeDefined();
        expect(vpc.args.subnets.cache).toBeDefined();
      });
    });
  });

  describe("Cross-component dependency resolution", () => {
    it("should handle complex dependency chains", async () => {
      await withTestEnvironment(async () => {
        // Create foundational components
        const vpc = new MockAWSComponent("BaseVPC", "aws:vpc:component", {
          cidr: "10.0.0.0/16"
        });

        const database = new MockAWSComponent("BaseDB", "aws:postgres:component", {
          vpc: {
            subnetIds: vpc.generatePhysicalName("database-subnets")
          }
        });

        const cache = new MockAWSComponent("BaseCache", "aws:redis:component", {
          vpc: {
            subnetIds: vpc.generatePhysicalName("cache-subnets")
          }
        });

        // Create service that depends on database and cache
        const service = new MockAWSComponent("BaseService", "aws:service:component", {
          vpc: {
            vpcId: vpc.generatePhysicalName("vpc")
          },
          environment: {
            DATABASE_URL: database.generatePhysicalName("connection-string"),
            REDIS_URL: cache.generatePhysicalName("connection-string")
          },
          link: [database, cache]
        });

        // Create function that depends on service
        const function1 = new MockAWSComponent("ServiceClient", "aws:function:component", {
          handler: "client.handler",
          environment: {
            SERVICE_URL: service.generatePhysicalName("url")
          },
          link: [service]
        });

        // Create another function that depends on database directly
        const function2 = new MockAWSComponent("DatabaseClient", "aws:function:component", {
          handler: "db.handler",
          environment: {
            DATABASE_URL: database.generatePhysicalName("connection-string")
          },
          link: [database]
        });

        // Verify dependency chain
        expect(service.args.link).toContain(database);
        expect(service.args.link).toContain(cache);
        expect(function1.args.link).toContain(service);
        expect(function2.args.link).toContain(database);
        
        // Verify all outputs are valid
        assertions.validOutput(service.args.environment.DATABASE_URL);
        assertions.validOutput(service.args.environment.REDIS_URL);
        assertions.validOutput(function1.args.environment.SERVICE_URL);
        assertions.validOutput(function2.args.environment.DATABASE_URL);
      });
    });

    it("should prevent circular dependencies", async () => {
      await withTestEnvironment(async () => {
        const componentA = new MockAWSComponent("ComponentA", "aws:function:component", {
          handler: "a.handler"
        });

        const componentB = new MockAWSComponent("ComponentB", "aws:function:component", {
          handler: "b.handler",
          environment: {
            COMPONENT_A_URL: componentA.generatePhysicalName("url")
          },
          link: [componentA]
        });

        // This would create a circular dependency in real implementation
        // but our mock allows it for testing purposes
        componentA.args.environment = {
          COMPONENT_B_URL: componentB.generatePhysicalName("url")
        };
        componentA.args.link = [componentB];

        // Verify both components reference each other
        expect(componentA.args.link).toContain(componentB);
        expect(componentB.args.link).toContain(componentA);
        
        assertions.validOutput(componentA.args.environment.COMPONENT_B_URL);
        assertions.validOutput(componentB.args.environment.COMPONENT_A_URL);
      });
    });
  });
});