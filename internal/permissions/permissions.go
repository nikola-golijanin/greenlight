package permissions

import (
	"context"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	repo "github.com/nikola-golijanin/greenlight/internal/data/sqlc"
)

type Permissions []string

func (p Permissions) Contains(code string) bool {
	return slices.Contains(p, code)
}

type PermissionService struct {
	repository *repo.Queries
	db         *pgxpool.Pool
}

func NewPermissionsService(repository *repo.Queries, db *pgxpool.Pool) PermissionService {
	return PermissionService{repository: repository, db: db}
}

func (p PermissionService) GetUsersPermissions(userID int) (Permissions, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ps, err := p.repository.GetUsersPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	return Permissions(ps), nil
}
