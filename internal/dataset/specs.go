package dataset

// col constroi um Column ({header do TSE} -> {field normalizado}).
func col(field, header string) Column { return Column{Header: header, Field: field} }

// Grupos (pasta/zip nacional) no CDN do TSE.
const (
	grupoConsultaCand   = "consulta_cand"
	grupoBemCandidato   = "bem_candidato"
	grupoPrestacaoCand  = "prestacao_de_contas_eleitorais_candidatos"
	grupoPrestacaoOrgao = "prestacao_de_contas_eleitorais_orgaos_partidarios"
)

// anosConsulta sao os anos com consulta_cand/bem_candidato (eleicoes gerais e
// municipais).
func anosConsulta() []int { return []int{2006, 2008, 2010, 2012, 2014, 2016, 2018, 2020, 2022, 2024} }

// anosPrestacao sao os anos com prestacao de contas (candidatos e orgaos).
func anosPrestacao() []int { return []int{2018, 2020, 2022, 2024} }

func defaultOrder() []string {
	return []string{
		"consulta_cand",
		"bem_candidato",
		"despesas_contratadas_candidatos",
		"receitas_candidatos",
		"receitas_candidatos_doador_originario",
		"despesas_contratadas_orgaos_partidarios",
		"receitas_orgaos_partidarios",
		"receitas_orgaos_partidarios_doador_originario",
		"despesas_pagas_candidatos",
		"despesas_pagas_orgaos_partidarios",
	}
}

