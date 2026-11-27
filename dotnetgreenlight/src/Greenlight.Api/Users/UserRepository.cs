using Dapper;
using Npgsql;

namespace Greenlight.Users;

public interface IUserRepository
{
    /// <exception cref="DuplicateEmailException">The email is already registered.</exception>
    Task InsertAsync(User user, CancellationToken ct = default);

    Task<User?> GetForTokenAsync(string scope, string tokenPlaintext, DateTimeOffset now, CancellationToken ct = default);

    /// <summary>Optimistic-locking update. Returns <c>false</c> on an edit conflict.</summary>
    Task<bool> UpdateAsync(User user, CancellationToken ct = default);
}

internal sealed class UserRepository(NpgsqlConnection connection) : IUserRepository
{
    private const string Columns = "users.id, users.created_at, users.name, users.email, users.password_hash, users.activated, users.version";

    public async Task InsertAsync(User user, CancellationToken ct = default)
    {
        try
        {
            var inserted = await connection.QuerySingleAsync<User>(new CommandDefinition(
                """
                INSERT INTO users (name, email, password_hash, activated)
                VALUES (@Name, @Email, @PasswordHash, @Activated)
                RETURNING id, created_at, version
                """,
                new { user.Name, user.Email, user.PasswordHash, user.Activated },
                cancellationToken: ct));

            user.Id = inserted.Id;
            user.CreatedAt = inserted.CreatedAt;
            user.Version = inserted.Version;
        }
        catch (PostgresException e) when (e is { SqlState: PostgresErrorCodes.UniqueViolation, ConstraintName: "users_email_key" })
        {
            throw new DuplicateEmailException();
        }
    }

    public Task<User?> GetForTokenAsync(string scope, string tokenPlaintext, DateTimeOffset now, CancellationToken ct = default) =>
        connection.QuerySingleOrDefaultAsync<User>(new CommandDefinition(
            $"""
            SELECT {Columns}
            FROM users
            INNER JOIN tokens ON users.id = tokens.user_id
            WHERE tokens.hash = @Hash AND tokens.scope = @Scope AND tokens.expiry > @Now
            """,
            new { Hash = Tokens.Hash(tokenPlaintext), Scope = scope, Now = now.UtcDateTime },
            cancellationToken: ct));

    public async Task<bool> UpdateAsync(User user, CancellationToken ct = default)
    {
        var version = await connection.ExecuteScalarAsync<int?>(new CommandDefinition(
            """
            UPDATE users
            SET name = @Name, email = @Email, password_hash = @PasswordHash, activated = @Activated, version = version + 1
            WHERE id = @Id AND version = @Version
            RETURNING version
            """,
            new { user.Id, user.Name, user.Email, user.PasswordHash, user.Activated, user.Version },
            cancellationToken: ct));

        if (version is null)
        {
            return false;
        }

        user.Version = version.Value;
        return true;
    }
}
