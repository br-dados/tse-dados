package dataset

import (
	"errors"
	"testing"

	"github.com/danyele/tse-dados/types"
)

func TestListarTiposSemConvenio(t *testing.T) {
	r := New()
	tipos := r.ListarTipos()
	if len(tipos) != 10 {
		t.Fatalf("esperado 10 tipos (sem convenio e sem prestacao_contas dedicado), obtido %d: %v", len(tipos), tipos)
	}
	for _, tt := range tipos {
		if tt == "convenio" || tt == "prestacao_contas" {
			t.Fatalf("tipo %q nao deve estar no registry", tt)
		}
	}
}

func TestBuildFetchNacional(t *testing.T) {
	r := New()
	specs, err := r.BuildFetch("https://cdn.tse.jus.br/estatistica/sead/odsele", []int{2022}, []string{"GO"}, []string{"consulta_cand"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 {
		t.Fatalf("esperado 1 spec, obtido %d", len(specs))
	}
	want := "https://cdn.tse.jus.br/estatistica/sead/odsele/consulta_cand/consulta_cand_2022.zip"
	if specs[0].URL != want {
		t.Fatalf("url %q, esperado %q", specs[0].URL, want)
	}
}

func TestBuildFetchAgrupaTipos(t *testing.T) {
	r := New()
	specs, err := r.BuildFetch("https://x", []int{2022}, []string{"GO"}, []string{
		"despesas_contratadas_candidatos", "despesas_pagas_candidatos", "receitas_candidatos",
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 {
		t.Fatalf("esperado 1 spec (mesmo grupo), obtido %d", len(specs))
	}
	if specs[0].Grupo != grupoPrestacaoCand {
		t.Fatalf("grupo %q, esperado %q", specs[0].Grupo, grupoPrestacaoCand)
	}
	if len(specs[0].Tipos) != 3 {
		t.Fatalf("esperado 3 tipos no mesmo zip, obtido %d: %v", len(specs[0].Tipos), specs[0].Tipos)
	}
}

func TestBuildFetchAnoNaoSuportado(t *testing.T) {
	r := New()
	_, err := r.BuildFetch("https://x", []int{2006}, []string{"GO"}, []string{"receitas_candidatos"}, true)
	if !errors.Is(err, types.ErrScopeNaoSuportado) {
		t.Fatalf("esperado ErrScopeNaoSuportado, obtido %v", err)
	}
}

func TestBuildFetchNaoEstritoIgnoraAnoNaoSuportado(t *testing.T) {
	r := New()
	tipos := r.ListarTipos()
	specs, err := r.BuildFetch("https://x", []int{2006}, []string{"GO"}, tipos, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 2 {
		t.Fatalf("esperado 2 specs (consulta_cand e bem_candidato), obtido %d", len(specs))
	}
}

func TestResolveTiposDesconhecido(t *testing.T) {
	r := New()
	if _, err := r.ResolveTipos([]string{"nao_existe"}); err == nil {
		t.Fatal("esperado erro para tipo desconhecido")
	}
}

func TestBuildFetchMultiAno(t *testing.T) {
	r := New()
	specs, err := r.BuildFetch("https://x", []int{2022, 2024}, []string{"GO", "SP"}, []string{"bem_candidato"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 2 {
		t.Fatalf("esperado 2 specs (1 zip nacional por ano), obtido %d", len(specs))
	}
}

func TestValidarDependenciasEmEscopo(t *testing.T) {
	r := New()
	if err := r.ValidarDependenciasEmEscopo([]string{"bem_candidato"}); !errors.Is(err, types.ErrDependenciaAusente) {
		t.Fatalf("esperado ErrDependenciaAusente, obtido %v", err)
	}
	if err := r.ValidarDependenciasEmEscopo([]string{"consulta_cand", "bem_candidato"}); err != nil {
		t.Fatalf("nao esperado erro com a dependencia no escopo: %v", err)
	}
	if err := r.ValidarDependenciasEmEscopo([]string{"receitas_candidatos_doador_originario"}); !errors.Is(err, types.ErrDependenciaAusente) {
		t.Fatalf("esperado ErrDependenciaAusente para doador originario, obtido %v", err)
	}
}

func TestOrdenarPorPrioridade(t *testing.T) {
	r := New()
	ordem := r.OrdenarPorPrioridade([]string{"bem_candidato", "despesas_pagas_orgaos_partidarios", "consulta_cand"})
	if ordem[0] != "consulta_cand" {
		t.Fatalf("esperado consulta_cand primeiro, obtido %v", ordem)
	}
	if ordem[len(ordem)-1] != "despesas_pagas_orgaos_partidarios" {
		t.Fatalf("esperado despesas_pagas_orgaos_partidarios por ultimo, obtido %v", ordem)
	}
}
