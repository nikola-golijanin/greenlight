using Dapper;
using Npgsql;

namespace Greenlight.Users;

public interface ITokenRepository
{
    Task InsertAsync(Token token, CancellationToken ct = default);
    Task DeleteAllForUserAsync(string scope, long userId, CancellationToken ct = default);
}

internal sealed class TokenRepository(NpgsqlConnection connection) : ITokenRepository
{
    public Task InsertAsync(Token token, CancellationToken ct = default) =>
        connection.ExecuteAsync(new CommandDefinition(
            "INSERT INTO tokens (hash, user_id, expiry, scope) VALUES (@Hash, @UserId, @Expiry, @Scope)",
            new { token.Hash, token.UserId, Expiry = token.Expiry.UtcDateTime, token.Scope },
            cancellationToken: ct));

    public Task DeleteAllForUserAsync(string scope, long userId, CancellationToken ct = default) =>
        connection.ExecuteAsync(new CommandDefinition(
            "DELETE FROM tokens WHERE scope = @scope AND user_id = @userId",
            new { scope, userId },
            cancellationToken: ct));
}
