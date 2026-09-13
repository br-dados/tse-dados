# tse-dados

Biblioteca Go que baixa as planilhas de dados abertos do TSE, extrai os CSVs
com um worker Python embutido e persiste os registros em Postgres. O download
sem persistência também é suportado.

| Modo | Descrição |
|------|-----------|
| `postgres` | Baixa, extrai e persiste no Postgres |
| `files` | Apenas baixa as planilhas para um diretório |

## Requisitos

- Go 1.22+
- Postgres 15+ (apenas para persistência com `postgres`)
- Python 3. O worker lê `.zip` e `.csv` com a biblioteca padrão. `.xlsx` exige
  `pandas` e `openpyxl`.

### Configuração

`tsedados.Options` configura a biblioteca.

```go
cliente, err := tsedados.New(ctx, tsedados.Options{
	Storage: "postgres",
	Postgres: &tsedados.PostgresConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "sua_senha",
		Database: "tsedados",
	},
	DownloadDir: "planilhas",
})
defer cliente.Close()
```

| Campo | Default | Descrição |
|-------|---------|-----------|
| `Storage string` | `postgres` | Backend de saída: `postgres` ou `files` |
| `Postgres *PostgresConfig` | `nil` | Conexão do backend `postgres`. Obrigatório nesse modo. |
| `DownloadDir string` | `.` | Diretório de saída das planilhas |
| `BaseURL string` | `https://cdn.tse.jus.br/estatistica/sead/odsele` | CDN do TSE |
| `Python *PythonConfig` | `python3` | Interpretador e timeout do worker de extração |
| `BatchSize int` | `2000` | Registros por lote na persistência |
| `MaxConcurrency int` | `1` | Downloads simultâneos |
| `ApplyMigrations *bool` | `true` | Aplica as migrations ao abrir (Postgres) |

`PostgresConfig` espelha a conexão:

| Campo | Default |
|-------|---------|
| `Host` | `127.0.0.1` |
| `Port` | `5432` |
| `User` | obrigatório |
| `Password` | obrigatório |
| `Database` | `tsedados` |
| `MaxConns` | `10` |
| `MinConns` | `2` |
| `MaxConnLifetime` | `30m` |
| `MaxConnIdleTime` | `5m` |

Host, porta, banco e pool têm default quando zerados. `User` e `Password` devem
ser informados, senha não tem default.

`PythonConfig` configura o worker:

| Campo | Default | Descrição |
|-------|---------|-----------|
| `Interpreter` | `python3` | Interpretador Python |
| `Timeout` | `0` | Limite de execução em segundos. `0` = sem limite. |

### Métodos do Cliente

| Método | Descrição |
|--------|-----------|
| `New(ctx, Options)` | Cria o cliente e abre o backend |
| `Download(ctx, Scope)` | Baixa as planilhas. Retorna `*types.DownloadSummary`. |
| `Import(ctx, Scope)` | Baixa, extrai e persiste. Retorna `*types.ImportSummary`. |
| `Progress()` | Andamento atual. Retorna `types.ProgressEvent`. |
| `Status(ctx)` | Estado agregado do índice (Postgres). Retorna `*types.IndexStatus`. |
| `CountRecords(ctx)` | Número de registros persistidos (Postgres) |
| `Close()` | Libera o pool Postgres |

### types.Scope

| Campo | Formato | Observação |
|-------|---------|------------|
| `Anos` | `[]int` | Obrigatório. Entre 1996 e 2100. |
| `UFs` | `[]string` | Siglas de 2 letras. Vazio = todas as 27 UFs. |
| `Tipos` | `[]string` | Tipos de planilha. Vazio = todos os suportados. |

Tipos explícitos que não existem no ano informado retornam
`types.ErrScopeNaoSuportado`.

### Exemplo mínimo (Postgres)

