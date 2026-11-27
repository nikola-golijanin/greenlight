using Greenlight.Common.Api;
using Microsoft.AspNetCore.Http.HttpResults;
using Greenlight.Common.Data;

namespace Greenlight.Movies;

public sealed class DeleteMovieEndpoint : IEndpoint
{
    public static void Map(IEndpointRouteBuilder app) =>
        app.MapDelete("/{id:long}", HandleAsync)
            .WithName("DeleteMovie")
            .WithSummary("Deletes a movie");

    private static async Task<Results<NoContent, ProblemHttpResult>> HandleAsync(
        long id,
        IUnitOfWork unitOfWork,
        CancellationToken ct)
    {
        if (id < 1)
        {
            return Problems.NotFound();
        }

        return await unitOfWork.Movies.DeleteAsync(id, ct)
            ? TypedResults.NoContent()
            : Problems.NotFound();
    }
}
