using FluentValidation;

namespace Greenlight.Common.Api;

internal sealed class RequestValidationFilter<TRequest>(IValidator<TRequest> validator) : IEndpointFilter
{
    public async ValueTask<object?> InvokeAsync(EndpointFilterInvocationContext context, EndpointFilterDelegate next)
    {
        var request = context.Arguments.OfType<TRequest>().First();
        var result = await validator.ValidateAsync(request, context.HttpContext.RequestAborted);

        return result.IsValid
            ? await next(context)
            : Problems.FailedValidation(result.ToDictionary());
    }
}

public static class RouteHandlerBuilderExtensions
{
    /// <summary>Validates the <typeparamref name="TRequest"/> argument with FluentValidation; 422 on failure.</summary>
    public static RouteHandlerBuilder WithRequestValidation<TRequest>(this RouteHandlerBuilder builder) =>
        builder
            .AddEndpointFilter<RequestValidationFilter<TRequest>>()
            .ProducesValidationProblem(StatusCodes.Status422UnprocessableEntity);
}
