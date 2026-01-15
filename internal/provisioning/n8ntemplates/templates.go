package n8ntemplates

import (
	"bytes"
	"embed"
	"fmt"
)

//go:embed templates/*.yaml
var templates embed.FS

type N8N_V1 struct {
	Namespace     string
	EncryptionKey string
	BaseURL       string
	DBHost        string
	DBName        string
	DBUser        string
	DBPassword    string
}

func (t *N8N_V1) Template() string {
	return "templates/n8n.yaml"
}

func (t *N8N_V1) Content() ([]byte, error) {
	return renderTemplate(t.Template(), map[string]string{
		"PLACEHOLDER_NAMESPACE":      t.Namespace,
		"PLACEHOLDER_ENCRYPTION_KEY": t.EncryptionKey,
		"PLACEHOLDER_BASE_URL":       t.BaseURL,
		"PLACEHOLDER_DB_HOST":        t.DBHost,
		"PLACEHOLDER_DB_NAME":        t.DBName,
		"PLACEHOLDER_DB_USER":        t.DBUser,
		"PLACEHOLDER_DB_PASSWORD":    t.DBPassword,
	})
}

func renderTemplate(template string, substitutions map[string]string) ([]byte, error) {

	// Read the single merged template file
	b, err := templates.ReadFile(template)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded template: %w", err)
	}
	for placeholder, value := range substitutions {
		b = bytes.ReplaceAll(b, []byte(placeholder), []byte(value))
	}

	return b, nil
}
