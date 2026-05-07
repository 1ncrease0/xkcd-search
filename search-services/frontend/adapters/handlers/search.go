package handlers

import (
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/1ncrease0/xkcd-search/frontend/core"
)

const defaultLimit = 10

func NewSearchHandler(log *slog.Logger, tmpl *template.Template, cl core.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := strings.TrimSpace(r.URL.Query().Get("phrase"))
		limit := defaultLimit
		if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
			limit = l
		}

		var data pageData

		if phrase != "" {
			comics, err := cl.Search(r.Context(), phrase, limit)
			if err != nil {
				log.Error("search", "error", err)
				data.Error = errorMessage(err)
			} else {
				data.Search = &searchData{Comics: comics}
			}
		}

		if wantsHXRequest(r) {
			renderFragment(w, log, tmpl, "main_inner.html", data)
			return
		}
		renderPage(w, log, tmpl, data)
	}
}
