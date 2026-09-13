package repositorios

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danyele/tse-dados/types"
)

// ImportacaoResultado acumula as metricas de uma persistencia.
type ImportacaoResultado struct {
	mu                 sync.Mutex
	RegistrosInseridos int64
	Operacoes          int
	TempoCOPY          time.Duration
	TempoMerge         time.Duration
	SetEntidade        func(string)
}

// Repositorio acessa o pool Postgres das tabelas tse_*.
type Repositorio struct {
	pool *pgxpool.Pool
}

// Novo cria um Repositorio sobre o pool informado.
func Novo(pool *pgxpool.Pool) *Repositorio { return &Repositorio{pool: pool} }

func (r *Repositorio) InserirEleicoesComRetorno(
	ctx context.Context, tx pgx.Tx, valores []*types.Eleicao, lote int, resultado *ImportacaoResultado,
) (map[uuid.UUID]uuid.UUID, error) {
	columns := []string{"id", "codigo_tse", "ano", "codigo_tipo_eleicao", "nome_tipo_eleicao", "descricao", "data_eleicao"}
	return copyInsertReturning(ctx, tx, valores, lote, "tse_eleicao", columns, "(codigo_tse)",
		"ano = EXCLUDED.ano, codigo_tipo_eleicao = EXCLUDED.codigo_tipo_eleicao, nome_tipo_eleicao = EXCLUDED.nome_tipo_eleicao, descricao = EXCLUDED.descricao, data_eleicao = EXCLUDED.data_eleicao, updated_at = NOW()",
		[]string{"id", "codigo_tse"},
		func(v *types.Eleicao) []any {
			return []any{v.ID, v.CodigoTSE, v.Ano, v.CodigoTipoEleicao, v.NomeTipoEleicao, v.Descricao, v.DataEleicao}
		},
		func(v *types.Eleicao) string { return strconv.Itoa(v.CodigoTSE) },
		resultado)
}

func (r *Repositorio) InserirUnidadesEleitoraisComRetorno(
	ctx context.Context, tx pgx.Tx, valores []*types.UnidadeEleitoral, lote int, resultado *ImportacaoResultado,
) (map[uuid.UUID]uuid.UUID, error) {
	columns := []string{"id", "sg_uf", "codigo_tse", "nome"}
	return copyInsertReturning(ctx, tx, valores, lote, "tse_unidade_eleitoral", columns, "(sg_uf, codigo_tse)",
		"nome = EXCLUDED.nome, updated_at = NOW()",
		[]string{"id", "sg_uf", "codigo_tse"},
		func(v *types.UnidadeEleitoral) []any {
			return []any{v.ID, v.UFSigla, v.CodigoTSE, v.Nome}
		},
		func(v *types.UnidadeEleitoral) string { return v.UFSigla + "|" + v.CodigoTSE },
		resultado)
}

func (r *Repositorio) InserirPartidosComRetorno(
	ctx context.Context, tx pgx.Tx, valores []*types.Partido, lote int, resultado *ImportacaoResultado,
) (map[uuid.UUID]uuid.UUID, error) {
	columns := []string{"id", "numero", "sigla", "nome", "federacao_codigo_tse", "federacao_sigla", "federacao_nome", "coligacao_codigo_tse", "coligacao_nome", "coligacao_composicao"}
	return copyInsertReturning(ctx, tx, valores, lote, "tse_partido", columns, "(numero)",
		"sigla = EXCLUDED.sigla, nome = EXCLUDED.nome, federacao_codigo_tse = EXCLUDED.federacao_codigo_tse, federacao_sigla = EXCLUDED.federacao_sigla, federacao_nome = EXCLUDED.federacao_nome, coligacao_codigo_tse = EXCLUDED.coligacao_codigo_tse, coligacao_nome = EXCLUDED.coligacao_nome, coligacao_composicao = EXCLUDED.coligacao_composicao, updated_at = NOW()",
		[]string{"id", "numero"},
		func(v *types.Partido) []any {
			return []any{v.ID, v.Numero, v.Sigla, v.Nome, v.FederacaoCodigoTSE, v.FederacaoSigla, v.FederacaoNome, v.ColigacaoCodigoTSE, v.ColigacaoNome, v.ColigacaoComposicao}
		},
		func(v *types.Partido) string { return strconv.Itoa(int(v.Numero)) },
		resultado)
}

