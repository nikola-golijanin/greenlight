package movies

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/nikola-golijanin/greenlight/internal/data"
	repo "github.com/nikola-golijanin/greenlight/internal/data/sqlc"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type MovieService struct {
	repository repo.Querier
}

func NewMovieService(repository repo.Querier) MovieService {
	return MovieService{
		repository: repository,
	}
}

func (m MovieService) GetAll(ctx context.Context, title string, genres []string, filters data.Filters) ([]*Movie, error) {
	movies, err := m.repository.GetAllMovies(ctx)
	if err != nil {
		return nil, err
	}

	mvs := make([]*Movie, len(movies))
	for i, m := range movies {
		mvs[i] = &Movie{
			ID:        m.ID,
			CreatedAt: m.CreatedAt.Time,
			Title:     m.Title,
			Year:      m.Year,
			Runtime:   Runtime(m.Runtime),
			Genres:    m.Genres,
			Version:   m.Version,
		}
	}
	return mvs, nil
}

func (m MovieService) Insert(ctx context.Context, movie *Movie) error {
	mv, err := m.repository.InsertMovie(ctx, repo.InsertMovieParams{
		Title:   movie.Title,
		Year:    movie.Year,
		Runtime: int(movie.Runtime),
		Genres:  movie.Genres,
	})
	if err != nil {
		return err
	}

	movie.ID = mv.ID
	movie.Version = mv.Version
	movie.CreatedAt = mv.CreatedAt.Time
	return nil
}

func (m MovieService) GetById(ctx context.Context, id int) (*Movie, error) {
	mv, err := m.repository.GetMovieById(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	movie := &Movie{
		ID:        mv.ID,
		CreatedAt: mv.CreatedAt.Time,
		Title:     mv.Title,
		Year:      mv.Year,
		Runtime:   Runtime(mv.Runtime),
		Genres:    mv.Genres,
		Version:   mv.Version,
	}

	return movie, nil
}

func (m MovieService) Update(ctx context.Context, movie *Movie) error {
	newVersion, err := m.repository.UpdateMovie(ctx, repo.UpdateMovieParams{
		ID:      movie.ID,
		Title:   movie.Title,
		Year:    movie.Year,
		Runtime: int(movie.Runtime),
		Genres:  movie.Genres,
		Version: movie.Version,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEditConflict
		}
		return err
	}

	movie.Version = newVersion
	return nil
}

func (m MovieService) Delete(ctx context.Context, id int) error {
	n, err := m.repository.DeleteMovie(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrRecordNotFound
	}
	return nil
}
