package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
)

type Service struct {
	log         *slog.Logger
	db          DB
	xkcd        XKCD
	words       Words
	notifier    UpdateNotifier
	concurrency int
	running     atomic.Bool
}

func NewService(
	log *slog.Logger, db DB, xkcd XKCD, words Words, notifier UpdateNotifier, concurrency int,
) (*Service, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("wrong concurrency specified: %d", concurrency)
	}
	return &Service{
		log:         log,
		db:          db,
		xkcd:        xkcd,
		words:       words,
		notifier:    notifier,
		concurrency: concurrency,
	}, nil
}

func (s *Service) Update(ctx context.Context) error {
	if !s.running.CompareAndSwap(false, true) {
		return ErrAlreadyExists
	}
	defer s.running.Store(false)

	missing, err := s.missingIDs(ctx)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		s.log.Info("database is up to date")
		return nil
	}

	s.log.Info("starting update", "missing", len(missing))
	s.runWorkers(ctx, missing)

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if err := s.notifier.Notify(ctx); err != nil {
		s.log.Error("notify after update", "error", err)
	}

	s.log.Info("update finished")
	return nil
}

func (s *Service) missingIDs(ctx context.Context) ([]int, error) {
	lastID, err := s.xkcd.LastID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get last comic id: %w", err)
	}

	existingIDs, err := s.db.IDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("get existing ids: %w", err)
	}

	existing := make(map[int]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existing[id] = struct{}{}
	}

	var missing []int
	for id := 1; id <= lastID; id++ {
		if _, ok := existing[id]; !ok {
			missing = append(missing, id)
		}
	}
	return missing, nil
}

func (s *Service) runWorkers(ctx context.Context, ids []int) {
	ch := make(chan int)
	var wg sync.WaitGroup

	for range s.concurrency {
		wg.Go(func() {
			for id := range ch {
				if err := s.processComicAdd(ctx, id); err != nil {
					if errors.Is(err, ErrNotFound) {
						s.log.Debug("comic not found, skipping", "id", id)
						continue
					}
					s.log.Error("process comic", "id", id, "error", err)
				}
			}
		})
	}

	for _, id := range ids {
		select {
		case ch <- id:
		case <-ctx.Done():
			close(ch)
			wg.Wait()
			return
		}
	}
	close(ch)
	wg.Wait()
}

func (s *Service) processComicAdd(ctx context.Context, id int) error {
	info, err := s.xkcd.Get(ctx, id)
	if err != nil {
		return err
	}

	words, err := s.words.Norm(ctx, info.Title+" "+info.Description)
	if err != nil {
		return fmt.Errorf("normalize words for comic %d: %w", id, err)
	}

	comic := Comics{
		ID:    info.ID,
		URL:   info.URL,
		Words: words,
	}

	if err := s.db.Add(ctx, comic); err != nil {
		return fmt.Errorf("save comic %d: %w", id, err)
	}

	s.log.Debug("comic processed", "id", id)
	return nil
}

func (s *Service) Stats(ctx context.Context) (ServiceStats, error) {
	dbStats, err := s.db.Stats(ctx)
	if err != nil {
		return ServiceStats{}, fmt.Errorf("get db stats: %w", err)
	}

	lastID, err := s.xkcd.LastID(ctx)
	if err != nil {
		return ServiceStats{}, fmt.Errorf("get last comic id: %w", err)
	}

	return ServiceStats{
		DBStats:     dbStats,
		ComicsTotal: lastID,
	}, nil
}

func (s *Service) Status(_ context.Context) ServiceStatus {
	if s.running.Load() {
		return StatusRunning
	}
	return StatusIdle
}

func (s *Service) Drop(ctx context.Context) error {
	if !s.running.CompareAndSwap(false, true) {
		return ErrAlreadyExists
	}
	defer s.running.Store(false)

	if err := s.db.Drop(ctx); err != nil {
		return err
	}

	if err := s.notifier.Notify(ctx); err != nil {
		s.log.Error("notify after drop", "error", err)
	}

	return nil
}
