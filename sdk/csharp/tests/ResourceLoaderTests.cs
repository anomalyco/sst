using System.Text.Json;
using Xunit;

namespace SST.Tests;

/// <summary>
/// Exercises the loader with plain dictionaries so nothing here touches the process environment.
/// </summary>
public class ResourceLoaderTests
{
    private static readonly Func<string, byte[]?> NoFiles = _ => null;

    private static Dictionary<string, JsonElement> Load(Dictionary<string, string> env, Func<string, byte[]?>? readFile = null) =>
        ResourceLoader.Load(env, readFile ?? NoFiles);

    [Fact]
    public void LoadsIndividualResourceVariables()
    {
        var resources = Load(new()
        {
            ["SST_RESOURCE_MyBucket"] = """{"name":"my-bucket","type":"aws.s3.Bucket"}""",
            ["SST_RESOURCE_MyTable"] = """{"name":"my-table"}""",
        });

        Assert.Equal(2, resources.Count);
        Assert.Equal("my-bucket", resources["MyBucket"].GetProperty("name").GetString());
        Assert.Equal("my-table", resources["MyTable"].GetProperty("name").GetString());
    }

    [Fact]
    public void IgnoresUnrelatedEmptyAndInvalidVariables()
    {
        var resources = Load(new()
        {
            ["PATH"] = "/usr/bin",
            ["NOT_SST"] = """{"name":"nope"}""",
            ["SST_RESOURCE_Empty"] = "",
            ["SST_RESOURCE_Broken"] = "not-json",
            ["SST_RESOURCE_Ok"] = """{"value":"yes"}""",
        });

        Assert.Single(resources);
        Assert.Equal("yes", resources["Ok"].GetProperty("value").GetString());
    }

    [Fact]
    public void LoadsConsolidatedJson()
    {
        var resources = Load(new()
        {
            ["SST_RESOURCES_JSON"] = """{"MyBucket":{"name":"my-bucket"},"App":{"name":"app","stage":"dev"}}""",
        });

        Assert.Equal(2, resources.Count);
        Assert.Equal("my-bucket", resources["MyBucket"].GetProperty("name").GetString());
        Assert.Equal("dev", resources["App"].GetProperty("stage").GetString());
    }

    [Fact]
    public void ConsolidatedJsonMergesWithAndOverridesIndividualVariables()
    {
        var resources = Load(new()
        {
            ["SST_RESOURCE_MyTable"] = """{"name":"my-table"}""",
            ["SST_RESOURCE_MyBucket"] = """{"name":"from-env-var"}""",
            ["SST_RESOURCES_JSON"] = """{"MyBucket":{"name":"from-json"}}""",
        });

        Assert.Equal(2, resources.Count);
        Assert.Equal("my-table", resources["MyTable"].GetProperty("name").GetString());
        Assert.Equal("from-json", resources["MyBucket"].GetProperty("name").GetString());
    }

    [Fact]
    public void InvalidConsolidatedJsonIsIgnored()
    {
        var resources = Load(new()
        {
            ["SST_RESOURCE_MyTable"] = """{"name":"my-table"}""",
            ["SST_RESOURCES_JSON"] = "not-json",
        });

        Assert.Single(resources);
        Assert.True(resources.ContainsKey("MyTable"));
    }

    [Fact]
    public void LoadsEncryptedFile()
    {
        var file = Crypto.Encrypt("""{"EncryptedBucket":{"name":"encrypted-bucket"},"EncryptedQueue":{"url":"https://sqs"}}""");
        var resources = Load(new()
        {
            ["SST_KEY"] = Crypto.KeyBase64,
            ["SST_KEY_FILE"] = "resource.enc",
        }, path => path == "resource.enc" ? file : null);

        Assert.Equal(2, resources.Count);
        Assert.Equal("encrypted-bucket", resources["EncryptedBucket"].GetProperty("name").GetString());
        Assert.Equal("https://sqs", resources["EncryptedQueue"].GetProperty("url").GetString());
    }

    [Fact]
    public void EnvironmentOverridesEncryptedFile()
    {
        var file = Crypto.Encrypt("""{"Shared":{"name":"from-file"},"OnlyFile":{"name":"file"}}""");
        var resources = Load(new()
        {
            ["SST_KEY"] = Crypto.KeyBase64,
            ["SST_KEY_FILE"] = "resource.enc",
            ["SST_RESOURCE_Shared"] = """{"name":"from-env"}""",
        }, _ => file);

        Assert.Equal(2, resources.Count);
        Assert.Equal("from-env", resources["Shared"].GetProperty("name").GetString());
        Assert.Equal("file", resources["OnlyFile"].GetProperty("name").GetString());
    }

    [Fact]
    public void MissingEncryptedFileIsSkipped()
    {
        var resources = Load(new()
        {
            ["SST_KEY"] = Crypto.KeyBase64,
            ["SST_KEY_FILE"] = "resource.enc",
            ["SST_RESOURCE_MyBucket"] = """{"name":"my-bucket"}""",
        });

        Assert.Single(resources);
        Assert.True(resources.ContainsKey("MyBucket"));
    }

    [Fact]
    public void OnlyKeyWithoutFileIsSkipped()
    {
        var resources = Load(new()
        {
            ["SST_KEY"] = Crypto.KeyBase64,
        }, _ => throw new InvalidOperationException("should not read"));

        Assert.Empty(resources);
    }

    [Fact]
    public void WrongKeyThrowsResourceException()
    {
        var file = Crypto.Encrypt("""{"MyBucket":{"name":"my-bucket"}}""");
        var wrongKey = Convert.ToBase64String(new byte[32]);

        var error = Assert.Throws<ResourceException>(() => Load(new()
        {
            ["SST_KEY"] = wrongKey,
            ["SST_KEY_FILE"] = "resource.enc",
        }, _ => file));

        Assert.Contains("resource.enc", error.Message);
        Assert.NotNull(error.InnerException);
    }

    [Fact]
    public void MalformedKeyThrowsResourceException()
    {
        var file = Crypto.Encrypt("""{"MyBucket":{"name":"my-bucket"}}""");

        Assert.Throws<ResourceException>(() => Load(new()
        {
            ["SST_KEY"] = "%%not-base64%%",
            ["SST_KEY_FILE"] = "resource.enc",
        }, _ => file));
    }

    [Fact]
    public void DecryptRoundTrips()
    {
        var plaintext = ResourceLoader.Decrypt(Crypto.Key, Crypto.Encrypt("""{"a":1}"""));

        Assert.Equal("""{"a":1}""", System.Text.Encoding.UTF8.GetString(plaintext));
    }

    [Fact]
    public void EmptyEnvironmentYieldsNoResources()
    {
        Assert.Empty(Load(new()));
    }
}
