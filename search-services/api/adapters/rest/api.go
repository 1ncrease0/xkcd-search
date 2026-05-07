package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/1ncrease0/xkcd-search/api/core"
	"github.com/VictoriaMetrics/metrics"
)

type Authenticator interface {
	Login(user, password string) (string, error)
}

func encodeReply(w io.Writer, reply any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(reply); err != nil {
		return fmt.Errorf("could not encode reply: %v", err)
	}
	return nil
}

func NewMetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics.WritePrometheus(w, true)
	}
}

type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func NewLoginHandler(log *slog.Logger, auth Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var login LoginRequest
		err := json.NewDecoder(r.Body).Decode(&login)
		if err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}

		token, err := auth.Login(login.Name, login.Password)
		if err != nil {
			if errors.Is(err, core.ErrUnauthorized) {
				log.Debug("unauthorized", slog.String("user", login.Name))
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			log.Error("login", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, err = w.Write([]byte(token))
		if err != nil {
			log.Error("could not write response", "error", err)
		}

	}
}

type PingReply struct {
	Replies map[string]string `json:"replies"`
}

func NewPingHandler(log *slog.Logger, pingers map[string]core.Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply := PingReply{
			Replies: make(map[string]string),
		}
		for name, pinger := range pingers {
			if err := pinger.Ping(r.Context()); err != nil {
				reply.Replies[name] = "unavailable"
				log.Error("one of services is not available", "service", name, "error", err)
				continue
			}
			reply.Replies[name] = "ok"
		}
		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}

func NewUpdateHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := updater.Update(r.Context()); err != nil {
			log.Error("error while update", "error", err)
			if errors.Is(err, core.ErrAlreadyExists) {
				http.Error(w, err.Error(), http.StatusAccepted)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

type StatsReply struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

func NewUpdateStatsHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := updater.Stats(r.Context())
		if err != nil {
			log.Error("stats", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		reply := StatsReply{
			WordsTotal:    stats.WordsTotal,
			WordsUnique:   stats.WordsUnique,
			ComicsFetched: stats.ComicsFetched,
			ComicsTotal:   stats.ComicsTotal,
		}
		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}

type StatusReply struct {
	Status string `json:"status"`
}

func NewUpdateStatusHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, err := updater.Status(r.Context())
		if err != nil {
			log.Error("status", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		reply := StatusReply{Status: string(st)}
		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}

func NewDropHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := updater.Drop(r.Context()); err != nil {
			if errors.Is(err, core.ErrAlreadyExists) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			log.Error("drop", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

type ComicsReply struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type SearchReply struct {
	Comics []ComicsReply `json:"comics"`
	Total  int           `json:"total"`
}

func NewSearchHandler(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var limit int
		var err error

		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			http.Error(w, "missing phrase", http.StatusBadRequest)
			return
		}

		limitStr := r.URL.Query().Get("limit")
		if limitStr != "" {
			limit, err = strconv.Atoi(limitStr)
			if err != nil {
				http.Error(w, "bad limit", http.StatusBadRequest)
				return
			}
			if limit <= 0 {
				http.Error(w, "bad limit", http.StatusBadRequest)
				return
			}
		}

		founded, err := searcher.Search(r.Context(), phrase, limit)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				http.Error(w, "comics not founded", http.StatusNotFound)
				return
			}
			log.Error("search", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		reply := SearchReply{
			Comics: make([]ComicsReply, 0, len(founded)),
			Total:  len(founded),
		}
		for _, c := range founded {
			reply.Comics = append(reply.Comics, ComicsReply{ID: c.ID, URL: c.URL})
		}

		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}

func NewISearchHandler(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var limit int
		var err error

		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			http.Error(w, "missing phrase", http.StatusBadRequest)
			return
		}

		limitStr := r.URL.Query().Get("limit")
		if limitStr != "" {
			limit, err = strconv.Atoi(limitStr)
			if err != nil {
				http.Error(w, "bad limit", http.StatusBadRequest)
				return
			}
			if limit <= 0 {
				http.Error(w, "bad limit", http.StatusBadRequest)
				return
			}
		}

		founded, err := searcher.ISearch(r.Context(), phrase, limit)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				http.Error(w, "comics not founded", http.StatusNotFound)
				return
			}
			log.Error("search", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		reply := SearchReply{
			Comics: make([]ComicsReply, 0, len(founded)),
			Total:  len(founded),
		}
		for _, c := range founded {
			reply.Comics = append(reply.Comics, ComicsReply{ID: c.ID, URL: c.URL})
		}

		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}
