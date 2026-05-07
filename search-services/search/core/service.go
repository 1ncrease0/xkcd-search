package core

import (
	"cmp"
	"context"
	"log/slog"
	"maps"
	"slices"
	"sync"
)

type Service struct {
	log    *slog.Logger
	db     DB
	words  Words
	index  map[string][]int
	comics map[int]Comics
	mu     sync.RWMutex
}

func NewService(log *slog.Logger, db DB, words Words) *Service {
	return &Service{
		log:    log,
		db:     db,
		words:  words,
		index:  make(map[string][]int),
		comics: make(map[int]Comics),
	}
}

func (s *Service) ISearch(ctx context.Context, phrase string, limit int) ([]Comics, error) {
	keywords, err := s.words.Norm(ctx, phrase)
	if err != nil {
		s.log.Error("failed to normalize phrase", "error", err)
		return nil, err
	}
	s.log.Debug("normalized query", "keywords", keywords)
	scores := map[int]int{}

	s.mu.RLock()

	for _, keyword := range keywords {
		ids := s.index[keyword]
		for _, id := range ids {
			scores[id]++
		}
	}
	sorted := slices.SortedFunc(maps.Keys(scores), func(a, b int) int {
		return cmp.Compare(scores[b], scores[a])
	})

	if len(sorted) < limit {
		limit = len(sorted)
	}

	sorted = sorted[:limit]
	result := make([]Comics, 0, len(sorted))
	for _, id := range sorted {
		c, ok := s.comics[id]
		if !ok {
			continue
		}
		result = append(result, c)
	}

	s.mu.RUnlock()

	s.log.Debug("returning comics", "count", len(result))

	return result, nil
}

func (s *Service) BuildIndex(ctx context.Context) error {
	comicsWords, err := s.db.List(ctx)
	if err != nil {
		s.log.Error("load comics for index", "error", err)
		return err
	}
	newIndex := make(map[string][]int)
	newComics := make(map[int]Comics, len(comicsWords))

	for _, c := range comicsWords {
		newComics[c.ID] = Comics{
			ID:  c.ID,
			URL: c.URL,
		}
		seenWords := make(map[string]struct{}, len(c.Words))
		for _, w := range c.Words {
			if _, ok := seenWords[w]; ok {
				continue
			}
			seenWords[w] = struct{}{}
			newIndex[w] = append(newIndex[w], c.ID)
		}
	}

	s.mu.Lock()
	s.index = newIndex
	s.comics = newComics
	s.mu.Unlock()

	s.log.Info("index rebuilt", "keywords", len(newIndex), "comics", len(newComics))

	return nil
}

func (s *Service) Search(ctx context.Context, phrase string, limit int) ([]Comics, error) {
	keywords, err := s.words.Norm(ctx, phrase)
	if err != nil {
		s.log.Error("failed to find keywords", "error", err)
		return nil, err
	}
	s.log.Debug("normalized query", "keywords", keywords)

	scores := map[int]int{}
	for _, keyword := range keywords {
		IDs, err := s.db.Search(ctx, keyword)
		if err != nil {
			s.log.Error("failed to search keyword in DB", "error", err)
			return nil, err
		}
		for _, ID := range IDs {
			scores[ID]++
		}
	}
	s.log.Debug("relevant comics", "count", len(scores))

	sorted := slices.SortedFunc(maps.Keys(scores), func(a, b int) int {
		return cmp.Compare(scores[b], scores[a])
	})

	if len(sorted) < limit {
		limit = len(sorted)
	}
	sorted = sorted[:limit]

	result := make([]Comics, 0, len(sorted))
	for _, ID := range sorted {
		comics, err := s.db.Get(ctx, ID)
		if err != nil {
			s.log.Error("failed to fetch comics", "id", ID, "error", err)
			return nil, err
		}
		result = append(result, comics)
	}
	s.log.Debug("returning comics", "count", len(result))

	return result, nil
}
