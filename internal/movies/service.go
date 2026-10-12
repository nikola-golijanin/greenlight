package movies

import (
	"context"
	"errors"
	"time"

	"github.com/nikola-golijanin/greenlight/internal/data"
	repo "github.com/nikola-golijanin/greenlight/internal/data/sqlc"
)

var ErrRecordNotFound = errors.New("record not found")

type MovieService struct {
	repo repo.Querier
}

func NewMovieService(repo repo.Querier) MovieService {
	return MovieService{
		repo: repo,
	}
}

func (m MovieService) GetAll(title string, genres []string, filters data.Filters) ([]*Movie, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	movies, err := m.repo.GetAllMovies(ctx)
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

func (m MovieService) Insert(movie *Movie) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	mv, err := m.repo.InsertMovie(ctx, repo.InsertMovieParams{
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

func (m MovieService) GetById(id int) (*Movie, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	mv, err := m.repo.GetMovieById(ctx, id)
	if err != nil {
		return nil, err
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

func (m MovieService) Update(movie *Movie) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.repo.UpdateMovie(ctx, repo.UpdateMovieParams{
		ID:      movie.ID,
		Title:   movie.Title,
		Year:    movie.Year,
		Runtime: int(movie.Runtime),
		Genres:  movie.Genres,
		Version: movie.Version,
	})

	return err
}

func (m MovieService) Delete(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.repo.DeleteMovie(ctx, id)
}
