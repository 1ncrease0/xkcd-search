package handlers

import (
	"errors"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/1ncrease0/xkcd-search/frontend/core"
)

func wantsHXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func errorMessage(err error) string {
	if errors.Is(err, core.ErrUnauthorized) {
		return "Введите верный логин и пароль для авторизации"
	}
	if errors.Is(err, core.ErrRateLimited) {
		return "Слишком много запросов, подождите немного"
	}
	return "Ошибка сервиса, попробуйте позже :("
}

func renderFragment(w http.ResponseWriter, log *slog.Logger, tmpl *template.Template, name string, data any) {
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Error("render template", "template", name, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func renderPage(w http.ResponseWriter, log *slog.Logger, tmpl *template.Template, data pageData) {
	renderFragment(w, log, tmpl, "index.html", data)
}
