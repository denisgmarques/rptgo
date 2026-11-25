package report

import (
	"os"
	"time"
)

// cfg lê uma variável de configuração do ambiente, com valor padrão.
func cfg(nome, padrao string) string {
	if v, ok := os.LookupEnv(nome); ok {
		return v
	}
	return padrao
}

// agora devolve a data/hora "atual". RPT_NOW (formato "2026-11-24 17:47" ou RFC3339) permite
// testes reprodutíveis, fixando o relógio.
func agora() time.Time {
	if v := os.Getenv("RPT_NOW"); v != "" {
		layouts := []string{"2006-01-02 15:04:05", "2006-01-02 15:04", time.RFC3339}
		for _, l := range layouts {
			if t, err := time.Parse(l, v); err == nil {
				return t
			}
		}
	}
	return time.Now()
}
