package handler

import (
	"net/http"

	"github.com/aliuygur/n8n-saas-api/internal/handler/components"
	"github.com/samber/lo"
)

// BlogIndex renders the blog index page
func (h *Handler) BlogIndex(w http.ResponseWriter, r *http.Request) {
	lo.Must0(components.BlogIndexPage().Render(r.Context(), w))
}

// BlogPost renders a single blog post page
func (h *Handler) BlogPost(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		http.Redirect(w, r, "/blog", http.StatusSeeOther)
		return
	}

	post := components.GetBlogPostBySlug(slug)
	if post == nil {
		http.NotFound(w, r)
		return
	}

	lo.Must0(components.BlogPostPage(*post).Render(r.Context(), w))
}
