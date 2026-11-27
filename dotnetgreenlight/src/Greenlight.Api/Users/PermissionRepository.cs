using Dapper;
using Npgsql;

namespace Greenlight.Users;

public interface IPermissionRepository
{
    Task AddForUserAsync(long userId, string[] codes, CancellationToken ct = default);
    Task<IReadOnlyList<string>> GetAllForUserAsync(long userId, CancellationToken ct = default);
}

internal sealed class PermissionRepository(NpgsqlConnection connection) : IPermissionRepository
{
    public Task AddForUserAsync(long userId, string[] codes, CancellationToken ct = default) =>
        connection.ExecuteAsync(new CommandDefinition(
            """
            INSERT INTO users_permissions (user_id, permission_id)
            SELECT @userId, permissions.id FROM permissions WHERE permissions.code = ANY(@codes)
            """,
            new { userId, codes },
            cancellationToken: ct));

    public async Task<IReadOnlyList<string>> GetAllForUserAsync(long userId, CancellationToken ct = default) =>
        [.. await connection.QueryAsync<string>(new CommandDefinition(
            """
            SELECT permissions.code
            FROM permissions
            INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
            WHERE users_permissions.user_id = @userId
            """,
            new { userId },
            cancellationToken: ct))];
}
