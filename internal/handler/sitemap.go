package handler

import (
	"encoding/xml"
	"net/http"
	"time"

	"github.com/aliuygur/n8n-saas-api/internal/handler/components"
)

// URLSet represents the root element of the sitemap
type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	XMLNS   string   `xml:"xmlns,attr"`
	URLs    []URL    `xml:"url"`
}

// URL represents a single URL entry in the sitemap
type URL struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod,omitempty"`
	ChangeFreq string  `xml:"changefreq,omitempty"`
	Priority   float64 `xml:"priority,omitempty"`
}

// Sitemap generates and serves the sitemap.xml
func (h *Handler) Sitemap(w http.ResponseWriter, r *http.Request) {
	// Get base URL from config
	baseURL := h.config.Server.APIBaseURL
	if baseURL == "" {
		// Fallback to request host if API_BASE_URL is not set
		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}
		baseURL = scheme + "://" + r.Host
	}

	// Create URL set
	urlSet := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  []URL{},
	}

	// Add static pages
	staticPages := []struct {
		path       string
		changefreq string
		priority   float64
	}{
		{"/", "daily", 1.0},
		{"/pricing", "weekly", 0.9},
		{"/terms", "monthly", 0.5},
		{"/privacy", "monthly", 0.5},
		{"/refund-policy", "monthly", 0.5},
		{"/blog", "weekly", 0.8},
		{"/login", "monthly", 0.6},
	}

	now := time.Now().Format("2006-01-02")

	for _, page := range staticPages {
		urlSet.URLs = append(urlSet.URLs, URL{
			Loc:        baseURL + page.path,
			LastMod:    now,
			ChangeFreq: page.changefreq,
			Priority:   page.priority,
		})
	}

	// Add blog posts
	blogPosts := components.GetAllBlogPosts()
	for _, post := range blogPosts {
		urlSet.URLs = append(urlSet.URLs, URL{
			Loc:        baseURL + "/blog/" + post.Slug,
			LastMod:    post.Date.Format("2006-01-02"),
			ChangeFreq: "monthly",
			Priority:   0.7,
		})
	}

	// Set XML content type
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Write XML header
	w.Write([]byte(xml.Header))

	// Encode and write sitemap
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")
	if err := encoder.Encode(urlSet); err != nil {
		http.Error(w, "Failed to generate sitemap", http.StatusInternalServerError)
		return
	}
}
