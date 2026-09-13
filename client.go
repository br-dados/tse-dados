// Package tsedados e a API publica da biblioteca tse-dados: download das
// planilhas do TSE e persistencia no Postgres (o processamento em lote do
// XLSX e delegado a um worker Python embutido).
//
// Uso minimo (Postgres):
//
//	cliente, err := tsedados.New(ctx, tsedados.Options{
//		Storage: "postgres",
//		Postgres: &tsedados.PostgresConfig{Host: "localhost", User: "app", Password: "s3cret", Database: "tsedados"},
//	})
//	resumo, err := cliente.Import(ctx, tsedados.Scope{Anos: []int{2022}, UFs: []string{"GO"}, Tipos: []string{"consulta_cand"}})
package tsedados

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danyele/tse-dados/internal/database"
	"github.com/danyele/tse-dados/internal/dataset"
	"github.com/danyele/tse-dados/internal/fetcher"
	"github.com/danyele/tse-dados/internal/parse"
	"github.com/danyele/tse-dados/internal/persist"
	"github.com/danyele/tse-dados/internal/postgres"
	"github.com/danyele/tse-dados/internal/progress"
	"github.com/danyele/tse-dados/internal/python"
	"github.com/danyele/tse-dados/internal/repositorios"
	"github.com/danyele/tse-dados/storage"
	"github.com/danyele/tse-dados/types"
)

// PostgresConfig descreve a conexao Postgres do backend "postgres".
type PostgresConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Database        string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// PythonConfig configura a execucao do worker Python.
type PythonConfig struct {
	Interpreter string
	Timeout     int
	OnProgress  func(python.ProgressEvent)
}

// Options configura a construcao do Client (configuracao 100% por codigo, sem
// variaveis de ambiente).
type Options struct {
	// Storage define o backend. Valores: "postgres" (default), "files".
	Storage string

	// Postgres define a conexao do backend "postgres". Obrigatorio nesse modo.
	Postgres *PostgresConfig

	// DownloadDir e o diretorio de saida das planilhas (modo download).
	DownloadDir string

	// BaseURL e o CDN do TSE (default types.DefaultBaseURL).
	BaseURL string

	// Python configura o worker.
	Python *PythonConfig

	// Datasets permite estender/sobrescrever o registry.
	Datasets *dataset.Registry

	// BatchSize e o numero de registros por lote (default 2000).
	BatchSize int

	// MaxConcurrency limita downloads simultaneos (default 1).
	MaxConcurrency int

	// ApplyMigrations aplica as migrations embutidas ao abrir (default true).
	ApplyMigrations *bool
}

// Client e o handle principal da biblioteca.
type Client struct {
	registry    *dataset.Registry
	baseURL     string
	storage     storage.Storage
	pool        *pgxpool.Pool
	repo        *repositorios.Repositorio
	pythonCfg   python.Config
	batchSize   int
	concurr     int
	downloadDir string
	progress    *progress.Progress
}

