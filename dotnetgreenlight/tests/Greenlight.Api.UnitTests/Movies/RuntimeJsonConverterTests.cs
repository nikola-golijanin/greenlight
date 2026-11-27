using System.Text.Json;
using Greenlight.Movies;

namespace Greenlight.UnitTests.Movies;

public sealed class RuntimeJsonConverterTests
{
    [Theory]
    [InlineData(107, "\"107 mins\"")]
    [InlineData(0, "\"0 mins\"")]
    [InlineData(1, "\"1 mins\"")]
    public void Serializes_as_minutes_string(int minutes, string expected) =>
        JsonSerializer.Serialize(new Runtime(minutes)).ShouldBe(expected);

    [Theory]
    [InlineData("\"107 mins\"", 107)]
    [InlineData("\"0 mins\"", 0)]
    [InlineData("\"-5 mins\"", -5)]
    public void Deserializes_valid_values(string json, int expected) =>
        JsonSerializer.Deserialize<Runtime>(json).Minutes.ShouldBe(expected);

    [Theory]
    [InlineData("107")]               // unquoted number
    [InlineData("\"107\"")]           // number as string without unit
    [InlineData("\"107 minutes\"")]   // wrong unit
    [InlineData("\"abc mins\"")]      // non-numeric
    [InlineData("\"107  mins\"")]     // double space
    [InlineData("\"\"")]              // empty string
    [InlineData("null")]
    public void Rejects_invalid_formats(string json)
    {
        var ex = Should.Throw<JsonException>(() => JsonSerializer.Deserialize<Runtime>(json));
        ex.Message.ShouldStartWith(RuntimeJsonConverter.InvalidFormatMessage);
    }

    [Fact]
    public void Round_trips_inside_an_object()
    {
        var json = JsonSerializer.Serialize(new { runtime = new Runtime(42) });
        json.ShouldBe("{\"runtime\":\"42 mins\"}");
    }
}
