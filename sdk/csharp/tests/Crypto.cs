using System.Security.Cryptography;
using System.Text;

namespace SST.Tests;

/// <summary>
/// Produces files in the same shape as the SST CLI: AES-256-GCM, zero nonce, tag appended.
/// </summary>
internal static class Crypto
{
    internal static readonly byte[] Key = Enumerable.Range(0, 32).Select(i => (byte)i).ToArray();
    internal static string KeyBase64 => Convert.ToBase64String(Key);

    internal static byte[] Encrypt(string json, byte[]? key = null)
    {
        var plaintext = Encoding.UTF8.GetBytes(json);
        var ciphertext = new byte[plaintext.Length];
        var tag = new byte[16];
        using var aes = new AesGcm(key ?? Key, 16);
        aes.Encrypt(new byte[12], plaintext, ciphertext, tag);
        return [.. ciphertext, .. tag];
    }
}
