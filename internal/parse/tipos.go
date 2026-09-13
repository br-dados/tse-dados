// Package parse le os CSVs normalizados produzidos pelo worker Python e monta o
// grafo de entidades, resolvendo as chaves naturais para UUIDs (mesma semantica
// do parser do projeto odp).
package parse

import (
	"log"
	"strconv"
	"strings"
	"time"
)

// quebraTexto normaliza o valor textual: remove espacos nas bordas e devolve
// vazio para valores considerados nulos pelo TSE.
func quebraTexto(valor string) string {
	texto := strings.TrimSpace(valor)
	switch strings.ToUpper(texto) {
	case "", "#NULO", "#NULO#", "#NE", "NÃO DIVULGÁVEL", "NAO DIVULGAVEL":
		return ""
	default:
		return texto
	}
}

// texto com fallback.
func textoFallback(valor, fallback string) string {
	if t := quebraTexto(valor); t != "" {
		return t
	}
	return fallback
}

// inteiro opcional (remove separadores de milhar).
func inteiroOpcional(valor string) *int {
	texto := quebraTexto(valor)
	if texto == "" {
		return nil
	}
	n := strings.ReplaceAll(texto, ".", "")
	n = strings.ReplaceAll(n, ",", "")
	i, err := strconv.Atoi(n)
	if err != nil || i < 0 {
		return nil
	}
	return &i
}

func inteiro16Opcional(valor string) *int16 {
	if i := inteiroOpcional(valor); i != nil {
		v := int16(*i)
		return &v
	}
	return nil
}

func inteiro64Opcional(valor string) *int64 {
	texto := quebraTexto(valor)
	if texto == "" {
		return nil
	}
	i, err := strconv.ParseInt(texto, 10, 64)
	if err != nil || i < 0 {
		return nil
	}
	return &i
}

// decimalOpcional aceita tanto o formato normalizado pelo worker ("1234.56")
// quanto o brasileiro ("1.234,56").
func decimalOpcional(valor string) *float64 {
	texto := quebraTexto(valor)
	if texto == "" {
		return nil
	}
	if d, err := strconv.ParseFloat(texto, 64); err == nil {
		return &d
	}
	n := strings.ReplaceAll(texto, ".", "")
	n = strings.ReplaceAll(n, ",", ".")
	d, err := strconv.ParseFloat(n, 64)
	if err != nil {
		return nil
	}
	return &d
}

func decimalZero(valor string) float64 {
	if d := decimalOpcional(valor); d != nil {
		return *d
	}
	return 0
}

// data opcional "dd/mm/aaaa" -> *time.Time.
func dataOpcional(valor string) *time.Time {
	texto := quebraTexto(valor)
	if texto == "" {
		return nil
	}
	if t, err := time.Parse("02/01/2006", texto); err == nil {
		return &t
	}
	return nil
}

// documento opcional: mantem apenas digitos; placeholder -> vazio.
func documentoOpcional(valor string) string {
	texto := quebraTexto(valor)
	if texto == "" {
		return ""
	}
	negativos := []string{"-4", "-1", "00000000000", "00000000000000"}
	for _, p := range negativos {
		if texto == p {
			return ""
		}
	}
	var b strings.Builder
	for _, r := range texto {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := b.String()
	if d == "" || strings.Trim(d, "0") == "" {
		return ""
	}
	return d
}

// str valor de campo (vazio -> fallback).
func str(v string) string { return quebraTexto(v) }

// warn loga um aviso simples (evita dependencia de logger externo).
func warn(format string, args ...any) { log.Printf("[parse] "+format, args...) }
