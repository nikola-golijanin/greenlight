using System.Text.Json;
using System.Text.Json.Serialization;
using System.Threading.RateLimiting;
using FluentValidation;
using Greenlight.Common.Api;
using Greenlight.Common.Data;
using Greenlight.Common.Observability;
using Greenlight.Movies;
using Greenlight.Users;
using Microsoft.AspNetCore.Cors.Infrastructure;
using Microsoft.AspNetCore.Http.Timeouts;
using Microsoft.AspNetCore.RateLimiting;
using Microsoft.Extensions.Options;
using Npgsql;
using OpenTelemetry;
using OpenTelemetry.Metrics;
using OpenTelemetry.Resources;
using OpenTelemetry.Trace;

namespace Greenlight;

public static class DependencyInjection
{
    public const string CorsPolicy = "TrustedOrigins";

    public static void AddApi(this WebApplicationBuilder builder)
    {
        var services = builder.Services;

        services.AddOptions<ApiOptions>()
            .BindConfiguration(ApiOptions.SectionName)
            .Validate(o => o.IsValid(), "Api options are invalid: timeouts, windows and limits must be positive.")
            .ValidateOnStart();

        services.ConfigureHttpJsonOptions(o =>
        {
            o.SerializerOptions.PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower;
            o.SerializerOptions.UnmappedMemberHandling = JsonUnmappedMemberHandling.Disallow;
            o.SerializerOptions.WriteIndented = true;
        });

        // Surface JSON/binding failures as exceptions so GlobalExceptionHandler can describe them.
        services.Configure<RouteHandlerOptions>(o => o.ThrowOnBadRequest = true);

        services.AddProblemDetails(o => o.CustomizeProblemDetails = ctx =>
        {
            var problem = ctx.ProblemDetails;
            problem.Instance ??= ctx.HttpContext.Request.Path;
            problem.Detail ??= problem.Status switch
            {
                StatusCodes.Status404NotFound => Problems.NotFoundDetail,
                StatusCodes.Status405MethodNotAllowed =>
                    $"the {ctx.HttpContext.Request.Method} method is not supported for this resource",
                _ => null,
            };
            if (System.Diagnostics.Activity.Current?.TraceId is { } traceId)
            {
                problem.Extensions["trace_id"] = traceId.ToString();
            }
        });
        services.AddExceptionHandler<GlobalExceptionHandler>();

        services.AddRequestTimeouts();
        services.AddOptions<RequestTimeoutOptions>()
            .Configure<IOptions<ApiOptions>>((o, api) =>
                o.DefaultPolicy = new RequestTimeoutPolicy { Timeout = api.Value.RequestTimeout });

        services.AddCors();
        services.AddOptions<CorsOptions>()
            .Configure<IOptions<ApiOptions>>((o, api) => o.AddPolicy(CorsPolicy, policy => policy
                .WithOrigins(api.Value.Cors.TrustedOrigins)
                .WithMethods("GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS")
                .WithHeaders("Accept", "Authorization", "Content-Type", "X-CSRF-Token")
                .WithExposedHeaders("Link")
                .SetPreflightMaxAge(TimeSpan.FromSeconds(300))));

        services.AddRateLimiter(o =>
        {
            o.RejectionStatusCode = StatusCodes.Status429TooManyRequests;
            o.OnRejected = (ctx, _) =>
            {
                var problems = ctx.HttpContext.RequestServices.GetRequiredService<IProblemDetailsService>();
                return problems.WriteAsync(new ProblemDetailsContext
                {
                    HttpContext = ctx.HttpContext,
                    ProblemDetails = { Status = StatusCodes.Status429TooManyRequests, Detail = "too many requests" },
                });
            };
        });
        services.AddOptions<RateLimiterOptions>()
            .Configure<IOptions<ApiOptions>>((o, api) =>
            {
                var limits = api.Value.RateLimiting;
                o.GlobalLimiter = PartitionedRateLimiter.Create<HttpContext, string>(ctx =>
                    RateLimitPartition.GetFixedWindowLimiter(
                        ctx.Connection.RemoteIpAddress?.ToString() ?? "unknown",
                        _ => new FixedWindowRateLimiterOptions
                        { PermitLimit = limits.PermitLimit, Window = limits.Window }));
            });

        services.AddOpenApi();
    }

    public static void AddData(this WebApplicationBuilder builder)
    {
        var connectionString = builder.Configuration.GetConnectionString("Greenlight")
                               ?? throw new InvalidOperationException(
                                   "Connection string 'Greenlight' is not configured (ConnectionStrings:Greenlight in appsettings.{Environment}.json).");

        // Registers NpgsqlDataSource (singleton, wired to ILoggerFactory) and a scoped NpgsqlConnection.
        builder.Services.AddNpgsqlDataSource(connectionString, connectionLifetime: ServiceLifetime.Scoped);

        builder.Services.AddScoped<IUnitOfWork, UnitOfWork>();
        builder.Services.AddScoped<IMovieRepository, MovieRepository>();
        builder.Services.AddScoped<IUserRepository, UserRepository>();
        builder.Services.AddScoped<ITokenRepository, TokenRepository>();
        builder.Services.AddScoped<IPermissionRepository, PermissionRepository>();
    }

    public static void AddObservability(this WebApplicationBuilder builder)
    {
        var otel = builder.Services.AddOpenTelemetry()
            .ConfigureResource(r => r
                .AddService(Telemetry.ServiceName, serviceVersion: Telemetry.Version)
                .AddAttributes([
                    new("deployment.environment.name", builder.Environment.EnvironmentName.ToLowerInvariant())
                ]))
            .WithTracing(t => t
                .AddSource(Telemetry.SourceName)
                .AddAspNetCoreInstrumentation()
                .AddHttpClientInstrumentation()
                .AddNpgsql())
            .WithMetrics(m => m
                .AddMeter("Npgsql")
                .AddAspNetCoreInstrumentation()
                .AddHttpClientInstrumentation()
                .AddRuntimeInstrumentation())
            .WithLogging(configureBuilder: null, configureOptions: o =>
            {
                o.IncludeFormattedMessage = true;
                o.IncludeScopes = true;
            });

        // Standard OTel env/config key; export only when an endpoint is configured.
        if (!string.IsNullOrWhiteSpace(builder.Configuration["OTEL_EXPORTER_OTLP_ENDPOINT"]))
        {
            otel.UseOtlpExporter();
        }
    }
}
