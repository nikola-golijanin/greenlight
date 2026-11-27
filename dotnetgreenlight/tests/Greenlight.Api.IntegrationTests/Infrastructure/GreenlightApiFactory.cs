using Greenlight.Common.Data;
using Greenlight.Users;
using Microsoft.AspNetCore.Hosting;
using Microsoft.AspNetCore.Mvc.Testing;
using Microsoft.AspNetCore.TestHost;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.DependencyInjection.Extensions;
using Microsoft.Extensions.Time.Testing;
using Npgsql;
using Respawn;
using Testcontainers.PostgreSql;

namespace Greenlight.IntegrationTests.Infrastructure;

/// <summary>
/// Boots the API in-memory against a throwaway PostgreSQL container that has the
/// Go app's Goose migrations applied.
/// </summary>
public sealed class GreenlightApiFactory : WebApplicationFactory<Program>, IAsyncLifetime
{
    private readonly PostgreSqlContainer _postgres = new PostgreSqlBuilder("postgres:17-alpine")
        .WithDatabase("greenlight")
        .WithUsername("greenlight")
        .WithPassword("pa55word")
        .Build();

    private Respawner _respawner = null!;

    public CapturingActivationNotifier ActivationNotifier { get; } = new();

    public FakeTimeProvider Time { get; } = new(DateTimeOffset.UtcNow);

    public string ConnectionString => _postgres.GetConnectionString();

    public async ValueTask InitializeAsync()
    {
        await _postgres.StartAsync();
        await Migrations.ApplyAsync(ConnectionString);

        await using var connection = new NpgsqlConnection(ConnectionString);
        await connection.OpenAsync();
        _respawner = await Respawner.CreateAsync(connection, new RespawnerOptions
        {
            DbAdapter = DbAdapter.Postgres,
            SchemasToInclude = ["public"],
            // permissions holds seed data inserted by a migration.
            TablesToIgnore = ["permissions"],
        });
    }

    public async Task ResetDatabaseAsync()
    {
        await using var connection = new NpgsqlConnection(ConnectionString);
        await connection.OpenAsync();
        await _respawner.ResetAsync(connection);
        ActivationNotifier.Clear();
    }

    protected override void ConfigureWebHost(IWebHostBuilder builder)
    {
        builder.UseEnvironment("Testing");
        builder.UseSetting("ConnectionStrings:Greenlight", ConnectionString);
        builder.UseSetting("Api:RateLimiting:Enabled", "false");

        builder.ConfigureTestServices(services =>
        {
            services.RemoveAll<IActivationNotifier>();
            services.AddSingleton<IActivationNotifier>(ActivationNotifier);
            services.RemoveAll<TimeProvider>();
            services.AddSingleton<TimeProvider>(Time);
        });
    }

    public override async ValueTask DisposeAsync()
    {
        await base.DisposeAsync();
        await _postgres.DisposeAsync();
    }
}

[CollectionDefinition(Name)]
public sealed class ApiCollection : ICollectionFixture<GreenlightApiFactory>
{
    public const string Name = "api";
}

/// <summary>Base class: a clean database and a fresh client for every test.</summary>
[Collection(ApiCollection.Name)]
public abstract class IntegrationTest(GreenlightApiFactory factory) : IAsyncLifetime
{
    protected GreenlightApiFactory Factory { get; } = factory;

    protected HttpClient Client { get; private set; } = null!;

    protected static CancellationToken Ct => TestContext.Current.CancellationToken;

    public async ValueTask InitializeAsync()
    {
        await Factory.ResetDatabaseAsync();
        Client = Factory.CreateClient();
    }

    public ValueTask DisposeAsync()
    {
        Client.Dispose();
        GC.SuppressFinalize(this);
        return ValueTask.CompletedTask;
    }

    /// <summary>Runs <paramref name="action"/> with a fresh request-scoped unit of work.</summary>
    protected async Task<T> WithUnitOfWorkAsync<T>(Func<IUnitOfWork, Task<T>> action)
    {
        await using var scope = Factory.Services.CreateAsyncScope();
        return await action(scope.ServiceProvider.GetRequiredService<IUnitOfWork>());
    }

    protected async Task WithUnitOfWorkAsync(Func<IUnitOfWork, Task> action)
    {
        await using var scope = Factory.Services.CreateAsyncScope();
        await action(scope.ServiceProvider.GetRequiredService<IUnitOfWork>());
    }
}
