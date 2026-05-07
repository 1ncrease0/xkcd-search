package core

import "context"

type Client interface {
	Search(ctx context.Context, phrase string, limit int) ([]Comic, error)
	Login(ctx context.Context, username, password string) (string, error)
	DBStats(ctx context.Context, token string) (AdminStats, error)
	Ping(ctx context.Context, token string) (map[string]string, error)
	DBJobStatus(ctx context.Context, token string) (JobStatus, error)
	DBUpdate(ctx context.Context, token string) error
	DBDrop(ctx context.Context, token string) error
}
