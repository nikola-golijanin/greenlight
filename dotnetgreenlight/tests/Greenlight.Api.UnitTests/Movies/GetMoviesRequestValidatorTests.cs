using Greenlight.Movies;

namespace Greenlight.UnitTests.Movies;

public sealed class GetMoviesRequestValidatorTests
{
    private readonly GetMoviesRequestValidator _validator = new();

    private Dictionary<string, string[]> Errors(GetMoviesRequest request) =>
        new(_validator.Validate(request).ToDictionary());

    [Fact]
    public void Defaults_are_valid() => _validator.Validate(new GetMoviesRequest()).IsValid.ShouldBeTrue();

    [Theory]
    [MemberData(nameof(SortValues))]
    public void Safelisted_sorts_are_valid(string sort) =>
        _validator.Validate(new GetMoviesRequest { Sort = sort }).IsValid.ShouldBeTrue();

    public static TheoryData<string> SortValues() => [.. GetMoviesRequestValidator.SortSafelist];

    [Fact]
    public void Unknown_sort_fails() =>
        Errors(new GetMoviesRequest { Sort = "created_at" })["sort"].ShouldBe(["invalid sort value"]);

    [Theory]
    [InlineData("abc", "must be an integer value")]
    [InlineData("0", "must be greater than zero")]
    [InlineData("-1", "must be greater than zero")]
    [InlineData("10000001", "must be a maximum of 10 million")]
    public void Invalid_page(string page, string message) =>
        Errors(new GetMoviesRequest { Page = page })["page"].ShouldBe([message]);

    [Theory]
    [InlineData("x", "must be an integer value")]
    [InlineData("0", "must be greater than zero")]
    [InlineData("101", "must be a maximum of 100")]
    public void Invalid_page_size(string pageSize, string message) =>
        Errors(new GetMoviesRequest { PageSize = pageSize })["page_size"].ShouldBe([message]);

    [Fact]
    public void Boundaries_are_valid() =>
        _validator.Validate(new GetMoviesRequest { Page = "10000000", PageSize = "100" }).IsValid.ShouldBeTrue();
}
