using System.Text;

namespace Greenlight.Users;

/// <summary>bcrypt (cost 12), stored as the ASCII hash bytes - compatible with the Go app.</summary>
public static class PasswordHasher
{
    public const int WorkFactor = 12;

    public static byte[] Hash(string plaintext) =>
        Encoding.UTF8.GetBytes(BCrypt.Net.BCrypt.HashPassword(plaintext, WorkFactor));

    public static bool Matches(string plaintext, byte[] hash) =>
        BCrypt.Net.BCrypt.Verify(plaintext, Encoding.UTF8.GetString(hash));
}
