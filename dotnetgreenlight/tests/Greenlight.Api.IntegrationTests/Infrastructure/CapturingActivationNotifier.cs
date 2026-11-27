using System.Collections.Concurrent;
using Greenlight.Users;

namespace Greenlight.IntegrationTests.Infrastructure;

public sealed class CapturingActivationNotifier : IActivationNotifier
{
    private readonly ConcurrentQueue<(User User, Token Token)> _sent = new();

    public Task NotifyAsync(User user, Token token, CancellationToken ct = default)
    {
        _sent.Enqueue((user, token));
        return Task.CompletedTask;
    }

    public (User User, Token Token) Single() => _sent.ShouldHaveSingleItem();

    public int Count => _sent.Count;

    public void Clear() => _sent.Clear();
}
