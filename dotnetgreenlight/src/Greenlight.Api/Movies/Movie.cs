namespace Greenlight.Movies;

public sealed class Movie : IMovieFields
{
    public long Id { get; set; }
    public DateTimeOffset CreatedAt { get; set; }
    public string Title { get; set; } = string.Empty;
    public int Year { get; set; }
    public Runtime Runtime { get; set; }
    public string[] Genres { get; set; } = [];
    public int Version { get; set; }
}

public sealed record MovieResponse(long Id, string Title, int Year, Runtime Runtime, string[] Genres, int Version)
{
    public static MovieResponse From(Movie movie) =>
        new(movie.Id, movie.Title, movie.Year, movie.Runtime, movie.Genres, movie.Version);
}
