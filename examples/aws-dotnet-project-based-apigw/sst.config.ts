/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS ApiGatewayV2 .NET
 *
 * Runs an ASP.NET Core Minimal API behind API Gateway V2 using
 * [`Amazon.Lambda.AspNetCoreServer.Hosting`](https://github.com/aws/aws-lambda-dotnet/tree/master/Libraries/src/Amazon.Lambda.AspNetCoreServer.Hosting).
 *
 * :::tip
 * One line swaps Kestrel for the Lambda runtime when the app runs on Lambda. Locally,
 * `dotnet run` still serves the same app on Kestrel.
 * :::
 *
 * So you write your API as you normally would.
 *
 * ```csharp title="src/Program.cs" {3}
 * var builder = WebApplication.CreateBuilder(args);
 *
 * builder.Services.AddAWSLambdaHosting(LambdaEventSource.HttpApi);
 *
 * var app = builder.Build();
 *
 * app.MapGet("/hello", () => Results.Json(new { message = "hello world" }));
 *
 * app.Run();
 * ```
 *
 * And point the function at the project. The Lambda handler is the assembly name.
 *
 * ```ts title="sst.config.ts" {3,4}
 * api.route("$default", {
 *   handler: "./src",
 *   runtime: "dotnet10",
 * });
 * ```
 *
 * The project is a regular `Microsoft.NET.Sdk.Web` project, so `sst dev` can build
 * and run it [_Live_](/docs/live) as well.
 */
export default $config({
  app(input) {
    return {
      name: "aws-dotnet-project-based-apigw",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const api = new sst.aws.ApiGatewayV2("DotnetApi");

    api.route("$default", {
      handler: "./src",
      runtime: "dotnet10",
    });
  },
});
