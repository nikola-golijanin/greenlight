namespace Greenlight.Users;

public sealed class User
{
    public long Id { get; set; }
    public DateTime CreatedAt { get; set; }
    public string Name { get; set; } = string.Empty;
    public string Email { get; set; } = string.Empty;
    public byte[] PasswordHash { get; set; } = [];
    public bool Activated { get; set; }
    public int Version { get; set; }
}

public sealed record UserResponse(long Id, DateTimeOffset CreatedAt, string Name, string Email, bool Activated)
{
    public static UserResponse From(User user) => new(
        user.Id,
        new DateTimeOffset(DateTime.SpecifyKind(user.CreatedAt, DateTimeKind.Utc)),
        user.Name,
        user.Email,
        user.Activated);
}

public sealed class DuplicateEmailException() : Exception("a user with this email already exists");