func defaultSpecs() map[string]Spec {
	m := map[string]Spec{}

	// ---------------------------------------------------------------------
	// consulta_cand -> eleicao, unidade_eleitoral, partido, candidato
	// ---------------------------------------------------------------------
	m["consulta_cand"] = Spec{
		Nome:       "consulta_cand",
		Grupo:      grupoConsultaCand,
		Prefijo:    "consulta_cand",
		Anos:       anosConsulta(),
		Prioridade: 1,
		Tables: []Table{
			{
				Target: "eleicao",
				Columns: []Column{
					col("codigo_tse", "CD_ELEICAO"),
					col("ano", "ANO_ELEICAO"),
					col("codigo_tipo_eleicao", "CD_TIPO_ELEICAO"),
					col("nome_tipo_eleicao", "NM_TIPO_ELEICAO"),
					col("descricao", "DS_ELEICAO"),
					col("data_eleicao", "DT_ELEICAO"),
				},
			},
			{
				Target: "unidade_eleitoral",
				Columns: []Column{
					col("sg_uf", "SG_UF"),
					col("codigo_tse", "SG_UE"),
					col("nome", "NM_UE"),
				},
			},
			{
				Target: "partido",
				Columns: []Column{
					col("numero", "NR_PARTIDO"),
					col("sigla", "SG_PARTIDO"),
					col("nome", "NM_PARTIDO"),
					col("federacao_codigo_tse", "NR_FEDERACAO"),
					col("federacao_sigla", "SG_FEDERACAO"),
					col("federacao_nome", "NM_FEDERACAO"),
					col("coligacao_codigo_tse", "SQ_COLIGACAO"),
					col("coligacao_nome", "NM_COLIGACAO"),
					col("coligacao_composicao", "DS_COMPOSICAO_COLIGACAO"),
				},
			},
			{
				Target: "candidato",
				Columns: []Column{
					col("sq_candidato", "SQ_CANDIDATO"),
					col("eleicao_codigo_tse", "CD_ELEICAO"),
					col("sg_uf", "SG_UF"),
					col("partido_numero", "NR_PARTIDO"),
					col("cargo_codigo", "CD_CARGO"),
					col("cargo_nome", "DS_CARGO"),
					col("genero_descricao", "DS_GENERO"),
					col("cor_raca_descricao", "DS_COR_RACA"),
					col("estado_civil_nome", "DS_ESTADO_CIVIL"),
					col("grau_instrucao_nome", "DS_GRAU_INSTRUCAO"),
					col("ocupacao_codigo", "CD_OCUPACAO"),
					col("ocupacao_nome", "DS_OCUPACAO"),
					col("numero_candidato", "NR_CANDIDATO"),
					col("cpf", "NR_CPF_CANDIDATO"),
					col("nome_completo", "NM_CANDIDATO"),
					col("nome_urna", "NM_URNA_CANDIDATO"),
					col("nome_social", "NM_SOCIAL_CANDIDATO"),
					col("data_nascimento", "DT_NASCIMENTO"),
					col("situacao_totalizacao_descricao", "DS_SIT_TOT_TURNO"),
				},
			},
		},
	}

	// ---------------------------------------------------------------------
	// bem_candidato -> bem_candidato
	// ---------------------------------------------------------------------
	m["bem_candidato"] = Spec{
		Nome:         "bem_candidato",
		Grupo:        grupoBemCandidato,
		Prefijo:      "bem_candidato",
		Anos:         anosConsulta(),
		Prioridade:   2,
		Dependencias: []string{"consulta_cand"},
		Tables: []Table{{
			Target: "bem_candidato",
			Columns: []Column{
				col("sq_candidato", "SQ_CANDIDATO"),
				col("tipo_bem_codigo", "CD_TIPO_BEM_CANDIDATO"),
				col("tipo_bem_nome", "DS_TIPO_BEM_CANDIDATO"),
				col("numero_ordem", "NR_ORDEM_BEM_CANDIDATO"),
				col("descricao", "DS_BEM_CANDIDATO"),
				col("valor", "VR_BEM_CANDIDATO"),
				col("data_ultima_atualizacao", "DT_ULT_ATUAL_BEM_CANDIDATO"),
				col("hora_ultima_atualizacao", "HH_ULT_ATUAL_BEM_CANDIDATO"),
			},
		}},
	}

	// ---------------------------------------------------------------------
	// despesas candidato -> prestacao_contas, fornecedor, despesa_candidato
	// ---------------------------------------------------------------------
	m["despesas_contratadas_candidatos"] = despesasCandidatoSpec("despesas_contratadas_candidatos", "despesas_contratadas_candidatos", "CONTRATADA", 3)
	m["despesas_pagas_candidatos"] = despesasCandidatoSpec("despesas_pagas_candidatos", "despesas_pagas_candidatos", "PAGA", 9)

	// ---------------------------------------------------------------------
	// receitas candidato -> prestacao_contas, doador, receita_candidato
	// ---------------------------------------------------------------------
	m["receitas_candidatos"] = receitasCandidatoSpec("receitas_candidatos", "receitas_candidatos", 4)

	m["receitas_candidatos_doador_originario"] = Spec{
		Nome:         "receitas_candidatos_doador_originario",
		Grupo:        grupoPrestacaoCand,
		Prefijo:      "receitas_candidatos_doador_originario",
		Anos:         anosPrestacao(),
		Prioridade:   5,
		Dependencias: []string{"receitas_candidatos"},
		Tables: []Table{{
			Target: "receita_doador_originario_candidato",
			Columns: []Column{
				col("sq_receita", "SQ_RECEITA"),
				col("receita_sq_receita", "SQ_RECEITA"),
				col("sq_prestador_contas", "SQ_PRESTADOR_CONTAS"),
				col("eleicao_codigo_tse", "CD_ELEICAO"),
				col("documento_doador", "NR_CPF_CNPJ_DOADOR_ORIGINARIO"),
				col("nome_doador", "NM_DOADOR_ORIGINARIO"),
				col("nome_doador_rfb", "NM_DOADOR_ORIGINARIO_RFB"),
				col("tipo_doador", "TP_DOADOR_ORIGINARIO"),
				col("cnae_codigo", "CD_CNAE_DOADOR_ORIGINARIO"),
				col("cnae_descricao", "DS_CNAE_DOADOR_ORIGINARIO"),
				col("data_receita", "DT_RECEITA"),
				col("descricao", "DS_RECEITA"),
				col("valor", "VR_RECEITA"),
			},
		}},
	}

	// ---------------------------------------------------------------------
	// despesas orgaos partidarios -> prestacao_contas, fornecedor,
	// despesa_orgao_partidario
	// ---------------------------------------------------------------------
	m["despesas_contratadas_orgaos_partidarios"] = despesasOrgaoSpec("despesas_contratadas_orgaos_partidarios", "despesas_contratadas_orgaos_partidarios", "CONTRATADA", 6)
	m["despesas_pagas_orgaos_partidarios"] = despesasOrgaoSpec("despesas_pagas_orgaos_partidarios", "despesas_pagas_orgaos_partidarios", "PAGA", 10)

	// ---------------------------------------------------------------------
	// receitas orgaos partidarios -> prestacao_contas, doador,
	// receita_orgao_partidario
	// ---------------------------------------------------------------------
	m["receitas_orgaos_partidarios"] = receitasOrgaoSpec("receitas_orgaos_partidarios", "receitas_orgaos_partidarios", 7)
	m["receitas_orgaos_partidarios_doador_originario"] = Spec{
		Nome:         "receitas_orgaos_partidarios_doador_originario",
		Grupo:        grupoPrestacaoOrgao,
		Prefijo:      "receitas_orgaos_partidarios_doador_originario",
		Anos:         anosPrestacao(),
		Prioridade:   8,
		Dependencias: []string{"receitas_orgaos_partidarios"},
		Tables: []Table{{
			Target: "receita_doador_originario_orgao_partidario",
			Columns: []Column{
				col("sq_receita", "SQ_RECEITA"),
				col("receita_sq_receita", "SQ_RECEITA"),
				col("sq_prestador_contas", "SQ_PRESTADOR_CONTAS"),
				col("eleicao_codigo_tse", "CD_ELEICAO"),
				col("documento_doador", "NR_CPF_CNPJ_DOADOR_ORIGINARIO"),
				col("nome_doador", "NM_DOADOR_ORIGINARIO"),
				col("nome_doador_rfb", "NM_DOADOR_ORIGINARIO_RFB"),
				col("tipo_doador", "TP_DOADOR_ORIGINARIO"),
				col("cnae_codigo", "CD_CNAE_DOADOR_ORIGINARIO"),
				col("cnae_descricao", "DS_CNAE_DOADOR_ORIGINARIO"),
				col("data_receita", "DT_RECEITA"),
				col("descricao", "DS_RECEITA"),
				col("valor", "VR_RECEITA"),
			},
		}},
	}

	return m
}

