using System.Globalization;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace Greenlight.Movies;

/// <summary>Movie runtime in minutes, serialized as <c>"107 mins"</c>.</summary>
[JsonConverter(typeof(RuntimeJsonConverter))]
public readonly record struct Runtime(int Minutes)
{
    public override string ToString() => string.Create(CultureInfo.InvariantCulture, $"{Minutes} mins");
}

public sealed class RuntimeJsonConverter : JsonConverter<Runtime>
{
    public const string InvalidFormatMessage = "invalid runtime format";

    public override Runtime Read(ref Utf8JsonReader reader, Type typeToConvert, JsonSerializerOptions options)
    {
        if (reader.TokenType != JsonTokenType.String)
        {
            throw new JsonException(InvalidFormatMessage);
        }

        var parts = reader.GetString()!.Split(' ');
        if (parts.Length != 2 || parts[1] != "mins" ||
            !int.TryParse(parts[0], NumberStyles.AllowLeadingSign, CultureInfo.InvariantCulture, out var minutes))
        {
            throw new JsonException(InvalidFormatMessage);
        }

        return new Runtime(minutes);
    }

    public override void Write(Utf8JsonWriter writer, Runtime value, JsonSerializerOptions options) =>
        writer.WriteStringValue(value.ToString());
}
