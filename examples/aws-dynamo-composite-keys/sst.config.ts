/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## DynamoDB composite keys
 *
 * Create a DynamoDB table with multi-attribute composite keys in a global secondary index.
 */
export default $config({
  app(input) {
    return {
      name: "aws-dynamo-composite-keys",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const table = new sst.aws.Dynamo("MyTable", {
      fields: {
        userId: "string",
        noteId: "string",
        region: "string",
        category: "string",
        createdAt: "number",
      },
      primaryIndex: { hashKey: "userId", rangeKey: "noteId" },
      globalIndexes: {
        RegionCategoryIndex: {
          hashKey: ["region", "category"],
          rangeKey: "createdAt",
        },
      },
    });

    const creator = new sst.aws.Function("MyCreator", {
      handler: "creator.handler",
      link: [table],
      url: true,
    });

    const reader = new sst.aws.Function("MyReader", {
      handler: "reader.handler",
      link: [table],
      url: true,
    });

    return {
      creator: creator.url,
      reader: reader.url,
      table: table.name,
    };
  },
});
