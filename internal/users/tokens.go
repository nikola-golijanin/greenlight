package users

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	repo "github.com/nikola-golijanin/greenlight/internal/data/sqlc"
	"github.com/nikola-golijanin/greenlight/internal/validator"
)

const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
)

type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int       `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

func generateToken(userID int, ttl time.Duration, scope string) *Token {
	token := &Token{
		Plaintext: rand.Text(),
		UserID:    userID,
		Expiry:    time.Now().Add(ttl),
		Scope:     scope,
	}

	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token
}

func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

type TokenService struct {
	repository repo.Querier
}

func NewTokenService(repository repo.Querier) TokenService {
	return TokenService{repository: repository}
}

func (t TokenService) New(userID int, ttl time.Duration, scope string) (*Token, error) {
	token := generateToken(userID, ttl, scope)

	err := t.Insert(token)
	return token, err
}

func (t TokenService) Insert(token *Token) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return t.repository.InsertToken(ctx, repo.InsertTokenParams{
		Hash:   token.Hash,
		UserID: token.UserID,
		Expiry: pgtype.Timestamptz{Time: token.Expiry, Valid: true},
		Scope:  token.Scope,
	})
}

func (t TokenService) DeleteAllTokensForUser(scope string, userID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return t.repository.DeleteAllTokensForUser(ctx, repo.DeleteAllTokensForUserParams{Scope: scope, UserID: userID})
}

func (u TokenService) Login(email, passwordPlaintext string) (*Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	usr, err := u.repository.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	pw := password{nil, usr.PasswordHash}

	match, err := pw.Matches(passwordPlaintext)
	if err != nil {
		return nil, err
	}

	if !match {
		return nil, ErrInvalidPassword
	}

	token, err := u.New(usr.ID, 24*time.Hour, ScopeAuthentication)
	if err != nil {
		return nil, nil
	}

	return token, nil
}
