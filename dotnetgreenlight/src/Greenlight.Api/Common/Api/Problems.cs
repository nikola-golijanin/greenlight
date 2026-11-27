using Microsoft.AspNetCore.Http.HttpResults;

namespace Greenlight.Common.Api;

/// <summary>RFC 7807 responses, keeping the Go API's status codes and messages.</summary>
public static class Problems
{
    public const string NotFoundDetail = "the requested resource could not be found";
    public const string EditConflictDetail = "unable to update the record due to an edit conflict, please try again";
    public const string ServerErrorDetail = "the server encountered a problem and could not process your request";

    public static ProblemHttpResult NotFound() =>
        TypedResults.Problem(statusCode: StatusCodes.Status404NotFound, detail: NotFoundDetail);

    public static ProblemHttpResult EditConflict() =>
        TypedResults.Problem(statusCode: StatusCodes.Status409Conflict, detail: EditConflictDetail);

    public static ProblemHttpResult FailedValidation(IDictionary<string, string[]> errors) =>
        TypedResults.Problem(new HttpValidationProblemDetails(errors)
        {
            Status = StatusCodes.Status422UnprocessableEntity,
        });

    public static ProblemHttpResult FailedValidation(string key, string message) =>
        FailedValidation(new Dictionary<string, string[]> { [key] = [message] });
}
