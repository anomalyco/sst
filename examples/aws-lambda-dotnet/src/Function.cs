using System.Text.Json;
using Amazon.Lambda.Core;
using Amazon.Lambda.RuntimeSupport;
using Amazon.Lambda.Serialization.SystemTextJson;
using SST;

// The function receives the raw event and returns the name of the linked bucket.
var handler = (JsonElement request, ILambdaContext context) =>
{
    context.Logger.LogInformation($"Handling request {context.AwsRequestId} in {Resource.App.Name}/{Resource.App.Stage}");
    return Task.FromResult(Resource.Get<string>("MyBucket", "name"));
};

// Executable assemblies talk to the Lambda Runtime API themselves. That is what
// lets `sst dev` run the same binary locally.
await LambdaBootstrapBuilder.Create(handler, new DefaultLambdaJsonSerializer())
    .Build()
    .RunAsync();
