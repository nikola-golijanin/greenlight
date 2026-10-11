package movies

import (
	"context"
	"errors"

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

func (m MovieService) Insert(movie *Movie) error {
	mv, err := m.repo.InsertMovie(context.Background(), repo.InsertMovieParams{
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
	mv, err := m.repo.GetMovieById(context.Background(), id)
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
	_, err := m.repo.UpdateMovie(context.Background(), repo.UpdateMovieParams{
		ID:      movie.ID,
		Title:   movie.Title,
		Year:    movie.Year,
		Runtime: int(movie.Runtime),
		Genres:  movie.Genres,
	})

	return err
}

func (m MovieService) Delete(id int) error {
	return m.repo.DeleteMovie(context.Background(), id)
}
