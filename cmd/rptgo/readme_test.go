package main

import (
	"os"
	"strings"
	"testing"
)

func lerArquivo(t *testing.T, caminho string) string {
	t.Helper()
	b, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// blocosPrimeirosPassos devolve o conteúdo dos blocos de código (```) da seção
// "## Primeiros passos" do README, na ordem em que aparecem.
func blocosPrimeirosPassos(t *testing.T) []string {
	t.Helper()
	readme := lerArquivo(t, "../../README.md")
	i := strings.Index(readme, "## Primeiros passos")
	if i < 0 {
		t.Fatal("README sem a seção 'Primeiros passos'")
	}
	sec := readme[i+len("## Primeiros passos"):]
	if j := strings.Index(sec, "\n## "); j >= 0 {
		sec = sec[:j]
	}
	var blocos []string
	partes := strings.Split(sec, "```\n")
	// partes: texto, bloco, texto, bloco, ...
	for k := 1; k < len(partes); k += 2 {
		blocos = append(blocos, strings.TrimSuffix(partes[k], "\n"))
	}
	return blocos
}

// TestReadmePrimeirosPassos: os comandos e as saídas documentados no README reproduzem
// exatamente o que o programa gera sobre os arquivos de examples/ (RF-12).
func TestReadmePrimeirosPassos(t *testing.T) {
	blocos := blocosPrimeirosPassos(t)
	if len(blocos) != 4 {
		t.Fatalf("esperados 4 blocos de código (2 comandos, 2 saídas), vieram %d", len(blocos))
	}
	casos := []struct {
		cmd, saida string
	}{{blocos[0], blocos[1]}, {blocos[2], blocos[3]}}
	for _, c := range casos {
		var args []string
		for _, ln := range strings.Split(c.cmd, "\n") {
			if strings.HasPrefix(ln, "./rptgo ") {
				args = strings.Fields(strings.TrimPrefix(ln, "./rptgo "))
			}
		}
		if args == nil {
			t.Fatalf("bloco sem linha './rptgo ...':\n%s", c.cmd)
		}
		for i, a := range args {
			if strings.HasPrefix(a, "examples/") {
				args[i] = "../../" + a
			}
		}
		code, out, errb := executa(t, "", args...)
		if code != 0 {
			t.Fatalf("%v: exit %d: %s", args, code, errb)
		}
		if got := strings.TrimRight(out, "\n"); got != c.saida {
			t.Errorf("a saída do README mudou para %v\n--- gerada ---\n%s\n--- README ---\n%s", args, got, c.saida)
		}
	}
}
