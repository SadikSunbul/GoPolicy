package handlers

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"
)

//go:embed templates/index.gohtml
var indexTemplateBytes []byte

type pageRenderer interface {
	Render(w http.ResponseWriter, data any) error
}

type htmlRenderer struct {
	tpl *template.Template
}

func newDefaultRenderer() pageRenderer {
	tpl := template.Must(template.New("index").Parse(string(indexTemplateBytes)))
	return &htmlRenderer{tpl: tpl}
}

func (r *htmlRenderer) Render(w http.ResponseWriter, data any) error {
	var buf bytes.Buffer
	if err := r.tpl.Execute(&buf, data); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := buf.WriteTo(w)
	return err
}
