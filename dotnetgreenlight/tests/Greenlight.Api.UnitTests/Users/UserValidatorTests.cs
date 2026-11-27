using Greenlight.Users;

namespace Greenlight.UnitTests.Users;

public sealed class UserValidatorTests
{
    private readonly RegisterUserRequestValidator _register = new();
    private readonly ActivateUserRequestValidator _activate = new();

    private static RegisterUserRequest Valid() => new() { Name = "Alice", Email = "alice@example.com", Password = "pa55word!" };

    private Dictionary<string, string[]> Errors(RegisterUserRequest request) =>
        new(_register.Validate(request).ToDictionary());

    [Fact]
    public void Valid_user_passes() => _register.Validate(Valid()).IsValid.ShouldBeTrue();

    [Fact]
    public void Missing_name() => Errors(Valid() with { Name = "" })["name"].ShouldBe(["must be provided"]);

    [Fact]
    public void Name_too_long() =>
        Errors(Valid() with { Name = new string('n', 501) })["name"].ShouldBe(["must not be more than 500 bytes long"]);

    [Theory]
    [InlineData("user@example.com", true)]
    [InlineData("", false)]
    [InlineData("not-an-email", false)]
    [InlineData("missing@tld.", false)]
    public void Email_rules(string email, bool valid) =>
        Errors(Valid() with { Email = email }).ContainsKey("email").ShouldBe(!valid);

    [Theory]
    [InlineData("", "must be provided")]
    [InlineData("1234567", "must be at least 8 bytes long")]
    public void Invalid_passwords(string password, string message) =>
        Errors(Valid() with { Password = password })["password"].ShouldBe([message]);

    [Fact]
    public void Password_over_72_bytes_fails() =>
        Errors(Valid() with { Password = new string('p', 73) })["password"].ShouldBe(["must not be more than 72 bytes long"]);

    [Theory]
    [InlineData("12345678", true)]
    [InlineData("pa55word!", true)]
    public void Valid_passwords(string password, bool valid) =>
        Errors(Valid() with { Password = password }).ContainsKey("password").ShouldBe(!valid);

    [Theory]
    [InlineData("", "must be provided")]
    [InlineData("AAAAAAAAAAAAAAAAAAAAAAAAA", "must be 26 bytes long")]
    [InlineData("AAAAAAAAAAAAAAAAAAAAAAAAAAA", "must be 26 bytes long")]
    public void Invalid_token(string token, string message) =>
        _activate.Validate(new ActivateUserRequest { Token = token }).ToDictionary()["token"].ShouldBe([message]);

    [Fact]
    public void Valid_token() =>
        _activate.Validate(new ActivateUserRequest { Token = new string('A', 26) }).IsValid.ShouldBeTrue();
}
