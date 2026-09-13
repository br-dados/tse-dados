package types

import "time"

// Backends (modo de saida) e arquivo/ diretorio padrao.
const (
	// StoragePostgres grava no Postgres (default).
	StoragePostgres = "postgres"

	// StorageFiles apenas baixa/extrai as planilhas (sem persistir).
	StorageFiles = "files"
)

// DefaultBaseURL e o CDN de estatisticas do TSE (arquivos odsele).
const DefaultBaseURL = "https://cdn.tse.jus.br/estatistica/sead/odsele"

// TipoPrestador para PrestacaoContas.
const (
	TipoPrestadorCandidato       = "CANDIDATO"
	TipoPrestadorOrgaoPartidario = "ORGAO_PARTIDARIO"
)

// Tipos de registro de despesa.
const (
	TipoRegistroDespesaContratada = "CONTRATADA"
	TipoRegistroDespesaPaga       = "PAGA"
)

// TamanhoLotePadraoImportacao e o numero de registros por lote.
const TamanhoLotePadraoImportacao = 2000

// Scope define o recorte do processo: ano(s), UF(s) e tipo(s) de planilha.
type Scope struct {
	// Anos da eleicao (ex.: 2022, 2024). Obrigatorio.
	Anos []int `json:"anos"`

	// UFs (siglas de 2 letras). Vazio = todas as UFs.
	UFs []string `json:"ufs"`

	// Tipos de planilha (ver internal/dataset). Vazio = todos suportados.
	Tipos []string `json:"tipos"`
}

// DownloadSummary e o resultado da etapa de download/extração.
type DownloadSummary struct {
	Anos        []string     `json:"anos"`
	UFs         []string     `json:"ufs"`
	Tipos       []string     `json:"tipos"`
	Arquivos    []FileResult `json:"arquivos"`
	Total       int          `json:"total"`
	Baixados    int          `json:"baixados"`
	Falhas      int          `json:"falhas"`
	Pulados     int          `json:"pulados"`
	DownloadDir string       `json:"diretorio_download"`
	UpdatedAt   time.Time    `json:"atualizado_em"`
}

// ImportSummary e o resultado do processo completo (download -> extract -> persist).
type ImportSummary struct {
	Download  DownloadSummary `json:"download"`
	Inseridos Inserted        `json:"inseridos"`
	Ignorados int             `json:"ignorados"`
	Total     int64           `json:"total"`
	UpdatedAt time.Time       `json:"atualizado_em"`
}

// Inserted contabiliza registros persistidos por grupo.
type Inserted struct {
	Dimensoes  int64 `json:"dimensoes"`
	Candidatos int64 `json:"candidatos"`
	Prestacoes int64 `json:"prestacoes"`
	Transacoes int64 `json:"transacoes"`
	Bens       int64 `json:"bens"`
}

// Total soma todos os grupos.
func (i Inserted) Total() int64 {
	return i.Dimensoes + i.Candidatos + i.Prestacoes + i.Transacoes + i.Bens
}

// FileResult descreve o desfecho de um arquivo (baixado/extraido ou erro).
// Um mesmo arquivo (zip nacional por grupo/ano) alimenta varios tipos; as UFs
// informam quais recortes foram extraidos dele.
type FileResult struct {
	Nome      string   `json:"nome"`
	Grupo     string   `json:"grupo"`
	Tipos     []string `json:"tipos"`
	Ano       int      `json:"ano"`
	UFs       []string `json:"ufs"`
	Caminho   string   `json:"caminho,omitempty"`
	Registros int      `json:"registros,omitempty"`
	Falha     string   `json:"falha,omitempty"`
	Erro      bool     `json:"erro"`
}

// IndexStatus descreve o estado agregado do indice.
type IndexStatus struct {
	Exists    bool       `json:"existe"`
	UpdatedAt *time.Time `json:"atualizado_em"`
	Records   int        `json:"registros"`
}

// ProgressEvent descreve o andamento do processo em um instante.
type ProgressEvent struct {
	Estagio         string  `json:"estagio"`
	Dataset         string  `json:"dataset,omitempty"`
	UF              string  `json:"uf,omitempty"`
	Arquivo         string  `json:"arquivo,omitempty"`
	Processados     int     `json:"processados"`
	Total           int     `json:"total"`
	Registros       int64   `json:"registros"`
	Erros           int     `json:"erros"`
	DuracaoSegundos float64 `json:"duracao_segundos"`
	Timestamp       string  `json:"timestamp"`
}
