package users

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	repo "github.com/nikola-golijanin/greenlight/internal/data/sqlc"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type UserService struct {
	repos *repo.Queries
	db    *pgxpool.Pool
}

func NewUserService(db *pgxpool.Pool) UserService {
	return UserService{repos: repo.New(db), db: db}
}

func (u UserService) Insert(user *User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	usr, err := u.repos.InsertUser(ctx, repo.InsertUserParams{
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

	usr, err := u.repos.GetUserByEmail(ctx, email)
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

	_, err := u.repos.UpdateUser(ctx, repo.UpdateUserParams{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.Password.hash,
		Activated:    user.Activated,
		Version:      user.Version,
	})

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return ErrEditConflict
	default:
		return err
	}
}

func (u UserService) GetUserForToken(scope string, tokenPlaintext string) (*User, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	usr, err := u.repos.GetUserForToken(ctx, repo.GetUserForTokenParams{
		Hash:   tokenHash[:],
		Scope:  scope,
		Expiry: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
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

	return user, err
}

func (u UserService) Activate(tokenPlaintext string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	hash := sha256.Sum256([]byte(tokenPlaintext))

	tx, err := u.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := u.repos.WithTx(tx)
	usr, err := qtx.GetUserForToken(ctx, repo.GetUserForTokenParams{
		Hash:   hash[:],
		Scope:  ScopeActivation,
		Expiry: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	usr.Activated = true

	newVersion, err := qtx.UpdateUser(ctx, repo.UpdateUserParams{
		ID:           usr.ID,
		Name:         usr.Name,
		Email:        usr.Email,
		PasswordHash: usr.PasswordHash,
		Activated:    usr.Activated,
		Version:      usr.Version,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEditConflict
		}
		return nil, err
	}

	if err := qtx.DeleteAllTokensForUser(ctx, repo.DeleteAllTokensForUserParams{
		Scope:  ScopeActivation,
		UserID: usr.ID,
	}); err != nil {
		return nil, err
	}

	user := &User{
		ID:        usr.ID,
		CreatedAt: usr.CreatedAt.Time,
		Name:      usr.Name,
		Email:     usr.Email,
		Password:  password{nil, usr.PasswordHash},
		Activated: usr.Activated,
		Version:   newVersion,
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return user, nil
}
