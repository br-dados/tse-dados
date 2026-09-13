package parse

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/danyele/tse-dados/types"
)

// Cache guarda IDs de entidades ja persistidas, por chave natural. Serve para
// resolver dependencias que nao foram baixadas neste escopo.
type Cache struct {
	Eleicoes      map[int]uuid.UUID
	Unidades      map[string]uuid.UUID
	Partidos      map[int16]uuid.UUID
	Candidatos    map[int64]uuid.UUID
	Prestacoes    map[string]uuid.UUID
	ReceitasCand  map[int64]uuid.UUID
	ReceitasOrgao map[int64]uuid.UUID
}

// Grafo agrega as entidades resultantes da leitura dos CSVs normalizados.
type Grafo struct {
	Eleicoes          map[int]*types.Eleicao
	Unidades          map[string]*types.UnidadeEleitoral
	Partidos          map[int16]*types.Partido
	Candidatos        map[int64]*types.Candidato
	Fornecedores      map[string]*types.Fornecedor
	Doadores          map[string]*types.Doador
	Prestacoes        map[string]*types.PrestacaoContas
	DespesasCandidato []*types.DespesaCandidato
	DespesasOrgao     []*types.DespesaOrgaoPartidario
	ReceitasCandidato []*types.ReceitaCandidato
	ReceitasOrgao     []*types.ReceitaOrgaoPartidario
	OrigemCand        []*types.ReceitaDoadorOriginarioCandidato
	OrigemOrgao       []*types.ReceitaDoadorOriginarioOrgaoPartidario
	Bens              map[string]*types.BemCandidato
	Ignorados         int
	Cache             *Cache
}

// NovoGrafo cria o grafo vazio.
func NovoGrafo() *Grafo {
	return &Grafo{
		Eleicoes:     make(map[int]*types.Eleicao),
		Unidades:     make(map[string]*types.UnidadeEleitoral),
		Partidos:     make(map[int16]*types.Partido),
		Candidatos:   make(map[int64]*types.Candidato),
		Fornecedores: make(map[string]*types.Fornecedor),
		Doadores:     make(map[string]*types.Doador),
		Prestacoes:   make(map[string]*types.PrestacaoContas),
		Bens:         make(map[string]*types.BemCandidato),
	}
}

