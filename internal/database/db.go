// Package database oferece a conexao Postgres (via struct de configuracao, sem
// variaveis de ambiente) e o runner de migracoes embutidas das tabelas tse_*.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB e a superficie de banco consumida pelos repositorios e migracoes.
type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
	Ping(ctx context.Context) error
}

// Config descreve a conexao Postgres (montada por codigo/flags).
type Config struct {
	Host            string
	Port            string
	User            string
	Password        string
	Database        string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// NewPool cria um pgxpool conectado (valida com Ping).
func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	if cfg.Password == "" {
		return nil, fmt.Errorf("senha do Postgres obrigatoria (Options.Postgres.Password)")
	}
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable&pool_max_conns=%d&pool_min_conns=%d",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database,
		cfg.MaxConns, cfg.MinConns,
	)
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}
