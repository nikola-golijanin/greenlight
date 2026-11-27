using FluentValidation;
using Greenlight.Common.Api;
using Greenlight.Common.Data;
using Microsoft.AspNetCore.Http.HttpResults;

namespace Greenlight.Users;

public sealed record ActivateUserRequest
{
    public string Token { get; init; } = string.Empty;
}

public sealed class ActivateUserEndpoint : IEndpoint
{
    public static void Map(IEndpointRouteBuilder app) =>
        app.MapPut("/activated", HandleAsync)
            .WithName("ActivateUser")
            .WithSummary("Activates a user with their activation token")
            .WithRequestValidation<ActivateUserRequest>();

    private static async Task<Results<Ok<UserResponse>, ProblemHttpResult>> HandleAsync(
        ActivateUserRequest request,
        IUnitOfWork unitOfWork,
        TimeProvider timeProvider,
        CancellationToken ct)
    {
        await unitOfWork.BeginTransactionAsync(ct);

        var user = await unitOfWork.Users.GetForTokenAsync(TokenScopes.Activation, request.Token, timeProvider.GetUtcNow(), ct);
        if (user is null)
        {
            await unitOfWork.RollbackAsync(ct);
            return Problems.FailedValidation("token", "invalid or expired activation token");
        }

        user.Activated = true;
        if (!await unitOfWork.Users.UpdateAsync(user, ct))
        {
            await unitOfWork.RollbackAsync(ct);
            return Problems.EditConflict();
        }

        await unitOfWork.Tokens.DeleteAllForUserAsync(TokenScopes.Activation, user.Id, ct);
        await unitOfWork.CommitAsync(ct);

        return TypedResults.Ok(UserResponse.From(user));
    }
}

public sealed class ActivateUserRequestValidator : AbstractValidator<ActivateUserRequest>
{
    public ActivateUserRequestValidator() => RuleFor(x => x.Token).TokenPlaintext();
}
