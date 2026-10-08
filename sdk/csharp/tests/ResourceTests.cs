using System.Text.Json;
using System.Text.Json.Serialization;
using Xunit;

[assembly: CollectionBehavior(DisableTestParallelization = true)]

namespace SST.Tests;

/// <summary>
/// Goes through the static <see cref="Resource"/> facade, so these tests set real environment
/// variables and clean up after themselves. Parallelization is disabled for the assembly.
/// </summary>
public sealed class ResourceTests : IDisposable
{
    private readonly List<string> _variables = new();
    private readonly List<string> _files = new();

    public ResourceTests()
    {
        // Make sure nothing from the host environment leaks into the tests.
        foreach (var key in Environment.GetEnvironmentVariables().Keys.OfType<string>())
        {
            if (key.StartsWith("SST_", StringComparison.Ordinal) || key == "AWS_LAMBDA_FUNCTION_NAME")
            {
                Set(key, null);
            }
        }
        Set("SST_RESOURCE_App", """{"name":"my-app","stage":"dev"}""");
    }

    private void Set(string name, string? value)
    {
        _variables.Add(name);
        Environment.SetEnvironmentVariable(name, value);
        Resource.Reload();
    }

    public void Dispose()
    {
        foreach (var name in _variables.Distinct())
        {
            Environment.SetEnvironmentVariable(name, null);
        }
        foreach (var file in _files)
        {
            File.Delete(file);
        }
        Resource.Reload();
    }

    [Fact]
    public void GetReturnsRawElement()
    {
        Set("SST_RESOURCE_MyBucket", """{"name":"my-bucket"}""");

        var bucket = Resource.Get("MyBucket");

        Assert.Equal(JsonValueKind.Object, bucket.ValueKind);
        Assert.Equal("my-bucket", bucket.GetProperty("name").GetString());
        Assert.Equal("my-bucket", Resource.Get("MyBucket", "name").GetString());
    }

    [Fact]
    public void GetNavigatesNestedProperties()
    {
        Set("SST_RESOURCE_MyApi", """{"url":"https://api","auth":{"type":"iam","roles":["a","b"]}}""");

        Assert.Equal("iam", Resource.Get<string>("MyApi", "auth", "type"));
        Assert.Equal(new[] { "a", "b" }, Resource.Get<string[]>("MyApi", "auth", "roles"));
    }

    [Fact]
    public void GetDeserializesTypedValues()
    {
        Set("SST_RESOURCE_MyBucket", """{"name":"my-bucket","arn":"arn:aws:s3:::my-bucket"}""");
        Set("SST_RESOURCE_MyConfig", """{"retries":3,"enabled":true}""");

        var bucket = Resource.Get<Bucket>("MyBucket");
        Assert.Equal("my-bucket", bucket.Name);
        Assert.Equal("arn:aws:s3:::my-bucket", bucket.Arn);
        Assert.Equal(3, Resource.Get<int>("MyConfig", "retries"));
        Assert.True(Resource.Get<bool>("MyConfig", "enabled"));
    }

    [Fact]
    public void GetWithTypeInfoDeserializesWithoutReflection()
    {
        Set("SST_RESOURCE_MyBucket", """{"name":"my-bucket","arn":"arn"}""");

        var bucket = Resource.Get(TestJsonContext.Default.Bucket, "MyBucket");

        Assert.Equal("my-bucket", bucket.Name);
    }

    [Fact]
    public void SecretsAreReadLikeAnyResource()
    {
        Set("SST_RESOURCE_MySecret", """{"value":"s3cret"}""");

        Assert.Equal("s3cret", Resource.Get<string>("MySecret", "value"));
    }

    [Fact]
    public void AppExposesNameAndStage()
    {
        var app = Resource.App;

        Assert.Equal("my-app", app.Name);
        Assert.Equal("dev", app.Stage);
    }

    [Fact]
    public void TryGetAndAll()
    {
        Set("SST_RESOURCE_MyBucket", """{"name":"my-bucket"}""");

        Assert.True(Resource.TryGet("MyBucket", out var bucket));
        Assert.Equal("my-bucket", bucket.GetProperty("name").GetString());
        Assert.False(Resource.TryGet("Nope", out _));
        Assert.Equal(new[] { "App", "MyBucket" }, Resource.All().Keys.OrderBy(k => k));
    }

