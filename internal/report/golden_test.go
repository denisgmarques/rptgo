package report

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// -update regrava os arquivos .txt de referência a partir do comportamento atual.
// Rede de segurança do modo texto: qualquer mudança aqui precisa ser intencional.
var update = flag.Bool("update", false, "regrava os arquivos golden")

const relogioFixo = "2026-11-24 17:47"

// executaGolden roda um script do testdata/golden sobre o arquivo de dados correspondente.
func executaGolden(t *testing.T, nome string, formFeed bool) string {
	t.Helper()
	t.Setenv("RPT_NOW", relogioFixo)
	// t.Setenv registra a restauração do valor original; depois disso remove a variável de
	// verdade (valor vazio é diferente de variável ausente para cfg()).
	for _, k := range []string{"DATEFMT", "CENTURY_CUTOFF", "TIMEFMT", "WIDTH", "PAGELN", "SEPARATOR", "RADIXSEP", "CURRSYM"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	dir := filepath.Join("testdata", "golden")
	src, err := os.ReadFile(filepath.Join(dir, nome+".rp"))
	if err != nil {
		t.Fatal(err)
	}
	dados, err := os.ReadFile(filepath.Join(dir, nome+".dat"))
	if err != nil {
		t.Fatal(err)
	}
	toks, err := Tokenize(string(src), false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	var regs [][]string
	for _, ln := range strings.Split(strings.TrimRight(string(dados), "\n"), "\n") {
		regs = append(regs, strings.Split(ln, s.Separator))
	}
	it, err := NovoInterp(s, regs, formFeed)
	if err != nil {
		t.Fatal(err)
	}
	out, err := it.Run()
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestGoldenTextoAtual(t *testing.T) {
	casos := []struct {
		nome     string
		formFeed bool
		arquivo  string
	}{
		{"vendas", false, "vendas.txt"},
		{"estoque", false, "estoque.txt"},
		{"misto", false, "misto.txt"},
		{"misto", true, "misto_formfeed.txt"},
	}
	for _, c := range casos {
		t.Run(c.arquivo, func(t *testing.T) {
			got := executaGolden(t, c.nome, c.formFeed)
			caminho := filepath.Join("testdata", "golden", c.arquivo)
			if *update {
				if err := os.WriteFile(caminho, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(caminho)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Errorf("saída diferente de %s\n--- got ---\n%s\n--- want ---\n%s", caminho, got, want)
			}
		})
	}
}
