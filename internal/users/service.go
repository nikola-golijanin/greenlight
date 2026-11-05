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

const activationTokenTTL = 3 * 24 * time.Hour

type UserService struct {
	repository *repo.Queries
	db         *pgxpool.Pool
}

func NewUserService(repository *repo.Queries, db *pgxpool.Pool) UserService {
	return UserService{repository: repository, db: db}
}

func (u UserService) Register(user *User) (*Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := u.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	qtx := u.repository.WithTx(tx)

	usr, err := qtx.InsertUser(ctx, repo.InsertUserParams{
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
			return nil, ErrDuplicateEmail
		}

		return nil, err
	}

	token := generateToken(usr.ID, activationTokenTTL, ScopeActivation)
	if err := qtx.InsertToken(ctx, repo.InsertTokenParams{
		Hash:   token.Hash,
		UserID: token.UserID,
		Expiry: pgtype.Timestamptz{Time: token.Expiry, Valid: true},
		Scope:  token.Scope,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	user.ID = usr.ID
	user.CreatedAt = usr.CreatedAt.Time
	user.Version = usr.Version
	user.Activated = false

	return token, nil
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
	qtx := u.repository.WithTx(tx)

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
