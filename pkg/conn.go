package pgfs

import (
	"context"
	"fmt"

	"github.com/ardanlabs/conf/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

func New(ctx context.Context, prefix string) (*pgxpool.Pool, error) {
	var cfg Config

	// logger

	_, err := conf.Parse(prefix, &cfg)
	if err != nil {
		// logger
		return nil, fmt.Errorf("parsing pg config from prefix [%s]: %w", prefix, err)
	}
	// TODO: resolve when impl logger
	// pgxConfig, err := pgxpool.ParseConfig(cfg.ConnString())
	// if err != nil {
	// 	// logger
	// 	return nil, fmt.Errorf("parsing pgxpool config from prefix [%s]: %w", prefix, err)
	// }
	// pgxConfig.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
	// 	if ctx.Err() != nil {
	// 		// logger
	// 		return false
	// 	}
	// 	// logger
	// 	return true
	// }
	// pgxConfig.AfterRelease = func(conn *pgx.Conn) bool {
	// 	// logger
	// 	return true
	// }
	// pool, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	pool, err := pgxpool.New(ctx, cfg.ConnString())
	if err != nil {
		// logger
		return nil, fmt.Errorf("creating pgxpool: %w", err)
	}
	if err = pool.Ping(ctx); err != nil {
		// logger
		return nil, fmt.Errorf("pinging pgxpool: %w", err)
	}
	// logger successfully
	return pool, nil
}
