using System.Net;
using Greenlight.IntegrationTests.Infrastructure;

namespace Greenlight.IntegrationTests.Monitoring;

public sealed class MonitoringEndpointsTests(GreenlightApiFactory factory) : IntegrationTest(factory)
{
    [Fact]
    public async Task Healthcheck_reports_status_environment_and_version()
    {
        var response = await Client.GetAsync("/v1/healthcheck", Ct);

        response.StatusCode.ShouldBe(HttpStatusCode.OK);
        var json = await response.ReadJsonAsync(Ct);
        json.GetProperty("status").GetString().ShouldBe("available");
        var info = json.GetProperty("system_info");
        info.GetProperty("environment").GetString().ShouldBe("testing");
        info.GetProperty("version").GetString().ShouldNotBeNullOrWhiteSpace();
    }

    [Fact]
    public async Task Debug_vars_exposes_a_runtime_snapshot()
    {
        var response = await Client.GetAsync("/debug/vars", Ct);

        response.StatusCode.ShouldBe(HttpStatusCode.OK);
        var vars = await response.ReadJsonAsync(Ct);
        vars.GetProperty("version").GetString().ShouldNotBeNullOrWhiteSpace();
        vars.GetProperty("timestamp").GetDateTimeOffset().ShouldBe(Factory.Time.GetUtcNow(), TimeSpan.FromSeconds(1));
        vars.GetProperty("threadpool").GetProperty("thread_count").GetInt64().ShouldBeGreaterThan(0);
        vars.GetProperty("memory").GetProperty("working_set_bytes").GetInt64().ShouldBeGreaterThan(0);
        vars.TryGetProperty("total_requests_received", out _).ShouldBeFalse();
    }
}
