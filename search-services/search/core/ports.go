package core

import (
	"context"
)

type Searcher interface {
	Search(ctx context.Context, phrase string, limit int) ([]Comics, error)
	ISearch(ctx context.Context, phrase string, limit int) ([]Comics, error)
	BuildIndex(ctx context.Context) error
}

type Initiator interface {
	Start(ctx context.Context)
}

type UpdateSubscriber interface {
	Subscribe(ctx context.Context) error
}

type DB interface {
	List(ctx context.Context) ([]ComicsWithWords, error)
	Search(ctx context.Context, keyword string) ([]int, error)
	Get(ctx context.Context, ID int) (Comics, error)
}

type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}