// LerDir le todos os CSVs normalizados (<dir>/<dataset>/<target>.csv) e devolve
// as linhas por tabela-alvo.
func LerDir(dir string) (map[string][]map[string]string, error) {
	rows := make(map[string][]map[string]string)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || strings.ToLower(filepath.Ext(d.Name())) != ".csv" {
			return nil
		}
		target := strings.TrimSuffix(d.Name(), ".csv")
		linhas, err := lerCSV(path)
		if err != nil {
			return err
		}
		rows[target] = append(rows[target], linhas...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func lerCSV(path string) ([]map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.LazyQuotes = true
	header, err := r.Read()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []map[string]string
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		m := make(map[string]string, len(header))
		for i, h := range header {
			if i < len(rec) {
				m[h] = rec[i]
			}
		}
		out = append(out, m)
	}
	return out, nil
}

// Construir monta o grafo (dimensoes -> candidatos -> fornecedores/doadores ->
// prestacoes -> transacoes -> bens/origem), resolvendo chaves naturais.
func Construir(rows map[string][]map[string]string) *Grafo {
	return ConstruirComCache(rows, nil)
}

// ConstruirComCache e igual a Construir, mas usa um cache de IDs ja
// persistidos para resolver dependencias que nao vieram nos arquivos.
func ConstruirComCache(rows map[string][]map[string]string, cache *Cache) *Grafo {
	g := NovoGrafo()
	g.Cache = cache

	for _, r := range rows["eleicao"] {
		cod := inteiroOpcional(r["codigo_tse"])
		if cod == nil {
			continue
		}
		if _, ok := g.Eleicoes[*cod]; ok {
			continue
		}
		e := &types.Eleicao{ModeloBase: novoID(), CodigoTSE: *cod}
		if a := inteiro16Opcional(r["ano"]); a != nil {
			e.Ano = *a
		}
		e.CodigoTipoEleicao = inteiroOpcional(r["codigo_tipo_eleicao"])
		e.NomeTipoEleicao = str(r["nome_tipo_eleicao"])
		e.Descricao = textoFallback(r["descricao"], "Eleicao sem descricao")
		e.DataEleicao = dataOpcional(r["data_eleicao"])
		g.Eleicoes[*cod] = e
	}

	for _, r := range rows["unidade_eleitoral"] {
		ue := &types.UnidadeEleitoral{ModeloBase: novoID(), UFSigla: strings.ToUpper(str(r["sg_uf"])), CodigoTSE: str(r["codigo_tse"]), Nome: textoFallback(r["nome"], str(r["codigo_tse"]))}
		chave := ue.UFSigla + "|" + ue.CodigoTSE
		if _, ok := g.Unidades[chave]; ok {
			continue
		}
		g.Unidades[chave] = ue
	}

	for _, r := range rows["partido"] {
		num := inteiro16Opcional(r["numero"])
		if num == nil {
			continue
		}
		if _, ok := g.Partidos[*num]; ok {
			continue
		}
		nomePartido := textoFallback(r["nome"], textoFallback(r["sigla"], strconv.Itoa(int(*num))))
		p := &types.Partido{ModeloBase: novoID(), Numero: *num, Sigla: textoFallback(r["sigla"], strconv.Itoa(int(*num))), Nome: nomePartido}
		p.FederacaoCodigoTSE = inteiro64Opcional(r["federacao_codigo_tse"])
		p.FederacaoSigla = str(r["federacao_sigla"])
		p.FederacaoNome = str(r["federacao_nome"])
		p.ColigacaoCodigoTSE = inteiro64Opcional(r["coligacao_codigo_tse"])
		p.ColigacaoNome = str(r["coligacao_nome"])
		p.ColigacaoComposicao = str(r["coligacao_composicao"])
		g.Partidos[*num] = p
	}

	for _, r := range rows["candidato"] {
		sq := inteiro64Opcional(r["sq_candidato"])
		if sq == nil {
			continue
		}
		if _, ok := g.Candidatos[*sq]; ok {
			continue
		}
		eleicaoID := g.resolveEleicao(r["eleicao_codigo_tse"])
		c := &types.Candidato{ModeloBase: novoID(), SQCandidato: *sq, EleicaoID: eleicaoID, UFSigla: strings.ToUpper(str(r["sg_uf"]))}
		c.PartidoID = g.resolvePartidoPtr(r["partido_numero"])
		c.CargoCodigo = inteiroOpcional(r["cargo_codigo"])
		c.CargoNome = str(r["cargo_nome"])
		c.GeneroDescricao = str(r["genero_descricao"])
		c.CorRacaDescricao = str(r["cor_raca_descricao"])
		c.EstadoCivilNome = str(r["estado_civil_nome"])
		c.GrauInstrucaoNome = str(r["grau_instrucao_nome"])
		c.OcupacaoCodigo = inteiroOpcional(r["ocupacao_codigo"])
		c.OcupacaoNome = str(r["ocupacao_nome"])
		c.NumeroCandidato = inteiroOpcional(r["numero_candidato"])
		c.CPF = documentoOpcional(r["cpf"])
		c.NomeCompleto = str(r["nome_completo"])
		c.NomeUrna = str(r["nome_urna"])
		c.NomeSocial = str(r["nome_social"])
		c.DataNascimento = dataOpcional(r["data_nascimento"])
		c.SituacaoTotalizacaoDescricao = str(r["situacao_totalizacao_descricao"])
		g.Candidatos[*sq] = c
	}

	for _, r := range rows["fornecedor"] {
		doc := documentoOpcional(r["cpf_cnpj"])
		if doc == "" {
			continue
		}
		if _, ok := g.Fornecedores[doc]; ok {
			continue
		}
		f := &types.Fornecedor{ModeloBase: novoID(), CPFCNPJ: doc, Nome: str(r["nome"])}
		f.NomeRFB = str(r["nome_rfb"])
		f.TipoFornecedorCodigo = inteiroOpcional(r["tipo_fornecedor_codigo"])
		f.TipoFornecedorDescricao = str(r["tipo_fornecedor_descricao"])
		f.CNAECodigo = str(r["cnae_codigo"])
		f.CNAEDescricao = str(r["cnae_descricao"])
		f.EsferaPartidariaCodigo = str(r["esfera_partidaria_codigo"])
		f.EsferaPartidariaDescricao = str(r["esfera_partidaria_descricao"])
		f.UFSigla = ptrStr(str(r["sg_uf"]))
		f.MunicipioNome = str(r["municipio_nome"])
		g.Fornecedores[doc] = f
	}

	for _, r := range rows["doador"] {
		doc := documentoOpcional(r["cpf_cnpj"])
		if doc == "" {
			continue
		}
		if _, ok := g.Doadores[doc]; ok {
			continue
		}
		d := &types.Doador{ModeloBase: novoID(), CPFCNPJ: doc, Nome: str(r["nome"])}
		d.NomeRFB = str(r["nome_rfb"])
		d.CNAECodigo = str(r["cnae_codigo"])
		d.CNAEDescricao = str(r["cnae_descricao"])
		d.EsferaPartidariaCodigo = str(r["esfera_partidaria_codigo"])
		d.EsferaPartidariaDescricao = str(r["esfera_partidaria_descricao"])
		d.UFSigla = ptrStr(str(r["sg_uf"]))
		d.MunicipioNome = str(r["municipio_nome"])
		g.Doadores[doc] = d
	}

	for _, r := range rows["prestacao_contas"] {
		sq := inteiro64Opcional(r["sq_prestador_contas"])
		cod := inteiroOpcional(r["eleicao_codigo_tse"])
		if sq == nil || cod == nil {
			continue
		}
		tipoPrestador := str(r["tipo_prestador"])
		if tipoPrestador == "" {
			tipoPrestador = ifThenCol(r["sq_candidato"], r["partido_numero"])
		}
		chave := tipoPrestador + "|" + strconv.Itoa(*cod) + "|" + strconv.FormatInt(*sq, 10)
		if _, ok := g.Prestacoes[chave]; ok {
			continue
		}
		p := &types.PrestacaoContas{ModeloBase: novoID(), SQPrestadorContas: *sq, EleicaoID: g.resolveEleicao(r["eleicao_codigo_tse"]), TipoPrestador: tipoPrestador}
		p.CandidatoID = g.resolveCandidatoPtr(r["sq_candidato"])
		p.PartidoID = g.resolvePartidoPtr(r["partido_numero"])
		p.UFSigla = ptrStr(str(r["sg_uf"]))
		if uf := str(r["sg_uf"]); uf != "" && str(r["unidade_codigo_tse"]) != "" {
			p.UnidadeEleitoralID = g.resolveUnidade(uf, str(r["unidade_codigo_tse"]))
		}
		p.TipoPrestacao = str(r["tipo_prestacao"])
		p.DataPrestacao = dataOpcional(r["data_prestacao"])
		p.Turno = inteiro16Opcional(r["turno"])
		p.CNPJPrestadorConta = documentoOpcional(r["cnpj_prestador_conta"])
		p.EsferaPartidariaCodigo = str(r["esfera_partidaria_codigo"])
		p.EsferaPartidariaDescricao = str(r["esfera_partidaria_descricao"])
		g.Prestacoes[chave] = p
	}

	for _, r := range rows["despesa_candidato"] {
		prestID, ok := g.resolvePrestacao(r["tipo_prestador"], r["eleicao_codigo_tse"], r["sq_prestador_contas"])
		if !ok {
			g.Ignorados++
			continue
		}
		candID := g.resolveCandidato(r["sq_candidato"])
		if candID == uuid.Nil {
			g.Ignorados++
			continue
		}
		d := &types.DespesaCandidato{ModeloBase: novoID(), PrestacaoContasID: prestID, CandidatoID: candID, FornecedorID: g.resolveFornecedorPtr(r["fornecedor_cpf_cnpj"])}
		d.SQDespesa = inteiro64Zero(r["sq_despesa"])
		d.TipoRegistro = tipoRegistro(r["tipo_registro"])
		d.TipoDocumento = str(r["tipo_documento"])
		d.NumeroDocumento = str(r["numero_documento"])
		d.OrigemDespesaCodigo = inteiroOpcional(r["origem_despesa_codigo"])
		d.OrigemDespesaDescricao = str(r["origem_despesa_descricao"])
		d.FonteDespesaCodigo = inteiroOpcional(r["fonte_despesa_codigo"])
		d.FonteDespesaDescricao = str(r["fonte_despesa_descricao"])
		d.NaturezaDespesaCodigo = inteiroOpcional(r["natureza_despesa_codigo"])
		d.NaturezaDespesaDescricao = str(r["natureza_despesa_descricao"])
		d.EspecieRecursoCodigo = inteiroOpcional(r["especie_recurso_codigo"])
		d.EspecieRecursoDescricao = str(r["especie_recurso_descricao"])
		d.SQPlanoParcelamento = inteiro64Opcional(r["sq_parcelamento_despesa"])
		d.DataDespesa = dataOpcional(r["data_despesa"])
		d.Descricao = textoFallback(r["descricao"], "DESPESA SEM DESCRICAO")
		d.Valor = decimalZero(r["valor"])
		g.DespesasCandidato = append(g.DespesasCandidato, d)
	}

	for _, r := range rows["despesa_orgao_partidario"] {
		prestID, ok := g.resolvePrestacao(r["tipo_prestador"], r["eleicao_codigo_tse"], r["sq_prestador_contas"])
		if !ok {
			g.Ignorados++
			continue
		}
		partID := g.resolvePartido(r["partido_numero"])
		if partID == uuid.Nil {
			g.Ignorados++
			continue
		}
		d := &types.DespesaOrgaoPartidario{ModeloBase: novoID(), PrestacaoContasID: prestID, PartidoID: partID, FornecedorID: g.resolveFornecedorPtr(r["fornecedor_cpf_cnpj"])}
		d.SQDespesa = inteiro64Zero(r["sq_despesa"])
		d.TipoRegistro = tipoRegistro(r["tipo_registro"])
		d.TipoDocumento = str(r["tipo_documento"])
		d.NumeroDocumento = str(r["numero_documento"])
		d.OrigemDespesaCodigo = inteiroOpcional(r["origem_despesa_codigo"])
		d.OrigemDespesaDescricao = str(r["origem_despesa_descricao"])
		d.FonteDespesaCodigo = inteiroOpcional(r["fonte_despesa_codigo"])
		d.FonteDespesaDescricao = str(r["fonte_despesa_descricao"])
		d.NaturezaDespesaCodigo = inteiroOpcional(r["natureza_despesa_codigo"])
		d.NaturezaDespesaDescricao = str(r["natureza_despesa_descricao"])
		d.EspecieRecursoCodigo = inteiroOpcional(r["especie_recurso_codigo"])
		d.EspecieRecursoDescricao = str(r["especie_recurso_descricao"])
		d.SQPlanoParcelamento = inteiro64Opcional(r["sq_parcelamento_despesa"])
		d.DataDespesa = dataOpcional(r["data_despesa"])
		d.Descricao = textoFallback(r["descricao"], "DESPESA SEM DESCRICAO")
		d.Valor = decimalZero(r["valor"])
		g.DespesasOrgao = append(g.DespesasOrgao, d)
	}

	for _, r := range rows["receita_candidato"] {
		prestID, ok := g.resolvePrestacao(r["tipo_prestador"], r["eleicao_codigo_tse"], r["sq_prestador_contas"])
		if !ok {
			g.Ignorados++
			continue
		}
		candID := g.resolveCandidato(r["sq_candidato"])
		if candID == uuid.Nil {
			g.Ignorados++
			continue
		}
		rc := &types.ReceitaCandidato{ModeloBase: novoID(), PrestacaoContasID: prestID, CandidatoID: candID, DoadorID: g.resolveDoadorPtr(r["doador_cpf_cnpj"])}
		rc.SQReceita = inteiro64Zero(r["sq_receita"])
		rc.FonteReceitaCodigo = inteiroOpcional(r["fonte_receita_codigo"])
		rc.FonteReceitaDescricao = str(r["fonte_receita_descricao"])
		rc.OrigemReceitaCodigo = inteiroOpcional(r["origem_receita_codigo"])
		rc.OrigemReceitaDescricao = str(r["origem_receita_descricao"])
		rc.NaturezaReceitaCodigo = inteiroOpcional(r["natureza_receita_codigo"])
		rc.NaturezaReceitaDescricao = str(r["natureza_receita_descricao"])
		rc.EspecieReceitaCodigo = inteiroOpcional(r["especie_receita_codigo"])
		rc.EspecieReceitaDescricao = str(r["especie_receita_descricao"])
		rc.NumeroReciboDoacao = str(r["numero_recibo_doacao"])
		rc.NumeroDocumentoDoacao = str(r["numero_documento_doacao"])
		rc.DataReceita = dataOpcional(r["data_receita"])
		rc.Descricao = textoFallback(r["descricao"], "RECEITA SEM DESCRICAO")
		rc.Valor = decimalZero(r["valor"])
		rc.NaturezaRecursoEstimavel = str(r["natureza_recurso_estimavel"])
		rc.Genero = str(r["genero"])
		rc.CorRaca = str(r["cor_raca"])
		g.ReceitasCandidato = append(g.ReceitasCandidato, rc)
	}

	for _, r := range rows["receita_orgao_partidario"] {
		prestID, ok := g.resolvePrestacao(r["tipo_prestador"], r["eleicao_codigo_tse"], r["sq_prestador_contas"])
		if !ok {
			g.Ignorados++
			continue
		}
		partID := g.resolvePartido(r["partido_numero"])
		if partID == uuid.Nil {
			g.Ignorados++
			continue
		}
		rc := &types.ReceitaOrgaoPartidario{ModeloBase: novoID(), PrestacaoContasID: prestID, PartidoID: partID, DoadorID: g.resolveDoadorPtr(r["doador_cpf_cnpj"])}
		rc.SQReceita = inteiro64Zero(r["sq_receita"])
		rc.FonteReceitaCodigo = inteiroOpcional(r["fonte_receita_codigo"])
		rc.FonteReceitaDescricao = str(r["fonte_receita_descricao"])
		rc.OrigemReceitaCodigo = inteiroOpcional(r["origem_receita_codigo"])
		rc.OrigemReceitaDescricao = str(r["origem_receita_descricao"])
		rc.NaturezaReceitaCodigo = inteiroOpcional(r["natureza_receita_codigo"])
		rc.NaturezaReceitaDescricao = str(r["natureza_receita_descricao"])
		rc.EspecieReceitaCodigo = inteiroOpcional(r["especie_receita_codigo"])
		rc.EspecieReceitaDescricao = str(r["especie_receita_descricao"])
		rc.NumeroReciboDoacao = str(r["numero_recibo_doacao"])
		rc.NumeroDocumentoDoacao = str(r["numero_documento_doacao"])
		rc.DataReceita = dataOpcional(r["data_receita"])
		rc.Descricao = textoFallback(r["descricao"], "RECEITA SEM DESCRICAO")
		rc.Valor = decimalZero(r["valor"])
		g.ReceitasOrgao = append(g.ReceitasOrgao, rc)
	}

	for _, r := range rows["bem_candidato"] {
		candID := g.resolveCandidato(r["sq_candidato"])
		if candID == uuid.Nil {
			g.Ignorados++
			continue
		}
		ordem := inteiroOpcional(r["numero_ordem"])
		if ordem == nil {
			continue
		}
		chave := candID.String() + "|" + strconv.Itoa(*ordem)
		if _, ok := g.Bens[chave]; ok {
			continue
		}
		b := &types.BemCandidato{ModeloBase: novoID(), CandidatoID: candID, NumeroOrdem: *ordem}
		b.TipoBemCodigo = inteiroOpcional(r["tipo_bem_codigo"])
		b.TipoBemNome = str(r["tipo_bem_nome"])
		b.Descricao = textoFallback(r["descricao"], "BEM SEM DESCRICAO")
		b.Valor = decimalZero(r["valor"])
		b.DataUltimaAtualizacao = dataOpcional(r["data_ultima_atualizacao"])
		b.HoraUltimaAtualizacao = str(r["hora_ultima_atualizacao"])
		g.Bens[chave] = b
	}

	for _, r := range rows["receita_doador_originario_candidato"] {
		prestID, ok := g.resolvePrestacao("CANDIDATO", r["eleicao_codigo_tse"], r["sq_prestador_contas"])
		if !ok {
			g.Ignorados++
			continue
		}
		var recID *uuid.UUID
		if sq := inteiro64Opcional(r["receita_sq_receita"]); sq != nil {
			if id, ok := g.resolveReceitaCand(*sq); ok {
				recID = &id
			}
		}
		o := &types.ReceitaDoadorOriginarioCandidato{ModeloBase: novoID(), PrestacaoContasID: prestID, ReceitaCandidatoID: recID}
		o.SQReceita = inteiro64Zero(r["sq_receita"])
		o.DocumentoDoador = documentoOpcional(r["documento_doador"])
		o.NomeDoador = textoFallback(r["nome_doador"], "DOADOR ORIGINARIO SEM NOME")
		o.NomeDoadorRFB = str(r["nome_doador_rfb"])
		o.TipoDoador = str(r["tipo_doador"])
		o.CNAECodigo = str(r["cnae_codigo"])
		o.CNAEDescricao = str(r["cnae_descricao"])
		o.DataReceita = dataOpcional(r["data_receita"])
		o.Descricao = textoFallback(r["descricao"], "RECEITA SEM DESCRICAO")
		o.Valor = decimalZero(r["valor"])
		g.OrigemCand = append(g.OrigemCand, o)
	}

	for _, r := range rows["receita_doador_originario_orgao_partidario"] {
		prestID, ok := g.resolvePrestacao("ORGAO_PARTIDARIO", r["eleicao_codigo_tse"], r["sq_prestador_contas"])
		if !ok {
			g.Ignorados++
			continue
		}
		var recID *uuid.UUID
		if sq := inteiro64Opcional(r["receita_sq_receita"]); sq != nil {
			if id, ok := g.resolveReceitaOrgao(*sq); ok {
				recID = &id
			}
		}
		o := &types.ReceitaDoadorOriginarioOrgaoPartidario{ModeloBase: novoID(), PrestacaoContasID: prestID, ReceitaOrgaoPartidarioID: recID}
		o.SQReceita = inteiro64Zero(r["sq_receita"])
		o.DocumentoDoador = documentoOpcional(r["documento_doador"])
		o.NomeDoador = textoFallback(r["nome_doador"], "DOADOR ORIGINARIO SEM NOME")
		o.NomeDoadorRFB = str(r["nome_doador_rfb"])
		o.TipoDoador = str(r["tipo_doador"])
		o.CNAECodigo = str(r["cnae_codigo"])
		o.CNAEDescricao = str(r["cnae_descricao"])
		o.DataReceita = dataOpcional(r["data_receita"])
		o.Descricao = textoFallback(r["descricao"], "RECEITA SEM DESCRICAO")
		o.Valor = decimalZero(r["valor"])
		g.OrigemOrgao = append(g.OrigemOrgao, o)
	}

	return g
}

// ---------------------------------------------------------------------------
// Resolucao de chaves naturais -> UUID
// ---------------------------------------------------------------------------

func (g *Grafo) resolveEleicao(codigo string) uuid.UUID {
	if c := inteiroOpcional(codigo); c != nil {
		if e, ok := g.Eleicoes[*c]; ok {
			return e.ID
		}
		if g.Cache != nil {
			if id, ok := g.Cache.Eleicoes[*c]; ok {
				return id
			}
		}
	}
	return uuid.Nil
}

func (g *Grafo) resolvePartido(numero string) uuid.UUID {
	if n := inteiro16Opcional(numero); n != nil {
		if p, ok := g.Partidos[*n]; ok {
			return p.ID
		}
		if g.Cache != nil {
			if id, ok := g.Cache.Partidos[*n]; ok {
				return id
			}
		}
	}
	return uuid.Nil
}

func (g *Grafo) resolvePartidoPtr(numero string) *uuid.UUID {
	if id := g.resolvePartido(numero); id != uuid.Nil {
		return &id
	}
	return nil
}

func (g *Grafo) resolveCandidato(sq string) uuid.UUID {
	if s := inteiro64Opcional(sq); s != nil {
		if c, ok := g.Candidatos[*s]; ok {
			return c.ID
		}
		if g.Cache != nil {
			if id, ok := g.Cache.Candidatos[*s]; ok {
				return id
			}
		}
	}
	return uuid.Nil
}

func (g *Grafo) resolveCandidatoPtr(sq string) *uuid.UUID {
	if id := g.resolveCandidato(sq); id != uuid.Nil {
		return &id
	}
	return nil
}

func (g *Grafo) resolveFornecedorPtr(doc string) *uuid.UUID {
	d := documentoOpcional(doc)
	if d == "" {
		return nil
	}
	if f, ok := g.Fornecedores[d]; ok {
		u := f.ID
		return &u
	}
	return nil
}

func (g *Grafo) resolveDoadorPtr(doc string) *uuid.UUID {
	d := documentoOpcional(doc)
	if d == "" {
		return nil
	}
	if f, ok := g.Doadores[d]; ok {
		u := f.ID
		return &u
	}
	return nil
}

func (g *Grafo) resolveUnidade(uf, codigo string) *uuid.UUID {
	chave := strings.ToUpper(uf) + "|" + codigo
	if u, ok := g.Unidades[chave]; ok {
		id := u.ID
		return &id
	}
	if g.Cache != nil {
		if id, ok := g.Cache.Unidades[chave]; ok {
			return &id
		}
	}
	return nil
}

func (g *Grafo) resolvePrestacao(tipo, eleicao, sq string) (uuid.UUID, bool) {
	cod := inteiroOpcional(eleicao)
	s := inteiro64Opcional(sq)
	if cod == nil || s == nil {
		return uuid.Nil, false
	}
	chave := tipoPreco(tipo) + "|" + strconv.Itoa(*cod) + "|" + strconv.FormatInt(*s, 10)
	if p, ok := g.Prestacoes[chave]; ok {
		return p.ID, true
	}
	if g.Cache != nil {
		if id, ok := g.Cache.Prestacoes[chave]; ok {
			return id, true
		}
	}
	return uuid.Nil, false
}

func (g *Grafo) resolveReceitaCand(sq int64) (uuid.UUID, bool) {
	for _, r := range g.ReceitasCandidato {
		if r.SQReceita == sq {
			return r.ID, true
		}
	}
	if g.Cache != nil {
		if id, ok := g.Cache.ReceitasCand[sq]; ok {
			return id, true
		}
	}
	return uuid.Nil, false
}

func (g *Grafo) resolveReceitaOrgao(sq int64) (uuid.UUID, bool) {
	for _, r := range g.ReceitasOrgao {
		if r.SQReceita == sq {
			return r.ID, true
		}
	}
	if g.Cache != nil {
		if id, ok := g.Cache.ReceitasOrgao[sq]; ok {
			return id, true
		}
	}
	return uuid.Nil, false
}

// tipoPreco normaliza o tipo de prestador (CANDIDATO | ORGAO_PARTIDARIO).
func tipoPreco(tipo string) string {
	t := strings.ToUpper(strings.TrimSpace(tipo))
	if t == "ORGAO_PARTIDARIO" || t == "ORGAO" {
		return "ORGAO_PARTIDARIO"
	}
	if t == "" {
		return ""
	}
	return "CANDIDATO"
}

// tipoRegistro mantem o valor do tipo de registro de despesa.
func tipoRegistro(tipo string) string {
	t := strings.ToUpper(strings.TrimSpace(tipo))
	if t == "PAGA" {
		return "PAGA"
	}
	if t == "CONTRATADA" {
		return "CONTRATADA"
	}
	return "PAGA"
}

func ifThenCol(sq, partido string) string {
	if strings.TrimSpace(partido) != "" && strings.TrimSpace(sq) == "" {
		return "ORGAO_PARTIDARIO"
	}
	return "CANDIDATO"
}

func novoID() types.ModeloBase {
	return types.ModeloBase{ID: uuid.Must(uuid.NewV7())}
}

func inteiro64Zero(valor string) int64 {
	if v := inteiro64Opcional(valor); v != nil {
		return *v
	}
	return 0
}

func ptrStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
