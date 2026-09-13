package types

import "errors"

// Erros de uso da biblioteca.
var (
	// ErrEmptyScope e retornado quando o escopo nao define nenhum ano.
	ErrEmptyScope = errors.New("escopo vazio: informe ao menos um ano (Scope.Anos)")

	// ErrInvalidYear e retornado para anos fora do intervalo aceito.
	ErrInvalidYear = errors.New("ano invalido: informe um ano entre 1996 e 2100")

	// ErrUnknownDataset e retornado para tipo de planilha desconhecido.
	ErrUnknownDataset = errors.New("tipo de planilha desconhecido (use internal/dataset.ListarTipos())")

	// ErrScopeNaoSuportado e retornado para combinacoes (ano x tipo) que nao
	// existem no inventario do TSE (ver matriz em internal/dataset).
	ErrScopeNaoSuportado = errors.New("escopo nao suportado: combinacao (ano x tipo) inexistente no TSE")

	// ErrDependenciaAusente e retornado quando um tipo pede outro que nao esta
	// no escopo nem foi importado antes.
	ErrDependenciaAusente = errors.New("dependencia ausente")

	// ErrInvalidUF e retornado para sigla de UF invalida.
	ErrInvalidUF = errors.New("uf invalida: sigla de 2 letras esperada")

	// ErrPostgresRequired e retornado quando Storage=postgres sem Options.Postgres.
	ErrPostgresRequired = errors.New("configuracao Postgres (Options.Postgres) e obrigatoria para o backend \"postgres\"")

	// ErrUnsupportedStorage e retornado para backend desconhecido.
	ErrUnsupportedStorage = errors.New("backends suportados: \"postgres\", \"files\"")
)
