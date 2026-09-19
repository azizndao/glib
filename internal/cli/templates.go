package cli

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"golang.org/x/text/cases"
)

//go:embed templates/**/*.templ
var templatesFS embed.FS

var cliTemplates *template.Template

func init() {
	var err error
	cliTemplates, err = template.New("").Funcs(template.FuncMap{
		"title": cases.Title, // Keep for backward compatibility in templates
	}).ParseFS(templatesFS, "templates/**/*.templ")
	if err != nil {
		panic(fmt.Sprintf("failed to parse CLI templates: %v", err))
	}
}

func executeTemplate(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := cliTemplates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}
	return buf.String(), nil
}

func renderTemplate(name string, data any) string {
	result, err := executeTemplate(name, data)
	if err != nil {
		panic(err)
	}
	return result
}

func writeGeneratedFile(rootDir, relativePath, content string) error {
	fullPath := filepath.Join(rootDir, relativePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", relativePath, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", relativePath, err)
	}
	return nil
}
