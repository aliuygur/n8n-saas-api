package services

import (
	"context"

	"github.com/aliuygur/n8n-saas-api/internal/db"
)

func (s *Service) RunInTransaction(ctx context.Context, fn func(tx *db.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	queries := db.New(tx)

	if err := fn(queries); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return rbErr
		}
		return err
	}

	return tx.Commit()
}
