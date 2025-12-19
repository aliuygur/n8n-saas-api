package services

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/appctx"
	"github.com/aliuygur/n8n-saas-api/internal/apperrs"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/aliuygur/n8n-saas-api/internal/provisioning/n8ntemplates"
	"github.com/samber/lo"
)

// Instance represents an instance for internal use (domain layer)
type Instance struct {
	ID         string
	UserID     string
	Status     string
	Namespace  string
	Subdomain  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeployedAt *time.Time
	DeletedAt  *time.Time
}

func (i *Instance) GetInstanceURL() string {
	return InstanceURL(i.Subdomain)
}

func InstanceURL(subdomain string) string {
	return fmt.Sprintf("https://%s.instol.cloud", subdomain)
}

// toDomainInstance maps a db.Instance to a types.Instance (domain layer)
func toDomainInstance(dbInst db.Instance) Instance {
	i := Instance{
		ID:        dbInst.ID,
		UserID:    dbInst.UserID,
		Status:    dbInst.Status,
		Namespace: dbInst.Namespace,
		Subdomain: dbInst.Subdomain,
		CreatedAt: dbInst.CreatedAt.Time,
		UpdatedAt: dbInst.UpdatedAt.Time,
	}
	if dbInst.DeployedAt.Valid {
		i.DeployedAt = &dbInst.DeployedAt.Time
	}
	if dbInst.DeletedAt.Valid {
		i.DeletedAt = &dbInst.DeletedAt.Time
	}
	return i
}

func (s *Service) GetInstancesByUser(ctx context.Context, userID string) ([]Instance, error) {

	queries := db.New(s.db)

	dbInstances, err := queries.ListInstancesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	return lo.Map(dbInstances, func(inst db.Instance, _ int) Instance {
		return toDomainInstance(inst)
	}), nil
}

func (s *Service) GetInstanceByID(ctx context.Context, instanceID string) (*Instance, error) {
	queries := db.New(s.db)

	dbInstance, err := queries.GetInstance(ctx, instanceID)
	if err != nil {
		if db.IsNotFoundError(err) {
			return nil, apperrs.Client(apperrs.CodeNotFound, "instance not found")
		}

		return nil, fmt.Errorf("failed to get instance: %w", err)
	}
	instance := toDomainInstance(dbInstance)
	return &instance, nil
}

func (s *Service) GetInstanceBySubdomain(ctx context.Context, subdomain string) (*Instance, error) {
	queries := db.New(s.db)

	dbInstance, err := queries.GetInstanceBySubdomain(ctx, subdomain)
	if err != nil {
		if db.IsNotFoundError(err) {
			return nil, apperrs.Client(apperrs.CodeNotFound, "instance not found")
		}

		return nil, fmt.Errorf("failed to get instance: %w", err)
	}
	instance := toDomainInstance(dbInstance)
	return &instance, nil
}

type DeleteInstanceParams struct {
	UserID     string
	InstanceID string
}

func (s *Service) DeleteInstance(ctx context.Context, params DeleteInstanceParams) error {
	queries, releaseLock := s.getDBWithLock(ctx, fmt.Sprintf("user_instance_delete_%s", params.UserID))
	defer releaseLock()

	l := appctx.GetLogger(ctx)

	instance, err := queries.GetInstance(ctx, params.InstanceID)
	if err != nil {
		if db.IsNotFoundError(err) {
			return apperrs.Client(apperrs.CodeNotFound, "instance not found")
		}
		return fmt.Errorf("failed to get instance: %w", err)
	}

	if instance.UserID != params.UserID {
		return apperrs.Client(apperrs.CodeForbidden, "user does not own the instance")
	}

	if err := s.gke.DeleteNamespace(ctx, instance.Namespace); err != nil {
		return apperrs.Server("failed to delete namespace from Kubernetes", err)
	}
	l.Debug("deleted namespace from Kubernetes", "namespace", instance.Namespace)

	domain := InstanceURL(instance.Subdomain)
	if err := s.cloudflare.RemoveTunnelRoute(ctx, domain); err != nil {
		return apperrs.Server("failed to delete DNS record from Cloudflare", err)
	}
	l.Debug("deleted DNS record from Cloudflare", "domain", domain)

	if err := queries.DeleteInstance(ctx, params.InstanceID); err != nil {
		return apperrs.Server("failed to delete instance from database", err)
	}
	l.Debug("deleted instance from database", "instance_id", params.InstanceID)

	return nil
}

