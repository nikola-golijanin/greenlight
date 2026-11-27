using Dapper;
using Npgsql;

namespace Greenlight.Movies;

public interface IMovieRepository
{
    Task InsertAsync(Movie movie, CancellationToken ct = default);
    Task<Movie?> GetByIdAsync(long id, CancellationToken ct = default);
    Task<IReadOnlyList<Movie>> GetAllAsync(CancellationToken ct = default);

    /// <summary>Optimistic-locking update. Returns <c>false</c> on an edit conflict.</summary>
    Task<bool> UpdateAsync(Movie movie, CancellationToken ct = default);

    /// <summary>Returns <c>false</c> when no movie with that id existed.</summary>
    Task<bool> DeleteAsync(long id, CancellationToken ct = default);
}

internal sealed class MovieRepository(NpgsqlConnection connection) : IMovieRepository
{
    private const string Columns = "id, created_at, title, year, runtime, genres, version";

    public async Task InsertAsync(Movie movie, CancellationToken ct = default)
    {
        var inserted = await connection.QuerySingleAsync<InsertedRow>(new CommandDefinition(
            """
            INSERT INTO movies (title, year, runtime, genres)
            VALUES (@Title, @Year, @Runtime, @Genres)
            RETURNING id, created_at, version
            """,
            new { movie.Title, movie.Year, Runtime = movie.Runtime.Minutes, movie.Genres },
            cancellationToken: ct));

        movie.Id = inserted.Id;
        movie.CreatedAt = inserted.CreatedAt;
        movie.Version = inserted.Version;
    }

    public async Task<Movie?> GetByIdAsync(long id, CancellationToken ct = default)
    {
        var row = await connection.QuerySingleOrDefaultAsync<MovieRow>(new CommandDefinition(
            $"SELECT {Columns} FROM movies WHERE id = @id", new { id }, cancellationToken: ct));
        return row?.ToMovie();
    }

    public async Task<IReadOnlyList<Movie>> GetAllAsync(CancellationToken ct = default)
    {
        var rows = await connection.QueryAsync<MovieRow>(new CommandDefinition(
            $"SELECT {Columns} FROM movies ORDER BY id", cancellationToken: ct));
        return [.. rows.Select(r => r.ToMovie())];
    }

    public async Task<bool> UpdateAsync(Movie movie, CancellationToken ct = default)
    {
        var version = await connection.ExecuteScalarAsync<int?>(new CommandDefinition(
            """
            UPDATE movies
            SET title = @Title, year = @Year, runtime = @Runtime, genres = @Genres, version = version + 1
            WHERE id = @Id AND version = @Version
            RETURNING version
            """,
            new { movie.Id, movie.Title, movie.Year, Runtime = movie.Runtime.Minutes, movie.Genres, movie.Version },
            cancellationToken: ct));

        if (version is null)
        {
            return false;
        }

        movie.Version = version.Value;
        return true;
    }

    public async Task<bool> DeleteAsync(long id, CancellationToken ct = default) =>
        await connection.ExecuteAsync(new CommandDefinition(
            "DELETE FROM movies WHERE id = @id", new { id }, cancellationToken: ct)) > 0;

    private sealed class InsertedRow
    {
        public long Id { get; init; }
        public DateTime CreatedAt { get; init; }
        public int Version { get; init; }
    }

    private sealed class MovieRow
    {
        public long Id { get; init; }
        public DateTime CreatedAt { get; init; }
        public string Title { get; init; } = string.Empty;
        public int Year { get; init; }
        public int Runtime { get; init; }
        public string[] Genres { get; init; } = [];
        public int Version { get; init; }

        public Movie ToMovie() => new()
        {
            Id = Id,
            CreatedAt = new DateTimeOffset(DateTime.SpecifyKind(CreatedAt, DateTimeKind.Utc)),
            Title = Title,
            Year = Year,
            Runtime = new Runtime(Runtime),
            Genres = Genres,
            Version = Version,
        };
    }
}
