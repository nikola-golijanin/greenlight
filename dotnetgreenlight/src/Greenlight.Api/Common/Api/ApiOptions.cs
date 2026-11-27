namespace Greenlight.Common.Api;

/// <summary>HTTP pipeline settings, bound from the <c>Api</c> configuration section.</summary>
public sealed class ApiOptions
{
    public const string SectionName = "Api";

    /// <summary>Per-request timeout (the Go app uses a 3s context timeout per handler).</summary>
    public TimeSpan RequestTimeout { get; set; } = TimeSpan.FromSeconds(3);

    public CorsSettings Cors { get; set; } = new();

    public RateLimitingSettings RateLimiting { get; set; } = new();

    public sealed class CorsSettings
    {
        public string[] TrustedOrigins { get; set; } = [];
    }

    public sealed class RateLimitingSettings
    {
        public bool Enabled { get; set; } = true;

        /// <summary>Requests allowed per client IP within <see cref="Window"/>.</summary>
        public int PermitLimit { get; set; } = 10;

        public TimeSpan Window { get; set; } = TimeSpan.FromSeconds(5);
    }

    internal bool IsValid() =>
        RequestTimeout > TimeSpan.Zero &&
        RateLimiting.PermitLimit > 0 &&
        RateLimiting.Window > TimeSpan.Zero;
}
