using System.Security.Cryptography;
using System.Text.Json;

namespace SST;

/// <summary>
/// Reads linked resources the way SST hands them to the runtime. In order of precedence,
/// lowest first:
/// <list type="number">
/// <item>The AES-256-GCM encrypted file named by <c>SST_KEY_FILE</c>, keyed by <c>SST_KEY</c>. This is
/// how deployed functions receive links.</item>
/// <item>Individual <c>SST_RESOURCE_{Name}</c> environment variables holding JSON.</item>
/// <item>The consolidated <c>SST_RESOURCES_JSON</c> variable, used on Windows where variable
/// names lose their casing.</item>
/// </list>
/// </summary>
internal static class ResourceLoader
{
    internal const string Prefix = "SST_RESOURCE_";
    private const int TagSize = 16;
    private const int NonceSize = 12;

    internal static Dictionary<string, JsonElement> Load(
        IReadOnlyDictionary<string, string> environment,
        Func<string, byte[]?> readFile)
    {
        var result = new Dictionary<string, JsonElement>(StringComparer.Ordinal);
        LoadEncryptedFile(result, environment, readFile);
        LoadEnvironment(result, environment);
        LoadConsolidated(result, environment);
        return result;
    }

    private static void LoadEncryptedFile(
        Dictionary<string, JsonElement> result,
        IReadOnlyDictionary<string, string> environment,
        Func<string, byte[]?> readFile)
    {
        if (!environment.TryGetValue("SST_KEY", out var key) || string.IsNullOrEmpty(key)) return;
        if (!environment.TryGetValue("SST_KEY_FILE", out var file) || string.IsNullOrEmpty(file)) return;

        // A missing file is not an error. Locally the links usually arrive through the
        // environment instead, and the other SDKs skip it too.
        var data = readFile(file);
        if (data == null) return;

        Dictionary<string, JsonElement> parsed;
        try
        {
            parsed = Parse(Decrypt(Convert.FromBase64String(key), data));
        }
        catch (Exception e) when (e is FormatException or CryptographicException or JsonException or ArgumentException)
        {
            throw new ResourceException(
                $"Failed to read SST links from {file}. Check that SST_KEY matches the key the file was encrypted with.", e);
        }
        foreach (var (name, value) in parsed)
        {
            result[name] = value;
        }
    }

    private static void LoadEnvironment(
        Dictionary<string, JsonElement> result,
        IReadOnlyDictionary<string, string> environment)
    {
        foreach (var (name, value) in environment)
        {
            if (!name.StartsWith(Prefix, StringComparison.Ordinal) || string.IsNullOrEmpty(value)) continue;
            try
            {
                using var document = JsonDocument.Parse(value);
                result[name[Prefix.Length..]] = document.RootElement.Clone();
            }
            catch (JsonException)
            {
                // Not something SST wrote. The JS SDK ignores these as well.
            }
        }
    }

    private static void LoadConsolidated(
        Dictionary<string, JsonElement> result,
        IReadOnlyDictionary<string, string> environment)
    {
        if (!environment.TryGetValue("SST_RESOURCES_JSON", out var json) || string.IsNullOrEmpty(json)) return;
        try
        {
            foreach (var (name, value) in Parse(json))
            {
                result[name] = value;
            }
        }
        catch (JsonException)
        {
            // Matches the other SDKs, which ignore an unparsable consolidated value.
        }
    }

    /// <summary>
    /// SST encrypts the links JSON with AES-256-GCM, a 12 byte zero nonce and the 16 byte
    /// authentication tag appended to the ciphertext.
    /// </summary>
    internal static byte[] Decrypt(byte[] key, byte[] data)
    {
        if (data.Length < TagSize)
        {
            throw new CryptographicException("Encrypted data is shorter than the authentication tag");
        }
        var ciphertext = data.AsSpan(0, data.Length - TagSize);
        var tag = data.AsSpan(data.Length - TagSize);
        var plaintext = new byte[ciphertext.Length];
        using var aes = new AesGcm(key, TagSize);
        aes.Decrypt(new byte[NonceSize], ciphertext, tag, plaintext);
        return plaintext;
    }

    private static Dictionary<string, JsonElement> Parse(byte[] json)
    {
        using var document = JsonDocument.Parse(json);
        return ToDictionary(document.RootElement);
    }

    private static Dictionary<string, JsonElement> Parse(string json)
    {
        using var document = JsonDocument.Parse(json);
        return ToDictionary(document.RootElement);
    }

    private static Dictionary<string, JsonElement> ToDictionary(JsonElement root)
    {
        if (root.ValueKind != JsonValueKind.Object)
        {
            throw new JsonException("Expected a JSON object of resources");
        }
        var result = new Dictionary<string, JsonElement>(StringComparer.Ordinal);
        foreach (var property in root.EnumerateObject())
        {
            result[property.Name] = property.Value.Clone();
        }
        return result;
    }
}
