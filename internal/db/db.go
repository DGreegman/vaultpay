package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds tunable connection-pool settings
type Config struct {
	URL					string
	MaxConns			int32 
	MinConns			int32
	MaxConnLifetime		time.Duration
	MaxConnIdleTime		time.Duration
}

// DefaultConfig returns pool settings suited to VaultPay's workload
func DefaultConfig(url string) Config {
	return Config {
		URL: url,
		MaxConns: 25,
		MinConns: 5,
		// Recycle connections periodically. Guards against a single
		// long-lived connection accumulating server-side state or
		// pinning a stale backend after a failover.
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}
}


// New opens a connection pool and verifies the database is reachable.
// It fails fast: if we cannot reach Postgres at startup, we do not start.
func New(ctx context.Context, cfg Config) (*pgxpool.Pool, error){
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("Db: parse config: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime

	// lock_timeout: a statement waiting on a row lock gives up after 3s
	// rather than blocking a pool connection indefinitely. See PRD §8.4 —
	// transfers on a hot wallet serialize on SELECT ... FOR UPDATE.
	//
	// statement_timeout: a hard ceiling on any single statement.

	poolCfg.ConnConfig.RuntimeParams["lock_timeout"] = "3000"
	poolCfg.ConnConfig.RuntimeParams["statement_timeout"] = "10000"

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("DB: Create pool: %w", err)
	}

	// Prove the database is actually reachable before returning.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("DB: Ping: %w", err)
	}

	return pool, nil
}