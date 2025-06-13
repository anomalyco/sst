import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { withTestEnvironment, createConsoleSpy, assertions, generators } from "../utils/test-helpers";
import { MockAWSComponent } from "../utils/mock-sst";

/**
 * Test Dynamo component creation and configuration
 * Tests DynamoDB table configuration and stream handling
 */
describe("Dynamo Component", () => {
  let consoleSpy: ReturnType<typeof createConsoleSpy>;

  beforeEach(() => {
    consoleSpy = createConsoleSpy('warn');
  });

  afterEach(() => {
    consoleSpy.restore();
  });

  describe("Dynamo table creation", () => {
    it("should create Dynamo table with basic configuration", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("TestTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          }
        });

        expect(table).toBeDefined();
        expect(table.originalName).toBe("TestTable"); expect(table.name).toMatch(/test-app-test-testtable-/);
        expect(table.args.fields.id).toBe("string");
        expect(table.args.primaryIndex.hashKey).toBe("id");
      });
    });

    it("should create Dynamo table with composite primary key", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("CompositeTable", "aws:dynamo:component", {
          fields: {
            userId: "string",
            timestamp: "number",
            data: "string"
          },
          primaryIndex: {
            hashKey: "userId",
            rangeKey: "timestamp"
          }
        });

        expect(table.args.fields.userId).toBe("string");
        expect(table.args.fields.timestamp).toBe("number");
        expect(table.args.primaryIndex.hashKey).toBe("userId");
        expect(table.args.primaryIndex.rangeKey).toBe("timestamp");
      });
    });

    it("should create Dynamo table with advanced configuration", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("AdvancedTable", "aws:dynamo:component", {
          fields: {
            id: "string",
            gsi1pk: "string",
            gsi1sk: "string",
            data: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          globalIndexes: {
            "GSI1": {
              hashKey: "gsi1pk",
              rangeKey: "gsi1sk"
            }
          },
          stream: "new-and-old-images",
          pointInTimeRecovery: true,
          deletionProtection: true,
          billing: {
            mode: "provisioned",
            readCapacity: 5,
            writeCapacity: 5
          }
        });

        expect(table.args.globalIndexes.GSI1.hashKey).toBe("gsi1pk");
        expect(table.args.stream).toBe("new-and-old-images");
        expect(table.args.pointInTimeRecovery).toBe(true);
        expect(table.args.deletionProtection).toBe(true);
        expect(table.args.billing.mode).toBe("provisioned");
      });
    });
  });

  describe("Dynamo field types", () => {
    it("should handle string fields", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("StringTable", "aws:dynamo:component", {
          fields: {
            id: "string",
            name: "string",
            email: "string"
          },
          primaryIndex: {
            hashKey: "id"
          }
        });

        expect(table.args.fields.id).toBe("string");
        expect(table.args.fields.name).toBe("string");
        expect(table.args.fields.email).toBe("string");
      });
    });

    it("should handle number fields", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("NumberTable", "aws:dynamo:component", {
          fields: {
            id: "string",
            count: "number",
            score: "number",
            timestamp: "number"
          },
          primaryIndex: {
            hashKey: "id"
          }
        });

        expect(table.args.fields.count).toBe("number");
        expect(table.args.fields.score).toBe("number");
        expect(table.args.fields.timestamp).toBe("number");
      });
    });

    it("should handle binary fields", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("BinaryTable", "aws:dynamo:component", {
          fields: {
            id: "string",
            data: "binary",
            image: "binary"
          },
          primaryIndex: {
            hashKey: "id"
          }
        });

        expect(table.args.fields.data).toBe("binary");
        expect(table.args.fields.image).toBe("binary");
      });
    });

    it("should handle mixed field types", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("MixedTable", "aws:dynamo:component", {
          fields: {
            id: "string",
            count: "number",
            data: "binary",
            active: "string" // DynamoDB doesn't have boolean type, stored as string
          },
          primaryIndex: {
            hashKey: "id"
          }
        });

        expect(table.args.fields.id).toBe("string");
        expect(table.args.fields.count).toBe("number");
        expect(table.args.fields.data).toBe("binary");
        expect(table.args.fields.active).toBe("string");
      });
    });
  });

  describe("Dynamo global secondary indexes", () => {
    it("should handle single GSI", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("GSITable", "aws:dynamo:component", {
          fields: {
            id: "string",
            status: "string",
            timestamp: "number"
          },
          primaryIndex: {
            hashKey: "id"
          },
          globalIndexes: {
            "StatusIndex": {
              hashKey: "status",
              rangeKey: "timestamp"
            }
          }
        });

        expect(table.args.globalIndexes.StatusIndex.hashKey).toBe("status");
        expect(table.args.globalIndexes.StatusIndex.rangeKey).toBe("timestamp");
      });
    });

    it("should handle multiple GSIs", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("MultiGSITable", "aws:dynamo:component", {
          fields: {
            id: "string",
            userId: "string",
            status: "string",
            category: "string",
            timestamp: "number"
          },
          primaryIndex: {
            hashKey: "id"
          },
          globalIndexes: {
            "UserIndex": {
              hashKey: "userId",
              rangeKey: "timestamp"
            },
            "StatusIndex": {
              hashKey: "status"
            },
            "CategoryIndex": {
              hashKey: "category",
              rangeKey: "timestamp"
            }
          }
        });

        expect(Object.keys(table.args.globalIndexes)).toHaveLength(3);
        expect(table.args.globalIndexes.UserIndex.hashKey).toBe("userId");
        expect(table.args.globalIndexes.StatusIndex.hashKey).toBe("status");
        expect(table.args.globalIndexes.CategoryIndex.hashKey).toBe("category");
      });
    });

    it("should handle GSI without range key", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("SimpleGSITable", "aws:dynamo:component", {
          fields: {
            id: "string",
            status: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          globalIndexes: {
            "StatusIndex": {
              hashKey: "status"
            }
          }
        });

        expect(table.args.globalIndexes.StatusIndex.hashKey).toBe("status");
        expect(table.args.globalIndexes.StatusIndex.rangeKey).toBeUndefined();
      });
    });
  });

  describe("Dynamo local secondary indexes", () => {
    it("should handle local secondary indexes", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("LSITable", "aws:dynamo:component", {
          fields: {
            userId: "string",
            timestamp: "number",
            status: "string",
            category: "string"
          },
          primaryIndex: {
            hashKey: "userId",
            rangeKey: "timestamp"
          },
          localIndexes: {
            "StatusIndex": {
              rangeKey: "status"
            },
            "CategoryIndex": {
              rangeKey: "category"
            }
          }
        });

        expect(table.args.localIndexes.StatusIndex.rangeKey).toBe("status");
        expect(table.args.localIndexes.CategoryIndex.rangeKey).toBe("category");
      });
    });
  });

  describe("Dynamo streams", () => {
    it("should handle keys-only stream", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("KeysOnlyStreamTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          stream: "keys-only"
        });

        expect(table.args.stream).toBe("keys-only");
      });
    });

    it("should handle new-image stream", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("NewImageStreamTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          stream: "new-image"
        });

        expect(table.args.stream).toBe("new-image");
      });
    });

    it("should handle old-image stream", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("OldImageStreamTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          stream: "old-image"
        });

        expect(table.args.stream).toBe("old-image");
      });
    });

    it("should handle new-and-old-images stream", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("FullStreamTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          stream: "new-and-old-images"
        });

        expect(table.args.stream).toBe("new-and-old-images");
      });
    });

    it("should handle disabled stream", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("NoStreamTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          stream: false
        });

        expect(table.args.stream).toBe(false);
      });
    });
  });

  describe("Dynamo billing configuration", () => {
    it("should handle on-demand billing", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("OnDemandTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          billing: {
            mode: "on-demand"
          }
        });

        expect(table.args.billing.mode).toBe("on-demand");
      });
    });

    it("should handle provisioned billing", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("ProvisionedTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          billing: {
            mode: "provisioned",
            readCapacity: 10,
            writeCapacity: 5
          }
        });

        expect(table.args.billing.mode).toBe("provisioned");
        expect(table.args.billing.readCapacity).toBe(10);
        expect(table.args.billing.writeCapacity).toBe(5);
      });
    });

    it("should handle auto-scaling configuration", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("AutoScalingTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          billing: {
            mode: "provisioned",
            readCapacity: 5,
            writeCapacity: 5,
            autoScaling: {
              read: {
                min: 5,
                max: 100,
                targetUtilization: 70
              },
              write: {
                min: 5,
                max: 100,
                targetUtilization: 70
              }
            }
          }
        });

        expect(table.args.billing.autoScaling.read.min).toBe(5);
        expect(table.args.billing.autoScaling.read.max).toBe(100);
        expect(table.args.billing.autoScaling.write.targetUtilization).toBe(70);
      });
    });
  });

  describe("Dynamo naming", () => {
    it("should generate valid DynamoDB table names", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("MyTestTable", "aws:dynamo:component");
        const physicalName = table.generatePhysicalName("table");
        
        assertions.validAWSName(physicalName);
        expect(physicalName.value).toMatch(/test-app-test-table-/);
      });
    });

    it("should handle table names with special characters", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("My-Test_Table.v2", "aws:dynamo:component");
        const physicalName = table.generatePhysicalName("table");
        
        assertions.validAWSName(physicalName);
        expect(physicalName.value).toMatch(/table/);
      });
    });
  });

  describe("Dynamo integration scenarios", () => {
    it("should integrate with Lambda function for stream processing", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("StreamTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          stream: "new-and-old-images"
        });

        const processor = new MockAWSComponent("StreamProcessor", "aws:function:component", {
          handler: "stream.handler",
          environment: {
            TABLE_NAME: table.generatePhysicalName("table")
          }
        });

        const streamSubscriber = new MockAWSComponent("StreamSubscriber", "aws:dynamo:stream:subscriber", {
          table: table.generatePhysicalName("table"),
          function: processor.generatePhysicalName("function"),
          batchSize: 10,
          startingPosition: "LATEST"
        });

        assertions.validOutput(processor.args.environment.TABLE_NAME);
        assertions.validOutput(streamSubscriber.args.table);
        assertions.validOutput(streamSubscriber.args.function);
      });
    });

    it("should integrate with API Gateway for CRUD operations", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("APITable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          }
        });

        const crudFunction = new MockAWSComponent("CRUDFunction", "aws:function:component", {
          handler: "crud.handler",
          environment: {
            TABLE_NAME: table.generatePhysicalName("table")
          }
        });

        const api = new MockAWSComponent("CRUDAPI", "aws:apigateway:component", {
          routes: {
            "GET /items": crudFunction.generatePhysicalName("function"),
            "POST /items": crudFunction.generatePhysicalName("function"),
            "PUT /items/{id}": crudFunction.generatePhysicalName("function"),
            "DELETE /items/{id}": crudFunction.generatePhysicalName("function")
          }
        });

        assertions.validOutput(crudFunction.args.environment.TABLE_NAME);
        assertions.validOutput(api.args.routes["GET /items"]);
      });
    });

    it("should integrate with EventBridge for change notifications", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("EventTable", "aws:dynamo:component", {
          fields: {
            id: "string",
            status: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          stream: "new-and-old-images"
        });

        const eventProcessor = new MockAWSComponent("EventProcessor", "aws:function:component", {
          handler: "events.handler"
        });

        const eventBridge = new MockAWSComponent("TableEventBridge", "aws:eventbridge:component", {
          rules: {
            "table-changes": {
              source: ["aws.dynamodb"],
              detailType: ["DynamoDB Stream Record"],
              target: eventProcessor.generatePhysicalName("function")
            }
          }
        });

        assertions.validOutput(eventBridge.args.rules["table-changes"].target);
      });
    });
  });

  describe("Dynamo error handling", () => {
    it("should handle missing primary key", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("NoPrimaryKeyTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          }
          // Missing primaryIndex
        });

        expect(table.args.primaryIndex).toBeUndefined();
      });
    });

    it("should handle invalid field types", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("InvalidFieldTable", "aws:dynamo:component", {
          fields: {
            id: "invalid-type"
          },
          primaryIndex: {
            hashKey: "id"
          }
        });

        expect(table.args.fields.id).toBe("invalid-type");
      });
    });

    it("should handle invalid stream type", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("InvalidStreamTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          stream: "invalid-stream-type"
        });

        expect(table.args.stream).toBe("invalid-stream-type");
      });
    });

    it("should handle invalid billing mode", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("InvalidBillingTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          billing: {
            mode: "invalid-mode"
          }
        });

        expect(table.args.billing.mode).toBe("invalid-mode");
      });
    });
  });

  describe("Dynamo backup and recovery", () => {
    it("should handle point-in-time recovery", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("PITRTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          pointInTimeRecovery: true
        });

        expect(table.args.pointInTimeRecovery).toBe(true);
      });
    });

    it("should handle deletion protection", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("ProtectedTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          deletionProtection: true
        });

        expect(table.args.deletionProtection).toBe(true);
      });
    });

    it("should handle backup configuration", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("BackupTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          backup: {
            enabled: true,
            retentionPeriod: "30 days"
          }
        });

        expect(table.args.backup.enabled).toBe(true);
        expect(table.args.backup.retentionPeriod).toBe("30 days");
      });
    });
  });

  describe("Dynamo encryption", () => {
    it("should handle default encryption", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("DefaultEncryptionTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          encryption: {
            type: "default"
          }
        });

        expect(table.args.encryption.type).toBe("default");
      });
    });

    it("should handle KMS encryption", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("KMSEncryptionTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: {
            hashKey: "id"
          },
          encryption: {
            type: "kms",
            kmsKeyId: "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
          }
        });

        expect(table.args.encryption.type).toBe("kms");
        expect(table.args.encryption.kmsKeyId).toContain("arn:aws:kms");
      });
    });
  });
});