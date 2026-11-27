using System.Security.Cryptography;
using System.Text;

namespace Greenlight.Users;

public static class TokenScopes
{
    public const string Activation = "activation";
    public const string Authentication = "authentication";
}

public sealed record Token(string Plaintext, byte[] Hash, long UserId, DateTimeOffset Expiry, string Scope);

public static class Tokens
{
    public const int PlaintextLength = 26;
    public static readonly TimeSpan ActivationTtl = TimeSpan.FromDays(3);

    // RFC 4648 base32 alphabet - the same output space as Go's crypto/rand.Text().
    private const string Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";

    public static Token Generate(long userId, TimeSpan ttl, string scope, TimeProvider timeProvider)
    {
        var plaintext = RandomNumberGenerator.GetString(Alphabet, PlaintextLength);
        return new Token(plaintext, Hash(plaintext), userId, timeProvider.GetUtcNow().Add(ttl), scope);
    }

    /// <summary>SHA-256 of the plaintext; only the hash is stored in the database.</summary>
    public static byte[] Hash(string plaintext) => SHA256.HashData(Encoding.UTF8.GetBytes(plaintext));
}
