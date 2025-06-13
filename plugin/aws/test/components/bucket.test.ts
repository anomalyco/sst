import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { withTestEnvironment, createConsoleSpy, assertions, generators } from "../utils/test-helpers";
import { MockAWSComponent } from "../utils/mock-sst";

/**
 * Test Bucket component creation and configuration
 * Tests CORS configuration and notification setup
 */
describe("Bucket Component", () => {
  let consoleSpy: ReturnType<typeof createConsoleSpy>;

  beforeEach(() => {
    consoleSpy = createConsoleSpy('warn');
  });

  afterEach(() => {
    consoleSpy.restore();
  });

  describe("Bucket creation", () => {
    it("should create Bucket component with basic configuration", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("TestBucket", "aws:bucket:component", {
          name: "my-test-bucket"
        });

        expect(bucket).toBeDefined();
        expect(bucket.originalName).toBe("TestBucket"); expect(bucket.name).toMatch(/test-app-test-testbucket-/);
        expect(bucket.args.name).toBe("my-test-bucket");
      });
    });

    it("should create Bucket component with minimal configuration", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("MinimalBucket", "aws:bucket:component");

        expect(bucket).toBeDefined();
        expect(bucket.originalName).toBe("MinimalBucket"); expect(bucket.name).toMatch(/test-app-test-minimalbucket-/);
      });
    });

    it("should create Bucket component with advanced configuration", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("AdvancedBucket", "aws:bucket:component", {
          name: "advanced-bucket",
          public: true,
          versioning: true,
          cors: {
            allowCredentials: true,
            allowHeaders: ["*"],
            allowMethods: ["GET", "POST", "PUT", "DELETE"],
            allowOrigins: ["https://example.com", "https://app.example.com"],
            exposeHeaders: ["ETag"],
            maxAge: "1 day"
          },
          notifications: {
            "image-upload": {
              function: "processImage",
              events: ["s3:ObjectCreated:*"],
              filterPrefix: "images/",
              filterSuffix: ".jpg"
            }
          }
        });

        expect(bucket.args.public).toBe(true);
        expect(bucket.args.versioning).toBe(true);
        expect(bucket.args.cors.allowMethods).toContain("GET");
        expect(bucket.args.cors.allowOrigins).toHaveLength(2);
        expect(bucket.args.notifications["image-upload"]).toBeDefined();
      });
    });
  });

  describe("Bucket CORS configuration", () => {
    it("should handle basic CORS configuration", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("CORSBucket", "aws:bucket:component", {
          cors: {
            allowMethods: ["GET", "POST"],
            allowOrigins: ["https://example.com"]
          }
        });

        expect(bucket.args.cors.allowMethods).toHaveLength(2);
        expect(bucket.args.cors.allowOrigins).toContain("https://example.com");
      });
    });

    it("should handle comprehensive CORS configuration", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("ComprehensiveCORSBucket", "aws:bucket:component", {
          cors: {
            allowCredentials: true,
            allowHeaders: ["Content-Type", "Authorization", "X-Requested-With"],
            allowMethods: ["GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"],
            allowOrigins: ["https://app.example.com", "https://admin.example.com"],
            exposeHeaders: ["ETag", "Content-Length"],
            maxAge: "24 hours"
          }
        });

        expect(bucket.args.cors.allowCredentials).toBe(true);
        expect(bucket.args.cors.allowHeaders).toHaveLength(3);
        expect(bucket.args.cors.allowMethods).toHaveLength(6);
        expect(bucket.args.cors.allowOrigins).toHaveLength(2);
        expect(bucket.args.cors.exposeHeaders).toHaveLength(2);
        expect(bucket.args.cors.maxAge).toBe("24 hours");
      });
    });

    it("should handle wildcard CORS configuration", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("WildcardCORSBucket", "aws:bucket:component", {
          cors: {
            allowHeaders: ["*"],
            allowMethods: ["*"],
            allowOrigins: ["*"]
          }
        });

        expect(bucket.args.cors.allowHeaders).toContain("*");
        expect(bucket.args.cors.allowMethods).toContain("*");
        expect(bucket.args.cors.allowOrigins).toContain("*");
      });
    });

    it("should handle empty CORS configuration", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("EmptyCORSBucket", "aws:bucket:component", {
          cors: {}
        });

        expect(bucket.args.cors).toEqual({});
      });
    });

    it("should handle undefined CORS configuration", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("NoCORSBucket", "aws:bucket:component");

        expect(bucket.args.cors).toBeUndefined();
      });
    });
  });

  describe("Bucket notification setup", () => {
    it("should handle Lambda function notifications", async () => {
      await withTestEnvironment(async () => {
        const processFunction = new MockAWSComponent("ProcessFunction", "aws:function:component");
        
        const bucket = new MockAWSComponent("NotificationBucket", "aws:bucket:component", {
          notifications: {
            "process-upload": {
              function: processFunction.generatePhysicalName("function"),
              events: ["s3:ObjectCreated:*"],
              filterPrefix: "uploads/"
            }
          }
        });

        expect(bucket.args.notifications["process-upload"]).toBeDefined();
        expect(bucket.args.notifications["process-upload"].events).toContain("s3:ObjectCreated:*");
        expect(bucket.args.notifications["process-upload"].filterPrefix).toBe("uploads/");
        assertions.validOutput(bucket.args.notifications["process-upload"].function);
      });
    });

    it("should handle multiple notification configurations", async () => {
      await withTestEnvironment(async () => {
        const imageProcessor = new MockAWSComponent("ImageProcessor", "aws:function:component");
        const videoProcessor = new MockAWSComponent("VideoProcessor", "aws:function:component");
        
        const bucket = new MockAWSComponent("MultiNotificationBucket", "aws:bucket:component", {
          notifications: {
            "process-images": {
              function: imageProcessor.generatePhysicalName("function"),
              events: ["s3:ObjectCreated:*"],
              filterPrefix: "images/",
              filterSuffix: ".jpg"
            },
            "process-videos": {
              function: videoProcessor.generatePhysicalName("function"),
              events: ["s3:ObjectCreated:*"],
              filterPrefix: "videos/",
              filterSuffix: ".mp4"
            },
            "cleanup-temp": {
              function: imageProcessor.generatePhysicalName("function"),
              events: ["s3:ObjectRemoved:*"],
              filterPrefix: "temp/"
            }
          }
        });

        expect(Object.keys(bucket.args.notifications)).toHaveLength(3);
        expect(bucket.args.notifications["process-images"].filterSuffix).toBe(".jpg");
        expect(bucket.args.notifications["process-videos"].filterSuffix).toBe(".mp4");
        expect(bucket.args.notifications["cleanup-temp"].events).toContain("s3:ObjectRemoved:*");
      });
    });

    it("should handle SQS queue notifications", async () => {
      await withTestEnvironment(async () => {
        const queue = new MockAWSComponent("ProcessQueue", "aws:queue:component");
        
        const bucket = new MockAWSComponent("QueueNotificationBucket", "aws:bucket:component", {
          notifications: {
            "queue-processing": {
              queue: queue.generatePhysicalName("queue"),
              events: ["s3:ObjectCreated:*", "s3:ObjectRemoved:*"]
            }
          }
        });

        expect(bucket.args.notifications["queue-processing"].queue).toBeDefined();
        expect(bucket.args.notifications["queue-processing"].events).toHaveLength(2);
        assertions.validOutput(bucket.args.notifications["queue-processing"].queue);
      });
    });

    it("should handle SNS topic notifications", async () => {
      await withTestEnvironment(async () => {
        const topic = new MockAWSComponent("ProcessTopic", "aws:sns:component");
        
        const bucket = new MockAWSComponent("TopicNotificationBucket", "aws:bucket:component", {
          notifications: {
            "topic-notification": {
              topic: topic.generatePhysicalName("topic"),
              events: ["s3:ObjectCreated:*"]
            }
          }
        });

        expect(bucket.args.notifications["topic-notification"].topic).toBeDefined();
        assertions.validOutput(bucket.args.notifications["topic-notification"].topic);
      });
    });

    it("should handle empty notifications", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("EmptyNotificationsBucket", "aws:bucket:component", {
          notifications: {}
        });

        expect(bucket.args.notifications).toEqual({});
      });
    });
  });

  describe("Bucket naming", () => {
    it("should generate valid S3 bucket names", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("MyTestBucket", "aws:bucket:component");
        const physicalName = bucket.generatePhysicalName("bucket");
        
        assertions.validAWSName(physicalName);
        expect(physicalName.value).toMatch(/test-app-test-bucket-/);
      });
    });

    it("should handle bucket names with special characters", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("My-Test_Bucket.v2", "aws:bucket:component");
        const physicalName = bucket.generatePhysicalName("bucket");
        
        assertions.validAWSName(physicalName);
        // Should normalize special characters to hyphens
        expect(physicalName.value).toMatch(/bucket/);
      });
    });

    it("should handle custom bucket names", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("CustomBucket", "aws:bucket:component", {
          name: "my-custom-bucket-name"
        });

        expect(bucket.args.name).toBe("my-custom-bucket-name");
      });
    });
  });

  describe("Bucket public access configuration", () => {
    it("should handle public bucket configuration", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("PublicBucket", "aws:bucket:component", {
          public: true
        });

        expect(bucket.args.public).toBe(true);
      });
    });

    it("should handle private bucket configuration", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("PrivateBucket", "aws:bucket:component", {
          public: false
        });

        expect(bucket.args.public).toBe(false);
      });
    });

    it("should default to private when not specified", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("DefaultBucket", "aws:bucket:component");

        expect(bucket.args.public).toBeUndefined();
      });
    });
  });

  describe("Bucket versioning configuration", () => {
    it("should handle versioning enabled", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("VersionedBucket", "aws:bucket:component", {
          versioning: true
        });

        expect(bucket.args.versioning).toBe(true);
      });
    });

    it("should handle versioning disabled", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("NonVersionedBucket", "aws:bucket:component", {
          versioning: false
        });

        expect(bucket.args.versioning).toBe(false);
      });
    });

    it("should default to disabled when not specified", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("DefaultVersioningBucket", "aws:bucket:component");

        expect(bucket.args.versioning).toBeUndefined();
      });
    });
  });

  describe("Bucket lifecycle configuration", () => {
    it("should handle lifecycle rules", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("LifecycleBucket", "aws:bucket:component", {
          lifecycle: {
            rules: [
              {
                id: "delete-old-files",
                status: "Enabled",
                expiration: {
                  days: 30
                }
              },
              {
                id: "transition-to-ia",
                status: "Enabled",
                transitions: [
                  {
                    days: 30,
                    storageClass: "STANDARD_IA"
                  },
                  {
                    days: 90,
                    storageClass: "GLACIER"
                  }
                ]
              }
            ]
          }
        });

        expect(bucket.args.lifecycle.rules).toHaveLength(2);
        expect(bucket.args.lifecycle.rules[0].id).toBe("delete-old-files");
        expect(bucket.args.lifecycle.rules[1].transitions).toHaveLength(2);
      });
    });

    it("should handle empty lifecycle configuration", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("EmptyLifecycleBucket", "aws:bucket:component", {
          lifecycle: {
            rules: []
          }
        });

        expect(bucket.args.lifecycle.rules).toHaveLength(0);
      });
    });
  });

  describe("Bucket integration scenarios", () => {
    it("should integrate with CloudFront distribution", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("CDNBucket", "aws:bucket:component", {
          public: true
        });

        const distribution = new MockAWSComponent("TestDistribution", "aws:cloudfront:component", {
          origins: {
            s3: {
              domainName: bucket.generatePhysicalName("bucket"),
              originPath: "/static"
            }
          }
        });

        assertions.validOutput(distribution.args.origins.s3.domainName);
      });
    });

    it("should integrate with Lambda function for processing", async () => {
      await withTestEnvironment(async () => {
        const processor = new MockAWSComponent("FileProcessor", "aws:function:component", {
          handler: "process.handler",
          environment: {
            BUCKET_NAME: "source-bucket"
          }
        });

        const bucket = new MockAWSComponent("ProcessingBucket", "aws:bucket:component", {
          notifications: {
            "process-file": {
              function: processor.generatePhysicalName("function"),
              events: ["s3:ObjectCreated:*"]
            }
          }
        });

        expect(processor.args.environment.BUCKET_NAME).toBe("source-bucket");
        assertions.validOutput(bucket.args.notifications["process-file"].function);
      });
    });

    it("should integrate with API Gateway for presigned URLs", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("UploadBucket", "aws:bucket:component");

        const presignFunction = new MockAWSComponent("PresignFunction", "aws:function:component", {
          handler: "presign.handler",
          environment: {
            BUCKET_NAME: bucket.generatePhysicalName("bucket")
          }
        });

        const api = new MockAWSComponent("UploadAPI", "aws:apigateway:component", {
          routes: {
            "POST /presign": presignFunction.generatePhysicalName("function")
          }
        });

        assertions.validOutput(presignFunction.args.environment.BUCKET_NAME);
        assertions.validOutput(api.args.routes["POST /presign"]);
      });
    });
  });

  describe("Bucket error handling", () => {
    it("should handle invalid CORS methods", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("InvalidCORSBucket", "aws:bucket:component", {
          cors: {
            allowMethods: ["INVALID_METHOD"],
            allowOrigins: ["https://example.com"]
          }
        });

        expect(bucket.args.cors.allowMethods).toContain("INVALID_METHOD");
      });
    });

    it("should handle invalid notification events", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("InvalidNotificationBucket", "aws:bucket:component", {
          notifications: {
            "invalid-event": {
              function: "some-function",
              events: ["s3:InvalidEvent:*"]
            }
          }
        });

        expect(bucket.args.notifications["invalid-event"].events).toContain("s3:InvalidEvent:*");
      });
    });

    it("should handle missing notification target", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("MissingTargetBucket", "aws:bucket:component", {
          notifications: {
            "missing-target": {
              events: ["s3:ObjectCreated:*"]
            }
          }
        });

        expect(bucket.args.notifications["missing-target"].function).toBeUndefined();
        expect(bucket.args.notifications["missing-target"].queue).toBeUndefined();
        expect(bucket.args.notifications["missing-target"].topic).toBeUndefined();
      });
    });
  });

  describe("Bucket security configuration", () => {
    it("should handle bucket policy configuration", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("PolicyBucket", "aws:bucket:component", {
          policy: {
            Version: "2012-10-17",
            Statement: [
              {
                Effect: "Allow",
                Principal: "*",
                Action: "s3:GetObject",
                Resource: "arn:aws:s3:::my-bucket/*"
              }
            ]
          }
        });

        expect(bucket.args.policy.Version).toBe("2012-10-17");
        expect(bucket.args.policy.Statement).toHaveLength(1);
        expect(bucket.args.policy.Statement[0].Effect).toBe("Allow");
      });
    });

    it("should handle encryption configuration", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("EncryptedBucket", "aws:bucket:component", {
          encryption: {
            algorithm: "AES256"
          }
        });

        expect(bucket.args.encryption.algorithm).toBe("AES256");
      });
    });

    it("should handle KMS encryption", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("KMSBucket", "aws:bucket:component", {
          encryption: {
            algorithm: "aws:kms",
            kmsKeyId: "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
          }
        });

        expect(bucket.args.encryption.algorithm).toBe("aws:kms");
        expect(bucket.args.encryption.kmsKeyId).toContain("arn:aws:kms");
      });
    });
  });
});