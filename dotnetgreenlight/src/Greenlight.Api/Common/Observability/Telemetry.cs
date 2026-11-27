using System.Diagnostics;
using System.Reflection;

namespace Greenlight.Common.Observability;

public static class Telemetry
{
    public const string ServiceName = "greenlight-api";
    public const string SourceName = "Greenlight.Api";

    public static readonly ActivitySource ActivitySource = new(SourceName, Version);

    public static string Version { get; } =
        typeof(Telemetry).Assembly.GetCustomAttribute<AssemblyInformationalVersionAttribute>()?.InformationalVersion
        ?? typeof(Telemetry).Assembly.GetName().Version?.ToString()
        ?? "unknown";
}
