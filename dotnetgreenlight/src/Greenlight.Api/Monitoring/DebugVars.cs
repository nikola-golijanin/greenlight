using System.Diagnostics;
using Greenlight.Common.Observability;
using Microsoft.AspNetCore.Http.HttpResults;

namespace Greenlight.Monitoring;

/// <summary>
/// Process snapshot, the counterpart of Go's expvar <c>/debug/vars</c>. Request counts, latency,
/// status codes and Npgsql pool stats come from the built-in ASP.NET Core / Npgsql meters,
/// exported via OpenTelemetry (see the "Greenlight API" Grafana dashboard).
/// </summary>
public sealed class DebugVarsEndpoint : IEndpoint
{
    public static void Map(IEndpointRouteBuilder app) =>
        app.MapGet("/debug/vars", Handle)
            .WithName("DebugVars")
            .WithTags("System");

    private static Ok<Dictionary<string, object>> Handle(TimeProvider timeProvider)
    {
        using var process = Process.GetCurrentProcess();
        var gc = GC.GetGCMemoryInfo();

        return TypedResults.Ok(new Dictionary<string, object>
        {
            ["version"] = Telemetry.Version,
            ["timestamp"] = timeProvider.GetUtcNow(),
            ["threadpool"] = new Dictionary<string, long>
            {
                ["thread_count"] = ThreadPool.ThreadCount,
                ["pending_work_items"] = ThreadPool.PendingWorkItemCount,
                ["completed_work_items"] = ThreadPool.CompletedWorkItemCount,
            },
            ["memory"] = new Dictionary<string, long>
            {
                ["working_set_bytes"] = process.WorkingSet64,
                ["gc_heap_size_bytes"] = gc.HeapSizeBytes,
                ["gc_total_allocated_bytes"] = GC.GetTotalAllocatedBytes(),
                ["gc_gen0_collections"] = GC.CollectionCount(0),
                ["gc_gen1_collections"] = GC.CollectionCount(1),
                ["gc_gen2_collections"] = GC.CollectionCount(2),
            },
        });
    }
}
