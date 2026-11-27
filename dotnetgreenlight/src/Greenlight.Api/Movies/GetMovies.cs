using System.Globalization;
using FluentValidation;
using Greenlight.Common.Api;
using Microsoft.AspNetCore.Http.HttpResults;
using Microsoft.AspNetCore.Mvc;
using Greenlight.Common.Data;

namespace Greenlight.Movies;

/// <summary>
/// Query parameters are bound as strings so that a non-integer page reports
/// "must be an integer value" (422) like the Go API, instead of a 400 binding error.
/// </summary>
public sealed record GetMoviesRequest
{
    [FromQuery(Name = "title")] public string? Title { get; init; }
    [FromQuery(Name = "genres")] public string? Genres { get; init; }
    [FromQuery(Name = "page")] public string? Page { get; init; }
    [FromQuery(Name = "page_size")] public string? PageSize { get; init; }
    [FromQuery(Name = "sort")] public string? Sort { get; init; }
}

public sealed class GetMoviesEndpoint : IEndpoint
{
    public static void Map(IEndpointRouteBuilder app) =>
        app.MapGet("/", HandleAsync)
            .WithName("GetMovies")
            .WithSummary("Lists all movies")
            .WithRequestValidation<GetMoviesRequest>();

    // Like the Go version, the filters are validated but not yet applied to the query.
    private static async Task<Ok<MovieResponse[]>> HandleAsync(
        [AsParameters] GetMoviesRequest request,
        IUnitOfWork unitOfWork,
        CancellationToken ct)
    {
        var all = await unitOfWork.Movies.GetAllAsync(ct);
        return TypedResults.Ok(all.Select(MovieResponse.From).ToArray());
    }
}

/// <summary>Port of <c>data.ValidateFilters</c>.</summary>
public sealed class GetMoviesRequestValidator : AbstractValidator<GetMoviesRequest>
{
    public static readonly string[] SortSafelist = ["id", "title", "year", "runtime", "-id", "-title", "-year", "-runtime"];

    public GetMoviesRequestValidator()
    {
        RuleFor(x => x.Page)
            .Must(BeIntegerOrEmpty).WithMessage("must be an integer value")
            .Must(p => ParseOr(p, 1) > 0).WithMessage("must be greater than zero")
            .Must(p => ParseOr(p, 1) <= 10_000_000).WithMessage("must be a maximum of 10 million");

        RuleFor(x => x.PageSize)
            .Must(BeIntegerOrEmpty).WithMessage("must be an integer value")
            .Must(p => ParseOr(p, 20) > 0).WithMessage("must be greater than zero")
            .Must(p => ParseOr(p, 20) <= 100).WithMessage("must be a maximum of 100");

        RuleFor(x => x.Sort)
            .Must(s => SortSafelist.Contains(string.IsNullOrEmpty(s) ? "id" : s, StringComparer.Ordinal))
            .WithMessage("invalid sort value");
    }

    private static bool BeIntegerOrEmpty(string? value) =>
        string.IsNullOrEmpty(value) || int.TryParse(value, NumberStyles.AllowLeadingSign, CultureInfo.InvariantCulture, out _);

    private static int ParseOr(string? value, int fallback) =>
        int.TryParse(value, NumberStyles.AllowLeadingSign, CultureInfo.InvariantCulture, out var parsed) ? parsed : fallback;
}