// New monta um Client conforme as Options (resolvendo defaults por codigo).
func New(ctx context.Context, options Options) (*Client, error) {
	storageType := options.Storage
	if storageType == "" {
		storageType = types.StoragePostgres
	}

	reg := options.Datasets
	if reg == nil {
		reg = dataset.New()
	}

	batchSize := options.BatchSize
	if batchSize <= 0 {
		batchSize = types.TamanhoLotePadraoImportacao
	}
	concurr := options.MaxConcurrency
	if concurr <= 0 {
		concurr = 1
	}
	baseURL := options.BaseURL
	if baseURL == "" {
		baseURL = types.DefaultBaseURL
	}
	downloadDir := options.DownloadDir
	if downloadDir == "" {
		downloadDir = "."
	}

	c := &Client{
		registry:    reg,
		baseURL:     baseURL,
		batchSize:   batchSize,
		concurr:     concurr,
		downloadDir: downloadDir,
		progress:    progress.New(),
		pythonCfg: python.Config{
			Interpreter: "python3",
		},
	}

	if options.Python != nil {
		c.pythonCfg.Interpreter = options.Python.Interpreter
		c.pythonCfg.Timeout = options.Python.Timeout
		c.pythonCfg.OnProgress = options.Python.OnProgress
	}

	switch storageType {
	case types.StoragePostgres:
		if options.Postgres == nil {
			return nil, types.ErrPostgresRequired
		}
		p := options.Postgres
		cfg := database.Config{
			Host: p.Host, Port: p.Port, User: p.User, Password: p.Password, Database: p.Database,
			MaxConns: p.MaxConns, MinConns: p.MinConns, MaxConnLifetime: p.MaxConnLifetime, MaxConnIdleTime: p.MaxConnIdleTime,
		}
		if cfg.Host == "" {
			cfg.Host = "127.0.0.1"
		}
		if cfg.Port == "" {
			cfg.Port = "5432"
		}
		if cfg.MaxConns <= 0 {
			cfg.MaxConns = 10
		}
		if cfg.MinConns <= 0 {
			cfg.MinConns = 2
		}
		if cfg.MaxConnLifetime == 0 {
			cfg.MaxConnLifetime = 30 * time.Minute
		}
		if cfg.MaxConnIdleTime == 0 {
			cfg.MaxConnIdleTime = 5 * time.Minute
		}
		if cfg.Database == "" {
			cfg.Database = "tsedados"
		}
		pool, err := database.NewPool(ctx, cfg)
		if err != nil {
			return nil, err
		}
		apply := true
		if options.ApplyMigrations != nil {
			apply = *options.ApplyMigrations
		}
		if apply {
			if err := database.Migrate(ctx, pool); err != nil && !errors.Is(err, database.ErrNoPendingMigrations) {
				pool.Close()
				return nil, fmt.Errorf("aplicar migracoes: %w", err)
			}
		}
		c.pool = pool
		c.repo = repositorios.Novo(pool)
		c.storage = postgres.New(pool)
	case types.StorageFiles:
		// sem pool
	default:
		return nil, fmt.Errorf("%w: %q", types.ErrUnsupportedStorage, storageType)
	}

	return c, nil
}

// Close libera o pool (se houver).
func (c *Client) Close() {
	if c.storage != nil {
		c.storage.Close()
	}
}

// Progress devolve o andamento do ultimo/atual processo.
func (c *Client) Progress() types.ProgressEvent { return c.progress.Evento() }

// Status devolve o estado agregado (requer Postgres).
func (c *Client) Status(ctx context.Context) (*types.IndexStatus, error) {
	if c.storage == nil {
		return nil, types.ErrUnsupportedStorage
	}
	return c.storage.Status(ctx)
}

// CountRecords conta os registros persistidos (requer Postgres).
func (c *Client) CountRecords(ctx context.Context) (int, error) {
	if c.storage == nil {
		return 0, types.ErrUnsupportedStorage
	}
	return c.storage.CountRecords(ctx)
}

// Download baixa as planilhas do escopo para c.downloadDir. No modo files as
// dependencias precisam estar no proprio escopo.
func (c *Client) Download(ctx context.Context, scope types.Scope) (*types.DownloadSummary, error) {
	anos, ufs, tipos, err := c.resolver(scope)
	if err != nil {
		return nil, err
	}
	if err := c.registry.ValidarDependenciasEmEscopo(tipos); err != nil {
		return nil, err
	}
	fetch, err := c.registry.BuildFetch(c.baseURL, anos, ufs, tipos, len(scope.Tipos) > 0)
	if err != nil {
		return nil, err
	}
	sum, err := c.download(ctx, fetch)
	if err != nil {
		return nil, err
	}
	sum.Anos = anosStr(scope.Anos)
	sum.UFs = scope.UFs
	sum.Tipos = scope.Tipos
	return sum, nil
}

// download executa o fetch de uma lista ja montada e resume o resultado.
func (c *Client) download(ctx context.Context, fetch []dataset.FetchSpec) (*types.DownloadSummary, error) {
	c.progress.SetEstagio("download")
	c.progress.SetTotal(len(fetch))

	res, err := fetcher.Baixar(ctx, fetch, c.downloadDir, c.concurr, func(pronto, total int) {
		c.progress.SetTotal(total)
		c.progress.AddProcessado(1)
	})
	if err != nil {
		return nil, err
	}
	sum := &types.DownloadSummary{DownloadDir: c.downloadDir, UpdatedAt: time.Now().UTC()}
	var baixados, falhas int
	for _, r := range res {
		sum.Arquivos = append(sum.Arquivos, r)
		if r.Erro {
			falhas++
			c.progress.AddErros(1)
		} else {
			baixados++
		}
	}
	sum.Total = len(res)
	sum.Baixados = baixados
	sum.Falhas = falhas
	return sum, nil
}