// prestacaoContasCandidatoCols sao as colunas comuns da prestacao para candidatos.
func prestacaoCandidatoColumns() []Column {
	return []Column{
		col("tipo_prestador", ""),
		col("sq_prestador_contas", "SQ_PRESTADOR_CONTAS"),
		col("eleicao_codigo_tse", "CD_ELEICAO"),
		col("sq_candidato", "SQ_CANDIDATO"),
		col("sg_uf", "SG_UF"),
		col("unidade_codigo_tse", "SG_UE"),
		col("nome", "NM_UE"),
		col("tipo_prestacao", "TP_PRESTACAO_CONTAS"),
		col("data_prestacao", "DT_PRESTACAO_CONTAS"),
		col("turno", "ST_TURNO"),
		col("cnpj_prestador_conta", "NR_CNPJ_PRESTADOR_CONTA"),
	}
}

func despesasCandidatoSpec(nome, folder, tipo string, prioridade int) Spec {
	return Spec{
		Nome:         nome,
		Grupo:        grupoPrestacaoCand,
		Prefijo:      folder,
		Anos:         anosPrestacao(),
		Prioridade:   prioridade,
		Dependencias: []string{"consulta_cand"},
		Tables: []Table{
			{
				Target:    "prestacao_contas",
				Prestador: "CANDIDATO",
				Columns:   prestacaoCandidatoColumns(),
			},
			{
				Target: "fornecedor",
				Columns: []Column{
					col("cpf_cnpj", "NR_CPF_CNPJ_FORNECEDOR"),
					col("nome", "NM_FORNECEDOR"),
					col("nome_rfb", "NM_FORNECEDOR_RFB"),
					col("tipo_fornecedor_codigo", "CD_TIPO_FORNECEDOR"),
					col("tipo_fornecedor_descricao", "DS_TIPO_FORNECEDOR"),
					col("cnae_codigo", "CD_CNAE_FORNECEDOR"),
					col("cnae_descricao", "DS_CNAE_FORNECEDOR"),
					col("esfera_partidaria_codigo", "CD_ESFERA_PART_FORNECEDOR"),
					col("esfera_partidaria_descricao", "DS_ESFERA_PART_FORNECEDOR"),
					col("sg_uf", "SG_UF_FORNECEDOR"),
					col("municipio_nome", "NM_MUNICIPIO_FORNECEDOR"),
				},
			},
			{
				Target:       "despesa_candidato",
				TipoRegistro: tipo,
				Prestador:    "CANDIDATO",
				Columns: []Column{
					col("tipo_prestador", ""),
					col("tipo_prestador", ""),
					col("tipo_registro", ""),
					col("sq_despesa", "SQ_DESPESA"),
					col("sq_prestador_contas", "SQ_PRESTADOR_CONTAS"),
					col("eleicao_codigo_tse", "CD_ELEICAO"),
					col("partido_numero", "NR_PARTIDO"),
					col("fornecedor_cpf_cnpj", "NR_CPF_CNPJ_FORNECEDOR"),
					col("tipo_documento", "DS_TIPO_DOCUMENTO"),
					col("numero_documento", "NR_DOCUMENTO"),
					col("origem_despesa_codigo", "CD_ORIGEM_DESPESA"),
					col("origem_despesa_descricao", "DS_ORIGEM_DESPESA"),
					col("fonte_despesa_codigo", "CD_FONTE_DESPESA"),
					col("fonte_despesa_descricao", "DS_FONTE_DESPESA"),
					col("natureza_despesa_codigo", "CD_NATUREZA_DESPESA"),
					col("natureza_despesa_descricao", "DS_NATUREZA_DESPESA"),
					col("especie_recurso_codigo", "CD_ESPECIE_RECURSO"),
					col("especie_recurso_descricao", "DS_ESPECIE_RECURSO"),
					col("sq_parcelamento_despesa", "SQ_PARCELAMENTO_DESPESA"),
					col("data_despesa", "DT_PAGTO_DESPESA"),
					col("descricao", "DS_DESPESA"),
					col("valor", "VR_PAGTO_DESPESA"),
				},
			},
		},
	}
}

