package handlers

import (
	"context"
	"errors"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/1ncrease0/xkcd-search/frontend/core"
)

const adminTokenCookieName = "admin_token"

func NewAdminHandler(log *slog.Logger, tmpl *template.Template, cl core.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := resolveAdminView(w, r, log, cl)
		if wantsHXRequest(r) {
			renderFragment(w, log, tmpl, "main_inner.html", data)
			return
		}
		renderPage(w, log, tmpl, data)
	}
}

func NewAdminStatsPartialHandler(log *slog.Logger, tmpl *template.Template, cl core.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !adminHXHasSessionCookie(w, r) {
			return
		}
		tok, _ := adminToken(r)
		s, err := cl.DBStats(r.Context(), tok)
		if err != nil {
			adminHXRenderPartialAPIError(w, log, tmpl, err)
			return
		}
		renderFragment(w, log, tmpl, "admin_stats_inner", &adminData{Stats: &s})
	}
}

func NewAdminPingPartialHandler(log *slog.Logger, tmpl *template.Template, cl core.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !adminHXHasSessionCookie(w, r) {
			return
		}
		tok, _ := adminToken(r)
		p, err := cl.Ping(r.Context(), tok)
		if err != nil {
			adminHXRenderPartialAPIError(w, log, tmpl, err)
			return
		}
		renderFragment(w, log, tmpl, "admin_ping_inner", &adminData{Ping: p})
	}
}

func NewAdminJobStatusPartialHandler(log *slog.Logger, tmpl *template.Template, cl core.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !adminHXHasSessionCookie(w, r) {
			return
		}
		tok, _ := adminToken(r)
		st, err := cl.DBJobStatus(r.Context(), tok)
		if err != nil {
			adminHXRenderPartialAPIError(w, log, tmpl, err)
			return
		}
		renderFragment(w, log, tmpl, "admin_job_status_inner", &adminData{JobStatus: st})
	}
}

func adminHXHasSessionCookie(w http.ResponseWriter, r *http.Request) bool {
	if _, ok := adminToken(r); !ok {
		w.Header().Set("HX-Redirect", "/admin")
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

func adminHXRenderPartialAPIError(w http.ResponseWriter, log *slog.Logger, tmpl *template.Template, err error) {
	if errors.Is(err, core.ErrUnauthorized) {
		clearAdminTokenCookie(w)
		w.Header().Set("HX-Redirect", "/admin")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	renderFragment(w, log, tmpl, "admin_partial_error", errorMessage(err))
}

func NewAdminUpdateHandler(log *slog.Logger, tmpl *template.Template, cl core.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := adminToken(r)
		if !ok {
			w.Header().Set("HX-Redirect", "/admin")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err := cl.DBUpdate(r.Context(), token); err != nil {
			log.Info("admin update", "error", err)
			if errors.Is(err, core.ErrUnauthorized) {
				clearAdminTokenCookie(w)
				w.Header().Set("HX-Redirect", "/admin")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			data := resolveAdminView(w, r, log, cl)
			data.Error = errorMessage(err)
			renderFragment(w, log, tmpl, "main_inner.html", data)
			return
		}
		data := resolveAdminView(w, r, log, cl)
		renderFragment(w, log, tmpl, "main_inner.html", data)
	}
}

func NewAdminDropHandler(log *slog.Logger, tmpl *template.Template, cl core.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := adminToken(r)
		if !ok {
			w.Header().Set("HX-Redirect", "/admin")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err := cl.DBDrop(r.Context(), token); err != nil {
			log.Info("admin drop", "error", err)
			if errors.Is(err, core.ErrUnauthorized) {
				clearAdminTokenCookie(w)
				w.Header().Set("HX-Redirect", "/admin")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			data := resolveAdminView(w, r, log, cl)
			data.Error = errorMessage(err)
			renderFragment(w, log, tmpl, "main_inner.html", data)
			return
		}
		data := resolveAdminView(w, r, log, cl)
		renderFragment(w, log, tmpl, "main_inner.html", data)
	}
}

func NewAdminLoginHandler(log *slog.Logger, tmpl *template.Template, cl core.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.FormValue("username")
		password := r.FormValue("password")

		token, err := cl.Login(r.Context(), username, password)
		if err != nil {
			log.Info("admin login", "error", err)
			data := pageData{
				Admin: &adminData{},
				Error: errorMessage(err),
			}
			if wantsHXRequest(r) {
				renderFragment(w, log, tmpl, "main_inner.html", data)
				return
			}
			renderPage(w, log, tmpl, data)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     adminTokenCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		stats, ping, job, err := loadAdminInfo(r.Context(), cl, token)
		data := pageData{
			Admin: &adminData{
				Stats:     stats,
				Ping:      ping,
				JobStatus: job,
			},
		}
		if err != nil {
			log.Info("admin bootstrap after login", "error", err)
			data.Error = errorMessage(err)
		}

		if wantsHXRequest(r) {
			renderFragment(w, log, tmpl, "main_inner.html", data)
			return
		}
		renderPage(w, log, tmpl, data)
	}
}

func resolveAdminView(w http.ResponseWriter, r *http.Request, log *slog.Logger, cl core.Client) pageData {
	token, ok := adminToken(r)
	if !ok {
		return pageData{Admin: &adminData{}}
	}

	stats, ping, job, err := loadAdminInfo(r.Context(), cl, token)
	if err != nil {
		log.Info("admin bootstrap", "error", err)
		if errors.Is(err, core.ErrUnauthorized) {
			clearAdminTokenCookie(w)
			return pageData{
				Admin: &adminData{},
				Error: errorMessage(err),
			}
		}
		return pageData{
			Admin: &adminData{},
			Error: errorMessage(err),
		}
	}

	return pageData{
		Admin: &adminData{
			Stats:     stats,
			Ping:      ping,
			JobStatus: job,
		},
	}
}

func loadAdminInfo(ctx context.Context, cl core.Client, token string) (stats *core.AdminStats, ping map[string]string, job core.JobStatus, err error) {
	s, err := cl.DBStats(ctx, token)
	if err != nil {
		return nil, nil, "", err
	}
	p, err := cl.Ping(ctx, token)
	if err != nil {
		return nil, nil, "", err
	}
	j, err := cl.DBJobStatus(ctx, token)
	if err != nil {
		return nil, nil, "", err
	}
	return &s, p, j, nil
}

func adminToken(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(adminTokenCookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}

func clearAdminTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}
