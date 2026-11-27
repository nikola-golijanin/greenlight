using System.Text;
using System.Text.RegularExpressions;
using FluentValidation;

namespace Greenlight.Users;

public static partial class UserRules
{
    // Same pattern as validator.EmailRX in the Go app.
    [GeneratedRegex("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")]
    public static partial Regex EmailRegex();

    public static IRuleBuilderOptions<T, string> UserName<T>(this IRuleBuilder<T, string> rule) =>
        rule.NotEmpty().WithMessage("must be provided")
            .Must(n => Encoding.UTF8.GetByteCount(n) <= 500).WithMessage("must not be more than 500 bytes long");

    public static IRuleBuilderOptions<T, string> Email<T>(this IRuleBuilder<T, string> rule) =>
        rule.NotEmpty().WithMessage("must be provided")
            .Matches(EmailRegex()).WithMessage("must be a valid email address");

    public static IRuleBuilderOptions<T, string> PlaintextPassword<T>(this IRuleBuilder<T, string> rule) =>
        rule.NotEmpty().WithMessage("must be provided")
            .Must(p => Encoding.UTF8.GetByteCount(p) >= 8).WithMessage("must be at least 8 bytes long")
            .Must(p => Encoding.UTF8.GetByteCount(p) <= 72).WithMessage("must not be more than 72 bytes long");

    public static IRuleBuilderOptions<T, string> TokenPlaintext<T>(this IRuleBuilder<T, string> rule) =>
        rule.NotEmpty().WithMessage("must be provided")
            .Length(Tokens.PlaintextLength).WithMessage($"must be {Tokens.PlaintextLength} bytes long");
}
