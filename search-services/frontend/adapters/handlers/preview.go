package handlers

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func allowedComicImageURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("only https allowed")
	}
	if strings.ToLower(u.Hostname()) != "imgs.xkcd.com" {
		return "", fmt.Errorf("unexpected host: %s", u.Hostname())
	}
	return u.String(), nil
}

func NewPreviewHandler(log *slog.Logger, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil || id <= 0 {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		imgURL, err := allowedComicImageURL(r.URL.Query().Get("u"))
		if err != nil {
			log.Debug("preview rejected url", "error", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		preview := &previewData{ID: id, URL: imgURL}

		if wantsHXRequest(r) {
			renderFragment(w, log, tmpl, "preview.html", preview)
			return
		}
		renderPage(w, log, tmpl, pageData{Preview: preview})
	}
}
