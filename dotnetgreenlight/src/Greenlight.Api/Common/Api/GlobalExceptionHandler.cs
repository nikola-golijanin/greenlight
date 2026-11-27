using System.Text.Json;
using Microsoft.AspNetCore.Diagnostics;

namespace Greenlight.Common.Api;

internal sealed class GlobalExceptionHandler(
    IProblemDetailsService problemDetails,
    ILogger<GlobalExceptionHandler> logger) : IExceptionHandler
{
    public async ValueTask<bool> TryHandleAsync(HttpContext context, Exception exception, CancellationToken ct)
    {
        var (status, detail) = exception switch
        {
            BadHttpRequestException bad => (bad.StatusCode, Describe(bad)),
            _ => (StatusCodes.Status500InternalServerError, Problems.ServerErrorDetail),
        };

        if (status >= StatusCodes.Status500InternalServerError)
        {
            logger.LogError(exception, "Unhandled exception for {Method} {Uri}",
                context.Request.Method, context.Request.Path + context.Request.QueryString);
        }

        context.Response.StatusCode = status;
        return await problemDetails.TryWriteAsync(new ProblemDetailsContext
        {
            HttpContext = context,
            Exception = exception,
            ProblemDetails = { Status = status, Detail = detail },
        });
    }

    private static string Describe(BadHttpRequestException exception) => exception.InnerException switch
    {
        JsonException json => json.Message,
        _ => exception.Message,
    };
}
