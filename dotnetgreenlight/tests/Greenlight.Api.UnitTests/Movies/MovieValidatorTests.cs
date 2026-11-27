using Greenlight.Movies;

namespace Greenlight.UnitTests.Movies;

public sealed class MovieValidatorTests
{
    private readonly CreateMovieRequestValidator _validator = new();

    private static CreateMovieRequest Valid() => new()
    {
        Title = "Moana",
        Year = 2016,
        Runtime = new Runtime(107),
        Genres = ["animation", "adventure"],
    };

    private Dictionary<string, string[]> Errors(CreateMovieRequest request) =>
        new(_validator.Validate(request).ToDictionary());

    [Fact]
    public void Valid_movie_passes() => _validator.Validate(Valid()).IsValid.ShouldBeTrue();

    [Fact]
    public void Empty_request_reports_every_field_as_missing()
    {
        var errors = Errors(new CreateMovieRequest());

        errors["title"].ShouldBe(["must be provided"]);
        errors["year"].ShouldBe(["must be provided"]);
        errors["runtime"].ShouldBe(["must be provided"]);
        errors["genres"].ShouldBe(["must be provided"]);
    }

    [Theory]
    [InlineData(1887, "must be greater than 1888")]
    [InlineData(3000, "must not be in the future")]
    public void Year_out_of_range(int year, string message) =>
        Errors(Valid() with { Year = year })["year"].ShouldBe([message]);

    [Fact]
    public void Current_year_is_allowed() =>
        _validator.Validate(Valid() with { Year = DateTime.UtcNow.Year }).IsValid.ShouldBeTrue();

    [Fact]
    public void Title_longer_than_500_bytes_fails() =>
        Errors(Valid() with { Title = new string('a', 501) })["title"].ShouldBe(["must not be more than 500 bytes long"]);

    [Fact]
    public void Title_length_is_measured_in_bytes() =>
        // 'č' is 2 bytes in UTF-8, so 251 chars = 502 bytes.
        Errors(Valid() with { Title = new string('č', 251) }).ShouldContainKey("title");

    [Fact]
    public void Negative_runtime_fails() =>
        Errors(Valid() with { Runtime = new Runtime(-1) })["runtime"].ShouldBe(["must be a positive integer"]);

    [Theory]
    [InlineData(new string[0], "must contain at least 1 genre")]
    [InlineData(new[] { "a", "b", "c", "d", "e", "f" }, "must not contain more than 5 genres")]
    [InlineData(new[] { "drama", "drama" }, "must not contain duplicate values")]
    public void Invalid_genres(string[] genres, string message) =>
        Errors(Valid() with { Genres = genres })["genres"].ShouldBe([message]);

    [Fact]
    public void Movie_validator_applies_the_same_rules()
    {
        var result = new MovieValidator().Validate(new Movie { Title = "", Year = 2000, Runtime = new Runtime(90), Genres = ["x"] });

        result.IsValid.ShouldBeFalse();
        result.ToDictionary().Keys.ShouldBe(["title"]);
    }
}
