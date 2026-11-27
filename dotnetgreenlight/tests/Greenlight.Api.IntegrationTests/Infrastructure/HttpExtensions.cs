using System.Net.Http.Json;
using System.Text;
using System.Text.Json;

namespace Greenlight.IntegrationTests.Infrastructure;

internal static class HttpExtensions
{
    public static readonly JsonSerializerOptions Json = new(JsonSerializerDefaults.Web)
    {
        PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower,
    };

    public static Task<HttpResponseMessage> PostJsonAsync(this HttpClient client, string url, object body, CancellationToken ct) =>
        client.PostAsJsonAsync(url, body, Json, ct);

    public static Task<HttpResponseMessage> PutJsonAsync(this HttpClient client, string url, object body, CancellationToken ct) =>
        client.PutAsJsonAsync(url, body, Json, ct);

    public static Task<HttpResponseMessage> PatchJsonAsync(this HttpClient client, string url, object body, CancellationToken ct) =>
        client.PatchAsJsonAsync(url, body, Json, ct);

    public static Task<HttpResponseMessage> PostRawAsync(this HttpClient client, string url, string json, CancellationToken ct) =>
        client.PostAsync(url, new StringContent(json, Encoding.UTF8, "application/json"), ct);

    public static async Task<T> ReadAsync<T>(this HttpResponseMessage response, CancellationToken ct) =>
        (await response.Content.ReadFromJsonAsync<T>(Json, ct))!;

    public static async Task<JsonElement> ReadJsonAsync(this HttpResponseMessage response, CancellationToken ct) =>
        (await response.Content.ReadFromJsonAsync<JsonElement>(Json, ct))!;

    /// <summary>Asserts a ProblemDetails response and returns its body.</summary>
    public static async Task<JsonElement> ShouldBeProblemAsync(this HttpResponseMessage response, int status, CancellationToken ct)
    {
        ((int)response.StatusCode).ShouldBe(status, await response.Content.ReadAsStringAsync(ct));
        response.Content.Headers.ContentType?.MediaType.ShouldBe("application/problem+json");
        var body = await response.ReadJsonAsync(ct);
        body.GetProperty("status").GetInt32().ShouldBe(status);
        return body;
    }

    public static string[] Errors(this JsonElement problem, string field) =>
        [.. problem.GetProperty("errors").GetProperty(field).EnumerateArray().Select(e => e.GetString()!)];
}

/// <summary>Shapes of the API's JSON responses (snake_case on the wire).</summary>
public sealed record MovieDto(long Id, string Title, int Year, string Runtime, string[] Genres, int Version)
{
    // Value equality for the genres array so whole DTOs can be compared in assertions.
    public bool Equals(MovieDto? other) =>
        other is not null && (Id, Title, Year, Runtime, Version) == (other.Id, other.Title, other.Year, other.Runtime, other.Version)
        && Genres.SequenceEqual(other.Genres);

    public override int GetHashCode() => HashCode.Combine(Id, Title, Year, Runtime, Version);

    public override string ToString() => $"MovieDto {{ Id = {Id}, Title = {Title}, Year = {Year}, Runtime = {Runtime}, Genres = [{string.Join(", ", Genres)}], Version = {Version} }}";
}

public sealed record UserDto(long Id, DateTimeOffset CreatedAt, string Name, string Email, bool Activated);