    [Fact]
    public void MissingResourceNamesTheFunction()
    {
        Set("AWS_LAMBDA_FUNCTION_NAME", "my-function");

        var error = Assert.Throws<ResourceNotFoundException>(() => Resource.Get("MyBucket"));

        Assert.Equal("\"MyBucket\" is not linked in your sst.config.ts to my-function", error.Message);
    }

    [Fact]
    public void MissingResourceWithoutFunctionName()
    {
        var error = Assert.Throws<ResourceNotFoundException>(() => Resource.Get<string>("MyBucket", "name"));

        Assert.Equal("\"MyBucket\" is not linked in your sst.config.ts", error.Message);
    }

    [Fact]
    public void MissingPropertyIsReported()
    {
        Set("SST_RESOURCE_MyBucket", """{"name":"my-bucket"}""");

        var error = Assert.Throws<ResourceNotFoundException>(() => Resource.Get("MyBucket", "arn"));

        Assert.Contains("has no property \"arn\"", error.Message);
    }

    [Fact]
    public void NullValueIsReported()
    {
        Set("SST_RESOURCE_MyBucket", """{"name":null}""");

        var error = Assert.Throws<ResourceException>(() => Resource.Get<string>("MyBucket", "name"));

        Assert.Contains("is null", error.Message);
    }

    [Fact]
    public void ExplainsWhenLinksAreNotActive()
    {
        Set("SST_RESOURCE_App", null);

        var error = Assert.Throws<ResourceNotFoundException>(() => Resource.Get("MyBucket"));

        Assert.StartsWith("It does not look like SST links are active", error.Message);
    }

    [Fact]
    public void ConsolidatedJsonCountsAsActiveLinks()
    {
        Set("SST_RESOURCE_App", null);
        Set("SST_RESOURCES_JSON", """{"MyBucket":{"name":"my-bucket"},"App":{"name":"app","stage":"prod"}}""");

        Assert.Equal("my-bucket", Resource.Get<string>("MyBucket", "name"));
        Assert.Equal("prod", Resource.App.Stage);
        var error = Assert.Throws<ResourceNotFoundException>(() => Resource.Get("Other"));
        Assert.Equal("\"Other\" is not linked in your sst.config.ts", error.Message);
    }

    [Fact]
    public void ReadsEncryptedKeyFile()
    {
        var path = Path.Combine(Path.GetTempPath(), $"sst-{Guid.NewGuid():N}.enc");
        File.WriteAllBytes(path, Crypto.Encrypt("""{"MyBucket":{"name":"encrypted-bucket"}}"""));
        _files.Add(path);
        Set("SST_KEY", Crypto.KeyBase64);
        Set("SST_KEY_FILE", path);

        Assert.Equal("encrypted-bucket", Resource.Get<string>("MyBucket", "name"));
    }

    [Fact]
    public void ReadsEncryptedKeyFileRelativeToApplicationDirectory()
    {
        var name = $"sst-{Guid.NewGuid():N}.enc";
        var path = Path.Combine(AppContext.BaseDirectory, name);
        File.WriteAllBytes(path, Crypto.Encrypt("""{"MyQueue":{"url":"https://sqs"}}"""));
        _files.Add(path);
        Set("SST_KEY", Crypto.KeyBase64);
        Set("SST_KEY_FILE", name);

        Assert.Equal("https://sqs", Resource.Get<string>("MyQueue", "url"));
    }

    [Fact]
    public void CachesUntilReloaded()
    {
        Set("SST_RESOURCE_MyBucket", """{"name":"first"}""");
        Assert.Equal("first", Resource.Get<string>("MyBucket", "name"));

        Environment.SetEnvironmentVariable("SST_RESOURCE_MyBucket", """{"name":"second"}""");
        Assert.Equal("first", Resource.Get<string>("MyBucket", "name"));

        Resource.Reload();
        Assert.Equal("second", Resource.Get<string>("MyBucket", "name"));
    }

    public sealed record Bucket(string Name, string Arn);
}

[JsonSourceGenerationOptions(PropertyNamingPolicy = JsonKnownNamingPolicy.CamelCase)]
[JsonSerializable(typeof(ResourceTests.Bucket))]
internal partial class TestJsonContext : JsonSerializerContext
{
}