func receitasCandidatoSpec(nome, folder string, prioridade int) Spec {
	return Spec{
		Nome:         nome,
		Grupo:        grupoPrestacaoCand,
		Prefijo:      folder,
		Anos:         anosPrestacao(),
		Prioridade:   prioridade,
		Dependencias: []string{"consulta_cand"},
		Tables: []Table{
			{
				Target:    "prestacao_contas",
				Prestador: "CANDIDATO",
				Columns:   prestacaoCandidatoColumns(),
			},
			{
				Target: "doador",
				Columns: []Column{
					col("cpf_cnpj", "NR_CPF_CNPJ_DOADOR"),
					col("nome", "NM_DOADOR"),
					col("nome_rfb", "NM_DOADOR_RFB"),
					col("cnae_codigo", "CD_CNAE_DOADOR"),
					col("cnae_descricao", "DS_CNAE_DOADOR"),
					col("esfera_partidaria_codigo", "CD_ESFERA_PART_DOADOR"),
					col("esfera_partidaria_descricao", "DS_ESFERA_PART_DOADOR"),
					col("sg_uf", "SG_UF_DOADOR"),
					col("municipio_nome", "NM_MUNICIPIO_DOADOR"),
				},
			},
			{
				Target:       "receita_candidato",
				TipoRegistro: "",
				Prestador:    "CANDIDATO",
				Columns: []Column{
					col("tipo_prestador", ""),
					col("tipo_prestador", ""),
					col("sq_receita", "SQ_RECEITA"),
					col("sq_prestador_contas", "SQ_PRESTADOR_CONTAS"),
					col("eleicao_codigo_tse", "CD_ELEICAO"),
					col("partido_numero", "NR_PARTIDO"),
					col("doador_cpf_cnpj", "NR_CPF_CNPJ_DOADOR"),
					col("fonte_receita_codigo", "CD_FONTE_RECEITA"),
					col("fonte_receita_descricao", "DS_FONTE_RECEITA"),
					col("origem_receita_codigo", "CD_ORIGEM_RECEITA"),
					col("origem_receita_descricao", "DS_ORIGEM_RECEITA"),
					col("natureza_receita_codigo", "CD_NATUREZA_RECEITA"),
					col("natureza_receita_descricao", "DS_NATUREZA_RECEITA"),
					col("especie_receita_codigo", "CD_ESPECIE_RECEITA"),
					col("especie_receita_descricao", "DS_ESPECIE_RECEITA"),
					col("numero_recibo_doacao", "NR_RECIBO_DOACAO"),
					col("numero_documento_doacao", "NR_DOCUMENTO_DOACAO"),
					col("data_receita", "DT_RECEITA"),
					col("descricao", "DS_RECEITA"),
					col("valor", "VR_RECEITA"),
					col("natureza_recurso_estimavel", "DS_NATUREZA_RECURSO_ESTIMAVEL"),
					col("genero", "DS_GENERO"),
					col("cor_raca", "DS_COR_RACA"),
				},
			},
		},
	}
}

