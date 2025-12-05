package subscription

import (
	"context"

	"encore.dev/cron"
	"encore.dev/rlog"
	"github.com/aliuygur/n8n-saas-api/internal/db"
)

// ExpireTrials is a cron job that runs every hour to check for expired trials
var _ = cron.NewJob("expire-trials", cron.JobConfig{
	Title:    "Expire Trial Subscriptions",
	Schedule: "0 * * * *", // Every hour at minute 0
	Endpoint: ExpireTrials,
})

// ExpireTrialsResponse represents the response from the cron job
type ExpireTrialsResponse struct {
	ExpiredCount int `json:"expired_count"`
}

// ExpireTrials checks for expired trial subscriptions and marks them as expired
//
//encore:api private
func (s *Service) ExpireTrials(ctx context.Context) (*ExpireTrialsResponse, error) {
	queries := db.New(s.db)

	// Get all expired trials
	expiredTrials, err := queries.GetExpiredTrials(ctx)
	if err != nil {
		rlog.Error("Failed to get expired trials", "error", err)
		return nil, err
	}

	if len(expiredTrials) == 0 {
		rlog.Info("No expired trials found")
		return &ExpireTrialsResponse{ExpiredCount: 0}, nil
	}

	rlog.Info("Found expired trials", "count", len(expiredTrials))

	// Update each expired trial
	expiredCount := 0
	for _, trial := range expiredTrials {
		err := queries.UpdateSubscriptionToExpired(ctx, trial.ID)
		if err != nil {
			rlog.Error("Failed to expire trial",
				"subscription_id", trial.ID,
				"user_id", trial.UserID,
				"error", err,
			)
			continue
		}

		rlog.Info("Trial expired",
			"subscription_id", trial.ID,
			"user_id", trial.UserID,
			"trial_ended_at", trial.TrialEndsAt,
		)

		expiredCount++

		// TODO: Optionally soft-delete instances after grace period
		// TODO: Send email notification to user about trial expiration
	}

	rlog.Info("Expired trials processed",
		"total", len(expiredTrials),
		"succeeded", expiredCount,
	)

	return &ExpireTrialsResponse{
		ExpiredCount: expiredCount,
	}, nil
}
