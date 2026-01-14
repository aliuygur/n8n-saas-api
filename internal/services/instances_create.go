package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/appctx"
	"github.com/aliuygur/n8n-saas-api/internal/apperrs"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/aliuygur/n8n-saas-api/internal/provisioning/n8ntemplates"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/samber/lo"
)

type CreateInstanceParams struct {
	UserID    string
	Subdomain string
}

func (s *Service) CreateInstance(ctx context.Context, params CreateInstanceParams) (*Instance, error) {
	l := appctx.GetLogger(ctx)

	queries, releaseLock := s.getDBWithLock(ctx, fmt.Sprintf("user_instance_create_%s", params.UserID))
	defer releaseLock()

	// Check if subdomain already exists
	exists, err := queries.CheckSubdomainExists(ctx, params.Subdomain)
	if err != nil {
		return nil, apperrs.Server("failed to check subdomain existence", err)
	}
	if exists {
		return nil, apperrs.Client(apperrs.CodeConflict, "subdomain already taken")
	}

	sub, err := queries.GetSubscriptionByUserID(ctx, params.UserID)
	if err != nil {
		return nil, apperrs.Server("failed to get subscription for user", err)
	}

	if sub.Status == SubscriptionStatusTrial {
		count, err := queries.CountActiveInstancesByUserID(ctx, params.UserID)
		if err != nil {
			return nil, apperrs.Server("failed to count instances", err)
		}
		if count > 0 {
			return nil, apperrs.Client(apperrs.CodeForbidden, "trial users can only have one instance")
		}
	}

	namespace, err := s.generateUniqueNamespace(ctx, queries)
	if err != nil {
		return nil, err
	}

	// Start trial if needed (idempotent, no rollback needed)
	if sub.Status == SubscriptionStatusTrial && !sub.TrialEndsAt.Valid {
		trialEndsAt := time.Now().Add(3 * 24 * time.Hour) // 3 days trial
		_, err = queries.UpdateSubscriptionTrialEndsAt(ctx, db.UpdateSubscriptionTrialEndsAtParams{
			ID: sub.ID,
			TrialEndsAt: pgtype.Timestamp{
				Time:  trialEndsAt,
				Valid: true,
			},
		})
		if err != nil {
			return nil, apperrs.Server("failed to start trial", err)
		}
		l.Debug("started trial subscription", "user_id", params.UserID, "subscription_id", sub.ID, "trial_ends_at", trialEndsAt)
	}

	// Create instance in database
	dbInst, err := queries.CreateInstance(ctx, db.CreateInstanceParams{
		UserID:    params.UserID,
		Namespace: namespace,
		Subdomain: params.Subdomain,
		Status:    InstanceStatusDeployed,
	})
	if err != nil {
		return nil, apperrs.Server("failed to create instance in database", err)
	}

	// Deploy to GKE
	domain := fmt.Sprintf("https://%s.ranx.cloud", params.Subdomain)
	n8nInstance := &n8ntemplates.N8N_V1{
		Namespace:     namespace,
		EncryptionKey: lo.RandomString(32, lo.AlphanumericCharset),
		BaseURL:       domain,
	}

	if err := s.gke.Apply(ctx, n8nInstance); err != nil {
		return nil, fmt.Errorf("failed to deploy n8n: %w", err)
	}

	// Sync subscription quantity with LemonSqueezy
	if err := s.SyncSubscriptionQuantity(ctx, params.UserID); err != nil {
		// Log error but don't fail the creation
		l.Error("failed to sync subscription quantity", "user_id", params.UserID, "error", err)
	}

	instance := toDomainInstance(dbInst)
	return &instance, nil
}

func (s *Service) generateUniqueNamespace(ctx context.Context, queries *db.Queries) (string, error) {
	// Try to find a unique namespace
	maxAttempts := 10
	for range maxAttempts {
		// Generate 8-character alphanumeric random string
		randomStr := lo.RandomString(16, append(lo.LowerCaseLettersCharset, lo.NumbersCharset...))

		// Format: n8n-{8-alphanumeric}
		namespace := fmt.Sprintf("n8n-%s", strings.ToLower(randomStr))
		// Truncate if too long (Kubernetes limit is 63 characters)
		if len(namespace) > 63 {
			namespace = namespace[:63]
		}

		// Check if namespace exists in database
		exists, err := queries.CheckNamespaceExists(ctx, namespace)
		if err != nil {
			return "", apperrs.Server("failed to check namespace existence", err)
		}

		if !exists {
			return namespace, nil
		}
	}

	return "", apperrs.Server(fmt.Sprintf("failed to generate unique namespace after %d attempts", maxAttempts), nil)
}
