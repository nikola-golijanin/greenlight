using System.Runtime.CompilerServices;
using System.Text.Json;
using FluentValidation;

namespace Greenlight.Common;

internal static class GlobalConfiguration
{
    /// <summary>Process-wide library settings; runs once when the assembly loads (also in tests).</summary>
    [ModuleInitializer]
    internal static void Initialize()
    {
        Dapper.DefaultTypeMap.MatchNamesWithUnderscores = true;

        // Like the Go validator: report only the first failure per field, keyed by its JSON name.
        ValidatorOptions.Global.DefaultRuleLevelCascadeMode = CascadeMode.Stop;
        ValidatorOptions.Global.PropertyNameResolver = (_, member, _) =>
            member is null ? null : JsonNamingPolicy.SnakeCaseLower.ConvertName(member.Name);
    }
}
