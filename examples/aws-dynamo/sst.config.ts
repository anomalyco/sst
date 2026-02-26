/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## DynamoDB streams
 *
 * Create a DynamoDB table, enable streams, and subscribe to it with a function.
 */
export default $config({
  app(input) {
    return {
      name: "aws-dynamo",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const table = new sst.aws.Dynamo("MyTable", {
      fields: {
        id: "string",
      },
      primaryIndex: { hashKey: "id" },
      stream: "new-and-old-images",
    });
    table.subscribe("MySubscriber", "subscriber.handler", {
      filters: [
        {
          dynamodb: {
            NewImage: {
              message: {
                S: ["Hello"],
              },
            },
          },
        },
      ],
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
