using System.Net;
using Greenlight.IntegrationTests.Infrastructure;

namespace Greenlight.IntegrationTests.Movies;

public sealed class MoviesEndpointsTests(GreenlightApiFactory factory) : IntegrationTest(factory)
{
    private static readonly object Moana = new { title = "Moana", year = 2016, runtime = "107 mins", genres = new[] { "animation", "adventure" } };

    private async Task<MovieDto> CreateAsync(object? body = null)
    {
        var response = await Client.PostJsonAsync("/v1/movies", body ?? Moana, Ct);
        response.StatusCode.ShouldBe(HttpStatusCode.Created, await response.Content.ReadAsStringAsync(Ct));
        return await response.ReadAsync<MovieDto>(Ct);
    }

    // ---- POST /v1/movies ----

    [Fact]
    public async Task Create_returns_201_with_location_and_movie()
    {
        var response = await Client.PostJsonAsync("/v1/movies", Moana, Ct);

        response.StatusCode.ShouldBe(HttpStatusCode.Created);
        var movie = await response.ReadAsync<MovieDto>(Ct);
        response.Headers.Location!.ToString().ShouldBe($"/v1/movies/{movie.Id}");
        movie.ShouldBe(new MovieDto(movie.Id, "Moana", 2016, "107 mins", ["animation", "adventure"], 1));
    }

    [Fact]
    public async Task Create_response_uses_go_json_shape()
    {
        var response = await Client.PostJsonAsync("/v1/movies", Moana, Ct);
        var json = await response.ReadJsonAsync(Ct);

        json.EnumerateObject().Select(p => p.Name).ShouldBe(["id", "title", "year", "runtime", "genres", "version"]);
    }

    [Fact]
    public async Task Create_with_invalid_fields_returns_422_with_field_errors()
    {
        var response = await Client.PostJsonAsync("/v1/movies",
            new { title = "", year = 1500, runtime = "107 mins", genres = new[] { "drama", "drama" } }, Ct);

        var problem = await response.ShouldBeProblemAsync(422, Ct);
        problem.Errors("title").ShouldBe(["must be provided"]);
        problem.Errors("year").ShouldBe(["must be greater than 1888"]);
        problem.Errors("genres").ShouldBe(["must not contain duplicate values"]);
    }

    [Fact]
    public async Task Create_with_empty_object_reports_missing_fields()
    {
        var response = await Client.PostRawAsync("/v1/movies", "{}", Ct);

        var problem = await response.ShouldBeProblemAsync(422, Ct);
        problem.GetProperty("errors").EnumerateObject().Select(p => p.Name)
            .ShouldBe(["title", "year", "runtime", "genres"], ignoreOrder: true);
    }

    [Theory]
    [InlineData("""{"title":"Moana","year":2016,"runtime":107,"genres":["animation"]}""", "invalid runtime format")]
    [InlineData("""{"title":"Moana","year":2016,"runtime":"107 minutes","genres":["animation"]}""", "invalid runtime format")]
    [InlineData("""{"title":"Moana","rating":5}""", "rating")]
    [InlineData("""{"title":"Moana",""", "")]
    [InlineData("""{"title":123}""", "")]
    public async Task Create_with_malformed_body_returns_400(string json, string detailContains)
    {
        var response = await Client.PostRawAsync("/v1/movies", json, Ct);

        var problem = await response.ShouldBeProblemAsync(400, Ct);
        problem.GetProperty("detail").GetString()!.ShouldContain(detailContains);
    }

    // ---- GET /v1/movies ----

    [Fact]
    public async Task List_returns_all_movies_ordered_by_id()
    {
        var first = await CreateAsync();
        var second = await CreateAsync(new { title = "Black Panther", year = 2018, runtime = "134 mins", genres = new[] { "action" } });

        var response = await Client.GetAsync("/v1/movies?title=panther&genres=action&page=1&page_size=5&sort=-year", Ct);

        response.StatusCode.ShouldBe(HttpStatusCode.OK);
        var movies = await response.ReadAsync<MovieDto[]>(Ct);
        movies.Select(m => m.Id).ShouldBe([first.Id, second.Id]);
    }

    [Fact]
    public async Task List_with_no_movies_returns_empty_array()
    {
        var response = await Client.GetAsync("/v1/movies", Ct);

        response.StatusCode.ShouldBe(HttpStatusCode.OK);
        (await response.ReadAsync<MovieDto[]>(Ct)).ShouldBeEmpty();
    }