func (r *Repositorio) InserirCandidatosComRetorno(
	ctx context.Context, tx pgx.Tx, valores []*types.Candidato, lote int, resultado *ImportacaoResultado,
) (map[uuid.UUID]uuid.UUID, error) {
	columns := []string{"id", "sq_candidato", "eleicao_id", "sg_uf", "partido_id", "cargo_codigo", "cargo_nome", "genero_descricao", "cor_raca_descricao", "estado_civil_nome", "grau_instrucao_nome", "ocupacao_codigo", "ocupacao_nome", "numero_candidato", "cpf", "cpf_vice", "nome_completo", "nome_urna", "nome_social", "data_nascimento", "situacao_totalizacao_descricao"}
	return copyInsertReturning(ctx, tx, valores, lote, "tse_candidato", columns, "(sq_candidato)",
		"eleicao_id = EXCLUDED.eleicao_id, sg_uf = EXCLUDED.sg_uf, partido_id = EXCLUDED.partido_id, cargo_codigo = EXCLUDED.cargo_codigo, cargo_nome = EXCLUDED.cargo_nome, genero_descricao = EXCLUDED.genero_descricao, cor_raca_descricao = EXCLUDED.cor_raca_descricao, estado_civil_nome = EXCLUDED.estado_civil_nome, grau_instrucao_nome = EXCLUDED.grau_instrucao_nome, ocupacao_codigo = EXCLUDED.ocupacao_codigo, ocupacao_nome = EXCLUDED.ocupacao_nome, numero_candidato = EXCLUDED.numero_candidato, cpf = EXCLUDED.cpf, cpf_vice = EXCLUDED.cpf_vice, nome_completo = EXCLUDED.nome_completo, nome_urna = EXCLUDED.nome_urna, nome_social = EXCLUDED.nome_social, data_nascimento = EXCLUDED.data_nascimento, situacao_totalizacao_descricao = EXCLUDED.situacao_totalizacao_descricao, updated_at = NOW()",
		[]string{"id", "sq_candidato"},
		func(v *types.Candidato) []any {
			return []any{v.ID, v.SQCandidato, v.EleicaoID, v.UFSigla, v.PartidoID, v.CargoCodigo, v.CargoNome, v.GeneroDescricao, v.CorRacaDescricao, v.EstadoCivilNome, v.GrauInstrucaoNome, v.OcupacaoCodigo, v.OcupacaoNome, v.NumeroCandidato, StrNil(v.CPF), StrNil(v.CPFVice), v.NomeCompleto, StrNil(v.NomeUrna), StrNil(v.NomeSocial), v.DataNascimento, StrNil(v.SituacaoTotalizacaoDescricao)}
		},
		func(v *types.Candidato) string { return strconv.FormatInt(v.SQCandidato, 10) },
		resultado)
}

