// Command download baixa as planilhas do TSE para um escopo (sem persistir).
// Alias de `harvest --persistencia files`.
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
	var anos, ufs, tipos, xlsxDir, baseURL string
	var concurrency int
	flag.StringVar(&anos, "ano", "", "ano(s) separados por virgula")
	flag.StringVar(&ufs, "uf", "", "UFs separadas por virgula (vazio = todas)")
	flag.StringVar(&tipos, "tipo", "", "tipos separados por virgula (vazio = todos)")
	flag.StringVar(&xlsxDir, "caminho-xlsx", ".", "diretorio de saida")
	flag.StringVar(&baseURL, "base-url", types.DefaultBaseURL, "CDN do TSE")
	flag.IntVar(&concurrency, "max-concurrency", 1, "downloads simultaneos")
	flag.Parse()

	cliente, err := tsedados.New(context.Background(), tsedados.Options{
		Storage: "files", DownloadDir: xlsxDir, BaseURL: baseURL, MaxConcurrency: concurrency,
	})
	if err != nil {
		fatal(err.Error())
	}
	defer cliente.Close()

	sum, err := cliente.Download(context.Background(), types.Scope{
		Anos: parseInts(anos), UFs: split(ufs), Tipos: split(tipos),
	})
	if err != nil {
		fatal(err.Error())
	}
	b, _ := json.MarshalIndent(sum, "", "  ")
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

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "erro:", msg)
	os.Exit(1)
}
