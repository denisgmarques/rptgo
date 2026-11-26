package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Os erros novos saem iguais do arquivo .rp e do -e, antes de qualquer leitura de dados.
func TestErrosDeScriptNaLinhaDeComando(t *testing.T) {
	for _, script := range []string{
		"input nome, valor [numerc] end",
		"input idade [numeric] sort idades end",
		"input a sort total(a) end",
		"input a,\nend",
	} {
		rp, dat := arquivosLinha(t, script, "x\n")
		_, _, want := executa(t, "", rp, dat)
		if !strings.HasPrefix(want, "rptgo: linha ") || !strings.Contains(want, "^") {
			t.Fatalf("%q: stderr=%q", script, want)
		}
		for _, formato := range []string{"text", "csv", "json", "html"} {
			for _, extra := range [][]string{nil, {"--check"}, {"-r"}} {
				for _, fonte := range [][]string{{rp}, {"-e", script}} {
					destino := filepath.Join(t.TempDir(), "saida")
					args := append([]string{"--format", formato, "-o", destino}, extra...)
					args = append(append(args, fonte...), "nao-existe.txt")
					entrada := &leitorVigiado{}
					var out, errb bytes.Buffer
					if code := executar(args, entrada, &out, &errb); code != 1 || out.Len() != 0 || errb.String() != want || entrada.lido {
						t.Errorf("%q: code=%d lido=%v stdout=%q stderr=%q", args, code, entrada.lido, out.String(), errb.String())
					}
					if _, err := os.Stat(destino); err == nil {
						t.Errorf("%q: criou o arquivo de -o", args)
					}
				}
			}
		}
	}
}

func TestMarcadorAlinhadoNasCodificacoes(t *testing.T) {
	script := "input a\ndetail\n\tprint 'São João', @\nend\n"
	const want = "  3 | \tprint 'São João', @\n      \t                  ^\n"
	for _, enc := range []string{"utf-8", "latin-1", "cp850"} {
		rp, dat := arquivosLinha(t, string(codificar(script, enc)), "x\n")
		if code, _, errb := executa(t, "", "--encoding", enc, rp, dat); code != 1 || !strings.HasSuffix(errb, want) {
			t.Errorf("%s: code=%d stderr=%q", enc, code, errb)
		}
	}
}