func despesasOrgaoSpec(nome, folder, tipo string, prioridade int) Spec {
	return Spec{
		Nome:         nome,
		Grupo:        grupoPrestacaoOrgao,
		Prefijo:      folder,
		Anos:         anosPrestacao(),
		Prioridade:   prioridade,
		Dependencias: []string{"consulta_cand"},
		Tables: []Table{
			{
				Target:    "prestacao_contas",
				Prestador: "ORGAO_PARTIDARIO",
				Columns: []Column{
					col("tipo_prestador", ""),
					col("sq_prestador_contas", "SQ_PRESTADOR_CONTAS"),
					col("eleicao_codigo_tse", "CD_ELEICAO"),
					col("partido_numero", "NR_PARTIDO"),
					col("sg_uf", "SG_UF"),
					col("unidade_codigo_tse", "SG_UE"),
					col("nome", "NM_UE"),
					col("tipo_prestacao", "TP_PRESTACAO_CONTAS"),
					col("data_prestacao", "DT_PRESTACAO_CONTAS"),
					col("turno", "ST_TURNO"),
					col("cnpj_prestador_conta", "NR_CNPJ_PRESTADOR_CONTA"),
					col("esfera_partidaria_codigo", "CD_ESFERA_PARTIDARIA"),
					col("esfera_partidaria_descricao", "DS_ESFERA_PARTIDARIA"),
				},
			},
			{
				Target: "fornecedor",
				Columns: []Column{
					col("cpf_cnpj", "NR_CPF_CNPJ_FORNECEDOR"),
					col("nome", "NM_FORNECEDOR"),
					col("nome_rfb", "NM_FORNECEDOR_RFB"),
					col("tipo_fornecedor_codigo", "CD_TIPO_FORNECEDOR"),
					col("tipo_fornecedor_descricao", "DS_TIPO_FORNECEDOR"),
					col("cnae_codigo", "CD_CNAE_FORNECEDOR"),
					col("cnae_descricao", "DS_CNAE_FORNECEDOR"),
					col("esfera_partidaria_codigo", "CD_ESFERA_PART_FORNECEDOR"),
					col("esfera_partidaria_descricao", "DS_ESFERA_PART_FORNECEDOR"),
					col("sg_uf", "SG_UF_FORNECEDOR"),
					col("municipio_nome", "NM_MUNICIPIO_FORNECEDOR"),
				},
			},
			{
				Target:       "despesa_orgao_partidario",
				TipoRegistro: tipo,
				Prestador:    "ORGAO_PARTIDARIO",
				Columns: []Column{
					col("sq_despesa", "SQ_DESPESA"),
					col("tipo_prestador", ""),
					col("sq_prestador_contas", "SQ_PRESTADOR_CONTAS"),
					col("eleicao_codigo_tse", "CD_ELEICAO"),
					col("partido_numero", "NR_PARTIDO"),
					col("fornecedor_cpf_cnpj", "NR_CPF_CNPJ_FORNECEDOR"),
					col("tipo_documento", "DS_TIPO_DOCUMENTO"),
					col("numero_documento", "NR_DOCUMENTO"),
					col("origem_despesa_codigo", "CD_ORIGEM_DESPESA"),
					col("origem_despesa_descricao", "DS_ORIGEM_DESPESA"),
					col("fonte_despesa_codigo", "CD_FONTE_DESPESA"),
					col("fonte_despesa_descricao", "DS_FONTE_DESPESA"),
					col("natureza_despesa_codigo", "CD_NATUREZA_DESPESA"),
					col("natureza_despesa_descricao", "DS_NATUREZA_DESPESA"),
					col("especie_recurso_codigo", "CD_ESPECIE_RECURSO"),
					col("especie_recurso_descricao", "DS_ESPECIE_RECURSO"),
					col("sq_parcelamento_despesa", "SQ_PARCELAMENTO_DESPESA"),
					col("data_despesa", "DT_PAGTO_DESPESA"),
					col("descricao", "DS_DESPESA"),
					col("valor", "VR_PAGTO_DESPESA"),
				},
			},
		},
	}
}

