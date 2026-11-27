using Greenlight.Movies;
using Greenlight.Monitoring;
using Greenlight.Users;

namespace Greenlight;

public static class Endpoints
{
    public static void MapEndpoints(this WebApplication app)
    {
        var endpoints = app.MapGroup(string.Empty);

        endpoints.MapSystemEndpoints();
        endpoints.MapMovieEndpoints();
        endpoints.MapUserEndpoints();
    }

    private static void MapSystemEndpoints(this IEndpointRouteBuilder app) =>
        app.MapEndpoint<HealthcheckEndpoint>()
            .MapEndpoint<DebugVarsEndpoint>();

    private static void MapMovieEndpoints(this IEndpointRouteBuilder app)
    {
        var endpoints = app.MapGroup("/v1/movies")
            .WithTags("Movies");

        endpoints
            .MapEndpoint<GetMoviesEndpoint>()
            .MapEndpoint<CreateMovieEndpoint>()
            .MapEndpoint<GetMovieByIdEndpoint>()
            .MapEndpoint<UpdateMovieEndpoint>()
            .MapEndpoint<DeleteMovieEndpoint>();
    }

    private static void MapUserEndpoints(this IEndpointRouteBuilder app)
    {
        var endpoints = app.MapGroup("/v1/users")
            .WithTags("Users");

        endpoints
            .MapEndpoint<RegisterUserEndpoint>()
            .MapEndpoint<ActivateUserEndpoint>();
    }

    public static IEndpointRouteBuilder MapEndpoint<TEndpoint>(this IEndpointRouteBuilder app) where TEndpoint : IEndpoint
    {
        TEndpoint.Map(app);
        return app;
    }
}

public interface IEndpoint
{
    static abstract void Map(IEndpointRouteBuilder app);
}
