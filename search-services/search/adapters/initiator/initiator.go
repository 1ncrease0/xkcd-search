package initiator

import (
	"context"
	"log/slog"
	"time"

	"github.com/1ncrease0/xkcd-search/search/core"
)

type Initiator struct {
	ttl      time.Duration
	searcher core.Searcher
	log      *slog.Logger
}

func NewInitiator(
	log *slog.Logger,
	ttl time.Duration,
	searcher core.Searcher,
) *Initiator {
	return &Initiator{
		ttl:      ttl,
		searcher: searcher,
		log:      log,
	}
}

func (i *Initiator) Start(ctx context.Context) {
	go func() {
		if err := i.searcher.BuildIndex(ctx); err != nil {
			i.log.Error("initial index build", "error", err)
		}

		ticker := time.NewTicker(i.ttl)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := i.searcher.BuildIndex(ctx); err != nil {
					i.log.Error("periodic index build", "error", err)
				}
			}
		}
	}()
}
