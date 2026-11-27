using System.Data;
using System.Diagnostics;
using Greenlight.Common.Observability;
using Greenlight.Movies;
using Greenlight.Users;
using Npgsql;

namespace Greenlight.Common.Data;

/// <summary>
/// Single entry point to the data layer for a request. Repositories share one connection; wrap
/// writes that must succeed or fail together in <see cref="BeginTransactionAsync"/> /
/// <see cref="CommitAsync"/>. A transaction that is still open when the request scope ends
/// (early return, exception) is rolled back.
/// </summary>
public interface IUnitOfWork
{
    IMovieRepository Movies { get; }
    IUserRepository Users { get; }
    ITokenRepository Tokens { get; }
    IPermissionRepository Permissions { get; }

    bool HasActiveTransaction { get; }

    Task BeginTransactionAsync(CancellationToken ct = default);
    Task CommitAsync(CancellationToken ct = default);
    Task RollbackAsync(CancellationToken ct = default);
}

/// <remarks>
/// Repositories and this class receive the same scoped <see cref="NpgsqlConnection"/> from DI.
/// Npgsql runs every command on a connection inside that connection's active transaction, so
/// repositories don't need the transaction passed to them. Outside a transaction, Dapper opens and
/// closes the connection around each call.
/// </remarks>
internal sealed class UnitOfWork(
    NpgsqlConnection connection,
    IMovieRepository movies,
    IUserRepository users,
    ITokenRepository tokens,
    IPermissionRepository permissions) : IUnitOfWork, IAsyncDisposable
{
    private NpgsqlTransaction? _transaction;
    private Activity? _activity;
    private bool _openedConnection;

    public IMovieRepository Movies => movies;
    public IUserRepository Users => users;
    public ITokenRepository Tokens => tokens;
    public IPermissionRepository Permissions => permissions;

    public bool HasActiveTransaction => _transaction is not null;

    public async Task BeginTransactionAsync(CancellationToken ct = default)
    {
        if (HasActiveTransaction)
        {
            throw new InvalidOperationException("A transaction is already in progress for this unit of work.");
        }

        // The connection must stay open for the whole transaction.
        if (connection.State != ConnectionState.Open)
        {
            await connection.OpenAsync(ct);
            _openedConnection = true;
        }

        _activity = Telemetry.ActivitySource.StartActivity("db.transaction");
        _transaction = await connection.BeginTransactionAsync(ct);
    }

    public async Task CommitAsync(CancellationToken ct = default)
    {
        var transaction = _transaction
            ?? throw new InvalidOperationException("There is no active transaction to commit.");

        await transaction.CommitAsync(ct);
        await EndAsync("commit");
    }

    public async Task RollbackAsync(CancellationToken ct = default)
    {
        if (_transaction is { } transaction)
        {
            await transaction.RollbackAsync(ct);
            await EndAsync("rollback");
        }
    }

    public async ValueTask DisposeAsync()
    {
        if (!HasActiveTransaction)
        {
            return;
        }

        try
        {
            await RollbackAsync();
        }
        catch (Exception e) when (e is NpgsqlException or InvalidOperationException)
        {
            // The connection may already be broken; disposing it discards the transaction anyway.
            await EndAsync("rollback");
        }
    }

    private async Task EndAsync(string outcome)
    {
        if (_transaction is { } transaction)
        {
            await transaction.DisposeAsync();
            _transaction = null;
        }

        // Return the connection to the pool early; later Dapper calls reopen it on demand.
        if (_openedConnection)
        {
            await connection.CloseAsync();
            _openedConnection = false;
        }

        _activity?.SetTag("db.transaction.outcome", outcome);
        _activity?.Dispose();
        _activity = null;
    }
}
