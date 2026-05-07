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

	wordspb "github.com/1ncrease0/xkcd-search/proto/words"
	wordsgrpc "github.com/1ncrease0/xkcd-search/words/grpc"
	"github.com/ilyakaznacheev/cleanenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	maxShutdownTime = 5 * time.Second
)

type Config struct {
	Port string `yaml:"words_address" env:"WORDS_ADDRESS" env-default:"11111"`
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic(err)
	}

	log := slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		},
	))

	if err := run(cfg, log); err != nil {
		log.Error("run", "error", err)
		os.Exit(1)
	}
}

func run(cfg Config, log *slog.Logger) error {
	listener, err := net.Listen("tcp", cfg.Port)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", cfg.Port, err)
	}

	s := grpc.NewServer()
	wordspb.RegisterWordsServer(s, wordsgrpc.NewServer())
	reflection.Register(s)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func() {
		log.Info("starting server", "port", cfg.Port)
		if err = s.Serve(listener); err != nil {
			log.Error("serve", "error", err)
			cancel()
		}
	}()

	<-ctx.Done()

	time.AfterFunc(maxShutdownTime, func() {
		log.Info("forcing server stop")
		s.Stop()
	})

	log.Info("starting graceful stop")
	s.GracefulStop()
	log.Info("server stopped")

	return err
}