func (r *Repositorio) InserirFornecedoresComRetorno(
	ctx context.Context, tx pgx.Tx, valores []*types.Fornecedor, lote int, resultado *ImportacaoResultado,
) (map[uuid.UUID]uuid.UUID, error) {
	columns := []string{"id", "cpf_cnpj", "nome", "nome_rfb", "tipo_fornecedor_codigo", "tipo_fornecedor_descricao", "cnae_codigo", "cnae_descricao", "esfera_partidaria_codigo", "esfera_partidaria_descricao", "sg_uf", "municipio_nome", "sq_candidato_relacionado", "numero_candidato_relacionado", "cargo_codigo_relacionado", "cargo_descricao_relacionada", "partido_numero_relacionado", "partido_sigla_relacionado", "partido_nome_relacionado"}
	return copyInsertReturning(ctx, tx, valores, lote, "tse_fornecedor", columns, "(cpf_cnpj)",
		"nome = EXCLUDED.nome, nome_rfb = EXCLUDED.nome_rfb, tipo_fornecedor_codigo = EXCLUDED.tipo_fornecedor_codigo, tipo_fornecedor_descricao = EXCLUDED.tipo_fornecedor_descricao, cnae_codigo = EXCLUDED.cnae_codigo, cnae_descricao = EXCLUDED.cnae_descricao, esfera_partidaria_codigo = EXCLUDED.esfera_partidaria_codigo, esfera_partidaria_descricao = EXCLUDED.esfera_partidaria_descricao, sg_uf = EXCLUDED.sg_uf, municipio_nome = EXCLUDED.municipio_nome, sq_candidato_relacionado = EXCLUDED.sq_candidato_relacionado, numero_candidato_relacionado = EXCLUDED.numero_candidato_relacionado, cargo_codigo_relacionado = EXCLUDED.cargo_codigo_relacionado, cargo_descricao_relacionada = EXCLUDED.cargo_descricao_relacionada, partido_numero_relacionado = EXCLUDED.partido_numero_relacionado, partido_sigla_relacionado = EXCLUDED.partido_sigla_relacionado, partido_nome_relacionado = EXCLUDED.partido_nome_relacionado, updated_at = NOW()",
		[]string{"id", "cpf_cnpj"},
		func(v *types.Fornecedor) []any {
			return []any{v.ID, v.CPFCNPJ, StrNil(v.Nome), StrNil(v.NomeRFB), v.TipoFornecedorCodigo, StrNil(v.TipoFornecedorDescricao), StrNil(v.CNAECodigo), StrNil(v.CNAEDescricao), StrNil(v.EsferaPartidariaCodigo), StrNil(v.EsferaPartidariaDescricao), strNilPtr(v.UFSigla), StrNil(v.MunicipioNome), v.SQCandidatoRelacionado, v.NumeroCandidatoRelacionado, v.CargoCodigoRelacionado, StrNil(v.CargoDescricaoRelacionada), v.PartidoNumeroRelacionado, StrNil(v.PartidoSiglaRelacionado), StrNil(v.PartidoNomeRelacionado)}
		},
		func(v *types.Fornecedor) string { return v.CPFCNPJ },
		resultado)
}

func (r *Repositorio) InserirDoadoresComRetorno(
	ctx context.Context, tx pgx.Tx, valores []*types.Doador, lote int, resultado *ImportacaoResultado,
) (map[uuid.UUID]uuid.UUID, error) {
	columns := []string{"id", "cpf_cnpj", "nome", "nome_rfb", "cnae_codigo", "cnae_descricao", "esfera_partidaria_codigo", "esfera_partidaria_descricao", "sg_uf", "municipio_nome", "sq_candidato_relacionado", "numero_candidato_relacionado", "cargo_codigo_relacionado", "cargo_descricao_relacionada", "partido_numero_relacionado", "partido_sigla_relacionado", "partido_nome_relacionado"}
	return copyInsertReturning(ctx, tx, valores, lote, "tse_doador", columns, "(cpf_cnpj)",
		"nome = EXCLUDED.nome, nome_rfb = EXCLUDED.nome_rfb, cnae_codigo = EXCLUDED.cnae_codigo, cnae_descricao = EXCLUDED.cnae_descricao, esfera_partidaria_codigo = EXCLUDED.esfera_partidaria_codigo, esfera_partidaria_descricao = EXCLUDED.esfera_partidaria_descricao, sg_uf = EXCLUDED.sg_uf, municipio_nome = EXCLUDED.municipio_nome, sq_candidato_relacionado = EXCLUDED.sq_candidato_relacionado, numero_candidato_relacionado = EXCLUDED.numero_candidato_relacionado, cargo_codigo_relacionado = EXCLUDED.cargo_codigo_relacionado, cargo_descricao_relacionada = EXCLUDED.cargo_descricao_relacionada, partido_numero_relacionado = EXCLUDED.partido_numero_relacionado, partido_sigla_relacionado = EXCLUDED.partido_sigla_relacionado, partido_nome_relacionado = EXCLUDED.partido_nome_relacionado, updated_at = NOW()",
		[]string{"id", "cpf_cnpj"},
		func(v *types.Doador) []any {
			return []any{v.ID, v.CPFCNPJ, StrNil(v.Nome), StrNil(v.NomeRFB), StrNil(v.CNAECodigo), StrNil(v.CNAEDescricao), StrNil(v.EsferaPartidariaCodigo), StrNil(v.EsferaPartidariaDescricao), strNilPtr(v.UFSigla), StrNil(v.MunicipioNome), v.SQCandidatoRelacionado, v.NumeroCandidatoRelacionado, v.CargoCodigoRelacionado, StrNil(v.CargoDescricaoRelacionada), v.PartidoNumeroRelacionado, StrNil(v.PartidoSiglaRelacionado), StrNil(v.PartidoNomeRelacionado)}
		},
		func(v *types.Doador) string { return v.CPFCNPJ },
		resultado)
}

