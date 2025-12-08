package components

import (
	"strings"
	"time"
)

// Helper functions for templates
func stripProtocol(url string) string {
	return strings.Replace(url, "https://", "", 1)
}

func getStatusClass(status string) string {
	if status == "running" {
		return "inline-block px-3 py-1 rounded-full text-xs font-medium bg-green-500/10 text-green-400 border border-green-500/20"
	}
	return "inline-block px-3 py-1 rounded-full text-xs font-medium bg-yellow-500/10 text-yellow-400 border border-yellow-500/20"
}

func formatDate(dateStr string) string {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("January 2, 2006")
}
