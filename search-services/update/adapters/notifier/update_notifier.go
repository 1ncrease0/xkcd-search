package notifier

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
)

type UpdateNotifier struct {
	conn  *nats.Conn
	topic string
	log   *slog.Logger
}

func NewUpdateNotifier(log *slog.Logger, address, topic string) (*UpdateNotifier, error) {
	nc, err := nats.Connect(address)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return &UpdateNotifier{
		conn:  nc,
		topic: topic,
		log:   log,
	}, nil
}

func (n *UpdateNotifier) Notify(ctx context.Context) error {
	if err := n.conn.Publish(n.topic, []byte("XKCD DB has been updated")); err != nil {
		return err
	}
	n.log.Debug("notify publish", "topic", n.topic)
	return nil
}

func (n *UpdateNotifier) Close() error {
	return n.conn.Drain()
}
