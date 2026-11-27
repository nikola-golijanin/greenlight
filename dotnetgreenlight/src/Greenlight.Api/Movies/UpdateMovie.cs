using FluentValidation;
using Greenlight.Common.Api;
using Microsoft.AspNetCore.Http.HttpResults;
using Greenlight.Common.Data;

namespace Greenlight.Movies;

/// <summary>Partial update: only the supplied fields are changed.</summary>
public sealed record UpdateMovieRequest
{
    public string? Title { get; init; }
    public int? Year { get; init; }
    public Runtime? Runtime { get; init; }
    public string[]? Genres { get; init; }
}

public sealed class UpdateMovieEndpoint : IEndpoint
{
    public static void Map(IEndpointRouteBuilder app) =>
        app.MapPatch("/{id:long}", HandleAsync)
            .WithName("UpdateMovie")
            .WithSummary("Partially updates a movie")
            .ProducesValidationProblem(StatusCodes.Status422UnprocessableEntity);

    private static async Task<Results<Ok<MovieResponse>, ProblemHttpResult>> HandleAsync(
        long id,
        UpdateMovieRequest request,
        IUnitOfWork unitOfWork,
        IValidator<Movie> validator,
        CancellationToken ct)
    {
        if (id < 1)
        {
            return Problems.NotFound();
        }

        var movie = await unitOfWork.Movies.GetByIdAsync(id, ct);
        if (movie is null)
        {
            return Problems.NotFound();
        }

        movie.Title = request.Title ?? movie.Title;
        movie.Year = request.Year ?? movie.Year;
        movie.Runtime = request.Runtime ?? movie.Runtime;
        movie.Genres = request.Genres ?? movie.Genres;

        var validation = await validator.ValidateAsync(movie, ct);
        if (!validation.IsValid)
        {
            return Problems.FailedValidation(validation.ToDictionary());
        }

        return await unitOfWork.Movies.UpdateAsync(movie, ct)
            ? TypedResults.Ok(MovieResponse.From(movie))
            : Problems.EditConflict();
    }
}
