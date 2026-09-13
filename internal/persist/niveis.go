// Package persist orquestra a persistencia das entidades em niveis (0..6),
// resolvendo as FKs com upsert + RETURNING e remapeando os IDs temporarios
// para os IDs reais do banco (mesma estrategia do projeto odp).
package persist

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/danyele/tse-dados/internal/parse"
	"github.com/danyele/tse-dados/internal/repositorios"
	"github.com/danyele/tse-dados/types"
)

// Executar roda os niveis em uma transacao.
func Executar(ctx context.Context, tx pgx.Tx, repo *repositorios.Repositorio, g *parse.Grafo, lote int, resultado *repositorios.ImportacaoResultado) (int64, error) {
	if err := lockImportacao(ctx, tx); err != nil {
		return 0, err
	}

	if err := nivelDimensoes(ctx, tx, repo, g, lote, resultado); err != nil {
		return 0, err
	}
	if err := nivelCandidatos(ctx, tx, repo, g, lote, resultado); err != nil {
		return 0, err
	}
	if err := nivelFornecedoresDoadores(ctx, tx, repo, g, lote, resultado); err != nil {
		return 0, err
	}
	if err := nivelPrestacoes(ctx, tx, repo, g, lote, resultado); err != nil {
		return 0, err
	}
	if err := nivelReceitasDespesas(ctx, tx, repo, g, lote, resultado); err != nil {
		return 0, err
	}
	if err := nivelBensOrigem(ctx, tx, repo, g, lote, resultado); err != nil {
		return 0, err
	}
	return resultado.RegistrosInseridos, nil
}

