using Greenlight.Common.Api;
using Microsoft.AspNetCore.Http.HttpResults;
using Greenlight.Common.Data;

namespace Greenlight.Movies;

public sealed record CreateMovieRequest : IMovieFields
{
    // Missing fields fall back to their zero values, so they are reported by the validator
    // ("must be provided") rather than as JSON binding errors - same behaviour as the Go API.
    public string Title { get; init; } = string.Empty;
    public int Year { get; init; }
    public Runtime Runtime { get; init; }
    public string[]? Genres { get; init; }
}

public sealed class CreateMovieEndpoint : IEndpoint
{
    public static void Map(IEndpointRouteBuilder app) =>
        app.MapPost("/", HandleAsync)
            .WithName("CreateMovie")
            .WithSummary("Creates a new movie")
            .WithRequestValidation<CreateMovieRequest>();

    private static async Task<Created<MovieResponse>> HandleAsync(
        CreateMovieRequest request,
        IUnitOfWork unitOfWork,
        CancellationToken ct)
    {
        var movie = new Movie
        {
            Title = request.Title,
            Year = request.Year,
            Runtime = request.Runtime,
            Genres = request.Genres!,
        };

        await unitOfWork.Movies.InsertAsync(movie, ct);

        return TypedResults.Created($"/v1/movies/{movie.Id}", MovieResponse.From(movie));
    }
}

public sealed class CreateMovieRequestValidator : MovieFieldsValidator<CreateMovieRequest>;