func (r *Repositorio) InserirPrestacoesComRetorno(
	ctx context.Context, tx pgx.Tx, valores []*types.PrestacaoContas, lote int, resultado *ImportacaoResultado,
) (map[uuid.UUID]uuid.UUID, error) {
	columns := []string{"id", "sq_prestador_contas", "eleicao_id", "candidato_id", "partido_id", "sg_uf", "unidade_eleitoral_id", "tipo_prestador", "tipo_prestacao", "data_prestacao", "turno", "cnpj_prestador_conta", "esfera_partidaria_codigo", "esfera_partidaria_descricao"}
	return copyInsertReturning(ctx, tx, valores, lote, "tse_prestacao_contas", columns, "(tipo_prestador, eleicao_id, sq_prestador_contas)",
		"candidato_id = EXCLUDED.candidato_id, partido_id = EXCLUDED.partido_id, sg_uf = EXCLUDED.sg_uf, unidade_eleitoral_id = EXCLUDED.unidade_eleitoral_id, tipo_prestacao = EXCLUDED.tipo_prestacao, data_prestacao = EXCLUDED.data_prestacao, turno = EXCLUDED.turno, cnpj_prestador_conta = EXCLUDED.cnpj_prestador_conta, esfera_partidaria_codigo = EXCLUDED.esfera_partidaria_codigo, esfera_partidaria_descricao = EXCLUDED.esfera_partidaria_descricao, updated_at = NOW()",
		[]string{"id", "tipo_prestador", "eleicao_id", "sq_prestador_contas"},
		func(v *types.PrestacaoContas) []any {
			return []any{v.ID, v.SQPrestadorContas, v.EleicaoID, v.CandidatoID, v.PartidoID, strNilPtr(v.UFSigla), v.UnidadeEleitoralID, v.TipoPrestador, StrNil(v.TipoPrestacao), v.DataPrestacao, v.Turno, StrNil(v.CNPJPrestadorConta), StrNil(v.EsferaPartidariaCodigo), StrNil(v.EsferaPartidariaDescricao)}
		},
		func(v *types.PrestacaoContas) string {
			return v.TipoPrestador + "|" + v.EleicaoID.String() + "|" + strconv.FormatInt(v.SQPrestadorContas, 10)
		},
		resultado)
}

func (r *Repositorio) InserirDespesasCandidato(
	ctx context.Context, tx pgx.Tx, valores []*types.DespesaCandidato, lote int, resultado *ImportacaoResultado,
) (int64, error) {
	columns := []string{"id", "prestacao_contas_id", "candidato_id", "fornecedor_id", "sq_despesa", "tipo_registro", "tipo_documento", "numero_documento", "origem_despesa_codigo", "origem_despesa_descricao", "fonte_despesa_codigo", "fonte_despesa_descricao", "natureza_despesa_codigo", "natureza_despesa_descricao", "especie_recurso_codigo", "especie_recurso_descricao", "sq_parcelamento_despesa", "data_despesa", "descricao", "valor"}
	return copyInsertEmLote(ctx, tx, valores, lote, "tse_despesa_candidato", columns, "",
		func(v *types.DespesaCandidato) []any {
			return []any{v.ID, v.PrestacaoContasID, v.CandidatoID, v.FornecedorID, v.SQDespesa, v.TipoRegistro, StrNil(v.TipoDocumento), StrNil(v.NumeroDocumento), v.OrigemDespesaCodigo, StrNil(v.OrigemDespesaDescricao), v.FonteDespesaCodigo, StrNil(v.FonteDespesaDescricao), v.NaturezaDespesaCodigo, StrNil(v.NaturezaDespesaDescricao), v.EspecieRecursoCodigo, StrNil(v.EspecieRecursoDescricao), v.SQPlanoParcelamento, v.DataDespesa, StrNil(v.Descricao), v.Valor}
		},
		"", resultado)
}