func lockImportacao(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('tse_importacao'))`)
	if err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	return nil
}

func valores[K comparable, V any](m map[K]V) []V {
	out := make([]V, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// ---------------------------------------------------------------------------
// Nivel 0: dimensoes (eleicao, unidade_eleitoral, partido)
// ---------------------------------------------------------------------------

func nivelDimensoes(ctx context.Context, tx pgx.Tx, repo *repositorios.Repositorio, g *parse.Grafo, lote int, r *repositorios.ImportacaoResultado) error {
	if len(g.Eleicoes) > 0 {
		m, err := repo.InserirEleicoesComRetorno(ctx, tx, valores(g.Eleicoes), lote, r)
		if err != nil {
			return err
		}
		remapearEleicao(g, m)
	}
	if len(g.Unidades) > 0 {
		m, err := repo.InserirUnidadesEleitoraisComRetorno(ctx, tx, valores(g.Unidades), lote, r)
		if err != nil {
			return err
		}
		remapearUnidade(g, m)
	}
	if len(g.Partidos) > 0 {
		m, err := repo.InserirPartidosComRetorno(ctx, tx, valores(g.Partidos), lote, r)
		if err != nil {
			return err
		}
		remapearPartido(g, m)
	}
	return nil
}

func nivelCandidatos(ctx context.Context, tx pgx.Tx, repo *repositorios.Repositorio, g *parse.Grafo, lote int, r *repositorios.ImportacaoResultado) error {
	if len(g.Candidatos) == 0 {
		return nil
	}
	m, err := repo.InserirCandidatosComRetorno(ctx, tx, valores(g.Candidatos), lote, r)
	if err != nil {
		return err
	}
	remapearCandidato(g, m)
	return nil
}

func nivelFornecedoresDoadores(ctx context.Context, tx pgx.Tx, repo *repositorios.Repositorio, g *parse.Grafo, lote int, r *repositorios.ImportacaoResultado) error {
	if len(g.Fornecedores) > 0 {
		m, err := repo.InserirFornecedoresComRetorno(ctx, tx, valores(g.Fornecedores), lote, r)
		if err != nil {
			return err
		}
		remapearFornecedor(g, m)
	}
	if len(g.Doadores) > 0 {
		m, err := repo.InserirDoadoresComRetorno(ctx, tx, valores(g.Doadores), lote, r)
		if err != nil {
			return err
		}
		remapearDoador(g, m)
	}
	return nil
}

func nivelPrestacoes(ctx context.Context, tx pgx.Tx, repo *repositorios.Repositorio, g *parse.Grafo, lote int, r *repositorios.ImportacaoResultado) error {
	if len(g.Prestacoes) == 0 {
		return nil
	}
	// Reconstroi mapa apos dedup por chave natural (evita 21000/23514).
	dedup := make(map[string]*types.PrestacaoContas, len(g.Prestacoes))
	for _, p := range g.Prestacoes {
		dedup[prestKey(p)] = p
	}
	m, err := repo.InserirPrestacoesComRetorno(ctx, tx, valores(dedup), lote, r)
	if err != nil {
		return err
	}
	remapearPrestacao(g, m)
	return nil
}

func nivelReceitasDespesas(ctx context.Context, tx pgx.Tx, repo *repositorios.Repositorio, g *parse.Grafo, lote int, r *repositorios.ImportacaoResultado) error {
	if len(g.DespesasCandidato) > 0 {
		if _, err := repo.InserirDespesasCandidato(ctx, tx, g.DespesasCandidato, lote, r); err != nil {
			return err
		}
	}
	if len(g.DespesasOrgao) > 0 {
		if _, err := repo.InserirDespesasOrgaoPartidario(ctx, tx, g.DespesasOrgao, lote, r); err != nil {
			return err
		}
	}
	if len(g.ReceitasCandidato) > 0 {
		m, err := repo.InserirReceitasCandidatoComRetorno(ctx, tx, g.ReceitasCandidato, lote, r)
		if err != nil {
			return err
		}
		remapearReceitaCand(g, m)
	}
	if len(g.ReceitasOrgao) > 0 {
		m, err := repo.InserirReceitasOrgaoComRetorno(ctx, tx, g.ReceitasOrgao, lote, r)
		if err != nil {
			return err
		}
		remapearReceitaOrgao(g, m)
	}
	return nil
}

func nivelBensOrigem(ctx context.Context, tx pgx.Tx, repo *repositorios.Repositorio, g *parse.Grafo, lote int, r *repositorios.ImportacaoResultado) error {
	if len(g.Bens) > 0 {
		if _, err := repo.InserirBensCandidato(ctx, tx, valores(g.Bens), lote, r); err != nil {
			return err
		}
	}
	if len(g.OrigemCand) > 0 {
		if _, err := repo.InserirReceitasDoadorOriginario(ctx, tx, g.OrigemCand, lote, r); err != nil {
			return err
		}
	}
	if len(g.OrigemOrgao) > 0 {
		if _, err := repo.InserirReceitasDoadorOriginarioOrgao(ctx, tx, g.OrigemOrgao, lote, r); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Remapeamento de IDs temporarios -> IDs reais do banco
// ---------------------------------------------------------------------------

func remapearEleicao(g *parse.Grafo, m map[uuid.UUID]uuid.UUID) {
	for _, c := range g.Candidatos {
		if n, ok := m[c.EleicaoID]; ok {
			c.EleicaoID = n
		}
	}
	for _, p := range g.Prestacoes {
		if n, ok := m[p.EleicaoID]; ok {
			p.EleicaoID = n
		}
	}
}

func remapearUnidade(g *parse.Grafo, m map[uuid.UUID]uuid.UUID) {
	for _, p := range g.Prestacoes {
		if p.UnidadeEleitoralID != nil {
			if n, ok := m[*p.UnidadeEleitoralID]; ok {
				p.UnidadeEleitoralID = &n
			}
		}
	}
}

func remapearPartido(g *parse.Grafo, m map[uuid.UUID]uuid.UUID) {
	for _, c := range g.Candidatos {
		if c.PartidoID != nil {
			if n, ok := m[*c.PartidoID]; ok {
				c.PartidoID = &n
			}
		}
	}
	for _, p := range g.Prestacoes {
		if p.PartidoID != nil {
			if n, ok := m[*p.PartidoID]; ok {
				p.PartidoID = &n
			}
		}
	}
	for _, d := range g.DespesasOrgao {
		if n, ok := m[d.PartidoID]; ok {
			d.PartidoID = n
		}
	}
	for _, rc := range g.ReceitasOrgao {
		if n, ok := m[rc.PartidoID]; ok {
			rc.PartidoID = n
		}
	}
}

func remapearCandidato(g *parse.Grafo, m map[uuid.UUID]uuid.UUID) {
	for _, p := range g.Prestacoes {
		if p.CandidatoID != nil {
			if n, ok := m[*p.CandidatoID]; ok {
				p.CandidatoID = &n
			}
		}
	}
	for _, d := range g.DespesasCandidato {
		if n, ok := m[d.CandidatoID]; ok {
			d.CandidatoID = n
		}
	}
	for _, rc := range g.ReceitasCandidato {
		if n, ok := m[rc.CandidatoID]; ok {
			rc.CandidatoID = n
		}
	}
	for _, b := range g.Bens {
		if n, ok := m[b.CandidatoID]; ok {
			b.CandidatoID = n
		}
	}
}

func remapearFornecedor(g *parse.Grafo, m map[uuid.UUID]uuid.UUID) {
	for _, d := range g.DespesasCandidato {
		if d.FornecedorID != nil {
			if n, ok := m[*d.FornecedorID]; ok {
				d.FornecedorID = &n
			}
		}
	}
	for _, d := range g.DespesasOrgao {
		if d.FornecedorID != nil {
			if n, ok := m[*d.FornecedorID]; ok {
				d.FornecedorID = &n
			}
		}
	}
}

func remapearDoador(g *parse.Grafo, m map[uuid.UUID]uuid.UUID) {
	for _, rc := range g.ReceitasCandidato {
		if rc.DoadorID != nil {
			if n, ok := m[*rc.DoadorID]; ok {
				rc.DoadorID = &n
			}
		}
	}
	for _, rc := range g.ReceitasOrgao {
		if rc.DoadorID != nil {
			if n, ok := m[*rc.DoadorID]; ok {
				rc.DoadorID = &n
			}
		}
	}
}

func remapearPrestacao(g *parse.Grafo, m map[uuid.UUID]uuid.UUID) {
	for _, d := range g.DespesasCandidato {
		if n, ok := m[d.PrestacaoContasID]; ok {
			d.PrestacaoContasID = n
		}
	}
	for _, d := range g.DespesasOrgao {
		if n, ok := m[d.PrestacaoContasID]; ok {
			d.PrestacaoContasID = n
		}
	}
	for _, rc := range g.ReceitasCandidato {
		if n, ok := m[rc.PrestacaoContasID]; ok {
			rc.PrestacaoContasID = n
		}
	}
	for _, rc := range g.ReceitasOrgao {
		if n, ok := m[rc.PrestacaoContasID]; ok {
			rc.PrestacaoContasID = n
		}
	}
	for _, o := range g.OrigemCand {
		if n, ok := m[o.PrestacaoContasID]; ok {
			o.PrestacaoContasID = n
		}
	}
	for _, o := range g.OrigemOrgao {
		if n, ok := m[o.PrestacaoContasID]; ok {
			o.PrestacaoContasID = n
		}
	}
}

func remapearReceitaCand(g *parse.Grafo, m map[uuid.UUID]uuid.UUID) {
	for _, o := range g.OrigemCand {
		if o.ReceitaCandidatoID != nil {
			if n, ok := m[*o.ReceitaCandidatoID]; ok {
				o.ReceitaCandidatoID = &n
			}
		}
	}
}

func remapearReceitaOrgao(g *parse.Grafo, m map[uuid.UUID]uuid.UUID) {
	for _, o := range g.OrigemOrgao {
		if o.ReceitaOrgaoPartidarioID != nil {
			if n, ok := m[*o.ReceitaOrgaoPartidarioID]; ok {
				o.ReceitaOrgaoPartidarioID = &n
			}
		}
	}
}

func prestKey(p *types.PrestacaoContas) string {
	return p.TipoPrestador + "|" + p.EleicaoID.String() + "|" + fmt.Sprintf("%d", p.SQPrestadorContas)
}
