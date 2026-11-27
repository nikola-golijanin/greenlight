using FluentValidation;
using Greenlight;
using Greenlight.Common.Api;
using Greenlight.Users;
using Microsoft.Extensions.Options;

var builder = WebApplication.CreateBuilder(args);

// Catch lifetime mistakes (e.g. a singleton capturing the scoped connection/unit of work)
// in every environment, not only Development.
builder.Host.UseDefaultServiceProvider(o =>
{
    o.ValidateScopes = true;
    o.ValidateOnBuild = true;
});

builder.AddApi();
builder.AddData();
builder.AddObservability();

builder.Services.AddValidatorsFromAssemblyContaining<Program>(includeInternalTypes: true);
builder.Services.AddSingleton(TimeProvider.System);
builder.Services.AddSingleton<IActivationNotifier, LoggingActivationNotifier>();

var app = builder.Build();

app.UseExceptionHandler();
app.UseStatusCodePages();
app.UseRouting();
app.UseCors(DependencyInjection.CorsPolicy);
if (app.Services.GetRequiredService<IOptions<ApiOptions>>().Value.RateLimiting.Enabled)
{
    app.UseRateLimiter();
}

app.UseRequestTimeouts();

if (!app.Environment.IsProduction())
{
    app.MapOpenApi();
}

app.MapEndpoints();

app.Run();

public partial class Program;