func (r *Repositorio) InserirDespesasOrgaoPartidario(
	ctx context.Context, tx pgx.Tx, valores []*types.DespesaOrgaoPartidario, lote int, resultado *ImportacaoResultado,
) (int64, error) {
	columns := []string{"id", "prestacao_contas_id", "partido_id", "fornecedor_id", "sq_despesa", "tipo_registro", "tipo_documento", "numero_documento", "origem_despesa_codigo", "origem_despesa_descricao", "fonte_despesa_codigo", "fonte_despesa_descricao", "natureza_despesa_codigo", "natureza_despesa_descricao", "especie_recurso_codigo", "especie_recurso_descricao", "sq_parcelamento_despesa", "data_despesa", "descricao", "valor"}
	return copyInsertEmLote(ctx, tx, valores, lote, "tse_despesa_orgao_partidario", columns, "",
		func(v *types.DespesaOrgaoPartidario) []any {
			return []any{v.ID, v.PrestacaoContasID, v.PartidoID, v.FornecedorID, v.SQDespesa, v.TipoRegistro, StrNil(v.TipoDocumento), StrNil(v.NumeroDocumento), v.OrigemDespesaCodigo, StrNil(v.OrigemDespesaDescricao), v.FonteDespesaCodigo, StrNil(v.FonteDespesaDescricao), v.NaturezaDespesaCodigo, StrNil(v.NaturezaDespesaDescricao), v.EspecieRecursoCodigo, StrNil(v.EspecieRecursoDescricao), v.SQPlanoParcelamento, v.DataDespesa, StrNil(v.Descricao), v.Valor}
		},
		"", resultado)
}

func (r *Repositorio) InserirReceitasCandidatoComRetorno(
	ctx context.Context, tx pgx.Tx, valores []*types.ReceitaCandidato, lote int, resultado *ImportacaoResultado,
) (map[uuid.UUID]uuid.UUID, error) {
	columns := []string{"id", "prestacao_contas_id", "candidato_id", "doador_id", "sq_receita", "fonte_receita_codigo", "fonte_receita_descricao", "origem_receita_codigo", "origem_receita_descricao", "natureza_receita_codigo", "natureza_receita_descricao", "especie_receita_codigo", "especie_receita_descricao", "numero_recibo_doacao", "numero_documento_doacao", "data_receita", "descricao", "valor", "natureza_recurso_estimavel", "genero", "cor_raca"}
	return copyInsertReturning(ctx, tx, valores, lote, "tse_receita_candidato", columns, "", "", []string{"id", "sq_receita"},
		func(v *types.ReceitaCandidato) []any {
			return []any{v.ID, v.PrestacaoContasID, v.CandidatoID, v.DoadorID, v.SQReceita, v.FonteReceitaCodigo, StrNil(v.FonteReceitaDescricao), v.OrigemReceitaCodigo, StrNil(v.OrigemReceitaDescricao), v.NaturezaReceitaCodigo, StrNil(v.NaturezaReceitaDescricao), v.EspecieReceitaCodigo, StrNil(v.EspecieReceitaDescricao), StrNil(v.NumeroReciboDoacao), StrNil(v.NumeroDocumentoDoacao), v.DataReceita, StrNil(v.Descricao), v.Valor, StrNil(v.NaturezaRecursoEstimavel), StrNil(v.Genero), StrNil(v.CorRaca)}
		},
		func(v *types.ReceitaCandidato) string { return strconv.FormatInt(v.SQReceita, 10) },
		resultado)
}

