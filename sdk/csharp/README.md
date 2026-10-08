# SST .NET SDK

The .NET SDK for [SST](https://sst.dev) lets you access linked resources, like buckets, tables and secrets, in your .NET Lambda functions and container applications.

## Installation

The package is not published to NuGet yet. Until it is, reference the project from this repo.

```xml
<ItemGroup>
  <ProjectReference Include="../../sdk/csharp/src/SST.Sdk.csproj" />
</ItemGroup>
```

Once published it will be installed like any other package.

```bash
dotnet add package SST.Sdk
```

The SDK targets `net8.0`, has no dependencies, and works with the `dotnet8` and `dotnet10` Lambda runtimes and with Native AOT.

## Usage

Link resources to your function in `sst.config.ts`.

```ts
const bucket = new sst.aws.Bucket("MyBucket");
const secret = new sst.Secret("StripeKey");

new sst.aws.Function("MyFunction", {
  runtime: "dotnet8",
  handler: "./src",
  link: [bucket, secret],
});
```

Then read them with `Resource` in your code.

```csharp
using SST;

// A property of a linked resource
var bucketName = Resource.Get<string>("MyBucket", "name");

// Secrets are resources with a `value` property
var stripeKey = Resource.Get<string>("StripeKey", "value");

// Info about the current app
Console.WriteLine($"{Resource.App.Name} / {Resource.App.Stage}");
```

Each resource is the JSON object described in the _Link_ section of that component's reference docs. You can read it as a whole, as a typed object, or as a raw `JsonElement`.

```csharp
public sealed record Bucket(string Name, string Arn);

var bucket = Resource.Get<Bucket>("MyBucket");   // camelCase JSON maps to PascalCase properties
var raw = Resource.Get("MyBucket");              // System.Text.Json.JsonElement
var nested = Resource.Get<string>("MyApi", "auth", "type");

if (Resource.TryGet("Optional", out var element)) { /* linked */ }

foreach (var (name, value) in Resource.All()) { /* everything linked */ }
```

A resource that is not linked throws `ResourceNotFoundException`, with a message that names the resource and, on Lambda, the function. If the process was not started through SST at all, the message says so and suggests `sst dev -- <command>`.

### Native AOT

`Resource.Get<T>(name, ...)` uses reflection based deserialization. For Native AOT, pass source generated metadata instead.

```csharp
[JsonSerializable(typeof(Bucket))]
internal partial class AppJsonContext : JsonSerializerContext { }

var bucket = Resource.Get(AppJsonContext.Default.Bucket, "MyBucket");
```

`Resource.Get(name, ...)` returning `JsonElement`, `Resource.App`, `TryGet` and `All` are AOT safe as they are.

## How it works

SST passes linked resources to your code in two ways, and the SDK reads both.

- On `sst deploy`, links are written to an encrypted `resource.enc` file next to your code. `SST_KEY_FILE` names the file and `SST_KEY` holds the AES-256-GCM key.
- On `sst dev`, and for some components, links arrive as `SST_RESOURCE_{Name}` environment variables holding JSON. On Windows the consolidated `SST_RESOURCES_JSON` variable is used instead.

Resources are read once, on first access, and cached for the life of the process.

## Development

```bash
cd sdk/csharp
dotnet test
```

The tests need the .NET 8 SDK or newer and no AWS access.

## Links

- [SST Documentation](https://sst.dev/docs/)
- [SDK Reference](https://sst.dev/docs/reference/sdk/#net)
- [.NET Example](https://github.com/anomalyco/sst/tree/dev/examples/aws-lambda-dotnet)
- [GitHub](https://github.com/anomalyco/sst)