```go
package main

import (
	"context"
	"fmt"

	"github.com/danyele/tse-dados"
	"github.com/danyele/tse-dados/types"
)

func main() {
	ctx := context.Background()

	cliente, err := tsedados.New(ctx, tsedados.Options{
		Storage: "postgres",
		Postgres: &tsedados.PostgresConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "postgres",
			Password: "sua_senha",
			Database: "tsedados",
		},
	})
	if err != nil {
		panic(err)
	}
	defer cliente.Close()

	resumo, err := cliente.Import(ctx, types.Scope{
		Anos:  []int{2024},
		UFs:   []string{"GO"},
		Tipos: []string{"consulta_cand"},
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("total=%d dimensoes=%d candidatos=%d\n",
		resumo.Total, resumo.Inseridos.Dimensoes, resumo.Inseridos.Candidatos)
}
```

## Tipos de planilha

| Tipo | Anos |
|------|------|
| `consulta_cand` | 2006, 2008, 2010, 2012, 2014, 2016, 2018, 2020, 2022, 2024 |
| `bem_candidato` | 2006, 2008, 2010, 2012, 2014, 2016, 2018, 2020, 2022, 2024 |
| `despesas_contratadas_candidatos` | 2018, 2020, 2022, 2024 |
| `despesas_pagas_candidatos` | 2018, 2020, 2022, 2024 |
| `receitas_candidatos` | 2018, 2020, 2022, 2024 |
| `receitas_candidatos_doador_originario` | 2018, 2020, 2022, 2024 |
| `despesas_contratadas_orgaos_partidarios` | 2018, 2020, 2022, 2024 |
| `despesas_pagas_orgaos_partidarios` | 2018, 2020, 2022, 2024 |
| `receitas_orgaos_partidarios` | 2018, 2020, 2022, 2024 |
| `receitas_orgaos_partidarios_doador_originario` | 2018, 2020, 2022, 2024 |

## Pipeline

1. Download (`internal/fetcher`): monta uma URL por grupo e ano no padrão
   `{base}/{grupo}/{grupo}_{ano}.zip`, envia `User-Agent` de navegador e repete
   em `429` e `5xx` com backoff. Resposta `403` retorna `ErrBlocked`. O
   diretório de saída é criado se não existir.
2. Extração (`internal/python`): o worker Python é embutido com `go:embed`,
   extraído para um diretório temporário e executado. Ele lê `.zip`, `.csv` e
   `.xlsx`, filtra os arquivos internos por prefixo, ano e UF, e escreve um CSV
   normalizado por tabela em `<tmp>/<dataset>/<target>.csv`. O progresso é
   emitido como linhas JSON no stdout.
3. Carga (`internal/parse`): lê os CSVs normalizados, resolve as chaves naturais
   (código TSE, SQ, CPF/CNPJ) e monta o grafo de entidades.
4. Persistência (`internal/persist`): em uma transação, insere por níveis
   (dimensões, candidatos, fornecedores e doadores, prestações, receitas e
   despesas, bens e origem) com `COPY` em tabela temporária, upsert com
   `ON CONFLICT` e `RETURNING`, remapeando os IDs temporários para os IDs do
   banco.
5. Registro: grava `tse_arquivo_importado` (deduplicação por arquivo e UF).

## Configuração por CLI

`cmd/harvest` lê a configuração por flags.

```bash
# baixar consulta_cand 2024 de GO e SP sem persistir
go run ./cmd/harvest --ano 2024 --uf GO,SP --tipo consulta_cand --persistencia files

# persistir consulta_cand 2024 de GO
go run ./cmd/harvest --ano 2024 --uf GO --tipo consulta_cand --pg-password s3cret --pg-db tsedados
```

| Flag | Default | Descrição |
|------|---------|-----------|
| `--ano` | vazio | Ano(s) separados por vírgula. Obrigatório. |
| `--uf` | vazio | UFs de 2 letras separadas por vírgula. Vazio = todas. |
| `--tipo` | vazio | Tipos separados por vírgula. Vazio = todos. |
| `--persistencia` | `postgres` | `postgres` ou `files` |
| `--caminho-xlsx` | `.` | Diretório de saída das planilhas |
| `--pg-host` | `127.0.0.1` | Host do Postgres |
| `--pg-port` | `5432` | Porta do Postgres |
| `--pg-user` | `postgres` | Usuário do Postgres |
| `--pg-password` | vazio | Senha. Obrigatória em `postgres`. |
| `--pg-db` | `tsedados` | Banco do Postgres |
| `--pg-max-conns` | `10` | Máximo de conexões no pool |
| `--base-url` | `https://cdn.tse.jus.br/estatistica/sead/odsele` | CDN do TSE |
| `--python` | `python3` | Interpretador do worker |
| `--batch-size` | `2000` | Registros por lote |
| `--max-concurrency` | `1` | Downloads simultâneos |
| `--no-migrate` | `false` | Não aplica as migrations ao abrir |

