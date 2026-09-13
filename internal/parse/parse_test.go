package parse

import (
	"os"
	"path/filepath"
	"testing"
)

func escrever(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestConstruirGrafoComFKs(t *testing.T) {
	dir := t.TempDir()

	escrever(t, dir, "consulta_cand/eleicao.csv", "codigo_tse,ano,descricao\n1,2022,Eleicao 2022\n")
	escrever(t, dir, "consulta_cand/partido.csv", "numero,sigla,nome\n13,PT,Partido dos Trabalhadores\n")
	escrever(t, dir, "consulta_cand/candidato.csv",
		"sq_candidato,eleicao_codigo_tse,sg_uf,partido_numero,nome_completo\n"+
			"1001,1,GO,13,FULANO DE TAL\n")
	escrever(t, dir, "despesas_pagas_candidatos/prestacao_contas.csv",
		"tipo_prestador,sq_prestador_contas,eleicao_codigo_tse,sq_candidato,sg_uf,tipo_prestacao\n"+
			"CANDIDATO,500,1,1001,GO,RELATORIO\n")
	escrever(t, dir, "despesas_pagas_candidatos/despesa_candidato.csv",
		"tipo_prestador,tipo_registro,sq_despesa,sq_prestador_contas,eleicao_codigo_tse,sq_candidato,descricao,valor\n"+
			"CANDIDATO,PAGA,777,500,1,1001,ALMOCO,1234.56\n")

	rows, err := LerDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	g := Construir(rows)

	if len(g.Eleicoes) != 1 {
		t.Fatalf("eleicoes=%d, esperado 1", len(g.Eleicoes))
	}
	if len(g.Candidatos) != 1 {
		t.Fatalf("candidatos=%d, esperado 1", len(g.Candidatos))
	}
	if len(g.Prestacoes) != 1 {
		t.Fatalf("prestacoes=%d, esperado 1", len(g.Prestacoes))
	}
	if len(g.DespesasCandidato) != 1 {
		t.Fatalf("despesas=%d, esperado 1", len(g.DespesasCandidato))
	}
	d := g.DespesasCandidato[0]
	cand := g.Candidatos[1001]
	if d.CandidatoID != cand.ID {
		t.Fatalf("candidato_id nao resolvido: %s != %s", d.CandidatoID, cand.ID)
	}
	if d.Valor != 1234.56 {
		t.Fatalf("valor=%v, esperado 1234.56", d.Valor)
	}
	prest := g.Prestacoes["CANDIDATO|1|500"]
	if prest.CandidatoID == nil || *prest.CandidatoID != cand.ID {
		t.Fatalf("prestacao.candidato_id nao resolvido")
	}
}
