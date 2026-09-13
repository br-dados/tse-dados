// Package repositorios implementa a persistencia das entidades tse_* usando
// COPY em tabela staging + merge (upsert) com RETURNING para remapear os IDs
// temporarios gerados em memoria pelos IDs reais do banco.
package repositorios

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func setID(v any, id uuid.UUID) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	f := rv.FieldByName("ID")
	if f.IsValid() && f.CanSet() {
		f.Set(reflect.ValueOf(id))
	}
}

func getID(v any) uuid.UUID {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	f := rv.FieldByName("ID")
	if f.IsValid() {
		return f.Interface().(uuid.UUID)
	}
	return uuid.Nil
}

type tableConfig struct {
	stagingName string
	targetName  string
}

func tabelaConfig(target string) tableConfig {
	return tableConfig{stagingName: "stg_" + target, targetName: target}
}

func garantirTabelaStaging(ctx context.Context, tx pgx.Tx, cfg tableConfig) error {
	if _, err := tx.Exec(ctx, fmt.Sprintf(
		`CREATE TEMP TABLE IF NOT EXISTS %s (LIKE %s INCLUDING DEFAULTS)`,
		cfg.stagingName, cfg.targetName)); err != nil {
		return fmt.Errorf("criar staging %s: %w", cfg.stagingName, err)
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf(`TRUNCATE TABLE %s`, cfg.stagingName)); err != nil {
		return fmt.Errorf("truncate staging %s: %w", cfg.stagingName, err)
	}
	return nil
}

