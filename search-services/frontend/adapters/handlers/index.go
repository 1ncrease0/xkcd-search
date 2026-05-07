package handlers

import (
	"html/template"
	"log/slog"
	"net/http"
)

func NewIndexHandler(log *slog.Logger, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderPage(w, log, tmpl, pageData{})
	}
}
