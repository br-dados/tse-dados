// Package postgres implementa storage.Storage sobre as tabelas tse_*.
package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danyele/tse-dados/types"
)

// Store e o backend Postgres.
type Store struct {
	pool *pgxpool.Pool
}

// New cria o Store sobre o pool informado.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Close libera o pool.
func (s *Store) Close() { s.pool.Close() }

// CountRecords soma as linhas das tabelas principais.
func (s *Store) CountRecords(ctx context.Context) (int, error) {
	tables := []string{
		"tse_eleicao", "tse_unidade_eleitoral", "tse_partido", "tse_candidato",
		"tse_fornecedor", "tse_doador", "tse_prestacao_contas", "tse_despesa_candidato",
		"tse_despesa_orgao_partidario", "tse_receita_candidato", "tse_receita_orgao_partidario",
		"tse_receita_doador_originario_candidato", "tse_receita_doador_originario_orgao_partidario",
		"tse_bem_candidato",
	}
	total := 0
	for _, t := range tables {
		var n int
		if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+t).Scan(&n); err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

func (s *Store) tableExists(ctx context.Context, name string) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = $1`, name).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Status monta o estado agregado do indice.
func (s *Store) Status(ctx context.Context) (*types.IndexStatus, error) {
	has, err := s.tableExists(ctx, "tse_eleicao")
	if err != nil {
		return nil, err
	}
	if !has {
		return &types.IndexStatus{Exists: false}, nil
	}
	records, err := s.CountRecords(ctx)
	if err != nil {
		return nil, err
	}
	status := &types.IndexStatus{Exists: true, Records: records}

	var updatedAt *time.Time
	err = s.pool.QueryRow(ctx, `SELECT MAX(criado_em) FROM tse_arquivo_importado`).Scan(&updatedAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	status.UpdatedAt = updatedAt
	return status, nil
}
