package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const dirGolden = "../../internal/report/testdata/golden"

// executa roda executar() com o relógio fixo e sem variáveis de configuração herdadas.
func executa(t *testing.T, stdin string, args ...string) (int, string, string) {
	t.Helper()
	t.Setenv("RPT_NOW", "2026-11-24 17:47")
	for _, k := range []string{"DATEFMT", "CENTURY_CUTOFF", "TIMEFMT", "WIDTH", "PAGELN", "SEPARATOR", "RADIXSEP", "CURRSYM"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	var out, errb bytes.Buffer
	code := executar(args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

func lerGolden(t *testing.T, nome string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dirGolden, nome))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestExecutarPadraoAtual fixa a saída de texto de executar() para combinações de flags
// existentes (T01); as mesmas referências valem para --format text depois (RF-03).
func TestExecutarPadraoAtual(t *testing.T) {
	script := filepath.Join(dirGolden, "misto.rp")
	dados := filepath.Join(dirGolden, "misto.dat")
	dadosComCabecalho := "id|nome|nasc|cadast|ativo|saldo\n" + lerGolden(t, "misto.dat")
	casos := []struct {
		nome  string
		stdin string
		args  []string
		want  string
	}{
		{"padrao", "", []string{script, dados}, lerGolden(t, "misto.txt")},
		{"stdin", lerGolden(t, "misto.dat"), []string{script, "-"}, lerGolden(t, "misto.txt")},
		{"form-feed", "", []string{"--form-feed", script, dados}, lerGolden(t, "misto_formfeed.txt")},
		{"newline crlf", "", []string{"--newline", "crlf", script, dados}, strings.ReplaceAll(lerGolden(t, "misto.txt"), "\n", "\r\n")},
		{"header sim", dadosComCabecalho, []string{"--header", "sim", script, "-"}, lerGolden(t, "misto.txt")},
		{"header auto", dadosComCabecalho, []string{script, "-"}, lerGolden(t, "misto.txt")},
		{"vendas", "", []string{filepath.Join(dirGolden, "vendas.rp"), filepath.Join(dirGolden, "vendas.dat")}, lerGolden(t, "vendas.txt")},
		{"estoque", "", []string{filepath.Join(dirGolden, "estoque.rp"), filepath.Join(dirGolden, "estoque.dat")}, lerGolden(t, "estoque.txt")},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			code, out, errb := executa(t, c.stdin, c.args...)
			if code != 0 {
				t.Fatalf("exit %d, stderr: %s", code, errb)
			}
			if out != c.want {
				t.Errorf("saída diferente da referência\n--- got ---\n%q\n--- want ---\n%q", out, c.want)
			}
		})
	}
}
