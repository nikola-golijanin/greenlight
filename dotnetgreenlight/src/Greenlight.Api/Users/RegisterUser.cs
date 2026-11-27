using FluentValidation;
using Greenlight.Common.Api;
using Greenlight.Common.Data;
using Microsoft.AspNetCore.Http.HttpResults;

namespace Greenlight.Users;

public sealed record RegisterUserRequest
{
    public string Name { get; init; } = string.Empty;
    public string Email { get; init; } = string.Empty;
    public string Password { get; init; } = string.Empty;
}

public sealed class RegisterUserEndpoint : IEndpoint
{
    public static readonly string[] DefaultPermissions = ["movies:read"];

    public static void Map(IEndpointRouteBuilder app) =>
        app.MapPost("/", HandleAsync)
            .WithName("RegisterUser")
            .WithSummary("Registers a user and issues an activation token")
            .WithRequestValidation<RegisterUserRequest>();

    private static async Task<Results<Accepted<UserResponse>, ProblemHttpResult>> HandleAsync(
        RegisterUserRequest request,
        IUnitOfWork unitOfWork,
        IActivationNotifier notifier,
        TimeProvider timeProvider,
        CancellationToken ct)
    {
        var user = new User
        {
            Name = request.Name,
            Email = request.Email,
            PasswordHash = PasswordHasher.Hash(request.Password),
            Activated = false,
        };

        await unitOfWork.BeginTransactionAsync(ct);
        try
        {
            await unitOfWork.Users.InsertAsync(user, ct);
        }
        catch (DuplicateEmailException e)
        {
            await unitOfWork.RollbackAsync(ct);
            return Problems.FailedValidation("email", e.Message);
        }

        await unitOfWork.Permissions.AddForUserAsync(user.Id, DefaultPermissions, ct);

        var token = Tokens.Generate(user.Id, Tokens.ActivationTtl, TokenScopes.Activation, timeProvider);
        await unitOfWork.Tokens.InsertAsync(token, ct);

        await unitOfWork.CommitAsync(ct);

        await notifier.NotifyAsync(user, token, ct);

        return TypedResults.Accepted((string?)null, UserResponse.From(user));
    }
}

public sealed class RegisterUserRequestValidator : AbstractValidator<RegisterUserRequest>
{
    public RegisterUserRequestValidator()
    {
        RuleFor(x => x.Name).UserName();
        RuleFor(x => x.Email).Email();
        RuleFor(x => x.Password).PlaintextPassword();
    }
}
