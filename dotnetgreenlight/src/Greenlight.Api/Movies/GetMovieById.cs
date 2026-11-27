using Greenlight.Common.Api;
using Microsoft.AspNetCore.Http.HttpResults;
using Greenlight.Common.Data;

namespace Greenlight.Movies;

public sealed class GetMovieByIdEndpoint : IEndpoint
{
    public static void Map(IEndpointRouteBuilder app) =>
        app.MapGet("/{id:long}", HandleAsync)
            .WithName("GetMovieById")
            .WithSummary("Gets a movie by id");

    private static async Task<Results<Ok<MovieResponse>, ProblemHttpResult>> HandleAsync(
        long id,
        IUnitOfWork unitOfWork,
        CancellationToken ct)
    {
        if (id < 1)
        {
            return Problems.NotFound();
        }

        var movie = await unitOfWork.Movies.GetByIdAsync(id, ct);
        return movie is null
            ? Problems.NotFound()
            : TypedResults.Ok(MovieResponse.From(movie));
    }
}
