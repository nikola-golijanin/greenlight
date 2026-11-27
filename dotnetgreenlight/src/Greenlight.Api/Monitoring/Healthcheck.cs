using Greenlight.Common.Observability;
using Microsoft.AspNetCore.Http.HttpResults;

namespace Greenlight.Monitoring;

public sealed record HealthcheckResponse(string Status, IReadOnlyDictionary<string, string> SystemInfo);

public sealed class HealthcheckEndpoint : IEndpoint
{
    public static void Map(IEndpointRouteBuilder app) =>
        app.MapGet("/v1/healthcheck", Handle)
            .WithName("Healthcheck")
            .WithTags("System");

    private static Ok<HealthcheckResponse> Handle(IHostEnvironment environment) =>
        TypedResults.Ok(new HealthcheckResponse("available", new Dictionary<string, string>
        {
            ["environment"] = environment.EnvironmentName.ToLowerInvariant(),
            ["version"] = Telemetry.Version,
        }));
}
