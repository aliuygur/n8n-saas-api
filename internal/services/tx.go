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

func (s *Service) getDB(ctx context.Context) *db.Queries {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		panic(err)
	}
	return db.New(conn)
}

func (s *Service) getDBWithLock(ctx context.Context, lockKey string) (*db.Queries, func()) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		panic(err)
	}
	queries := db.New(conn)

	if err := queries.AcquireLock(ctx, lockKey); err != nil {
		panic(err)
	}

	releaseFunc := func() {
		if err := queries.ReleaseLock(ctx, lockKey); err != nil {
			panic(err)
		}
	}

	return queries, releaseFunc
}
