package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/1ncrease0/xkcd-search/frontend/adapters/client"
	"github.com/1ncrease0/xkcd-search/frontend/adapters/handlers"
	"github.com/1ncrease0/xkcd-search/frontend/config"
	"github.com/1ncrease0/xkcd-search/shared/logger"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)

	log := logger.MustMake(cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		log.Error("run", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting server")
	log.Debug("debug messages are enabled")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	httpClient := client.New(cfg.APIBaseURL, cfg.ClientTimeout, log)

	tmpl := template.Must(template.New("").ParseGlob("templates/*.html"))

	mux := http.NewServeMux()
	mux.Handle("GET /", handlers.NewIndexHandler(log, tmpl))
	mux.Handle("GET /admin", handlers.NewAdminHandler(log, tmpl, httpClient))
	mux.Handle("GET /admin/partials/stats", handlers.NewAdminStatsPartialHandler(log, tmpl, httpClient))
	mux.Handle("GET /admin/partials/ping", handlers.NewAdminPingPartialHandler(log, tmpl, httpClient))
	mux.Handle("GET /admin/partials/job-status", handlers.NewAdminJobStatusPartialHandler(log, tmpl, httpClient))
	mux.Handle("POST /admin/login", handlers.NewAdminLoginHandler(log, tmpl, httpClient))
	mux.Handle("POST /admin/update", handlers.NewAdminUpdateHandler(log, tmpl, httpClient))
	mux.Handle("DELETE /admin/drop", handlers.NewAdminDropHandler(log, tmpl, httpClient))
	mux.Handle("GET /search", handlers.NewSearchHandler(log, tmpl, httpClient))
	mux.Handle("GET /preview", handlers.NewPreviewHandler(log, tmpl))
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	server := http.Server{
		Addr:        cfg.HTTPConfig.Address,
		ReadTimeout: cfg.HTTPConfig.Timeout,
		Handler:     mux,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error("erroneous shutdown", "error", err)
		}
	}()

	log.Info("running HTTP server", "address", cfg.HTTPConfig.Address)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server closed unexpectedly: %v", err)
	}

	return nil
}
