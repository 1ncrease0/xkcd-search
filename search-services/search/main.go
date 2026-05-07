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

	searchpb "github.com/1ncrease0/xkcd-search/proto/search"
	"github.com/1ncrease0/xkcd-search/search/adapters/db"
	searchgrpc "github.com/1ncrease0/xkcd-search/search/adapters/grpc"
	"github.com/1ncrease0/xkcd-search/search/adapters/initiator"
	"github.com/1ncrease0/xkcd-search/search/adapters/subscriber"
	"github.com/1ncrease0/xkcd-search/search/adapters/words"
	"github.com/1ncrease0/xkcd-search/search/config"
	"github.com/1ncrease0/xkcd-search/search/core"
	"github.com/1ncrease0/xkcd-search/shared/closers"
	"github.com/1ncrease0/xkcd-search/shared/logger"
	"golang.org/x/sync/errgroup"
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

	wordsClient, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("failed create Words client: %v", err)
	}
	defer closers.CloseOrLog(wordsClient, log)

	searchService := core.NewService(log, storage, wordsClient)

	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	searchpb.RegisterSearchServer(s, searchgrpc.NewServer(searchService))
	reflection.Register(s)

	indexInitiator := initiator.NewInitiator(
		log,
		cfg.IndexTTL,
		searchService,
	)
	updateSubscriber := subscriber.NewUpdateSubscriber(
		log,
		cfg.Nats.Address,
		cfg.Nats.UpdateTopic,
		searchService,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	group, groupCtx := errgroup.WithContext(ctx)

	indexInitiator.Start(groupCtx)

	group.Go(func() error {
		return updateSubscriber.Subscribe(groupCtx)
	})

	group.Go(func() error {
		log.Info("starting grpc server", "addr", cfg.Address)
		return s.Serve(listener)
	})

	group.Go(func() error {
		<-groupCtx.Done()
		log.Info("shutting down server")
		time.AfterFunc(shutdownTimeout, func() {
			log.Info("forcing server stop")
			s.Stop()
		})
		s.GracefulStop()
		return nil
	})

	return group.Wait()
}
