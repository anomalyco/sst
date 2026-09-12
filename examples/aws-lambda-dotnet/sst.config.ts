/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda .NET
 *
 * This example shows how to use the [`dotnet8`](https://dotnet.microsoft.com/) runtime in
 * your Lambda functions.
 *
 * Our .NET project is in the `src` directory and we point to it in our function.
 *
 * ```ts title="sst.config.ts" {5}
 * new sst.aws.Function("MyFunction", {
 *   url: true,
 *   runtime: "dotnet8",
 *   link: [bucket],
 *   handler: "./src",
 * });
 * ```
 *
 * The project is an executable assembly that bootstraps itself with
 * `Amazon.Lambda.RuntimeSupport`. This is what lets `sst dev` run it
 * [_Live_](/docs/live).
 *
 * ```csharp title="src/Function.cs"
 * await LambdaBootstrapBuilder.Create(handler, new DefaultLambdaJsonSerializer())
 *   .Build()
 *   .RunAsync();
 * ```
 *
 * We are also linking it to an S3 bucket. We can reference the bucket in our function
 * with the SST .NET SDK.
 *
 * ```csharp title="src/Function.cs" {1}
 * using SST;
 *
 * var bucket = Resource.Get<string>("MyBucket", "name");
 * ```
 *
 * The SDK lives in `sdk/csharp` and the example references it as a project.
 *
 * ```xml title="src/MyFunction.csproj"
 * <ProjectReference Include="../../../sdk/csharp/src/SST.Sdk.csproj" />
 * ```
 */
export default $config({
  app(input) {
    return {
      name: "aws-lambda-dotnet",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const bucket = new sst.aws.Bucket("MyBucket");

    new sst.aws.Function("MyFunction", {
      url: true,
      runtime: "dotnet8",
      link: [bucket],
      handler: "./src",
    });
  },
});
