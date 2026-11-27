using Greenlight.Common.Data;
using Greenlight.IntegrationTests.Infrastructure;
using Greenlight.Movies;
using Npgsql;

namespace Greenlight.IntegrationTests.Data;

public sealed class UnitOfWorkTests(GreenlightApiFactory factory) : IntegrationTest(factory)
{
    private static Movie NewMovie(string title, int year = 2000) =>
        new() { Title = title, Year = year, Runtime = new Runtime(90), Genres = ["drama"] };

    private Task<IReadOnlyList<Movie>> AllMoviesAsync() => WithUnitOfWorkAsync(uow => uow.Movies.GetAllAsync(Ct));

    [Fact]
    public async Task Committed_work_is_persisted()
    {
        await WithUnitOfWorkAsync(async uow =>
        {
            await uow.BeginTransactionAsync(Ct);
            await uow.Movies.InsertAsync(NewMovie("a"), Ct);
            await uow.Movies.InsertAsync(NewMovie("b"), Ct);
            await uow.CommitAsync(Ct);
            uow.HasActiveTransaction.ShouldBeFalse();
        });

        (await AllMoviesAsync()).Select(m => m.Title).ShouldBe(["a", "b"]);
    }

    [Fact]
    public async Task Explicit_rollback_discards_work_and_the_unit_of_work_stays_usable()
    {
        await WithUnitOfWorkAsync(async uow =>
        {
            await uow.BeginTransactionAsync(Ct);
            await uow.Movies.InsertAsync(NewMovie("never committed"), Ct);
            (await uow.Movies.GetAllAsync(Ct)).Count.ShouldBe(1); // visible inside the transaction

            await uow.RollbackAsync(Ct);

            (await uow.Movies.GetAllAsync(Ct)).ShouldBeEmpty();
            await uow.Movies.InsertAsync(NewMovie("auto-committed"), Ct); // no transaction: autocommit
        });

        (await AllMoviesAsync()).Select(m => m.Title).ShouldBe(["auto-committed"]);
    }

    [Fact]
    public async Task Uncommitted_transaction_is_rolled_back_when_the_scope_ends()
    {
        await WithUnitOfWorkAsync(async uow =>
        {
            await uow.BeginTransactionAsync(Ct);
            await uow.Movies.InsertAsync(NewMovie("forgotten"), Ct);
            // no commit - e.g. a handler returned early
        });

        (await AllMoviesAsync()).ShouldBeEmpty();
    }

    [Fact]
    public async Task Exception_inside_transaction_rolls_back()
    {
        await Should.ThrowAsync<InvalidOperationException>(() => WithUnitOfWorkAsync(async uow =>
        {
            await uow.BeginTransactionAsync(Ct);
            await uow.Movies.InsertAsync(NewMovie("partial"), Ct);
            throw new InvalidOperationException("boom");
        }));

        (await AllMoviesAsync()).ShouldBeEmpty();
    }

    [Fact]
    public async Task Database_error_rolls_back_earlier_statements()
    {
        await Should.ThrowAsync<PostgresException>(() => WithUnitOfWorkAsync(async uow =>
        {
            await uow.BeginTransactionAsync(Ct);
            await uow.Movies.InsertAsync(NewMovie("ok"), Ct);
            await uow.Movies.InsertAsync(NewMovie("bad", year: 1000), Ct); // violates movies_year_check
            await uow.CommitAsync(Ct);
        }));

        (await AllMoviesAsync()).ShouldBeEmpty();
    }

    [Fact]
    public async Task Nested_transactions_are_rejected()
    {
        await WithUnitOfWorkAsync(async uow =>
        {
            await uow.BeginTransactionAsync(Ct);
            await Should.ThrowAsync<InvalidOperationException>(() => uow.BeginTransactionAsync(Ct));
        });
    }

    [Fact]
    public async Task Commit_without_transaction_throws()
    {
        await WithUnitOfWorkAsync(async uow =>
            await Should.ThrowAsync<InvalidOperationException>(() => uow.CommitAsync(Ct)));
    }

    [Fact]
    public async Task Repositories_share_the_unit_of_work_transaction()
    {
        await WithUnitOfWorkAsync(async uow =>
        {
            await uow.BeginTransactionAsync(Ct);
            var user = new Greenlight.Users.User { Name = "a", Email = "tx@example.com", PasswordHash = [1], Activated = false };
            await uow.Users.InsertAsync(user, Ct);
            await uow.Permissions.AddForUserAsync(user.Id, ["movies:read", "movies:write"], Ct);
            (await uow.Permissions.GetAllForUserAsync(user.Id, Ct)).Count.ShouldBe(2);
            await uow.RollbackAsync(Ct);

            (await uow.Permissions.GetAllForUserAsync(user.Id, Ct)).ShouldBeEmpty();
        });
    }
}
