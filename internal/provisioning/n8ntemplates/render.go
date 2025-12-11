package n8ntemplates

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed templates/*.yaml
var templates embed.FS

// Config holds the configuration for rendering N8N templates
type Config struct {
	Namespace     string
	EncryptionKey string
	BaseURL       string
	CPURequest    string
	MemoryRequest string
	CPULimit      string
	MemoryLimit   string
	StorageSize   string
}

// Render renders all N8N templates and returns them as a single multi-document YAML string
// This combines all resources into one YAML document separated by ---
// Uses embedded templates (go:embed), no filesystem access required
func Render(config Config) (string, error) {
	// Read the single merged template file
	content, err := templates.ReadFile("templates/n8n.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to read embedded template: %w", err)
	}

	// Replace placeholders with actual values
	return renderTemplate(string(content), config), nil
}

// GetTemplate returns the template file content with placeholders replaced
// This is now equivalent to Render() since we use a single template file
func GetTemplate(config Config) (string, error) {
	return Render(config)
}

// renderTemplate replaces all placeholders in the template content with actual values
func renderTemplate(content string, config Config) string {
	replacements := map[string]string{
		"PLACEHOLDER_NAMESPACE":       config.Namespace,
		"PLACEHOLDER_ENCRYPTION_KEY":  config.EncryptionKey,
		"PLACEHOLDER_BASE_URL":        config.BaseURL,
		"PLACEHOLDER_CPU_REQUEST":     config.CPURequest,
		"PLACEHOLDER_MEMORY_REQUEST":  config.MemoryRequest,
		"PLACEHOLDER_CPU_LIMIT":       config.CPULimit,
		"PLACEHOLDER_MEMORY_LIMIT":    config.MemoryLimit,
		"PLACEHOLDER_STORAGE_SIZE":    config.StorageSize,
	}

	result := content
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}
