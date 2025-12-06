package domainutils

import (
	"fmt"
	"regexp"
	"strings"
)

// reservedSubdomains contains subdomains that cannot be used
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

// ValidateSubdomain validates that a subdomain is safe and appropriate
func ValidateSubdomain(subdomain string) error {
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
