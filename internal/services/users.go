package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/apperrs"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type User struct {
	ID        string
	Email     string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	queries := s.getDB()

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
		CreatedAt: dbUser.CreatedAt.Time,
		UpdatedAt: dbUser.UpdatedAt.Time,
	}

	return user, nil
}

func (s *Service) GetUser(ctx context.Context, userID string) (*User, error) {
	queries := s.getDB()

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
		CreatedAt: dbUser.CreatedAt.Time,
		UpdatedAt: dbUser.UpdatedAt.Time,
	}

	return user, nil
}

func (s *Service) UpdateUserLastLogin(ctx context.Context, userID string) error {
	queries := s.getDB()

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
	queries := s.getDB()

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
		CreatedAt: dbUser.CreatedAt.Time,
		UpdatedAt: dbUser.UpdatedAt.Time,
	}

	return user, nil
}

// GetOrCreateUser gets a user by email or creates a new one if not found
// When creating a new user, also creates a trial subscription
func (s *Service) GetOrCreateUser(ctx context.Context, params CreateUserParams) (*User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	queries := db.New(tx)

	dbUser, err := queries.GetUserByEmail(ctx, params.Email)
	if err != nil {
		if db.IsNotFoundError(err) {
			// Create new user
			dbUser, err = queries.CreateUser(ctx, db.CreateUserParams{
				Email: params.Email,
				Name:  params.Name,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create user: %w", err)
			}

			// Create trial subscription for new user (trial starts when first instance is created)
			_, err = queries.CreateSubscription(ctx, db.CreateSubscriptionParams{
				UserID:         dbUser.ID,
				ProductID:      "", // Empty for trial
				CustomerID:     "", // Empty for trial
				SubscriptionID: "", // Empty for trial
				TrialEndsAt: pgtype.Timestamp{
					Valid: false, // Will be set when first instance is created
				},
				Status:   SubscriptionStatusTrial,
				Quantity: 1, // Default quantity for trial
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create trial subscription: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get user by email: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	user := &User{
		ID:        dbUser.ID,
		Email:     dbUser.Email,
		Name:      dbUser.Name,
		CreatedAt: dbUser.CreatedAt.Time,
		UpdatedAt: dbUser.UpdatedAt.Time,
	}

	return user, nil
}
