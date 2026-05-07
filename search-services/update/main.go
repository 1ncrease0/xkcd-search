package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	updatepb "github.com/1ncrease0/xkcd-search/proto/update"
	"github.com/1ncrease0/xkcd-search/shared/closers"
	"github.com/1ncrease0/xkcd-search/shared/logger"
	"github.com/1ncrease0/xkcd-search/update/adapters/db"
	updategrpc "github.com/1ncrease0/xkcd-search/update/adapters/grpc"
	"github.com/1ncrease0/xkcd-search/update/adapters/notifier"
	"github.com/1ncrease0/xkcd-search/update/adapters/words"
	"github.com/1ncrease0/xkcd-search/update/adapters/xkcd"
	"github.com/1ncrease0/xkcd-search/update/config"
	"github.com/1ncrease0/xkcd-search/update/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const shutdownTimeout = 5 * time.Second

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()
	cfg := config.MustLoad(configPath)

	log := logger.MustMake(cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		log.Error("server", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting server")
	log.Debug("debug messages are enabled")

	storage, err := db.New(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %v", err)
	}
	defer closers.CloseOrLog(storage, log)
	if err := storage.Migrate(); err != nil {
		return fmt.Errorf("failed to migrate db: %v", err)
	}

	xkcdClient, err := xkcd.NewClient(cfg.XKCD.URL, cfg.XKCD.Timeout, log)
	if err != nil {
		return fmt.Errorf("failed create XKCD client: %v", err)
	}

	wordsClient, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("failed create Words client: %v", err)
	}
	defer closers.CloseOrLog(wordsClient, log)

	updateNotifier, err := notifier.NewUpdateNotifier(log, cfg.Nats.Address, cfg.Nats.UpdateTopic)
	if err != nil {
		return fmt.Errorf("failed create nats update notifier: %v", err)
	}
	defer closers.CloseOrLog(updateNotifier, log)

	updater, err := core.NewService(log, storage, xkcdClient, wordsClient, updateNotifier, cfg.XKCD.Concurrency)
	if err != nil {
		return fmt.Errorf("failed create Update service: %v", err)
	}

	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	updatepb.RegisterUpdateServer(s, updategrpc.NewServer(updater))
	reflection.Register(s)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Info("shutting down server")
		time.AfterFunc(shutdownTimeout, func() {
			log.Info("forcing server stop")
			s.Stop()
		})
		s.GracefulStop()
	}()

	if err := s.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}
	return nil
}
