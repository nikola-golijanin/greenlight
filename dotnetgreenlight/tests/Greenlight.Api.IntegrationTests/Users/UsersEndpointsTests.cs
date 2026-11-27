using System.Text.Json;
using System.Net;
using Greenlight.IntegrationTests.Infrastructure;
using Greenlight.Users;

namespace Greenlight.IntegrationTests.Users;

public sealed class UsersEndpointsTests(GreenlightApiFactory factory) : IntegrationTest(factory)
{
    private static object Alice(string email = "alice@example.com") => new { name = "Alice", email, password = "pa55word!" };

    private async Task<(UserDto User, string Token)> RegisterAsync(string email = "alice@example.com")
    {
        Factory.ActivationNotifier.Clear();
        var response = await Client.PostJsonAsync("/v1/users", Alice(email), Ct);
        response.StatusCode.ShouldBe(HttpStatusCode.Accepted, await response.Content.ReadAsStringAsync(Ct));
        return (await response.ReadAsync<UserDto>(Ct), Factory.ActivationNotifier.Single().Token.Plaintext);
    }

    // ---- POST /v1/users ----

    [Fact]
    public async Task Register_returns_202_with_inactive_user()
    {
        var response = await Client.PostJsonAsync("/v1/users", Alice(), Ct);

        response.StatusCode.ShouldBe(HttpStatusCode.Accepted);
        var json = await response.ReadJsonAsync(Ct);
        json.EnumerateObject().Select(p => p.Name).ShouldBe(["id", "created_at", "name", "email", "activated"]);

        var user = json.Deserialize<UserDto>(HttpExtensions.Json)!;
        user.Name.ShouldBe("Alice");
        user.Email.ShouldBe("alice@example.com");
        user.Activated.ShouldBeFalse();
        user.Id.ShouldBeGreaterThan(0);
    }

    [Fact]
    public async Task Register_issues_an_activation_token_and_default_permission()
    {
        var (user, _) = await RegisterAsync();

        var (notifiedUser, token) = Factory.ActivationNotifier.Single();
        notifiedUser.Id.ShouldBe(user.Id);
        token.Scope.ShouldBe(TokenScopes.Activation);
        token.Plaintext.Length.ShouldBe(26);
        token.Expiry.ShouldBe(Factory.Time.GetUtcNow() + Tokens.ActivationTtl, TimeSpan.FromSeconds(1));

        var permissions = await WithUnitOfWorkAsync(uow => uow.Permissions.GetAllForUserAsync(user.Id, Ct));
        permissions.ShouldBe(["movies:read"]);
    }

    [Fact]
    public async Task Register_stores_a_bcrypt_hash_not_the_password()
    {
        var (_, token) = await RegisterAsync();

        var stored = await WithUnitOfWorkAsync(uow =>
            uow.Users.GetForTokenAsync(TokenScopes.Activation, token, Factory.Time.GetUtcNow(), Ct));

        stored.ShouldNotBeNull();
        PasswordHasher.Matches("pa55word!", stored.PasswordHash).ShouldBeTrue();
    }

    [Fact]
    public async Task Register_duplicate_email_returns_422_and_rolls_back()
    {
        await RegisterAsync("dup@example.com");
        Factory.ActivationNotifier.Clear();

        // citext column: the check is case-insensitive.
        var response = await Client.PostJsonAsync("/v1/users", Alice("DUP@example.com"), Ct);

        var problem = await response.ShouldBeProblemAsync(422, Ct);
        problem.Errors("email").ShouldBe(["a user with this email already exists"]);
        Factory.ActivationNotifier.Count.ShouldBe(0);
    }

    [Fact]
    public async Task Register_with_invalid_input_returns_422()
    {
        var response = await Client.PostJsonAsync("/v1/users", new { name = "", email = "not-an-email", password = "short" }, Ct);

        var problem = await response.ShouldBeProblemAsync(422, Ct);
        problem.Errors("name").ShouldBe(["must be provided"]);
        problem.Errors("email").ShouldBe(["must be a valid email address"]);
        problem.Errors("password").ShouldBe(["must be at least 8 bytes long"]);
        Factory.ActivationNotifier.Count.ShouldBe(0);
    }

    [Fact]
    public async Task Register_with_unknown_field_returns_400()
    {
        var response = await Client.PostRawAsync("/v1/users", """{"name":"a","email":"a@b.c","password":"12345678","admin":true}""", Ct);

        await response.ShouldBeProblemAsync(400, Ct);
    }

    // ---- PUT /v1/users/activated ----

    [Fact]
    public async Task Activate_with_valid_token_activates_user()
    {
        var (user, token) = await RegisterAsync();

        var response = await Client.PutJsonAsync("/v1/users/activated", new { token }, Ct);

        response.StatusCode.ShouldBe(HttpStatusCode.OK);
        var activated = await response.ReadAsync<UserDto>(Ct);
        activated.ShouldBe(user with { Activated = true });
    }

    [Fact]
    public async Task Activation_token_cannot_be_reused()
    {
        var (_, token) = await RegisterAsync();
        (await Client.PutJsonAsync("/v1/users/activated", new { token }, Ct)).StatusCode.ShouldBe(HttpStatusCode.OK);

        var response = await Client.PutJsonAsync("/v1/users/activated", new { token }, Ct);

        var problem = await response.ShouldBeProblemAsync(422, Ct);
        problem.Errors("token").ShouldBe(["invalid or expired activation token"]);
    }

    [Fact]
    public async Task Expired_activation_token_is_rejected()
    {
        var (_, token) = await RegisterAsync();
        Factory.Time.Advance(Tokens.ActivationTtl + TimeSpan.FromMinutes(1));

        var response = await Client.PutJsonAsync("/v1/users/activated", new { token }, Ct);

        var problem = await response.ShouldBeProblemAsync(422, Ct);
        problem.Errors("token").ShouldBe(["invalid or expired activation token"]);
    }

    [Fact]
    public async Task Unknown_token_is_rejected()
    {
        var response = await Client.PutJsonAsync("/v1/users/activated", new { token = new string('A', 26) }, Ct);

        var problem = await response.ShouldBeProblemAsync(422, Ct);
        problem.Errors("token").ShouldBe(["invalid or expired activation token"]);
    }

    [Theory]
    [InlineData("", "must be provided")]
    [InlineData("TOOSHORT", "must be 26 bytes long")]
    public async Task Malformed_token_returns_422(string token, string message)
    {
        var response = await Client.PutJsonAsync("/v1/users/activated", new { token }, Ct);

        var problem = await response.ShouldBeProblemAsync(422, Ct);
        problem.Errors("token").ShouldBe([message]);
    }
}