// Import baixa, extrai (worker Python) e persiste no Postgres. Com
// Storage=files, limita-se ao download.
//
// Em Postgres, arquivos ja importados para uma UF nao sao baixados de novo, e
// os zips baixados sao apagados apos a persistencia bem-sucedida.
//
// Dependencias: tipos que exigem outros (ex.: bem_candidato depende de
// consulta_cand) precisam do tipo exigido no escopo ou ja importado no banco.
// Quando ja esta no banco, os IDs sao resolvidos por cache, sem rebaixar.
func (c *Client) Import(ctx context.Context, scope types.Scope) (*types.ImportSummary, error) {
	anos, ufs, tipos, err := c.resolver(scope)
	if err != nil {
		return nil, err
	}

	var importados map[string]bool
	if c.repo != nil {
		importados, err = c.repo.ListarArquivosImportados(ctx)
		if err != nil {
			return nil, err
		}
	}
	if err := c.validarDependencias(tipos, anos, ufs, importados); err != nil {
		return nil, err
	}

	fetch, err := c.registry.BuildFetch(c.baseURL, anos, ufs, tipos, len(scope.Tipos) > 0)
	if err != nil {
		return nil, err
	}

	pulados := 0
	if c.repo != nil {
		filtrados := make([]dataset.FetchSpec, 0, len(fetch))
		for _, s := range fetch {
			faltantes := ufsFaltantes(s, importados)
			if len(faltantes) == 0 {
				pulados++
				continue
			}
			s.UFs = faltantes
			filtrados = append(filtrados, s)
		}
		fetch = filtrados
	}

	var download *types.DownloadSummary
	if len(fetch) == 0 {
		download = &types.DownloadSummary{DownloadDir: c.downloadDir, UpdatedAt: time.Now().UTC()}
	} else {
		download, err = c.download(ctx, fetch)
		if err != nil {
			return nil, err
		}
	}
	download.Anos = anosStr(scope.Anos)
	download.UFs = scope.UFs
	download.Tipos = scope.Tipos
	download.Pulados = pulados

	if c.repo == nil {
		// modo "files": apenas baixou
		return &types.ImportSummary{Download: *download, UpdatedAt: time.Now().UTC()}, nil
	}

	// Somente arquivos baixados com sucesso.
	var files []dataset.Extracted
	for _, r := range download.Arquivos {
		if r.Erro || r.Caminho == "" {
			continue
		}
		files = append(files, dataset.Extracted{
			Path:   r.Caminho,
			Grupo:  r.Grupo,
			Tipos:  r.Tipos,
			Ano:    r.Ano,
			UFs:    r.UFs,
			Format: fetcher.FormatoDetectado(r.Caminho),
		})
	}
	if len(files) == 0 {
		return &types.ImportSummary{Download: *download, UpdatedAt: time.Now().UTC()}, nil
	}

	tmpOut, err := createWorkTmp()
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpOut)

	manifest, err := c.registry.BuildManifest(files, tmpOut)
	if err != nil {
		return nil, err
	}
	c.progress.SetEstagio("extract")
	if err := python.Run(ctx, manifest, c.pythonCfg); err != nil {
		return nil, err
	}

	c.progress.SetEstagio("load")
	rows, err := parse.LerDir(tmpOut)
	if err != nil {
		return nil, err
	}
	cache, err := c.carregarCache(ctx, tipos, fetch, importados)
	if err != nil {
		return nil, err
	}
	grafo := parse.ConstruirComCache(rows, cache)

	if _, err := c.persist(ctx, grafo, download); err != nil {
		return nil, err
	}

	// Apaga os zips baixados apos serem consultados/persistidos.
	for _, r := range download.Arquivos {
		if r.Erro || r.Caminho == "" {
			continue
		}
		_ = os.Remove(r.Caminho)
	}

	inseridos := insertFromGrafo(grafo)
	return &types.ImportSummary{
		Download:  *download,
		Inseridos: inseridos,
		Ignorados: grafo.Ignorados,
		Total:     inseridos.Total(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

// ufsFaltantes devolve as UFs de um zip que ainda nao foram importadas.
func ufsFaltantes(s dataset.FetchSpec, importados map[string]bool) []string {
	out := make([]string, 0, len(s.UFs))
	for _, uf := range s.UFs {
		if importados[s.Nome+"|"+uf] {
			continue
		}
		out = append(out, uf)
	}
	return out
}

func (c *Client) persist(ctx context.Context, grafo *parse.Grafo, download *types.DownloadSummary) (types.Inserted, error) {
	conn, err := c.pool.Acquire(ctx)
	if err != nil {
		return types.Inserted{}, fmt.Errorf("adquirir conexao: %w", err)
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return types.Inserted{}, err
	}
	defer tx.Rollback(ctx)

	res := &repositorios.ImportacaoResultado{}
	if _, err := persist.Executar(ctx, tx, c.repo, grafo, c.batchSize, res); err != nil {
		return types.Inserted{}, err
	}

	importados, err := c.repo.ListarArquivosImportados(ctx)
	if err == nil {
		tipos := func(r types.FileResult) string { return strings.Join(r.Tipos, ",") }
		for _, r := range download.Arquivos {
			if r.Erro || r.Caminho == "" {
				continue
			}
			for _, uf := range r.UFs {
				if importados[r.Nome+"|"+uf] {
					continue
				}
				_ = c.repo.RegistrarArquivoImportado(ctx, tx, r.Caminho, r.Nome, tipos(r), uf, r.Ano, r.Registros, "")
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return types.Inserted{}, fmt.Errorf("commit: %w", err)
	}
	return types.Inserted{}, nil
}

// resolver normaliza o escopo: anos, UFs e tipos (estes ordenados por
// prioridade, com dependencias antes dos dependentes).
func (c *Client) resolver(scope types.Scope) ([]int, []string, []string, error) {
	anos, err := validarAnos(scope.Anos)
	if err != nil {
		return nil, nil, nil, err
	}
	ufs, err := validarUFs(scope.UFs)
	if err != nil {
		return nil, nil, nil, err
	}
	tipos, err := c.registry.ResolveTipos(scope.Tipos)
	if err != nil {
		return nil, nil, nil, err
	}
	return anos, ufs, c.registry.OrdenarPorPrioridade(tipos), nil
}

// validarDependencias exige que cada dependencia esteja no escopo ou ja tenha
// sido importada para todos os anos e UFs pedidos.
func (c *Client) validarDependencias(tipos []string, anos []int, ufs []string, importados map[string]bool) error {
	if c.repo == nil {
		return c.registry.ValidarDependenciasEmEscopo(tipos)
	}
	presentes := make(map[string]bool, len(tipos))
	for _, t := range tipos {
		presentes[t] = true
	}
	for _, t := range tipos {
		for _, dep := range c.registry.Dependencias(t) {
			if presentes[dep] {
				continue
			}
			if c.dependenciaImportada(dep, anos, ufs, importados) {
				continue
			}
			return fmt.Errorf("%w: %q depende de %q; inclua %q no escopo ou importe-o antes",
				types.ErrDependenciaAusente, t, dep, dep)
		}
	}
	return nil
}

// dependenciaImportada informa se o zip de um tipo ja foi importado para todos
// os anos e UFs informados.
func (c *Client) dependenciaImportada(dep string, anos []int, ufs []string, importados map[string]bool) bool {
	spec, ok := c.registry.Spec(dep)
	if !ok {
		return false
	}
	for _, ano := range anos {
		nome := fmt.Sprintf("%s_%d.zip", spec.Grupo, ano)
		for _, uf := range ufs {
			if !importados[nome+"|"+uf] {
				return false
			}
		}
	}
	return true
}

// carregarCache monta o cache de IDs das dependencias que serao resolvidas pelo
// banco (ou seja, que nao estao sendo baixadas neste escopo). Devolve nil quando
// nao ha dependencia servida pelo banco.
func (c *Client) carregarCache(ctx context.Context, tipos []string, fetch []dataset.FetchSpec, importados map[string]bool) (*parse.Cache, error) {
	noFetch := map[string]bool{}
	for _, s := range fetch {
		for _, t := range s.Tipos {
			noFetch[t] = true
		}
	}

	var precisaConsulta, precisaReceitasCand, precisaReceitasOrgao bool
	for _, t := range tipos {
		for _, dep := range c.registry.Dependencias(t) {
			if noFetch[dep] {
				continue
			}
			switch dep {
			case "consulta_cand":
				precisaConsulta = true
			case "receitas_candidatos":
				precisaReceitasCand = true
			case "receitas_orgaos_partidarios":
				precisaReceitasOrgao = true
			}
		}
	}
	if !precisaConsulta && !precisaReceitasCand && !precisaReceitasOrgao {
		return nil, nil
	}

	cache := &parse.Cache{}
	var err error
	if precisaConsulta {
		if cache.Eleicoes, err = c.repo.CarregarIDsEleicoes(ctx); err != nil {
			return nil, err
		}
		if cache.Unidades, err = c.repo.CarregarIDsUnidades(ctx); err != nil {
			return nil, err
		}
		if cache.Partidos, err = c.repo.CarregarIDsPartidos(ctx); err != nil {
			return nil, err
		}
		if cache.Candidatos, err = c.repo.CarregarIDsCandidatos(ctx); err != nil {
			return nil, err
		}
	}
	if precisaReceitasCand || precisaReceitasOrgao {
		if cache.Prestacoes, err = c.repo.CarregarIDsPrestacoes(ctx); err != nil {
			return nil, err
		}
	}
	if precisaReceitasCand {
		if cache.ReceitasCand, err = c.repo.CarregarIDsReceitasCandidato(ctx); err != nil {
			return nil, err
		}
	}
	if precisaReceitasOrgao {
		if cache.ReceitasOrgao, err = c.repo.CarregarIDsReceitasOrgao(ctx); err != nil {
			return nil, err
		}
	}
	return cache, nil
}

func validarAnos(anos []int) ([]int, error) {
	if len(anos) == 0 {
		return nil, types.ErrEmptyScope
	}
	out := make([]int, 0, len(anos))
	for _, a := range anos {
		if a < 1996 || a > 2100 {
			return nil, fmt.Errorf("%w: %d", types.ErrInvalidYear, a)
		}
		out = append(out, a)
	}
	return out, nil
}

func validarUFs(ufs []string) ([]string, error) {
	if len(ufs) == 0 {
		return defaultUFs(), nil
	}
	out := make([]string, 0, len(ufs))
	for _, u := range ufs {
		u = strings.ToUpper(strings.TrimSpace(u))
		if !dataset.IsValidUF(u) {
			return nil, fmt.Errorf("%w: %q", types.ErrInvalidUF, u)
		}
		out = append(out, u)
	}
	return out, nil
}

func defaultUFs() []string {
	return []string{"AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO", "MA", "MT", "MS", "MG", "PA", "PB", "PR", "PE", "PI", "RJ", "RN", "RS", "RO", "RR", "SC", "SP", "SE", "TO"}
}

func anosStr(anos []int) []string {
	out := make([]string, 0, len(anos))
	for _, a := range anos {
		out = append(out, fmt.Sprintf("%d", a))
	}
	return out
}

// insertFromGrafo resume os conjuntos processados.
func insertFromGrafo(g *parse.Grafo) types.Inserted {
	return types.Inserted{
		Dimensoes:  int64(len(g.Eleicoes) + len(g.Unidades) + len(g.Partidos)),
		Candidatos: int64(len(g.Candidatos)),
		Prestacoes: int64(len(g.Prestacoes)),
		Transacoes: int64(len(g.DespesasCandidato) + len(g.DespesasOrgao) + len(g.ReceitasCandidato) + len(g.ReceitasOrgao)),
		Bens:       int64(len(g.Bens) + len(g.OrigemCand) + len(g.OrigemOrgao)),
	}
}

func createWorkTmp() (string, error) {
	d, err := os.MkdirTemp("", "tsedados-out-*")
	if err != nil {
		return "", err
	}
	return d, nil
}
