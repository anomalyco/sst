namespace SST;

/// <summary>
/// Thrown when linked resources cannot be read or converted.
/// </summary>
public class ResourceException : Exception
{
    /// <inheritdoc />
    public ResourceException(string message) : base(message)
    {
    }

    /// <inheritdoc />
    public ResourceException(string message, Exception innerException) : base(message, innerException)
    {
    }
}

/// <summary>
/// Thrown when a resource is not linked to the function, or a requested property does not exist.
/// </summary>
public sealed class ResourceNotFoundException : ResourceException
{
    /// <inheritdoc />
    public ResourceNotFoundException(string message) : base(message)
    {
    }
}
