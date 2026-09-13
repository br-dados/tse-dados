// Command harvest baixa e (opcionalmente) persiste as planilhas do TSE para um
// escopo (ano, UF, tipo). Exemplos:
//
//	// baixar consulta_cand 2022 de GO e SP para o diretorio atual
//	go run ./cmd/harvest --ano 2022 --uf GO,SP --tipo consulta_cand --persistencia files
//
//	// persistir tudo no Postgres de teste
//	go run ./cmd/harvest --ano 2022 --uf GO --pg-user app --pg-password s3cret --pg-db tsedados
//
// Configuracao 100% por flags (sem variaveis de ambiente).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/danyele/tse-dados"
	"github.com/danyele/tse-dados/types"
)

func main() {
	var (
		anos                                 string
		ufs                                  string
		tipos                                string
		persistencia                         string
		xlsxDir                              string
		pgHost, pgPort, pgUser, pgPass, pgDB string
		pgMaxConns                           int
		baseURL                              string
		python                               string
		batchSize                            int
		concurrency                          int
		noMigrate                            bool
	)
	flag.StringVar(&anos, "ano", "", "ano(s) da eleicao, separados por virgula")
	flag.StringVar(&ufs, "uf", "", "UFs de 2 letras, separadas por virgula (vazio = todas)")
	flag.StringVar(&tipos, "tipo", "", "tipos de planilha, separados por virgula (vazio = todos)")
	flag.StringVar(&persistencia, "persistencia", "postgres", "postgres ou files")
	flag.StringVar(&xlsxDir, "caminho-xlsx", ".", "diretorio de saida/extração das planilhas")
	flag.StringVar(&pgHost, "pg-host", "127.0.0.1", "host do Postgres")
	flag.StringVar(&pgPort, "pg-port", "5432", "porta do Postgres")
	flag.StringVar(&pgUser, "pg-user", "postgres", "usuario do Postgres")
	flag.StringVar(&pgPass, "pg-password", "", "senha do Postgres (obrigatoria em modo postgres)")
	flag.StringVar(&pgDB, "pg-db", "tsedados", "banco do Postgres")
	flag.IntVar(&pgMaxConns, "pg-max-conns", 10, "maximo de conexoes do pool")
	flag.StringVar(&baseURL, "base-url", types.DefaultBaseURL, "CDN do TSE")
	flag.StringVar(&python, "python", "python3", "interpretador Python do worker")
	flag.IntVar(&batchSize, "batch-size", types.TamanhoLotePadraoImportacao, "registros por lote")
	flag.IntVar(&concurrency, "max-concurrency", 1, "downloads simultaneos")
	flag.BoolVar(&noMigrate, "no-migrate", false, "nao aplicar migrations ao abrir")
	flag.Parse()

	ctx := context.Background()

	options := tsedados.Options{
		Storage:         persistencia,
		DownloadDir:     xlsxDir,
		BaseURL:         baseURL,
		BatchSize:       batchSize,
		MaxConcurrency:  concurrency,
		Python:          &tsedados.PythonConfig{Interpreter: python},
		ApplyMigrations: boolPtr(!noMigrate),
	}
	if persistencia == "postgres" {
		if pgPass == "" {
			fatal("--pg-password e obrigatoria no modo postgres")
		}
		options.Postgres = &tsedados.PostgresConfig{
			Host: pgHost, Port: pgPort, User: pgUser, Password: pgPass, Database: pgDB,
			MaxConns: int32(pgMaxConns), MinConns: 2,
		}
	}

	cliente, err := tsedados.New(ctx, options)
	if err != nil {
		fatal("ergo ao iniciar: " + err.Error())
	}
	defer cliente.Close()

	scope := types.Scope{Anos: parseInts(anos), UFs: split(ufs), Tipos: split(tipos)}

	var out any
	if persistencia == "files" {
		sum, err := cliente.Download(ctx, scope)
		if err != nil {
			fatal("download: " + err.Error())
		}
		out = sum
	} else {
		sum, err := cliente.Import(ctx, scope)
		if err != nil {
			fatal("import: " + err.Error())
		}
		out = sum
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
}

func split(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseInts(s string) []int {
	var out []int
	for _, p := range split(s) {
		if n, err := strconv.Atoi(p); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func boolPtr(b bool) *bool { return &b }

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "erro:", msg)
	os.Exit(1)
}