func (r *Repositorio) InserirReceitasOrgaoComRetorno(
	ctx context.Context, tx pgx.Tx, valores []*types.ReceitaOrgaoPartidario, lote int, resultado *ImportacaoResultado,
) (map[uuid.UUID]uuid.UUID, error) {
	columns := []string{"id", "prestacao_contas_id", "partido_id", "doador_id", "sq_receita", "fonte_receita_codigo", "fonte_receita_descricao", "origem_receita_codigo", "origem_receita_descricao", "natureza_receita_codigo", "natureza_receita_descricao", "especie_receita_codigo", "especie_receita_descricao", "numero_recibo_doacao", "numero_documento_doacao", "data_receita", "descricao", "valor"}
	return copyInsertReturning(ctx, tx, valores, lote, "tse_receita_orgao_partidario", columns, "", "", []string{"id", "sq_receita"},
		func(v *types.ReceitaOrgaoPartidario) []any {
			return []any{v.ID, v.PrestacaoContasID, v.PartidoID, v.DoadorID, v.SQReceita, v.FonteReceitaCodigo, StrNil(v.FonteReceitaDescricao), v.OrigemReceitaCodigo, StrNil(v.OrigemReceitaDescricao), v.NaturezaReceitaCodigo, StrNil(v.NaturezaReceitaDescricao), v.EspecieReceitaCodigo, StrNil(v.EspecieReceitaDescricao), StrNil(v.NumeroReciboDoacao), StrNil(v.NumeroDocumentoDoacao), v.DataReceita, StrNil(v.Descricao), v.Valor}
		},
		func(v *types.ReceitaOrgaoPartidario) string { return strconv.FormatInt(v.SQReceita, 10) },
		resultado)
}

func (r *Repositorio) InserirBensCandidato(
	ctx context.Context, tx pgx.Tx, valores []*types.BemCandidato, lote int, resultado *ImportacaoResultado,
) (int64, error) {
	columns := []string{"id", "candidato_id", "tipo_bem_codigo", "tipo_bem_nome", "numero_ordem", "descricao", "valor", "data_ultima_atualizacao", "hora_ultima_atualizacao"}
	return copyInsertEmLote(ctx, tx, valores, lote, "tse_bem_candidato", columns, "(candidato_id, numero_ordem)",
		func(v *types.BemCandidato) []any {
			return []any{v.ID, v.CandidatoID, v.TipoBemCodigo, StrNil(v.TipoBemNome), v.NumeroOrdem, StrNil(v.Descricao), v.Valor, v.DataUltimaAtualizacao, StrNil(v.HoraUltimaAtualizacao)}
		},
		"candidato_id = EXCLUDED.candidato_id, tipo_bem_codigo = EXCLUDED.tipo_bem_codigo, tipo_bem_nome = EXCLUDED.tipo_bem_nome, numero_ordem = EXCLUDED.numero_ordem, descricao = EXCLUDED.descricao, valor = EXCLUDED.valor, data_ultima_atualizacao = EXCLUDED.data_ultima_atualizacao, hora_ultima_atualizacao = EXCLUDED.hora_ultima_atualizacao, updated_at = NOW()",
		resultado)
}

func (r *Repositorio) InserirReceitasDoadorOriginario(
	ctx context.Context, tx pgx.Tx, valores []*types.ReceitaDoadorOriginarioCandidato, lote int, resultado *ImportacaoResultado,
) (int64, error) {
	columns := []string{"id", "prestacao_contas_id", "receita_candidato_id", "sq_receita", "documento_doador", "nome_doador", "nome_doador_rfb", "tipo_doador", "cnae_codigo", "cnae_descricao", "data_receita", "descricao", "valor"}
	return copyInsertEmLote(ctx, tx, valores, lote, "tse_receita_doador_originario_candidato", columns, "",
		func(v *types.ReceitaDoadorOriginarioCandidato) []any {
			return []any{v.ID, v.PrestacaoContasID, v.ReceitaCandidatoID, v.SQReceita, StrNil(v.DocumentoDoador), StrNil(v.NomeDoador), StrNil(v.NomeDoadorRFB), StrNil(v.TipoDoador), StrNil(v.CNAECodigo), StrNil(v.CNAEDescricao), v.DataReceita, StrNil(v.Descricao), v.Valor}
		},
		"", resultado)
}

