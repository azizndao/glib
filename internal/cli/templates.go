package cli

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	"golang.org/x/text/cases"
)

//go:embed templates/**/*.tmpl
var templatesFS embed.FS

var cliTemplates *template.Template

func init() {
	var err error
	cliTemplates, err = template.New("").Funcs(template.FuncMap{
		"title": cases.Title, // Keep for backward compatibility in templates
	}).ParseFS(templatesFS, "templates/**/*.tmpl")
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