type UpdateInstanceStatusParams struct {
	InstanceID string
	Status     string
}

func (s *Service) UpdateInstanceStatus(ctx context.Context, params UpdateInstanceStatusParams) error {
	queries := db.New(s.db)

	_, err := queries.UpdateInstanceStatus(ctx, db.UpdateInstanceStatusParams{
		ID:     params.InstanceID,
		Status: params.Status,
	})
	if err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}
	return nil
}

type CreateInstanceParams struct {
	UserID    string
	Subdomain string
}

func (s *Service) CreateInstance(ctx context.Context, params CreateInstanceParams) (*Instance, error) {

	queries, releaseLock := s.getDBWithLock(ctx, fmt.Sprintf("user_instance_create_%s", params.UserID))
	defer releaseLock()

	// if err := s.canCreateInstance(ctx, queries, params.UserID); err != nil {
	// 	return nil, err
	// }

	// Check if subdomain already exists
	exists, err := queries.CheckSubdomainExists(ctx, params.Subdomain)
	if err != nil {
		return nil, apperrs.Server("failed to check subdomain existence", err)
	}
	if exists {
		return nil, apperrs.Client(apperrs.CodeConflict, "subdomain already taken")
	}

	return s.createInstanceInternal(ctx, queries, params)
}

func (s *Service) CheckSubdomainExists(ctx context.Context, subdomain string) (bool, error) {
	queries := db.New(s.db)

	exists, err := queries.CheckSubdomainExists(ctx, subdomain)
	if err != nil {
		return false, fmt.Errorf("failed to check subdomain existence: %w", err)
	}

	return exists, nil
}

func (s *Service) CheckInstanceURLActive(ctx context.Context, id string) (bool, error) {

	queries := s.getDB(ctx)

	instance, err := queries.GetInstance(ctx, id)
	if err != nil {
		return false, fmt.Errorf("failed to get instance: %w", err)
	}

	// if instance created less than 1 minute ago, skip check
	if time.Since(instance.CreatedAt.Time) < time.Minute {
		return false, nil
	}

	healthURL := fmt.Sprintf("https://%s.instol.cloud/healthz/readiness", instance.Subdomain)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Make GET request to instance URL
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		// check if context deadline exceeded
		if ctx.Err() == context.DeadlineExceeded {
			return false, nil
		}
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Instance not ready yet
		return false, nil
	}
	defer resp.Body.Close()

	// Check if we got a 200 status
	return resp.StatusCode == http.StatusOK, nil
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

type instanceCreationState struct {
	instanceID     string
	userID         string
	namespace      string
	subdomain      string
	subscriptionID string
}