func (r *Repositorio) InserirReceitasDoadorOriginarioOrgao(
	ctx context.Context, tx pgx.Tx, valores []*types.ReceitaDoadorOriginarioOrgaoPartidario, lote int, resultado *ImportacaoResultado,
) (int64, error) {
	columns := []string{"id", "prestacao_contas_id", "receita_orgao_partidario_id", "sq_receita", "documento_doador", "nome_doador", "nome_doador_rfb", "tipo_doador", "cnae_codigo", "cnae_descricao", "data_receita", "descricao", "valor"}
	return copyInsertEmLote(ctx, tx, valores, lote, "tse_receita_doador_originario_orgao_partidario", columns, "",
		func(v *types.ReceitaDoadorOriginarioOrgaoPartidario) []any {
			return []any{v.ID, v.PrestacaoContasID, v.ReceitaOrgaoPartidarioID, v.SQReceita, StrNil(v.DocumentoDoador), StrNil(v.NomeDoador), StrNil(v.NomeDoadorRFB), StrNil(v.TipoDoador), StrNil(v.CNAECodigo), StrNil(v.CNAEDescricao), v.DataReceita, StrNil(v.Descricao), v.Valor}
		},
		"", resultado)
}

// -----------------------------------------------------------------------------
// Metadados: registro de arquivos importados
// -----------------------------------------------------------------------------

