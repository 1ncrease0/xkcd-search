package db

import (
	"context"
	"log/slog"

	"github.com/lib/pq"

	"github.com/1ncrease0/xkcd-search/search/core"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {
	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}

	return &DB{
		log:  log,
		conn: db,
	}, nil
}

type comics struct {
	ID  int    `db:"id"`
	URL string `db:"url"`
}

func (db *DB) Search(ctx context.Context, keyword string) ([]int, error) {
	var IDs []int
	err := db.conn.SelectContext(
		ctx, &IDs,
		"SELECT id FROM comics WHERE $1 = ANY(words)",
		keyword,
	)

	return IDs, err
}

func (db *DB) Get(ctx context.Context, id int) (core.Comics, error) {
	var comics comics
	err := db.conn.GetContext(
		ctx, &comics,
		"SELECT id, url FROM comics WHERE id = $1",
		id,
	)

	return core.Comics{ID: comics.ID, URL: comics.URL}, err
}

type comicsWithWords struct {
	ID    int            `db:"id"`
	URL   string         `db:"url"`
	Words pq.StringArray `db:"words"`
}

func (db *DB) List(ctx context.Context) ([]core.ComicsWithWords, error) {
	var rows []comicsWithWords
	err := db.conn.SelectContext(
		ctx,
		&rows,
		`SELECT id, url, words FROM comics`,
	)
	if err != nil {
		return nil, err
	}
	result := make([]core.ComicsWithWords, 0, len(rows))
	for _, r := range rows {
		result = append(result, core.ComicsWithWords{
			ID:    r.ID,
			URL:   r.URL,
			Words: r.Words,
		})
	}
	return result, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
