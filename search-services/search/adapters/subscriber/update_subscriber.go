package subscriber

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/1ncrease0/xkcd-search/search/core"
	"github.com/nats-io/nats.go"
)

type UpdateSubscriber struct {
	address  string
	topic    string
	searcher core.Searcher
	log      *slog.Logger
}

func NewUpdateSubscriber(
	log *slog.Logger,
	address, topic string,
	searcher core.Searcher,
) *UpdateSubscriber {
	return &UpdateSubscriber{
		address:  address,
		topic:    topic,
		searcher: searcher,
		log:      log,
	}
}

func (s *UpdateSubscriber) Subscribe(ctx context.Context) error {
	nc, err := nats.Connect(s.address)
	if err != nil {
		return fmt.Errorf("nats connect: %w", err)
	}
	defer nc.Close()

	trigger := make(chan struct{}, 1)
	sub, err := nc.Subscribe(s.topic, func(_ *nats.Msg) {
		select {
		case trigger <- struct{}{}:
		default:
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe to %s: %w", s.topic, err)
	}
	defer func() {
		if err := sub.Unsubscribe(); err != nil {
			s.log.Error("unsubscribe", "error", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-trigger:
			if err := s.searcher.BuildIndex(ctx); err != nil {
				s.log.Error("index build after event", "error", err)
			}
		}
	}
}