func receitasOrgaoSpec(nome, folder string, prioridade int) Spec {
	return Spec{
		Nome:         nome,
		Grupo:        grupoPrestacaoOrgao,
		Prefijo:      folder,
		Anos:         anosPrestacao(),
		Prioridade:   prioridade,
		Dependencias: []string{"consulta_cand"},
		Tables: []Table{
			{
				Target:    "prestacao_contas",
				Prestador: "ORGAO_PARTIDARIO",
				Columns: []Column{
					col("tipo_prestador", ""),
					col("sq_prestador_contas", "SQ_PRESTADOR_CONTAS"),
					col("eleicao_codigo_tse", "CD_ELEICAO"),
					col("partido_numero", "NR_PARTIDO"),
					col("sg_uf", "SG_UF"),
					col("unidade_codigo_tse", "SG_UE"),
					col("nome", "NM_UE"),
					col("tipo_prestacao", "TP_PRESTACAO_CONTAS"),
					col("data_prestacao", "DT_PRESTACAO_CONTAS"),
					col("turno", "ST_TURNO"),
					col("cnpj_prestador_conta", "NR_CNPJ_PRESTADOR_CONTA"),
					col("esfera_partidaria_codigo", "CD_ESFERA_PARTIDARIA"),
					col("esfera_partidaria_descricao", "DS_ESFERA_PARTIDARIA"),
				},
			},
			{
				Target: "doador",
				Columns: []Column{
					col("cpf_cnpj", "NR_CPF_CNPJ_DOADOR"),
					col("nome", "NM_DOADOR"),
					col("nome_rfb", "NM_DOADOR_RFB"),
					col("cnae_codigo", "CD_CNAE_DOADOR"),
					col("cnae_descricao", "DS_CNAE_DOADOR"),
					col("esfera_partidaria_codigo", "CD_ESFERA_PART_DOADOR"),
					col("esfera_partidaria_descricao", "DS_ESFERA_PART_DOADOR"),
					col("sg_uf", "SG_UF_DOADOR"),
					col("municipio_nome", "NM_MUNICIPIO_DOADOR"),
				},
			},
			{
				Target:       "receita_orgao_partidario",
				TipoRegistro: "",
				Prestador:    "ORGAO_PARTIDARIO",
				Columns: []Column{
					col("sq_receita", "SQ_RECEITA"),
					col("tipo_prestador", ""),
					col("sq_prestador_contas", "SQ_PRESTADOR_CONTAS"),
					col("eleicao_codigo_tse", "CD_ELEICAO"),
					col("partido_numero", "NR_PARTIDO"),
					col("doador_cpf_cnpj", "NR_CPF_CNPJ_DOADOR"),
					col("fonte_receita_codigo", "CD_FONTE_RECEITA"),
					col("fonte_receita_descricao", "DS_FONTE_RECEITA"),
					col("origem_receita_codigo", "CD_ORIGEM_RECEITA"),
					col("origem_receita_descricao", "DS_ORIGEM_RECEITA"),
					col("natureza_receita_codigo", "CD_NATUREZA_RECEITA"),
					col("natureza_receita_descricao", "DS_NATUREZA_RECEITA"),
					col("especie_receita_codigo", "CD_ESPECIE_RECEITA"),
					col("especie_receita_descricao", "DS_ESPECIE_RECEITA"),
					col("numero_recibo_doacao", "NR_RECIBO_DOACAO"),
					col("numero_documento_doacao", "NR_DOCUMENTO_DOACAO"),
					col("data_receita", "DT_RECEITA"),
					col("descricao", "DS_RECEITA"),
					col("valor", "VR_RECEITA"),
				},
			},
		},
	}
}
