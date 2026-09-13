// Package progress mantem o andamento do processo de download/extracao/persist.
package progress

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/danyele/tse-dados/types"
)

// Progress acompanha eventos atomicos do processo.
type Progress struct {
	mu          sync.Mutex
	estagio     atomic.Value
	dataset     atomic.Value
	uf          atomic.Value
	arquivo     atomic.Value
	processados atomic.Int64
	total       atomic.Int64
	registros   atomic.Int64
	erros       atomic.Int64
	inicio      time.Time
}

// New cria o tracker.
func New() *Progress { return &Progress{inicio: time.Now()} }

func (p *Progress) SetEstagio(s string)  { p.estagio.Store(s) }
func (p *Progress) SetDataset(d string)  { p.dataset.Store(d) }
func (p *Progress) SetUF(u string)       { p.uf.Store(u) }
func (p *Progress) SetArquivo(a string)  { p.arquivo.Store(a) }
func (p *Progress) SetTotal(n int)       { p.total.Store(int64(n)) }
func (p *Progress) AddProcessado(n int)  { p.processados.Add(int64(n)) }
func (p *Progress) AddRegistros(n int64) { p.registros.Add(n) }
func (p *Progress) AddErros(n int)       { p.erros.Add(int64(n)) }

// Evento devolve o snapshot atual.
func (p *Progress) Evento() types.ProgressEvent {
	get := func(v atomic.Value) string {
		if x := v.Load(); x != nil {
			return x.(string)
		}
		return ""
	}
	return types.ProgressEvent{
		Estagio:         get(p.estagio),
		Dataset:         get(p.dataset),
		UF:              get(p.uf),
		Arquivo:         get(p.arquivo),
		Processados:     int(p.processados.Load()),
		Total:           int(p.total.Load()),
		Registros:       p.registros.Load(),
		Erros:           int(p.erros.Load()),
		DuracaoSegundos: time.Since(p.inicio).Seconds(),
		Timestamp:       time.Now().Format(time.RFC3339Nano),
	}
}
