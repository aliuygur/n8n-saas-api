package components

import "time"

// BlogPost represents a single blog post
type BlogPost struct {
	Slug        string
	Title       string
	Description string
	Author      string
	AuthorBio   string
	AuthorImage string
	Date        time.Time
	ReadTime    string
	Content     string
	Keywords    []string
	OGImage     string
	CoverImage  string
	Category    string
}

// GetAllBlogPosts returns all blog posts sorted by date (newest first)
func GetAllBlogPosts() []BlogPost {
	posts := []BlogPost{
		{
			Slug:        "what-is-n8n-workflow-automation",
			Title:       "What is n8n? A Complete Guide to Workflow Automation",
			Description: "Discover n8n, the powerful open-source workflow automation tool that helps you connect apps and automate repetitive tasks without writing code.",
			Author:      "instol.cloud Team",
			AuthorBio:   "Helping teams deploy and manage n8n workflow automation on Google Cloud Platform.",
			AuthorImage: "https://ui-avatars.com/api/?name=instol.cloud&background=6366f1&color=fff&size=128",
			Date:        time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			ReadTime:    "5 min read",
			Keywords:    []string{"n8n", "workflow automation", "no-code automation", "open source automation"},
			OGImage:     "https://instol.cloud/static/android-chrome-512x512.png",
			CoverImage:  "https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&h=630&fit=crop",
			Category:    "Tutorial",
		},
		{
			Slug:        "n8n-vs-zapier-comparison",
			Title:       "n8n vs Zapier: Which Automation Tool is Right for You?",
			Description: "Compare n8n and Zapier to understand the key differences, pricing, features, and which automation platform best fits your needs.",
			Author:      "instol.cloud Team",
			AuthorBio:   "Helping teams deploy and manage n8n workflow automation on Google Cloud Platform.",
			AuthorImage: "https://ui-avatars.com/api/?name=instol.cloud&background=6366f1&color=fff&size=128",
			Date:        time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
			ReadTime:    "7 min read",
			Keywords:    []string{"n8n vs zapier", "automation comparison", "workflow tools", "n8n alternative"},
			OGImage:     "https://instol.cloud/static/android-chrome-512x512.png",
			CoverImage:  "https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=1200&h=630&fit=crop",
			Category:    "Comparison",
		},
		{
			Slug:        "getting-started-with-n8n",
			Title:       "Getting Started with n8n: Your First Workflow in 10 Minutes",
			Description: "Learn how to create your first n8n workflow in just 10 minutes. Step-by-step guide for beginners to start automating tasks today.",
			Author:      "instol.cloud Team",
			AuthorBio:   "Helping teams deploy and manage n8n workflow automation on Google Cloud Platform.",
			AuthorImage: "https://ui-avatars.com/api/?name=instol.cloud&background=6366f1&color=fff&size=128",
			Date:        time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC),
			ReadTime:    "8 min read",
			Keywords:    []string{"n8n tutorial", "n8n getting started", "workflow automation tutorial", "n8n beginner guide"},
			OGImage:     "https://instol.cloud/static/android-chrome-512x512.png",
			CoverImage:  "https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=1200&h=630&fit=crop",
			Category:    "Tutorial",
		},
		{
			Slug:        "n8n-use-cases-examples",
			Title:       "10 Powerful n8n Use Cases to Automate Your Business",
			Description: "Explore real-world n8n use cases and automation examples that can save you hours of manual work every week.",
			Author:      "instol.cloud Team",
			AuthorBio:   "Helping teams deploy and manage n8n workflow automation on Google Cloud Platform.",
			AuthorImage: "https://ui-avatars.com/api/?name=instol.cloud&background=6366f1&color=fff&size=128",
			Date:        time.Date(2024, 12, 28, 0, 0, 0, 0, time.UTC),
			ReadTime:    "10 min read",
			Keywords:    []string{"n8n use cases", "automation examples", "n8n workflows", "business automation"},
			OGImage:     "https://instol.cloud/static/android-chrome-512x512.png",
			CoverImage:  "https://images.unsplash.com/photo-1557804506-669a67965ba0?w=1200&h=630&fit=crop",
			Category:    "Guide",
		},
		{
			Slug:        "self-hosted-n8n-vs-cloud",
			Title:       "Self-Hosted n8n vs n8n Cloud: Which Should You Choose?",
			Description: "Understand the differences between self-hosted n8n and n8n Cloud, including cost, control, and which option fits your requirements.",
			Author:      "instol.cloud Team",
			AuthorBio:   "Helping teams deploy and manage n8n workflow automation on Google Cloud Platform.",
			AuthorImage: "https://ui-avatars.com/api/?name=instol.cloud&background=6366f1&color=fff&size=128",
			Date:        time.Date(2024, 12, 20, 0, 0, 0, 0, time.UTC),
			ReadTime:    "6 min read",
			Keywords:    []string{"self-hosted n8n", "n8n cloud", "n8n hosting", "n8n deployment"},
			OGImage:     "https://instol.cloud/static/android-chrome-512x512.png",
			CoverImage:  "https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=1200&h=630&fit=crop",
			Category:    "Comparison",
		},
		{
			Slug:        "why-choose-instol-cloud-for-n8n",
			Title:       "Why Choose instol.cloud for Your n8n Hosting?",
			Description: "Discover why instol.cloud is the easiest way to deploy and manage n8n instances on Google Cloud Platform with automatic SSL and updates.",
			Author:      "instol.cloud Team",
			AuthorBio:   "Helping teams deploy and manage n8n workflow automation on Google Cloud Platform.",
			AuthorImage: "https://ui-avatars.com/api/?name=instol.cloud&background=6366f1&color=fff&size=128",
			Date:        time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC),
			ReadTime:    "4 min read",
			Keywords:    []string{"instol.cloud", "n8n hosting", "managed n8n", "n8n gcp"},
			OGImage:     "https://instol.cloud/static/android-chrome-512x512.png",
			CoverImage:  "https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=1200&h=630&fit=crop",
			Category:    "Product",
		},
	}
	return posts
}

// GetBlogPostBySlug returns a single blog post by its slug
func GetBlogPostBySlug(slug string) *BlogPost {
	posts := GetAllBlogPosts()
	for i := range posts {
		if posts[i].Slug == slug {
			return &posts[i]
		}
	}
	return nil
}
