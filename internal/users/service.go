package users

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	repo "github.com/nikola-golijanin/greenlight/internal/data/sqlc"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
	ErrRecordNotFound = errors.New("record not found")
)

type UserService struct {
	repo repo.Querier
}

func NewUserService(repo repo.Querier) UserService {
	return UserService{repo: repo}
}

func (u UserService) Insert(user *User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	usr, err := u.repo.InsertUser(ctx, repo.InsertUserParams{
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.Password.hash,
		Activated:    user.Activated,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			pgErr.ConstraintName == "users_email_key" {
			return ErrDuplicateEmail
		}

		return err
	}
	user.ID = usr.ID
	user.CreatedAt = usr.CreatedAt.Time
	user.Version = usr.Version
	return nil
}

func (u UserService) GetUserByEmail(email string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	usr, err := u.repo.GetUserByEmail(ctx, email)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	user := &User{
		ID:        usr.ID,
		CreatedAt: usr.CreatedAt.Time,
		Name:      usr.Name,
		Email:     usr.Email,
		Password:  password{nil, usr.PasswordHash},
		Activated: usr.Activated,
		Version:   usr.Version,
	}
	return user, nil
}

func (u UserService) Update(user *User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := u.repo.UpdateUser(ctx, repo.UpdateUserParams{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.Password.hash,
		Activated:    user.Activated,
		Version:      user.Version,
	})

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return ErrRecordNotFound
	default:
		return err
	}
}
