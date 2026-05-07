package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/1ncrease0/xkcd-search/api/adapters/aaa"
	"github.com/1ncrease0/xkcd-search/api/adapters/rest"
	"github.com/1ncrease0/xkcd-search/api/adapters/rest/middleware"
	"github.com/1ncrease0/xkcd-search/api/adapters/search"
	"github.com/1ncrease0/xkcd-search/api/adapters/update"
	"github.com/1ncrease0/xkcd-search/api/adapters/words"
	"github.com/1ncrease0/xkcd-search/api/config"
	"github.com/1ncrease0/xkcd-search/api/core"
	"github.com/1ncrease0/xkcd-search/shared/closers"
	"github.com/1ncrease0/xkcd-search/shared/logger"
)

const shutdownTimeout = time.Second * 5

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
	wordsClient, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("cannot init words adapter: %w", err)
	}
	defer closers.CloseOrLog(wordsClient, log)

	updateClient, err := update.NewClient(cfg.UpdateAddress, log)
	if err != nil {
		return fmt.Errorf("cannot init update adapter: %w", err)
	}
	defer closers.CloseOrLog(updateClient, log)

	searchClient, err := search.NewClient(cfg.SearchAddress, log)
	if err != nil {
		return fmt.Errorf("cannot init search adapter: %w", err)
	}
	defer closers.CloseOrLog(searchClient, log)

	auth, err := aaa.New(cfg.TokenTTL, log)
	if err != nil {
		return fmt.Errorf("cannot init auth adapter: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", rest.NewMetricsHandler())
	mux.Handle("GET /api/ping", middleware.Auth(rest.NewPingHandler(log, map[string]core.Pinger{
		"words":  wordsClient,
		"update": updateClient,
		"search": searchClient,
	}), auth))
	mux.Handle("POST /api/login", rest.NewLoginHandler(log, auth))
	mux.Handle("GET /api/search", middleware.Concurrency(rest.NewSearchHandler(log, searchClient), cfg.SearchConcurrency))
	mux.Handle("GET /api/isearch", middleware.Rate(rest.NewISearchHandler(log, searchClient), cfg.SearchRate))
	mux.Handle("POST /api/db/update", middleware.Auth(rest.NewUpdateHandler(log, updateClient), auth))
	mux.Handle("GET /api/db/stats", middleware.Auth(rest.NewUpdateStatsHandler(log, updateClient), auth))
	mux.Handle("GET /api/db/status", middleware.Auth(rest.NewUpdateStatusHandler(log, updateClient), auth))
	mux.Handle("DELETE /api/db", middleware.Auth(rest.NewDropHandler(log, updateClient), auth))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := http.Server{
		Addr:        cfg.HTTPConfig.Address,
		ReadTimeout: cfg.HTTPConfig.Timeout,
		Handler:     middleware.WithMetrics(mux),
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}

	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		<-groupCtx.Done()

		shCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		log.Info("starting graceful shutdown")
		if err := server.Shutdown(shCtx); err != nil {
			return fmt.Errorf("shutdown failed: %w", err)
		}
		log.Info("server stopped")
		return nil
	})

	group.Go(func() error {
		log.Info("starting http server", "addr", cfg.HTTPConfig.Address)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server failed: %w", err)
		}
		return nil
	})

	return group.Wait()
}
