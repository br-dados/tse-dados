// Package fetcher baixa os arquivos do TSE (zip/xlsx/csv) tratando os bloqueios
// de clientes automatizados (User-Agent de navegador) e retries com backoff.
package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/danyele/tse-dados/internal/dataset"
	"github.com/danyele/tse-dados/types"
)

const browserUA = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// ErrBlocked e retornado quando a origem responde com status anti-bot (403).
type ErrBlocked struct{ URL string }

func (e *ErrBlocked) Error() string {
	return fmt.Sprintf("acesso bloqueado pela origem (%s): o CDN do TSE pode bloquear clientes automatizados; baixe manualmente ou use Options.BaseURL", e.URL)
}

// Baixar baixa as specs em paralelo (limitado por concurrency) para dir.
// Notifica progresso por callback. Retorna o resultado por arquivo.
func Baixar(ctx context.Context, specs []dataset.FetchSpec, dir string, concurrency int, progress func(pronto, total int)) ([]types.FileResult, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("criar diretorio %q: %w", dir, err)
	}
	if concurrency < 1 {
		concurrency = 1
	}
	results := make([]types.FileResult, len(specs))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	pronto := 0

	for i, spec := range specs {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, spec dataset.FetchSpec) {
			defer wg.Done()
			defer func() { <-sem }()
			res := types.FileResult{Nome: spec.Nome, Grupo: spec.Grupo, Tipos: spec.Tipos, Ano: spec.Ano, UFs: spec.UFs}
			p, err := BaixarUm(ctx, spec, dir, 4)
			if err != nil {
				res.Erro = true
				res.Falha = err.Error()
				mu.Lock()
				results[i] = res
				pronto++
				mu.Unlock()
				return
			}
			res.Caminho = p
			mu.Lock()
			results[i] = res
			pronto++
			mu.Unlock()
			if progress != nil {
				progress(pronto, len(specs))
			}
		}(i, spec)
	}
	wg.Wait()
	return results, nil
}

// BaixarUm baixa um arquivo com retries de backoff.
func BaixarUm(ctx context.Context, spec dataset.FetchSpec, dir string, tentativas int) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 30 * time.Minute}
	var lastErr error
	for attempt := 1; attempt <= tentativas; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, spec.URL, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", browserUA)
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if !sleepBackoff(ctx, attempt) {
				return "", err
			}
			continue
		}
		if resp.StatusCode == http.StatusForbidden {
			resp.Body.Close()
			return "", &ErrBlocked{URL: spec.URL}
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d em %s", resp.StatusCode, spec.URL)
			if !sleepBackoff(ctx, attempt) {
				return "", lastErr
			}
			continue
		}

		dst := filepath.Join(dir, spec.Nome)
		if abs, aerr := filepath.Abs(dst); aerr == nil {
			dst = abs
		}
		out, err := os.Create(dst)
		if err != nil {
			resp.Body.Close()
			return "", err
		}
		_, cerr := io.Copy(out, resp.Body)
		out.Close()
		resp.Body.Close()
		if cerr != nil {
			os.Remove(dst)
			lastErr = cerr
			if !sleepBackoff(ctx, attempt) {
				return "", cerr
			}
			continue
		}
		return dst, nil
	}
	return "", lastErr
}

// FormatoDetectado informa o formato de um arquivo pela extensao.
func FormatoDetectado(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".zip":
		return "zip"
	case ".xlsx":
		return "xlsx"
	case ".csv":
		return "csv"
	default:
		return "zip"
	}
}

func sleepBackoff(ctx context.Context, attempt int) bool {
	backoff := time.Duration(1<<uint(attempt-1)) * time.Second
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}
	select {
	case <-ctx.Done():
		return false
	case <-time.After(backoff):
		return true
	}
}
