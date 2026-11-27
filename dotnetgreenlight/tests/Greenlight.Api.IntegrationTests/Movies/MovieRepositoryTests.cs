using Greenlight.IntegrationTests.Infrastructure;
using Greenlight.Movies;

namespace Greenlight.IntegrationTests.Movies;

public sealed class MovieRepositoryTests(GreenlightApiFactory factory) : IntegrationTest(factory)
{
    private static Movie NewMovie() => new() { Title = "Heat", Year = 1995, Runtime = new Runtime(170), Genres = ["crime"] };

    [Fact]
    public async Task Insert_then_get_round_trips_all_columns()
    {
        var movie = NewMovie();

        var loaded = await WithUnitOfWorkAsync(async uow =>
        {
            await uow.Movies.InsertAsync(movie, Ct);
            return await uow.Movies.GetByIdAsync(movie.Id, Ct);
        });

        loaded.ShouldNotBeNull();
        loaded.Id.ShouldBe(movie.Id);
        loaded.Title.ShouldBe("Heat");
        loaded.Runtime.ShouldBe(new Runtime(170));
        loaded.Genres.ShouldBe(["crime"]);
        loaded.Version.ShouldBe(1);
        loaded.CreatedAt.ShouldBeInRange(DateTimeOffset.UtcNow.AddMinutes(-1), DateTimeOffset.UtcNow.AddMinutes(1));
    }

    [Fact]
    public async Task Update_with_stale_version_is_an_edit_conflict()
    {
        var movie = NewMovie();
        await WithUnitOfWorkAsync(uow => uow.Movies.InsertAsync(movie, Ct));

        // Two "requests" load the same version...
        var first = await WithUnitOfWorkAsync(uow => uow.Movies.GetByIdAsync(movie.Id, Ct));
        var second = await WithUnitOfWorkAsync(uow => uow.Movies.GetByIdAsync(movie.Id, Ct));

        first!.Title = "first writer";
        second!.Title = "second writer";

        // ...the first update wins, the second sees a version mismatch.
        (await WithUnitOfWorkAsync(uow => uow.Movies.UpdateAsync(first, Ct))).ShouldBeTrue();
        (await WithUnitOfWorkAsync(uow => uow.Movies.UpdateAsync(second, Ct))).ShouldBeFalse();

        var stored = await WithUnitOfWorkAsync(uow => uow.Movies.GetByIdAsync(movie.Id, Ct));
        stored!.Title.ShouldBe("first writer");
        stored.Version.ShouldBe(2);
    }
}
