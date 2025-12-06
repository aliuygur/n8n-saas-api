package provisioning

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/samber/lo"
)

// ValidateSubdomainRequest represents a request to validate subdomain
type ValidateSubdomainRequest struct {
	Subdomain string
}

// ValidateSubdomainResponse represents the response
type ValidateSubdomainResponse struct {
	Valid        bool
	ErrorMessage string
}

// ValidateSubdomain validates a subdomain's format and availability rules
//
//encore:api private
func (s *Service) ValidateSubdomain(ctx context.Context, req *ValidateSubdomainRequest) (*ValidateSubdomainResponse, error) {
	err := validateSubdomain(req.Subdomain)
	if err != nil {
		return &ValidateSubdomainResponse{
			Valid:        false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &ValidateSubdomainResponse{
		Valid: true,
	}, nil
}

// CheckSubdomainExistsRequest represents a request to check if subdomain exists
type CheckSubdomainExistsRequest struct {
	Subdomain string
}

// CheckSubdomainExistsResponse represents the response
type CheckSubdomainExistsResponse struct {
	Exists bool
}

// CheckSubdomainExists checks if a subdomain already exists in the database
//
//encore:api private
func (s *Service) CheckSubdomainExists(ctx context.Context, req *CheckSubdomainExistsRequest) (*CheckSubdomainExistsResponse, error) {
	// Check if subdomain exists in database
	queries := db.New(s.db)
	_, err := queries.GetInstanceBySubdomain(ctx, req.Subdomain)
	if err != nil {
		// If not found (no rows), subdomain is available
		if errors.Is(err, sql.ErrNoRows) {
			return &CheckSubdomainExistsResponse{
				Exists: false,
			}, nil
		}
		return nil, fmt.Errorf("failed to check subdomain: %w", err)
	}

	// If we got here, instance was found, so subdomain exists
	return &CheckSubdomainExistsResponse{
		Exists: true,
	}, nil
}

// Reserved subdomains that cannot be used
var reservedSubdomains = map[string]bool{
	"www":          true,
	"ftp":          true,
	"mail":         true,
	"smtp":         true,
	"pop":          true,
	"imap":         true,
	"admin":        true,
	"root":         true,
	"api":          true,
	"app":          true,
	"blog":         true,
	"shop":         true,
	"store":        true,
	"support":      true,
	"help":         true,
	"docs":         true,
	"status":       true,
	"dashboard":    true,
	"portal":       true,
	"cdn":          true,
	"static":       true,
	"assets":       true,
	"ns1":          true,
	"ns2":          true,
	"ns3":          true,
	"ns4":          true,
	"localhost":    true,
	"webmail":      true,
	"cpanel":       true,
	"whm":          true,
	"autoconfig":   true,
	"autodiscover": true,
}

// validateSubdomain validates that a subdomain is safe and appropriate
func validateSubdomain(subdomain string) error {
	// Check minimum length
	if len(subdomain) < 3 {
		return fmt.Errorf("subdomain must be at least 3 characters long")
	}

	// Check maximum length
	if len(subdomain) > 63 {
		return fmt.Errorf("subdomain must be at most 63 characters long")
	}

	// Check if it's a reserved subdomain
	if reservedSubdomains[strings.ToLower(subdomain)] {
		return fmt.Errorf("subdomain '%s' is reserved and cannot be used", subdomain)
	}

	// Validate format: must start and end with alphanumeric, can contain hyphens in the middle
	validSubdomain := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	if !validSubdomain.MatchString(strings.ToLower(subdomain)) {
		return fmt.Errorf("subdomain must contain only lowercase letters, numbers, and hyphens, and must start and end with a letter or number")
	}

	// Check for consecutive hyphens
	if strings.Contains(subdomain, "--") {
		return fmt.Errorf("subdomain cannot contain consecutive hyphens")
	}

	return nil
}

// generateUniqueNamespace creates a unique namespace by checking database and handling GKE conflicts
func (s *Service) generateUniqueNamespace(ctx context.Context, userID string) (string, error) {
	queries := db.New(s.db)

	// Sanitize userID to be kubernetes-compliant
	sanitized := regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(strings.ToLower(userID), "")

	// Try to find a unique namespace
	maxAttempts := 10
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Generate 8-character alphanumeric random string using lo
		randomStr := lo.RandomString(8, append(lo.LowerCaseLettersCharset, lo.NumbersCharset...))

		// Format: n8n-{userid}-{8-alphanumeric}
		namespace := fmt.Sprintf("n8n-%s-%s", sanitized, strings.ToLower(randomStr))

		// Truncate if too long (Kubernetes limit is 63 characters)
		if len(namespace) > 63 {
			// Keep the format but truncate the userID part
			maxUserIDLength := 63 - 4 - 8 - 2 // 63 - "n8n-" - random - "-"
			if len(sanitized) > maxUserIDLength {
				sanitized = sanitized[:maxUserIDLength]
			}
			namespace = fmt.Sprintf("n8n-%s-%s", sanitized, strings.ToLower(randomStr))
		}

		// Check if namespace exists in database
		exists, err := s.namespaceExistsInDB(ctx, queries, namespace)
		if err != nil {
			return "", fmt.Errorf("failed to check namespace in database: %w", err)
		}

		if !exists {
			return namespace, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique namespace after %d attempts", maxAttempts)
}

// namespaceExistsInDB checks if a namespace exists in the database
func (s *Service) namespaceExistsInDB(ctx context.Context, queries *db.Queries, namespace string) (bool, error) {
	exists, err := queries.CheckNamespaceExists(ctx, namespace)
	if err != nil {
		return false, fmt.Errorf("failed to check namespace existence: %w", err)
	}
	return exists, nil
}
