// Package dataset mantem o registro de tipos de planilha do TSE: diretorio no
// CDN, padrao de nome (por UF ou nacional), tabelas-alvo e o mapa de colunas
// TSE -> coluna normalizada. E a fonte unica de verdade consumida tanto pelo
// downloader (montagem de URL) quanto pelo worker Python (manifest) e pela
// camada de parse/esquema.
package dataset

import (
	"fmt"
	"sort"
	"strings"

	"github.com/danyele/tse-dados/types"
)

// Column descreve uma coluna de saida normalizada e a fonte no arquivo TSE.
// Header e o nome da coluna no arquivo TSE; Field e o nome da coluna no CSV
// normalizado (usado pelo worker Python e pelo parse em Go).
type Column struct {
	Header string `json:"header"`
	Field  string `json:"field"`
}

// Table descreve uma tabela-alvo alimentada por um dataset.
type Table struct {
	// Target e o nome da entidade/tabela no esquema tse_* (sem prefixo).
	Target string
	// Columns mapeia cada coluna normalizada para a fonte TSE.
	Columns []Column
	// TipoRegistro fixa o tipo de registro (despesas): CONTRATADA ou PAGA.
	TipoRegistro string
	// Prestador fixa o tipo de prestador de contas: CANDIDATO ou ORGAO_PARTIDARIO.
	Prestador string
}

// Spec descreve um dataset (tipo de planilha).
//
// O TSE publica um zip nacional por (grupo, ano) contendo os CSVs de cada UF.
// Grupo e o nome da pasta/zip no CDN (ex.: "prestacao_de_contas_eleitorais_candidatos")
// e Prefijo e o prefixo do CSV interno que corresponde a este tipo
// (ex.: "despesas_contratadas_candidatos" dentro do grupo acima).
type Spec struct {
	// Nome e o identificador do tipo (usado por Scope.Tipos).
	Nome string
	// Grupo e o diretorio/zip nacional no CDN (odsele) deste tipo.
	Grupo string
	// Prefijo e o prefixo do arquivo CSV interno do zip que alimenta o tipo.
	Prefijo string
	// Anos sao os anos em que o tipo existe (nil = qualquer ano).
	Anos []int
	// Prioridade define a ordem de processamento. Menor valor e processado
	// antes. Dependencias sempre tem prioridade menor que os dependentes.
	Prioridade int
	// Dependencias lista os tipos que precisam existir (no escopo ou no banco)
	// antes deste ser importado.
	Dependencias []string
	// Tables sao as tabelas-alvo alimentadas.
	Tables []Table
}

// SuportaAno informa se o tipo existe no ano informado.
func (s Spec) SuportaAno(ano int) bool {
	if len(s.Anos) == 0 {
		return true
	}
	for _, a := range s.Anos {
		if a == ano {
			return true
		}
	}
	return false
}

// Registry agrupa os datasets suportados.
type Registry struct {
	specs map[string]Spec
	order []string
}

// New cria o registry com todos os datasets suportados (sem convenio).
func New() *Registry {
	return &Registry{specs: defaultSpecs(), order: defaultOrder()}
}

// WithSpecs permite sobrescrever/estender o registry.
func (r *Registry) WithSpecs(specs ...Spec) *Registry {
	for _, s := range specs {
		if _, ok := r.specs[s.Nome]; !ok {
			r.order = append(r.order, s.Nome)
		}
		r.specs[s.Nome] = s
	}
	return r
}

// ListarTipos devolve os tipos suportados, em ordem estavel.
func (r *Registry) ListarTipos() []string {
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

// Spec devolve a spec de um tipo.
func (r *Registry) Spec(nome string) (Spec, bool) {
	s, ok := r.specs[nome]
	return s, ok
}

// Dependencias devolve os tipos exigidos por um tipo (vazio se nao houver).
func (r *Registry) Dependencias(nome string) []string {
	s, ok := r.specs[nome]
	if !ok {
		return nil
	}
	out := make([]string, len(s.Dependencias))
	copy(out, s.Dependencias)
	return out
}

// OrdenarPorPrioridade devolve os tipos na ordem de processamento, mantendo a
// ordem original como desempate.
func (r *Registry) OrdenarPorPrioridade(tipos []string) []string {
	out := make([]string, len(tipos))
	copy(out, tipos)
	sort.SliceStable(out, func(i, j int) bool {
		return r.prioridade(out[i]) < r.prioridade(out[j])
	})
	return out
}

func (r *Registry) prioridade(nome string) int {
	if s, ok := r.specs[nome]; ok {
		return s.Prioridade
	}
	return 99
}

// ValidarDependenciasEmEscopo verifica se todos os tipos exigidos por cada
// tipo informado tambem estao no escopo. Retorna um erro apontando o que falta.
func (r *Registry) ValidarDependenciasEmEscopo(tipos []string) error {
	presentes := make(map[string]bool, len(tipos))
	for _, t := range tipos {
		presentes[t] = true
	}
	for _, t := range tipos {
		for _, dep := range r.Dependencias(t) {
			if !presentes[dep] {
				return fmt.Errorf("%w: %q depende de %q; inclua %q no escopo", types.ErrDependenciaAusente, t, dep, dep)
			}
		}
	}
	return nil
}

// TabelasDe devolve os nomes de tabelas-alvo de um tipo.
func (r *Registry) TabelasDe(nome string) []string {
	s, ok := r.specs[nome]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(s.Tables))
	for _, t := range s.Tables {
		out = append(out, t.Target)
	}
	return out
}

