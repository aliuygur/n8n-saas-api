package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aliuygur/n8n-saas-api/internal/config"
)

func TestSitemap(t *testing.T) {
	// Create a test config
	cfg := &config.Config{
		Server: config.ServerConfig{
			APIBaseURL: "https://ranx.cloud",
		},
	}

	// Create handler
	h := &Handler{
		config: cfg,
	}

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	w := httptest.NewRecorder()

	// Call the handler
	h.Sitemap(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/xml") {
		t.Errorf("Expected content type to contain 'application/xml', got %s", contentType)
	}

	// Check response body contains expected elements
	body := w.Body.String()

	// Check for XML declaration
	if !strings.Contains(body, "<?xml version") {
		t.Error("Expected XML declaration in response")
	}

	// Check for urlset element
	if !strings.Contains(body, "<urlset") {
		t.Error("Expected <urlset> element in response")
	}

	// Check for required URLs
	expectedURLs := []string{
		"<loc>https://ranx.cloud/</loc>",
		"<loc>https://ranx.cloud/pricing</loc>",
		"<loc>https://ranx.cloud/blog</loc>",
		"<loc>https://ranx.cloud/blog/what-is-n8n-workflow-automation</loc>",
		"<loc>https://ranx.cloud/blog/n8n-vs-zapier-comparison</loc>",
	}

	for _, expected := range expectedURLs {
		if !strings.Contains(body, expected) {
			t.Errorf("Expected URL not found in sitemap: %s", expected)
		}
	}

	// Check for SEO attributes
	if !strings.Contains(body, "<changefreq>") {
		t.Error("Expected <changefreq> elements in sitemap")
	}

	if !strings.Contains(body, "<priority>") {
		t.Error("Expected <priority> elements in sitemap")
	}

	if !strings.Contains(body, "<lastmod>") {
		t.Error("Expected <lastmod> elements in sitemap")
	}
}

func TestSitemap_NoAPIBaseURL(t *testing.T) {
	// Create a test config without API base URL
	cfg := &config.Config{
		Server: config.ServerConfig{
			APIBaseURL: "",
		},
	}

	// Create handler
	h := &Handler{
		config: cfg,
	}

	// Create test request with host
	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	// Call the handler
	h.Sitemap(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Check that URLs use the request host
	body := w.Body.String()
	if !strings.Contains(body, "http://example.com/") {
		t.Error("Expected URLs to use request host when API_BASE_URL is not set")
	}
}
