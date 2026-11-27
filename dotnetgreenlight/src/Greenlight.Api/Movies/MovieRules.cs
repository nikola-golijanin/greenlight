using System.Text;
using FluentValidation;

namespace Greenlight.Movies;

/// <summary>The shape shared by a create request and a (merged) movie; validated by the same rules.</summary>
public interface IMovieFields
{
    string Title { get; }
    int Year { get; }
    Runtime Runtime { get; }
    string[]? Genres { get; }
}

/// <summary>Port of <c>movies.ValidateMovie</c> from the Go app.</summary>
public abstract class MovieFieldsValidator<T> : AbstractValidator<T> where T : IMovieFields
{
    public const int FirstFilmYear = 1888;

    protected MovieFieldsValidator()
    {
        RuleFor(x => x.Title)
            .NotEmpty().WithMessage("must be provided")
            .Must(t => Encoding.UTF8.GetByteCount(t) <= 500).WithMessage("must not be more than 500 bytes long");

        RuleFor(x => x.Year)
            .NotEqual(0).WithMessage("must be provided")
            .GreaterThanOrEqualTo(FirstFilmYear).WithMessage($"must be greater than {FirstFilmYear}")
            .Must(y => y <= DateTime.UtcNow.Year).WithMessage("must not be in the future");

        RuleFor(x => x.Runtime)
            .Must(r => r.Minutes != 0).WithMessage("must be provided")
            .Must(r => r.Minutes > 0).WithMessage("must be a positive integer");

        RuleFor(x => x.Genres)
            .NotNull().WithMessage("must be provided")
            .Must(g => g!.Length >= 1).WithMessage("must contain at least 1 genre")
            .Must(g => g!.Length <= 5).WithMessage("must not contain more than 5 genres")
            .Must(g => g!.Distinct(StringComparer.Ordinal).Count() == g!.Length).WithMessage("must not contain duplicate values");
    }
}

public sealed class MovieValidator : MovieFieldsValidator<Movie>;
