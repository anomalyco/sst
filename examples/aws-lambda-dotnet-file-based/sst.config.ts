/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda .NET file-based app with DynamoDB
 *
 * A .NET Lambda function written as a single C# file, using
 * [file-based apps](https://learn.microsoft.com/dotnet/core/sdk/file-based-apps)
 * from the .NET 10 SDK, that stores notes in a linked DynamoDB table. There is no
 * `.csproj`; packages and build settings are declared with `#:` directives at the
 * top of the file.
 *
 * ```csharp title="src/Api.cs"
 * #:property PublishAot=false
 * #:package Amazon.Lambda.RuntimeSupport@2.2.0
 * #:package AWSSDK.DynamoDBv2@4.0.103.7
 * ```
 *
 * We link the table to the function and point the function at the file. The Lambda
 * handler is the file name, `Api`.
 *
 * ```ts title="sst.config.ts" {2,4}
 * new sst.aws.Function("DotnetFunction", {
 *   handler: "./src/Api.cs",
 *   runtime: "dotnet10",
 *   link: [table],
 *   url: true,
 * });
 * ```
 *
 * The function reads the table name from the link with the SST .NET SDK and uses it
 * with the DynamoDB client. The link also grants the function permissions on the table.
 *
 * ```csharp title="src/Api.cs" {1}
 * var tableName = Resource.Get<string>("DotnetTable", "name");
 *
 * await dynamo.PutItemAsync(new PutItemRequest { TableName = tableName, Item = ... });
 * ```
 *
 * Once deployed, exercise it through the function URL.
 *
 * ```bash
 * curl "$URL/"                                   # reports the linked table name
 * curl -X POST "$URL/notes" -d '{"userId":"u1","content":"hello"}'
 * curl "$URL/notes?userId=u1"                    # lists u1's notes
 * curl "$URL/notes?userId=u1&noteId=<noteId>"    # reads one
 * curl -X DELETE "$URL/notes?userId=u1&noteId=<noteId>"
 * ```
 *
 * The app bootstraps itself with `Amazon.Lambda.RuntimeSupport`, so `sst dev` can run
 * the same file locally and still talk to the real table.
 *
 * :::note
 * File-based apps need the .NET 10 SDK or later on the machine running `sst deploy`
 * or `sst dev`.
 * :::
 */
export default $config({
  app(input) {
    return {
      name: "aws-lambda-dotnet-file-based",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const table = new sst.aws.Dynamo("DotnetTable", {
      fields: {
        userId: "string",
        noteId: "string",
      },
      primaryIndex: { hashKey: "userId", rangeKey: "noteId" },
    });

    const fn = new sst.aws.Function("DotnetFunction", {
      handler: "./src/Api.cs",
      runtime: "dotnet10",
      link: [table],
      url: true,
    });

    return { url: fn.url };
  },
});