`cmd/download` baixa sem persistir, equivalente a `--persistencia files`.
`cmd/migrate` aplica as migrations.

## Aplicar as migrations

Ao abrir o backend `postgres`, as migrations embutidas são aplicadas. Desligue
com `ApplyMigrations` apontando para `false`.

As migrations são idempotentes. Rodar de novo responde `nenhuma migracao
pendente`. A versão aplicada fica em `tse_migrations`. As definições ficam em
`internal/database/migrations/`.

## Ordem e dependências

Os tipos são processados na ordem de dependência. Um tipo que exige outro não é
importado sem que a dependência esteja no escopo ou já tenha sido importada para
o ano e a UF.

| Tipo | Prioridade | Depende de |
|------|-----------|------------|
| `consulta_cand` | 1 | |
| `bem_candidato` | 2 | `consulta_cand` |
| `despesas_contratadas_candidatos` | 3 | `consulta_cand` |
| `receitas_candidatos` | 4 | `consulta_cand` |
| `receitas_candidatos_doador_originario` | 5 | `receitas_candidatos` |
| `despesas_contratadas_orgaos_partidarios` | 6 | `consulta_cand` |
| `receitas_orgaos_partidarios` | 7 | `consulta_cand` |
| `receitas_orgaos_partidarios_doador_originario` | 8 | `receitas_orgaos_partidarios` |
| `despesas_pagas_candidatos` | 9 | `consulta_cand` |
| `despesas_pagas_orgaos_partidarios` | 10 | `consulta_cand` |

Se o escopo pedir um tipo e a dependência não estiver no escopo nem importada,
`Import` retorna `types.ErrDependenciaAusente` indicando o que falta. Quando a
dependência já está no banco, os IDs são resolvidos por cache, sem rebaixar o
arquivo.

## Deduplicação e reexecução

Cada arquivo importado é registrado em `tse_arquivo_importado` pela chave
`(nome, uf)`. Um zip nacional alimenta várias UFs, então cada UF tem sua própria
linha.

No modo `postgres`, arquivos já importados para uma UF não são baixados de novo.
Os zips baixados são apagados após a persistência bem-sucedida. Reexecutar o
mesmo escopo retorna `pulados` maior que zero e não insere nada.

## Schema

Tabelas de dimensão:

- `tse_eleicao`: código TSE, ano, tipo, descrição e data.
- `tse_unidade_eleitoral`: UF, código TSE e nome.
- `tse_partido`: número, sigla, nome, federação e coligação.

Tabelas de entidade:

- `tse_candidato`: SQ, eleição, UF, partido, cargo, dados pessoais e situação.
- `tse_fornecedor`: CPF/CNPJ, nome, CNAE, esfera partidária e localização.
- `tse_doador`: CPF/CNPJ, nome, CNAE, esfera partidária e localização.
- `tse_prestacao_contas`: SQ do prestador, eleição, candidato, partido, UF, tipo
  e data.

Tabelas de movimentação:

- `tse_despesa_candidato` e `tse_despesa_orgao_partidario`: prestação, candidato
  ou partido, fornecedor, documento, classificação e valor.
- `tse_receita_candidato` e `tse_receita_orgao_partidario`: prestação, candidato
  ou partido, doador, documento, classificação e valor.
- `tse_receita_doador_originario_candidato` e
  `tse_receita_doador_originario_orgao_partidario`: doador originário, documento,
  data e valor.
- `tse_bem_candidato`: candidato, tipo, número de ordem, descrição e valor.

Tabelas de controle:

- `tse_arquivo_importado`: arquivos importados por nome e UF.
- `tse_migrations`: versões de migration aplicadas.

## Testes

```bash
go test ./...
```
