// Package python embute o worker Python (go:embed) que le as planilhas do TSE e
// emite CSVs normalizados, e o executa como subprocesso, consumindo o progresso.
package python

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/danyele/tse-dados/internal/dataset"
)

//go:embed all:worker
var workerFS embed.FS

// Config configura a execucao do worker.
type Config struct {
	// Interpreter e o binario python (default "python3").
	Interpreter string
	// Timeout limita a execucao do subprocesso (0 = sem limite).
	Timeout int
	// OnProgress recebe o progresso do worker (event by event).
	OnProgress func(ProgressEvent)
}

// ProgressEvent reflete a saida do worker.
type ProgressEvent struct {
	Estagio   string `json:"estagio"`
	Dataset   string `json:"dataset,omitempty"`
	UF        string `json:"uf,omitempty"`
	Arquivo   string `json:"arquivo,omitempty"`
	Registros int    `json:"registros"`
	Erro      string `json:"erro,omitempty"`
}

// Run extrai o worker embutido para um diretorio temporario e executa
// `python -m tse_dados_extract --manifest <path> --output <dir>`.
func Run(ctx context.Context, manifest dataset.Manifest, cfg Config) error {
	interpreter := cfg.Interpreter
	if interpreter == "" {
		interpreter = "python3"
	}

	work, err := os.MkdirTemp("", "tsedados-worker-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	if err := copyFS(work, workerFS); err != nil {
		return err
	}

	manifestPath := filepath.Join(work, "manifest.json")
	b, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(manifestPath, b, 0o600); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, interpreter, "-m", "tse_dados_extract", "--manifest", manifestPath)
	cmd.Dir = filepath.Join(work, "worker")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("iniciar worker python (%s): %w (instale python3 e 'pip install pandas openpyxl psycopg2')", interpreter, err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var workerErr string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev ProgressEvent
		if err := json.Unmarshal([]byte(line), &ev); err == nil {
			if ev.Erro != "" {
				workerErr = ev.Erro
			}
			if cfg.OnProgress != nil {
				cfg.OnProgress(ev)
			}
		}
	}
	scanErr := scanner.Err()
	waitErr := cmd.Wait()

	if scanErr != nil {
		return fmt.Errorf("ler saida do worker: %w", scanErr)
	}
	if waitErr != nil {
		msg := strings.TrimSpace(stderr.String())
		if workerErr != "" {
			if msg != "" {
				msg = workerErr + ": " + msg
			} else {
				msg = workerErr
			}
		}
		return fmt.Errorf("worker python falhou: %w: %s", waitErr, msg)
	}
	return nil
}

func copyFS(dst string, fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dst, path)
		if d.IsDir() {
			if d.Name() == "__pycache__" {
				return fs.SkipDir
			}
			return os.MkdirAll(target, 0o755)
		}
		// ignora bytecode .pyc embutido (evita usar versao obsoleta)
		if strings.HasSuffix(d.Name(), ".pyc") {
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