// StrNil converte string vazia em NULL.
func StrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func formatKey(v any) string {
	switch val := v.(type) {
	case [16]byte:
		return uuid.UUID(val).String()
	case string:
		return val
	case int64:
		return strconv.FormatInt(val, 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int16:
		return strconv.FormatInt(int64(val), 10)
	case int:
		return strconv.Itoa(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// copyInsertReturning copia um lote para a staging, faz merge no destino com
// titulo ON CONFLICT e devolve o mapa {ID temporario -> ID real do banco}.
func copyInsertReturning[T any](
	ctx context.Context, tx pgx.Tx,
	valores []*T, lote int,
	nomeTabela string,
	columns []string,
	conflictTarget string,
	setClause string,
	returningColumns []string,
	extractValues func(v *T) []any,
	keyFunc func(v *T) string,
	resultado *ImportacaoResultado,
) (map[uuid.UUID]uuid.UUID, error) {
	resultadoMap := make(map[uuid.UUID]uuid.UUID)
	if len(valores) == 0 {
		return resultadoMap, nil
	}
	cfg := tabelaConfig(nomeTabela)
	if err := garantirTabelaStaging(ctx, tx, cfg); err != nil {
		return nil, err
	}
	colList := strings.Join(columns, ", ")

	for i := 0; i < len(valores); i += lote {
		end := i + lote
		if end > len(valores) {
			end = len(valores)
		}
		batch := valores[i:end]
		rows := make([][]any, len(batch))
		for j, v := range batch {
			rows[j] = extractValues(v)
		}
		start := time.Now()
		copied, err := tx.CopyFrom(ctx, pgx.Identifier{cfg.stagingName}, columns, pgx.CopyFromRows(rows))
		if err != nil {
			return nil, fmt.Errorf("copy %s: %w", nomeTabela, err)
		}
		if resultado != nil {
			resultado.TempoCOPY += time.Since(start)
			resultado.Operacoes++
		}
		if copied != int64(len(batch)) {
			return nil, fmt.Errorf("copy %s: esperado %d, copiado %d", nomeTabela, len(batch), copied)
		}

		returningList := strings.Join(returningColumns, ", ")
		var mergeSQL string
		if conflictTarget == "" {
			mergeSQL = fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM %s RETURNING %s`,
				cfg.targetName, colList, colList, cfg.stagingName, returningList)
		} else {
			mergeSQL = fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM %s ON CONFLICT %s DO UPDATE SET %s RETURNING %s`,
				cfg.targetName, colList, colList, cfg.stagingName, conflictTarget, setClause, returningList)
		}
		start = time.Now()
		rowsResult, err := tx.Query(ctx, mergeSQL)
		if err != nil {
			return nil, fmt.Errorf("merge %s: %w", nomeTabela, err)
		}
		if resultado != nil {
			resultado.TempoMerge += time.Since(start)
		}

		isPlainInsert := conflictTarget == ""
		var keyToList map[string][]*T
		var lookup map[string]*T
		var origIDs map[*T]uuid.UUID
		if !isPlainInsert {
			keyToList = make(map[string][]*T, len(batch))
			lookup = make(map[string]*T, len(batch))
			origIDs = make(map[*T]uuid.UUID, len(batch))
			for _, v := range batch {
				k := keyFunc(v)
				keyToList[k] = append(keyToList[k], v)
				lookup[k] = v
				origIDs[v] = getID(v)
			}
		}

		countReturning := 0
		returningIdx := 0
		for rowsResult.Next() {
			vals, err := rowsResult.Values()
			if err != nil {
				rowsResult.Close()
				return nil, fmt.Errorf("scan values %s: %w", nomeTabela, err)
			}
			idRaw := vals[0].([16]byte)
			idUUID := uuid.UUID(idRaw)
			if isPlainInsert {
				if returningIdx < len(batch) {
					entry := batch[returningIdx]
					resultadoMap[getID(entry)] = idUUID
					setID(entry, idUUID)
					countReturning++
				}
			} else {
				if len(vals) < 2 {
					continue
				}
				var chave string
				if len(vals) == 2 {
					chave = formatKey(vals[1])
				} else {
					parts := make([]string, len(vals)-1)
					for idx, v := range vals[1:] {
						parts[idx] = formatKey(v)
					}
					chave = strings.Join(parts, "|")
				}
				if list, ok := keyToList[chave]; ok {
					for _, entry := range list {
						resultadoMap[origIDs[entry]] = idUUID
						setID(entry, idUUID)
						countReturning++
					}
				} else if v, ok := lookup[chave]; ok {
					resultadoMap[getID(v)] = idUUID
					setID(v, idUUID)
					countReturning++
				}
			}
			returningIdx++
		}
		rowsResult.Close()
		if err := rowsResult.Err(); err != nil {
			return nil, fmt.Errorf("iteracao %s: %w", nomeTabela, err)
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf(`TRUNCATE TABLE %s`, cfg.stagingName)); err != nil {
			return nil, fmt.Errorf("truncate staging %s: %w", nomeTabela, err)
		}
		_ = countReturning
	}
	return resultadoMap, nil
}

// copyInsertEmLote copia um lote para a staging e faz merge no destino
// (sem RETURNING). Devolve o total de linhas afetadas.
func copyInsertEmLote[T any](
	ctx context.Context, tx pgx.Tx,
	valores []*T, lote int,
	nomeTabela string,
	columns []string,
	conflictTarget string,
	extractValues func(v *T) []any,
	setClause string,
	resultado *ImportacaoResultado,
) (int64, error) {
	if len(valores) == 0 {
		return 0, nil
	}
	cfg := tabelaConfig(nomeTabela)
	if err := garantirTabelaStaging(ctx, tx, cfg); err != nil {
		return 0, err
	}
	colList := strings.Join(columns, ", ")
	var totalInserido int64

	for i := 0; i < len(valores); i += lote {
		end := i + lote
		if end > len(valores) {
			end = len(valores)
		}
		batch := valores[i:end]
		rows := make([][]any, len(batch))
		for j, v := range batch {
			rows[j] = extractValues(v)
		}
		start := time.Now()
		copied, err := tx.CopyFrom(ctx, pgx.Identifier{cfg.stagingName}, columns, pgx.CopyFromRows(rows))
		if err != nil {
			return totalInserido, fmt.Errorf("copy %s: %w", nomeTabela, err)
		}
		if resultado != nil {
			resultado.TempoCOPY += time.Since(start)
			resultado.Operacoes++
		}
		if copied != int64(len(batch)) {
			return totalInserido, fmt.Errorf("copy %s: esperado %d, copiado %d", nomeTabela, len(batch), copied)
		}

		var mergeSQL string
		if conflictTarget == "" {
			mergeSQL = fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM %s`,
				cfg.targetName, colList, colList, cfg.stagingName)
		} else {
			mergeSQL = fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM %s ON CONFLICT %s DO UPDATE SET %s`,
				cfg.targetName, colList, colList, cfg.stagingName, conflictTarget, setClause)
		}
		start = time.Now()
		tag, err := tx.Exec(ctx, mergeSQL)
		if err != nil {
			return totalInserido, fmt.Errorf("merge %s: %w", nomeTabela, err)
		}
		if resultado != nil {
			resultado.TempoMerge += time.Since(start)
			resultado.RegistrosInseridos += tag.RowsAffected()
		}
		totalInserido += tag.RowsAffected()
		if _, err := tx.Exec(ctx, fmt.Sprintf(`TRUNCATE TABLE %s`, cfg.stagingName)); err != nil {
			return totalInserido, fmt.Errorf("truncate staging %s: %w", nomeTabela, err)
		}
	}
	return totalInserido, nil
}