func (s *Service) createInstanceInternal(ctx context.Context, queries *db.Queries, params CreateInstanceParams) (*Instance, error) {
	l := appctx.GetLogger(ctx)

	createInstance := func(state *instanceCreationState) (*instanceCreationState, error) {
		dbInst, err := queries.CreateInstance(ctx, db.CreateInstanceParams{
			UserID:    state.userID,
			Namespace: state.namespace,
			Subdomain: state.subdomain,
			Status:    InstanceStatusDeployed,
		})
		if err != nil {
			return state, apperrs.Server("failed to create instance in database", err)
		}

		state.instanceID = dbInst.ID
		state.namespace = dbInst.Namespace
		state.subdomain = dbInst.Subdomain
		return state, nil
	}

	deleteInstance := func(state *instanceCreationState) *instanceCreationState {
		if err := queries.DeleteInstance(ctx, state.instanceID); err != nil {
			l.Error("failed to revert instance creation", "instance_id", state.instanceID, "error", err)
		} else {
			l.Debug("reverted instance creation", "instance_id", state.instanceID)
		}
		return state
	}

	createTrialSubscription := func(state *instanceCreationState) (*instanceCreationState, error) {
		// Check if user already has a subscription
		subscriptions, err := queries.GetAllSubscriptionsByUserID(ctx, state.userID)
		if err != nil {
			return state, apperrs.Server("failed to check existing subscriptions", err)
		}

		// Only create trial subscription if user has no subscriptions
		if len(subscriptions) == 0 {
			trialEndsAt := time.Now().Add(3 * 24 * time.Hour) // 3 days trial
			subscription, err := queries.CreateSubscription(ctx, db.CreateSubscriptionParams{
				UserID:              state.userID,
				InstanceID:          state.instanceID,
				PolarProductID:      "", // Empty for trial
				PolarCustomerID:     "", // Empty for trial
				PolarSubscriptionID: "", // Empty for trial
				TrialEndsAt: sql.NullTime{
					Time:  trialEndsAt,
					Valid: true,
				},
				Status: SubscriptionStatusTrial,
			})
			if err != nil {
				return state, apperrs.Server("failed to create trial subscription", err)
			}
			state.subscriptionID = subscription.ID
			l.Debug("created trial subscription", "instance_id", state.instanceID, "subscription_id", subscription.ID, "trial_ends_at", trialEndsAt)
		}

		return state, nil
	}
	revertTrialSubscription := func(state *instanceCreationState) *instanceCreationState {
		// Only delete if we actually created a subscription
		if state.subscriptionID != "" {
			if err := queries.DeleteSubscriptionByInstanceID(ctx, state.instanceID); err != nil {
				l.Error("failed to revert trial subscription creation", "instance_id", state.instanceID, "subscription_id", state.subscriptionID, "error", err)
			}
			l.Debug("reverted trial subscription creation", "instance_id", state.instanceID, "subscription_id", state.subscriptionID)
		}
		return state
	}

	deployGke := func(state *instanceCreationState) (*instanceCreationState, error) {
		// Deploy n8n instance
		domain := fmt.Sprintf("https://%s.instol.cloud", state.subdomain)
		n8nInstance := &n8ntemplates.N8N_V1{
			Namespace:     state.namespace,
			EncryptionKey: lo.RandomString(32, lo.AlphanumericCharset),
			BaseURL:       domain,
		}

		if err := s.gke.Apply(ctx, n8nInstance); err != nil {
			return state, fmt.Errorf("failed to deploy n8n: %w", err)
		}
		return state, nil
	}
	revertGke := func(state *instanceCreationState) *instanceCreationState {
		if err := s.gke.DeleteNamespace(ctx, state.namespace); err != nil {
			l.Error("failed to revert GKE deployment", "namespace", state.namespace, "error", err)
		} else {
			l.Debug("reverted GKE deployment", "namespace", state.namespace)
		}
		return state
	}

	addDNSRoute := func(state *instanceCreationState) (*instanceCreationState, error) {
		domain := fmt.Sprintf("https://%s.instol.cloud", state.subdomain)
		serviceURL := fmt.Sprintf("http://n8n-main.%s.svc.cluster.local", state.namespace)
		if err := s.cloudflare.AddTunnelRoute(ctx, domain, serviceURL); err != nil {
			return state, apperrs.Server("failed to add Cloudflare tunnel route", err)
		}
		return state, nil
	}
	revertDNSRoute := func(state *instanceCreationState) *instanceCreationState {
		domain := fmt.Sprintf("https://%s.instol.cloud", state.subdomain)
		if err := s.cloudflare.DeleteDNSRecord(ctx, domain); err != nil {
			l.Error("failed to revert DNS route", "domain", domain, "error", err)
		} else {
			l.Debug("reverted DNS route", "domain", domain)
		}
		return state
	}

	namespace, err := s.generateUniqueNamespace(ctx, queries)
	if err != nil {
		return nil, err
	}

	initialState := &instanceCreationState{
		userID:    params.UserID,
		subdomain: params.Subdomain,
		namespace: namespace,
	}
	saga := lo.NewTransaction[*instanceCreationState]().
		Then(createInstance, deleteInstance).
		Then(createTrialSubscription, revertTrialSubscription).
		Then(deployGke, revertGke).
		Then(addDNSRoute, revertDNSRoute)

	finalState, err := saga.Process(initialState)
	if err != nil {
		return nil, err
	}

	// Fetch the full instance from the database to return
	dbInstance, err := queries.GetInstance(ctx, finalState.instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created instance: %w", err)
	}

	instance := toDomainInstance(dbInstance)
	return &instance, nil
}

func (s *Service) canCreateInstance(ctx context.Context, queries *db.Queries, userID string) error {
	ss, err := queries.GetAllSubscriptionsByUserID(ctx, userID)
	if err != nil {
		return apperrs.Server("failed to get subscriptions for user", err)
	}

	// if has active subscription, can create instance
	if lo.SomeBy(ss, func(st db.Subscription) bool {
		return st.Status == InstanceStatusActive
	}) {
		return nil
	}

	// if no subscriptions, can create instance (trial)
	if len(ss) == 0 {
		return nil
	}

	return apperrs.Client(apperrs.CodeForbidden, "user cannot create instance, no active subscription")
}
