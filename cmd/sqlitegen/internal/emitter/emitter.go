package emitter

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

func EmitTemplate(name string, data any) (string, error) {
	tmpl, err := template.New("root").ParseFS(templateFS, "templates/*.tmpl")
	if err != nil {
		return "", fmt.Errorf("parse templates: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("execute %s: %w", name, err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("format %s: %w\n%s", name, err, buf.String())
	}
	return string(formatted), nil
}