// ResolveTipos normaliza Scope.Tipos: vazio = todos. Erro para desconhecido.
func (r *Registry) ResolveTipos(tipos []string) ([]string, error) {
	if len(tipos) == 0 {
		return r.ListarTipos(), nil
	}
	out := make([]string, 0, len(tipos))
	for _, t := range tipos {
		t = strings.TrimSpace(t)
		if _, ok := r.specs[t]; !ok {
			return nil, fmt.Errorf("%w: %q", types.ErrUnknownDataset, t)
		}
		out = append(out, t)
	}
	return out, nil
}

// FetchSpec descreve um arquivo (zip nacional) a baixar. Varios tipos podem
// compartilhar o mesmo zip (mesmo Grupo/Ano), por isso Tipos e uma lista.
type FetchSpec struct {
	URL   string
	Nome  string
	Grupo string
	Ano   int
	Tipos []string
	UFs   []string
}

// BuildFetch monta a lista de downloads (1 por Grupo x Ano) para um escopo ja
// resolvido. Quando estrito, uma combinacao (tipo, ano) inexistente e erro
// imediato; quando nao estrito (tipos omitidos), ela e simplesmente ignorada.
func (r *Registry) BuildFetch(base string, anos []int, ufs []string, tipos []string, estrito bool) ([]FetchSpec, error) {
	type chave struct {
		grupo string
		ano   int
	}
	indice := map[chave]int{}
	var out []FetchSpec
	for _, tipo := range tipos {
		spec, ok := r.specs[tipo]
		if !ok {
			return nil, fmt.Errorf("%w: %q", types.ErrUnknownDataset, tipo)
		}
		for _, ano := range anos {
			if !spec.SuportaAno(ano) {
				if estrito {
					return nil, fmt.Errorf("%w: tipo %q ano %d", types.ErrScopeNaoSuportado, tipo, ano)
				}
				continue
			}
			k := chave{spec.Grupo, ano}
			if i, ok := indice[k]; ok {
				out[i].Tipos = append(out[i].Tipos, tipo)
				continue
			}
			nome := fmt.Sprintf("%s_%d.zip", spec.Grupo, ano)
			out = append(out, FetchSpec{
				URL:   fmt.Sprintf("%s/%s/%s", strings.TrimRight(base, "/"), spec.Grupo, nome),
				Nome:  nome,
				Grupo: spec.Grupo,
				Ano:   ano,
				Tipos: []string{tipo},
				UFs:   append([]string(nil), ufs...),
			})
			indice[k] = len(out) - 1
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: nenhum tipo disponivel para os anos informados", types.ErrScopeNaoSuportado)
	}
	return out, nil
}

// Manifest descreve a entrada do worker Python para um conjunto de arquivos
// baixados. Vai para o subprocesso python via arquivo temporario.
type Manifest struct {
	OutputDir string         `json:"output_dir"`
	Files     []ManifestFile `json:"files"`
}

// ManifestFile descreve um zip baixado e os grupos (tipos) extraidos dele.
type ManifestFile struct {
	Path   string          `json:"path"`
	Format string          `json:"format"`
	Ano    int             `json:"ano"`
	UFs    []string        `json:"ufs,omitempty"`
	Grupos []ManifestGroup `json:"grupos"`
}

// ManifestGroup descreve como extrair um tipo (dataset) de um zip nacional:
// filtra os CSVs internos por Prefijo e ano, e escreve as tabelas-alvo.
type ManifestGroup struct {
	Dataset string          `json:"dataset"`
	Prefijo string          `json:"prefijo"`
	Tables  []ManifestTable `json:"tables"`
}

// ManifestTable descreve a extração de uma tabela de um arquivo.
type ManifestTable struct {
	Target       string   `json:"target"`
	Columns      []Column `json:"columns"`
	TipoRegistro string   `json:"tipo_registro,omitempty"`
	Prestador    string   `json:"prestador,omitempty"`
}

// BuildManifest monta o manifest para os arquivos baixados. Cada zip carrega
// um grupo por tipo solicitado, com prefixo e tabelas-alvo.
func (r *Registry) BuildManifest(files []Extracted, outputDir string) (Manifest, error) {
	f := make([]ManifestFile, 0, len(files))
	for _, ext := range files {
		mf := ManifestFile{Path: ext.Path, Format: ext.Format, Ano: ext.Ano, UFs: ext.UFs}
		for _, tipo := range ext.Tipos {
			spec, ok := r.specs[tipo]
			if !ok {
				return Manifest{}, fmt.Errorf("%w: %q", types.ErrUnknownDataset, tipo)
			}
			mg := ManifestGroup{Dataset: tipo, Prefijo: spec.Prefijo}
			for _, t := range spec.Tables {
				mg.Tables = append(mg.Tables, ManifestTable{
					Target:       t.Target,
					Columns:      t.Columns,
					TipoRegistro: t.TipoRegistro,
					Prestador:    t.Prestador,
				})
			}
			mf.Grupos = append(mf.Grupos, mg)
		}
		f = append(f, mf)
	}
	return Manifest{OutputDir: outputDir, Files: f}, nil
}

// Extracted descreve um zip ja baixado pronto para o worker.
type Extracted struct {
	Path   string
	Grupo  string
	Tipos  []string
	Ano    int
	UFs    []string
	Format string
}

// ufsMap valida UFs (usado pelo client).
var ufsMap = map[string]bool{
	"AC": true, "AL": true, "AP": true, "AM": true, "BA": true, "CE": true,
	"DF": true, "ES": true, "GO": true, "MA": true, "MT": true, "MS": true,
	"MG": true, "PA": true, "PB": true, "PR": true, "PE": true, "PI": true,
	"RJ": true, "RN": true, "RS": true, "RO": true, "RR": true, "SC": true,
	"SP": true, "SE": true, "TO": true,
}

// IsValidUF informa se ah mais uma sigla de UF valida.
func IsValidUF(uf string) bool { return ufsMap[strings.ToUpper(strings.TrimSpace(uf))] }
