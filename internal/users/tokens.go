package users

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	repo "github.com/nikola-golijanin/greenlight/internal/data/sqlc"
	"github.com/nikola-golijanin/greenlight/internal/validator"
)

const (
	ScopeActivation = "activation"
)

type Token struct {
	Plaintext string
	Hash      []byte
	UserID    int
	Expiry    time.Time
	Scope     string
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
	repo repo.Querier
}

func NewTokenService(repo repo.Querier) TokenService {
	return TokenService{repo: repo}
}

func (t TokenService) New(userID int, ttl time.Duration, scope string) (*Token, error) {
	token := generateToken(userID, ttl, scope)

	err := t.Insert(token)
	return token, err
}

func (t TokenService) Insert(token *Token) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return t.repo.InsertToken(ctx, repo.InsertTokenParams{
		Hash:   token.Hash,
		UserID: token.UserID,
		Expiry: pgtype.Timestamptz{Time: token.Expiry, Valid: true},
		Scope:  token.Scope,
	})
}

func (t TokenService) DeleteAllTokensForUser(scope string, userID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return t.repo.DeleteAllTokensForUser(ctx, repo.DeleteAllTokensForUserParams{Scope: scope, UserID: userID})
}