// ListarArquivosImportados devolve os pares (nome do arquivo, UF) ja
// persistidos, com chave "nome|uf". Um mesmo zip nacional pode ser importado
// para varias UFs, entao a deduplicacao de download e por (arquivo, UF).
func (r *Repositorio) ListarArquivosImportados(ctx context.Context) (map[string]bool, error) {
	importados := make(map[string]bool)
	rows, err := r.pool.Query(ctx, `SELECT nome, COALESCE(uf, '') FROM tse_arquivo_importado`)
	if err != nil {
		return nil, fmt.Errorf("listar arquivos importados: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var nome, uf string
		if err := rows.Scan(&nome, &uf); err != nil {
			return nil, err
		}
		importados[chaveArquivoUF(nome, uf)] = true
	}
	return importados, rows.Err()
}

// chaveArquivoUF identifica um arquivo importado para uma UF.
func chaveArquivoUF(nome, uf string) string {
	return nome + "|" + uf
}

// RegistrarArquivoImportado insere a linha de dedup de um arquivo.
func (r *Repositorio) RegistrarArquivoImportado(ctx context.Context, tx pgx.Tx, caminhoRelativo, nome, tipo, uf string, ano int, totalRegistros int, hash string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO tse_arquivo_importado (caminho_relativo, nome, tipo, uf, ano, total_registros, hash_sha256)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (nome, uf) DO UPDATE SET
			caminho_relativo = EXCLUDED.caminho_relativo,
			tipo = EXCLUDED.tipo,
			ano = EXCLUDED.ano,
			total_registros = EXCLUDED.total_registros,
			hash_sha256 = EXCLUDED.hash_sha256`,
		caminhoRelativo, nome, tipo, uf, ano, totalRegistros, hash)
	if err != nil {
		return fmt.Errorf("registrar arquivo importado: %w", err)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Cache de chaves naturais para resolver dependencias ja importadas
// -----------------------------------------------------------------------------

// CarregarIDsEleicoes devolve codigo_tse -> id.
func (r *Repositorio) CarregarIDsEleicoes(ctx context.Context) (map[int]uuid.UUID, error) {
	out := map[int]uuid.UUID{}
	rows, err := r.pool.Query(ctx, `SELECT codigo_tse, id FROM tse_eleicao WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("carregar eleicoes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var codigo int
		var id uuid.UUID
		if err := rows.Scan(&codigo, &id); err != nil {
			return nil, err
		}
		out[codigo] = id
	}
	return out, rows.Err()
}

// CarregarIDsUnidades devolve "UF|codigo_tse" -> id.
func (r *Repositorio) CarregarIDsUnidades(ctx context.Context) (map[string]uuid.UUID, error) {
	out := map[string]uuid.UUID{}
	rows, err := r.pool.Query(ctx, `SELECT sg_uf, codigo_tse, id FROM tse_unidade_eleitoral WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("carregar unidades eleitorais: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var uf, codigo string
		var id uuid.UUID
		if err := rows.Scan(&uf, &codigo, &id); err != nil {
			return nil, err
		}
		out[uf+"|"+codigo] = id
	}
	return out, rows.Err()
}

// CarregarIDsPartidos devolve numero -> id.
func (r *Repositorio) CarregarIDsPartidos(ctx context.Context) (map[int16]uuid.UUID, error) {
	out := map[int16]uuid.UUID{}
	rows, err := r.pool.Query(ctx, `SELECT numero, id FROM tse_partido WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("carregar partidos: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var numero int16
		var id uuid.UUID
		if err := rows.Scan(&numero, &id); err != nil {
			return nil, err
		}
		out[numero] = id
	}
	return out, rows.Err()
}

// CarregarIDsCandidatos devolve sq_candidato -> id.
func (r *Repositorio) CarregarIDsCandidatos(ctx context.Context) (map[int64]uuid.UUID, error) {
	out := map[int64]uuid.UUID{}
	rows, err := r.pool.Query(ctx, `SELECT sq_candidato, id FROM tse_candidato WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("carregar candidatos: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sq int64
		var id uuid.UUID
		if err := rows.Scan(&sq, &id); err != nil {
			return nil, err
		}
		out[sq] = id
	}
	return out, rows.Err()
}

// CarregarIDsPrestacoes devolve "tipo_prestador|codigo_tse|sq_prestador_contas" -> id.
func (r *Repositorio) CarregarIDsPrestacoes(ctx context.Context) (map[string]uuid.UUID, error) {
	out := map[string]uuid.UUID{}
	rows, err := r.pool.Query(ctx, `
		SELECT p.tipo_prestador, e.codigo_tse, p.sq_prestador_contas, p.id
		FROM tse_prestacao_contas p
		JOIN tse_eleicao e ON e.id = p.eleicao_id
		WHERE p.deleted_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("carregar prestacoes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tipo string
		var codigo int
		var sq int64
		var id uuid.UUID
		if err := rows.Scan(&tipo, &codigo, &sq, &id); err != nil {
			return nil, err
		}
		out[chavePrestacaoCache(tipo, codigo, sq)] = id
	}
	return out, rows.Err()
}

// CarregarIDsReceitasCandidato devolve sq_receita -> id.
func (r *Repositorio) CarregarIDsReceitasCandidato(ctx context.Context) (map[int64]uuid.UUID, error) {
	out := map[int64]uuid.UUID{}
	rows, err := r.pool.Query(ctx, `SELECT sq_receita, id FROM tse_receita_candidato WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("carregar receitas de candidato: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sq int64
		var id uuid.UUID
		if err := rows.Scan(&sq, &id); err != nil {
			return nil, err
		}
		out[sq] = id
	}
	return out, rows.Err()
}

// CarregarIDsReceitasOrgao devolve sq_receita -> id.
func (r *Repositorio) CarregarIDsReceitasOrgao(ctx context.Context) (map[int64]uuid.UUID, error) {
	out := map[int64]uuid.UUID{}
	rows, err := r.pool.Query(ctx, `SELECT sq_receita, id FROM tse_receita_orgao_partidario WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("carregar receitas de orgao: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sq int64
		var id uuid.UUID
		if err := rows.Scan(&sq, &id); err != nil {
			return nil, err
		}
		out[sq] = id
	}
	return out, rows.Err()
}

// chavePrestacaoCache normaliza a chave da prestacao usada pelo parse.
func chavePrestacaoCache(tipo string, codigo int, sq int64) string {
	return tipo + "|" + strconv.Itoa(codigo) + "|" + strconv.FormatInt(sq, 10)
}

func strNilPtr(s *string) any {
	if s == nil {
		return nil
	}
	if *s == "" {
		return nil
	}
	return *s
}

// ErrNoRows e reexportado para conveniencia.
var ErrNoRows = errors.New("sem linhas")
