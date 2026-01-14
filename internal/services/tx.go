package services

import (
	"context"

	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/jackc/pgx/v5"
)

func (s *Service) RunInTransaction(ctx context.Context, fn func(tx *db.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	queries := db.New(tx)

	if err := fn(queries); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *Service) getDB() *db.Queries {
	return db.New(s.pool)
}

func (s *Service) getDBWithLock(ctx context.Context, lockKey string) (*db.Queries, func()) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		panic(err)
	}
	queries := db.New(conn)

	if err := queries.AcquireLock(ctx, lockKey); err != nil {
		conn.Release()
		panic(err)
	}

	releaseFunc := func() {
		if err := queries.ReleaseLock(ctx, lockKey); err != nil {
			conn.Release()
			panic(err)
		}
		conn.Release()
	}

	return queries, releaseFunc
}

func (s *Service) RunInTransactionWithOptions(ctx context.Context, txOptions pgx.TxOptions, fn func(tx *db.Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, txOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	queries := db.New(tx)

	if err := fn(queries); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
