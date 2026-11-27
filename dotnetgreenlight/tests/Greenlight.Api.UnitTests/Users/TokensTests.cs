using System.Security.Cryptography;
using System.Text;
using Greenlight.Users;

namespace Greenlight.UnitTests.Users;

public sealed class TokensTests
{
    [Fact]
    public void Generate_produces_a_hashed_26_char_base32_token()
    {
        var ttl = TimeSpan.FromHours(24);
        var before = DateTimeOffset.UtcNow;

        var token = Tokens.Generate(42, ttl, TokenScopes.Authentication, TimeProvider.System);

        token.Plaintext.Length.ShouldBe(26);
        token.Plaintext.ShouldMatch("^[A-Z2-7]{26}$");
        token.Hash.ShouldBe(SHA256.HashData(Encoding.UTF8.GetBytes(token.Plaintext)));
        token.Hash.ShouldNotBe(Encoding.UTF8.GetBytes(token.Plaintext));
        token.UserId.ShouldBe(42);
        token.Scope.ShouldBe(TokenScopes.Authentication);
        token.Expiry.ShouldBeInRange(before + ttl, DateTimeOffset.UtcNow + ttl);
    }

    [Fact]
    public void Tokens_are_unique()
    {
        var a = Tokens.Generate(1, TimeSpan.FromHours(1), TokenScopes.Activation, TimeProvider.System);
        var b = Tokens.Generate(1, TimeSpan.FromHours(1), TokenScopes.Activation, TimeProvider.System);

        a.Plaintext.ShouldNotBe(b.Plaintext);
        a.Hash.ShouldNotBe(b.Hash);
    }

    [Fact]
    public void Hash_is_deterministic() => Tokens.Hash("ABC").ShouldBe(Tokens.Hash("ABC"));
}

public sealed class PasswordHasherTests
{
    [Fact]
    public void Hash_is_bcrypt_with_cost_12_and_matches_only_the_original()
    {
        var hash = PasswordHasher.Hash("pa55word");

        Encoding.UTF8.GetString(hash).ShouldStartWith("$2");
        Encoding.UTF8.GetString(hash).ShouldContain("$12$");
        PasswordHasher.Matches("pa55word", hash).ShouldBeTrue();
        PasswordHasher.Matches("wrongpassword", hash).ShouldBeFalse();
    }

    [Fact]
    public void Verifies_hashes_produced_by_the_go_app()
    {
        // Generated with golang.org/x/crypto/bcrypt for "pa55word" (cost 4 to keep the test fast).
        var goHash = Encoding.UTF8.GetBytes("$2a$04$XRKePuHjSAlIk4BlCAurfur04YUeu6rJcP1AvZ3/3AAjdbFUxD9Ve");
        PasswordHasher.Matches("pa55word", goHash).ShouldBeTrue();
    }
}