    [Theory]
    [InlineData("page=abc", "page", "must be an integer value")]
    [InlineData("page=0", "page", "must be greater than zero")]
    [InlineData("page_size=101", "page_size", "must be a maximum of 100")]
    [InlineData("sort=foo", "sort", "invalid sort value")]
    public async Task List_with_invalid_filters_returns_422(string query, string field, string message)
    {
        var response = await Client.GetAsync($"/v1/movies?{query}", Ct);

        var problem = await response.ShouldBeProblemAsync(422, Ct);
        problem.Errors(field).ShouldBe([message]);
    }

    // ---- GET /v1/movies/{id} ----

    [Fact]
    public async Task Get_returns_the_movie()
    {
        var created = await CreateAsync();

        var response = await Client.GetAsync($"/v1/movies/{created.Id}", Ct);

        response.StatusCode.ShouldBe(HttpStatusCode.OK);
        (await response.ReadAsync<MovieDto>(Ct)).ShouldBe(created);
    }

    [Theory]
    [InlineData("999999")]
    [InlineData("0")]
    [InlineData("-1")]
    [InlineData("abc")]
    public async Task Get_unknown_or_invalid_id_returns_404(string id)
    {
        var response = await Client.GetAsync($"/v1/movies/{id}", Ct);

        var problem = await response.ShouldBeProblemAsync(404, Ct);
        problem.GetProperty("detail").GetString().ShouldBe("the requested resource could not be found");
    }

    // ---- PATCH /v1/movies/{id} ----

    [Fact]
    public async Task Patch_updates_all_fields_and_bumps_version()
    {
        var created = await CreateAsync();

        var response = await Client.PatchJsonAsync($"/v1/movies/{created.Id}",
            new { title = "Movie1", year = 2000, runtime = "90 mins", genres = new[] { "drama" } }, Ct);

        response.StatusCode.ShouldBe(HttpStatusCode.OK);
        var updated = await response.ReadAsync<MovieDto>(Ct);
        updated.ShouldBe(new MovieDto(created.Id, "Movie1", 2000, "90 mins", ["drama"], 2));

        (await (await Client.GetAsync($"/v1/movies/{created.Id}", Ct)).ReadAsync<MovieDto>(Ct)).ShouldBe(updated);
    }

    [Fact]
    public async Task Patch_with_partial_body_keeps_other_fields()
    {
        var created = await CreateAsync();

        var response = await Client.PatchJsonAsync($"/v1/movies/{created.Id}", new { title = "Movie2", year = 2005 }, Ct);

        response.StatusCode.ShouldBe(HttpStatusCode.OK);
        var updated = await response.ReadAsync<MovieDto>(Ct);
        updated.ShouldBe(created with { Title = "Movie2", Year = 2005, Version = 2 });
    }

    [Fact]
    public async Task Patch_with_invalid_merged_movie_returns_422()
    {
        var created = await CreateAsync();

        var response = await Client.PatchJsonAsync($"/v1/movies/{created.Id}", new { genres = Array.Empty<string>() }, Ct);

        var problem = await response.ShouldBeProblemAsync(422, Ct);
        problem.Errors("genres").ShouldBe(["must contain at least 1 genre"]);
    }

    [Fact]
    public async Task Patch_unknown_movie_returns_404()
    {
        var response = await Client.PatchJsonAsync("/v1/movies/999999", new { title = "x" }, Ct);

        await response.ShouldBeProblemAsync(404, Ct);
    }

    // ---- DELETE /v1/movies/{id} ----

    [Fact]
    public async Task Delete_returns_204_then_404()
    {
        var created = await CreateAsync();

        var deleted = await Client.DeleteAsync($"/v1/movies/{created.Id}", Ct);
        deleted.StatusCode.ShouldBe(HttpStatusCode.NoContent);

        await (await Client.GetAsync($"/v1/movies/{created.Id}", Ct)).ShouldBeProblemAsync(404, Ct);
        await (await Client.DeleteAsync($"/v1/movies/{created.Id}", Ct)).ShouldBeProblemAsync(404, Ct);
    }

    [Fact]
    public async Task Delete_invalid_id_returns_404()
    {
        await (await Client.DeleteAsync("/v1/movies/0", Ct)).ShouldBeProblemAsync(404, Ct);
    }

    // ---- routing ----

    [Fact]
    public async Task Unsupported_method_returns_405_problem()
    {
        var response = await Client.PutJsonAsync("/v1/movies/1", new { }, Ct);

        var problem = await response.ShouldBeProblemAsync(405, Ct);
        problem.GetProperty("detail").GetString().ShouldBe("the PUT method is not supported for this resource");
    }

    [Fact]
    public async Task Unknown_route_returns_404_problem()
    {
        await (await Client.GetAsync("/v1/nope", Ct)).ShouldBeProblemAsync(404, Ct);
    }
}
