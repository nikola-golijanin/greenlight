namespace Greenlight.Users;

/// <summary>Delivers the activation token to a newly registered user.</summary>
public interface IActivationNotifier
{
    Task NotifyAsync(User user, Token token, CancellationToken ct = default);
}

/// <summary>Stand-in for the Go app's welcome email: the token is written to the log to copy-paste.</summary>
internal sealed class LoggingActivationNotifier(ILogger<LoggingActivationNotifier> logger) : IActivationNotifier
{
    public Task NotifyAsync(User user, Token token, CancellationToken ct = default)
    {
        logger.LogInformation(
            "Activation token for user {UserId}: {ActivationToken} (expires {Expiry:O})",
            user.Id, token.Plaintext, token.Expiry);
        return Task.CompletedTask;
    }
}
