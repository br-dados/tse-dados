// Package storage define a superficie consultavel de saida da biblioteca
// tse-dados. O pipeline de escrita segue o fluxo descendente
// fetch -> python(extract) -> parse -> persist (repositorios), mas as
// consultas de estado/cobertura expostas pelo cliente passam por esta interface,
// permitindo implementacoes alternativas (ex.: planilha/arquivo) no futuro.
package storage

import (
	"context"

	"github.com/danyele/tse-dados/types"
)

// Storage e o conjunto minimo de consultas que a biblioteca expoe.
type Storage interface {
	// Status monta o estado agregado do indice.
	Status(ctx context.Context) (*types.IndexStatus, error)

	// CountRecords conta as linhas persistidas.
	CountRecords(ctx context.Context) (int, error)

	// Close libera os recursos do backend.
	Close()
}
