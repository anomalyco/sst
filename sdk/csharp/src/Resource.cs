using System.Collections;
using System.Diagnostics.CodeAnalysis;
using System.Text.Json;
using System.Text.Json.Serialization.Metadata;

namespace SST;

/// <summary>
/// Access the resources linked to your function or container with <c>link: [...]</c>
/// in <c>sst.config.ts</c>.
/// </summary>
/// <example>
/// <code>
/// var bucket = Resource.Get&lt;string&gt;("MyBucket", "name");
/// var secret = Resource.Get&lt;string&gt;("MySecret", "value");
/// var stage = Resource.App.Stage;
/// </code>
/// </example>
public static class Resource
{
    private static readonly object Gate = new();
    private static readonly JsonSerializerOptions WebOptions = new(JsonSerializerDefaults.Web);
    private static IReadOnlyDictionary<string, JsonElement>? _resources;
    private static IReadOnlyDictionary<string, string>? _environment;

    /// <summary>
    /// Info about the current app. Available in every linked function.
    /// </summary>
    public static App App
    {
        get
        {
            var app = Get("App");
            return new App(
                app.TryGetProperty("name", out var name) ? name.GetString() ?? "" : "",
                app.TryGetProperty("stage", out var stage) ? stage.GetString() ?? "" : "");
        }
    }

    /// <summary>
    /// Returns a linked resource, or a property of it, as a raw <see cref="JsonElement"/>.
    /// </summary>
    /// <param name="name">The name of the resource in <c>sst.config.ts</c>, for example <c>MyBucket</c>.</param>
    /// <param name="path">Optional property path, for example <c>"name"</c>.</param>
    /// <exception cref="ResourceNotFoundException">The resource is not linked or the property does not exist.</exception>
    public static JsonElement Get(string name, params string[] path)
    {
        var resources = Resources;
        if (!resources.TryGetValue(name, out var current))
        {
            throw NotLinked(name);
        }
        foreach (var segment in path)
        {
            if (current.ValueKind != JsonValueKind.Object || !current.TryGetProperty(segment, out var next))
            {
                throw new ResourceNotFoundException($"\"{name}\" has no property \"{string.Join(".", path)}\"");
            }
            current = next;
        }
        return current;
    }

    /// <summary>
    /// Returns a linked resource, or a property of it, deserialized to <typeparamref name="T"/>.
    /// Property names are matched case-insensitively so <c>record Bucket(string Name)</c>
    /// works with the camelCase properties SST produces.
    /// </summary>
    /// <exception cref="ResourceNotFoundException">The resource is not linked or the property does not exist.</exception>
    /// <exception cref="ResourceException">The value is JSON <c>null</c> or cannot be deserialized to <typeparamref name="T"/>.</exception>
    [RequiresUnreferencedCode("Uses reflection based JSON deserialization. For Native AOT use the overload that takes a JsonTypeInfo.")]
    [RequiresDynamicCode("Uses reflection based JSON deserialization. For Native AOT use the overload that takes a JsonTypeInfo.")]
    public static T Get<T>(string name, params string[] path)
    {
        var element = Get(name, path);
        try
        {
            return JsonSerializer.Deserialize<T>(element, WebOptions) ?? throw Null(name, path);
        }
        catch (JsonException e)
        {
            throw new ResourceException($"Failed to deserialize {Describe(name, path)} to {typeof(T).Name}", e);
        }
    }

    /// <summary>
    /// Returns a linked resource, or a property of it, deserialized with source generated
    /// metadata. Safe for Native AOT.
    /// </summary>
    /// <exception cref="ResourceNotFoundException">The resource is not linked or the property does not exist.</exception>
    /// <exception cref="ResourceException">The value is JSON <c>null</c> or cannot be deserialized to <typeparamref name="T"/>.</exception>
    public static T Get<T>(JsonTypeInfo<T> typeInfo, string name, params string[] path)
    {
        var element = Get(name, path);
        try
        {
            return JsonSerializer.Deserialize(element, typeInfo) ?? throw Null(name, path);
        }
        catch (JsonException e)
        {
            throw new ResourceException($"Failed to deserialize {Describe(name, path)} to {typeof(T).Name}", e);
        }
    }

    /// <summary>
    /// Returns whether a resource is linked and, if so, its raw value.
    /// </summary>
    public static bool TryGet(string name, out JsonElement value) => Resources.TryGetValue(name, out value);

    /// <summary>
    /// Returns every linked resource keyed by name.
    /// </summary>
    public static IReadOnlyDictionary<string, JsonElement> All() => Resources;

    /// <summary>
    /// Drops the cached resources so the next access reads the environment again.
    /// </summary>
    internal static void Reload()
    {
        lock (Gate)
        {
            _resources = null;
            _environment = null;
        }
    }

    private static IReadOnlyDictionary<string, JsonElement> Resources
    {
        get
        {
            lock (Gate)
            {
                if (_resources == null)
                {
                    _environment = ReadEnvironment();
                    _resources = ResourceLoader.Load(_environment, ReadFile);
                }
                return _resources;
            }
        }
    }

    private static ResourceNotFoundException NotLinked(string name)
    {
        var env = _environment ?? ReadEnvironment();
        if (!env.ContainsKey("SST_RESOURCE_App") && !env.ContainsKey("SST_RESOURCES_JSON"))
        {
            return new ResourceNotFoundException(
                "It does not look like SST links are active. If this is in local development and you are not starting this process through the multiplexer, wrap your command with `sst dev -- <command>`");
        }
        var message = $"\"{name}\" is not linked in your sst.config.ts";
        if (env.TryGetValue("AWS_LAMBDA_FUNCTION_NAME", out var function) && !string.IsNullOrEmpty(function))
        {
            message += $" to {function}";
        }
        return new ResourceNotFoundException(message);
    }

    private static ResourceException Null(string name, string[] path) =>
        new($"{Describe(name, path)} is null");

    private static string Describe(string name, string[] path) =>
        path.Length == 0 ? $"\"{name}\"" : $"\"{name}.{string.Join(".", path)}\"";

    /// <summary>
    /// SST_KEY_FILE is relative to the function's working directory. Lambda starts in
    /// /var/task, but fall back to the application directory in case the host does not.
    /// </summary>
    private static byte[]? ReadFile(string path)
    {
        if (File.Exists(path))
        {
            return File.ReadAllBytes(path);
        }
        var fallback = Path.Combine(AppContext.BaseDirectory, path);
        return File.Exists(fallback) ? File.ReadAllBytes(fallback) : null;
    }

    private static IReadOnlyDictionary<string, string> ReadEnvironment()
    {
        var result = new Dictionary<string, string>(StringComparer.Ordinal);
        foreach (DictionaryEntry entry in Environment.GetEnvironmentVariables())
        {
            if (entry.Key is string key && entry.Value is string value)
            {
                result[key] = value;
            }
        }
        return result;
    }
}

/// <summary>
/// The current SST app.
/// </summary>
/// <param name="Name">The name of the app.</param>
/// <param name="Stage">The stage the app is deployed to.</param>
public sealed record App(string Name, string Stage);
