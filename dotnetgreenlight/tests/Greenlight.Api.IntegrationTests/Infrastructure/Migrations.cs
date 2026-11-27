using Npgsql;

namespace Greenlight.IntegrationTests.Infrastructure;

/// <summary>
/// Applies the "-- +goose Up" section of each migration in internal/data/migrations
/// (linked into the test output), the same way cmd/api/testmain_test.go does.
/// </summary>
internal static class Migrations
{
    private const string UpMarker = "-- +goose Up";
    private const string DownMarker = "-- +goose Down";

    public static async Task ApplyAsync(string connectionString)
    {
        var directory = Path.Combine(AppContext.BaseDirectory, "migrations");
        var files = Directory.GetFiles(directory, "*.sql").Order(StringComparer.Ordinal).ToArray();
        if (files.Length == 0)
        {
            throw new InvalidOperationException($"No migrations found in {directory}");
        }

        await using var connection = new NpgsqlConnection(connectionString);
        await connection.OpenAsync();

        await using (var extension = new NpgsqlCommand("CREATE EXTENSION IF NOT EXISTS citext", connection))
        {
            await extension.ExecuteNonQueryAsync();
        }

        await connection.ReloadTypesAsync();

        foreach (var file in files)
        {
            var up = ExtractUp(await File.ReadAllTextAsync(file));
            if (string.IsNullOrWhiteSpace(up))
            {
                continue;
            }

            await using var command = new NpgsqlCommand(up, connection);
            await command.ExecuteNonQueryAsync();
        }
    }

    internal static string ExtractUp(string migration)
    {
        var start = migration.IndexOf(UpMarker, StringComparison.Ordinal);
        if (start < 0)
        {
            return string.Empty;
        }

        start += UpMarker.Length;
        var end = migration.IndexOf(DownMarker, start, StringComparison.Ordinal);
        var up = end < 0 ? migration[start..] : migration[start..end];

        return string.Join('\n', up.Split('\n')
            .Where(line => !line.TrimStart().StartsWith("-- +goose", StringComparison.Ordinal)));
    }
}
