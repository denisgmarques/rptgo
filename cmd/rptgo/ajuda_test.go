package main

import (
	"strings"
	"testing"
)

func TestAjudaListaTodasAsOpcoes(t *testing.T) {
	for _, flag := range []string{"-h", "--help"} {
		code, out, errb := executa(t, "", flag)
		if code != 0 || errb != "" {
			t.Fatalf("%s: code=%d stderr=%q", flag, code, errb)
		}
		if !strings.HasPrefix(out, "Uso: rptgo [OPÇÃO]... SCRIPT.rp [ENTRADA.txt | -]") {
			t.Errorf("%s: linha de uso inesperada:\n%s", flag, out)
		}
		for _, op := range []string{"-o, --saida, --output", "--format", "--encoding", "--header", "--form-feed",
			"--newline", "--strict", "-r, --check", "-h, --help", "text", "csv", "json", "latin-1", "cp850", "crlf", "Exemplos:", "Código de saída"} {
			if !strings.Contains(out, op) {
				t.Errorf("%s: a ajuda não menciona %q", flag, op)
			}
		}
	}
	// --help vence mesmo junto de outras opções, inclusive inválidas
	if code, out, _ := executa(t, "", "--format", "xml", "--help"); code != 0 || !strings.Contains(out, "Uso:") {
		t.Errorf("--help junto de opção inválida: code=%d", code)
	}
}

func TestSemArgumentosMostraUsoCompleto(t *testing.T) {
	code, out, errb := executa(t, "")
	if code != 2 || out != "" {
		t.Fatalf("code=%d stdout=%q", code, out)
	}
	if !strings.HasPrefix(errb, "rptgo: falta o arquivo de script\n") || !strings.Contains(errb, "Uso: rptgo") || !strings.Contains(errb, "Opções:") {
		t.Errorf("stderr inesperado:\n%s", errb)
	}
}

func TestNomeDoProgramaNaAjudaENosErros(t *testing.T) {
	antes := nomePrograma
	nomePrograma = "rpt"
	defer func() { nomePrograma = antes }()
	if _, out, _ := executa(t, "", "--help"); !strings.Contains(out, "Uso: rpt [OPÇÃO]...") || !strings.Contains(out, "  rpt --format json") {
		t.Errorf("a ajuda deveria usar o nome rpt:\n%s", out)
	}
	_, _, errb := executa(t, "", "--format", "xml", "s.rp")
	if !strings.HasPrefix(errb, "rpt: ") || !strings.Contains(errb, "Tente 'rpt --help' para mais informações.") {
		t.Errorf("erro de uso: %q", errb)
	}
}

func TestValoresInvalidosDeOpcoes(t *testing.T) {
	casos := [][]string{
		{"--header", "talvez", "s.rp"},
		{"--newline", "cr", "s.rp"},
		{"--encoding", "ebcdic", "s.rp"},
		{"--format", "xml", "s.rp"},
		{"--saida"},
		{"--desconhecida", "s.rp"},
	}
	for _, args := range casos {
		code, out, errb := executa(t, "", args...)
		if code != 2 || out != "" || !strings.HasPrefix(errb, "rptgo: ") || !strings.Contains(errb, "--help") {
			t.Errorf("%v: code=%d stdout=%q stderr=%q", args, code, out, errb)
		}
	}
	// valores válidos continuam aceitos (inclusive encoding sem diferenciar maiúsculas)
	for _, args := range [][]string{
		{"--header", "nao", "--newline", "lf", "--encoding", "UTF-8", "--check", "../../examples/relatorio-vendas-regiao/relatorio.rp"},
		{"--encoding", "Latin1", "--check", "../../examples/relatorio-vendas-regiao/relatorio.rp"},
		{"--encoding", "CP850", "--check", "../../examples/relatorio-vendas-regiao/relatorio.rp"},
	} {
		if code, _, errb := executa(t, "", args...); code != 0 {
			t.Errorf("%v: code=%d stderr=%q", args, code, errb)
		}
	}
}

func TestCheckMensagem(t *testing.T) {
	code, out, errb := executa(t, "", "--check", "../../examples/relatorio-vendas-regiao/relatorio.rp")
	if code != 0 || out != "Script sem erros de sintaxe.\n" || errb != "" {
		t.Errorf("code=%d stdout=%q stderr=%q", code, out, errb)
	}
}
