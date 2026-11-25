package main

import (
	"strings"
	"testing"
)

func TestConverteCaixasTabela(t *testing.T) {
	entrada := strings.Join([]string{
		"+----------------+-------------+",
		"| Cliente        | Valor       |",
		"+----------------+-------------+",
		"| Acme           |    1.500,00 |",
		"+----------------+-------------+",
	}, "\n")
	want := strings.Join([]string{
		"┌────────────────┬─────────────┐",
		"│ Cliente        │ Valor       │",
		"├────────────────┼─────────────┤",
		"│ Acme           │    1.500,00 │",
		"└────────────────┴─────────────┘",
	}, "\n")
	if got := converteCaixas(entrada); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestConverteCaixasNaoMexeNoTexto(t *testing.T) {
	// nada disso forma linha: tem que sair idêntico
	for _, s := range []string{
		"2026-06-30", "a - b", "x | y", "1 + 2 = 3", "C++", "a--b? sim", "- item", "-5",
		"| solto |", "texto | com | barras", "+ sozinho", "total = a + b - c",
	} {
		got := converteCaixas(s)
		// "a--b" é uma sequência de 2 traços: vira linha de propósito (regra: run de 2+)
		if s == "a--b? sim" {
			if got != "a──b? sim" {
				t.Errorf("%q → %q", s, got)
			}
			continue
		}
		if got != s {
			t.Errorf("%q não devia mudar, virou %q", s, got)
		}
	}
}

func TestConverteCaixasCantosEJuncoes(t *testing.T) {
	entrada := strings.Join([]string{
		"+--+--+",
		"|  |  |",
		"+--+--+",
		"|  |  |",
		"+--+--+",
		"",
		"  +---+",
		"  |   |",
		"  +---+",
	}, "\n")
	want := strings.Join([]string{
		"┌──┬──┐",
		"│  │  │",
		"├──┼──┤",
		"│  │  │",
		"└──┴──┘",
		"",
		"  ┌───┐",
		"  │   │",
		"  └───┘",
	}, "\n")
	if got := converteCaixas(entrada); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestConverteCaixasLinhasDeTamanhosDiferentes(t *testing.T) {
	// a 3ª linha é mais curta: sem '|' acima, o '+' final não tem 2 ligações e fica como está
	got := converteCaixas("+--+\n|  |\n|\n+--+")
	if got != "┌──┐\n│  │\n│\n└──+" {
		t.Errorf("got %q", got)
	}
}

func TestConverteCaixasPaginasFormFeed(t *testing.T) {
	// a caixa não atravessa o \f: o '|' do fim da página 1 não vê a '+' da página 2
	got := converteCaixas("+--+\n|  |\f|  |\n+--+")
	want := "┌──┐\n│  │\f│  │\n└──┘"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
	if got := converteCaixas("texto\f|  |\fmais"); got != "texto\f|  |\fmais" {
		t.Errorf("| solto numa página: %q", got)
	}
}

func TestBoxEmExecutar(t *testing.T) {
	dir := t.TempDir()
	rp := dir + "/s.rp"
	dat := dir + "/d.txt"
	script := "input\n  nome [string 10],\n  v [numeric]\nbefore report\n  print '+------------+------+'\n  print '| Nome       | Valor|'\n  print '+------------+------+'\ndetail\n  print '|' in col 1, nome in col 3, '|' in col 14, v in col 16 using '####', '|' in col 21\nafter report\n  print '+------------+------+'\nend\n"
	writeFile(t, rp, script)
	writeFile(t, dat, "Acme|10\nBeta|200\n")
	code, out, errb := executa(t, "", "--box", rp, dat)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{"┌────────────┬──────┐", "│ Nome       │ Valor│", "├────────────┼──────┤", "└────────────┴──────┘"} {
		if !strings.Contains(out, want) {
			t.Errorf("saída sem %q:\n%s", want, out)
		}
	}
	// sem --box nada muda
	_, out2, _ := executa(t, "", rp, dat)
	if strings.ContainsAny(out2, "┌─│└┘┐├┤┬┴┼") || !strings.Contains(out2, "+------------+------+") {
		t.Errorf("sem --box a saída não deveria ter caracteres de linha:\n%s", out2)
	}
	// cp850: bytes de desenho de caixa do DOS
	_, out3, _ := executa(t, "", "--box", "--encoding", "cp850", rp, dat)
	if !strings.Contains(out3, "\xda\xc4\xc4\xc4\xc4\xc4\xc4\xc4\xc4\xc4\xc4\xc4\xc4\xc2\xc4\xc4\xc4\xc4\xc4\xc4\xbf") {
		t.Errorf("cp850: sem os bytes ┌─…┬─…┐ (0xDA 0xC4… 0xC2… 0xBF):\n%q", out3)
	}
}

func TestBoxValidacoes(t *testing.T) {
	casos := [][]string{
		{"--box", "--format", "csv", "s.rp"},
		{"--box", "--format", "json", "s.rp"},
		{"--box", "--encoding", "latin-1", "s.rp"},
		{"--box", "--encoding", "ISO-8859-1", "s.rp"},
	}
	for _, args := range casos {
		code, out, errb := executa(t, "", args...)
		if code != 2 || out != "" || !strings.Contains(errb, "--box") {
			t.Errorf("%v: code=%d stderr=%q", args, code, errb)
		}
	}
}
