package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/apperrs"
	"github.com/aliuygur/n8n-saas-api/internal/db"
)

type User struct {
	ID        string
	Email     string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	queries := db.New(s.db)

	dbUser, err := queries.GetUserByEmail(ctx, email)
	if err != nil {
		if db.IsNotFoundError(err) {
			return nil, apperrs.Client(apperrs.CodeNotFound, "user not found")
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	user := &User{
		ID:        dbUser.ID,
		Email:     dbUser.Email,
		Name:      dbUser.Name,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
	}

	return user, nil
}

func (s *Service) GetUser(ctx context.Context, userID string) (*User, error) {
	queries := db.New(s.db)

	dbUser, err := queries.GetUserByID(ctx, userID)
	if err != nil {
		if db.IsNotFoundError(err) {
			return nil, apperrs.Client(apperrs.CodeNotFound, "user not found")
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	user := &User{
		ID:        dbUser.ID,
		Email:     dbUser.Email,
		Name:      dbUser.Name,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
	}

	return user, nil
}

func (s *Service) UpdateUserLastLogin(ctx context.Context, userID string) error {
	queries := db.New(s.db)

	_, err := queries.UpdateUserLastLogin(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to update user last login: %w", err)
	}

	return nil
}

type CreateUserParams struct {
	Email string
	Name  string
}

func (s *Service) CreateUser(ctx context.Context, params CreateUserParams) (*User, error) {
	queries := db.New(s.db)

	dbUser, err := queries.CreateUser(ctx, db.CreateUserParams{
		Email: params.Email,
		Name:  params.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	user := &User{
		ID:        dbUser.ID,
		Email:     dbUser.Email,
		Name:      dbUser.Name,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
	}

	return user, nil
}

// GetOrCreateUser gets a user by email or creates a new one if not found
func (s *Service) GetOrCreateUser(ctx context.Context, params CreateUserParams) (*User, error) {
	queries := db.New(s.db)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	queries = queries.WithTx(tx)

	dbUser, err := queries.GetUserByEmail(ctx, params.Email)
	if err != nil {
		if db.IsNotFoundError(err) {
			dbUser, err = queries.CreateUser(ctx, db.CreateUserParams{
				Email: params.Email,
				Name:  params.Name,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create user: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get user by email: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	user := &User{
		ID:        dbUser.ID,
		Email:     dbUser.Email,
		Name:      dbUser.Name,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
	}

	return user, nil
}
