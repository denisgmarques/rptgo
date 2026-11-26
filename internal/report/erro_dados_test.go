package report

import (
	"strings"
	"testing"
)

// fonteDeDados registra o texto de dados do teste e restaura o estado global no fim.
func fonteDeDados(t *testing.T, texto string) {
	t.Helper()
	fonte, localiza := FonteDados, localizadorDados
	t.Cleanup(func() { FonteDados, localizadorDados = fonte, localiza })
	DefFonteDados(texto)
}

// localizaEm registra um localizador que põe qualquer registro na posição dada.
func localizaEm(linha, coluna int) {
	DefLocalizadorDados(func(int) PosicaoDados { return PosicaoDados{Linha: linha, Coluna: coluna} })
}

// sequencia devolve n runas distintas entre vizinhas, pra o teste perceber um corte deslocado.
func sequencia(n int) string {
	const digitos = "0123456789"
	return strings.Repeat(digitos, n/len(digitos)+1)[:n]
}

func circunflexos(n int) string { return strings.Repeat("^", n) }

func TestErroDeDadosTrechoDaLinha(t *testing.T) {
	s := sequencia(200)
	acentos := strings.Repeat("áé€𝄞ç", 30) // 150 runas de vários bytes
	r := []rune(acentos)
	for _, c := range []struct {
		nome, linha     string
		coluna          int
		mostrada, marca string
	}{
		{"72 runas saem inteiras", s[:72], 0,
			s[:72], circunflexos(40)},
		{"72 runas ignoram a coluna", s[:72], 30,
			s[:72], circunflexos(40)},
		{"73 runas são cortadas", s[:73], 0,
			s[:72] + "...", circunflexos(40)},
		{"corte no começo da linha", s, 0,
			s[:72] + "...", circunflexos(40)},
		{"corte no meio da linha", s, 50,
			"..." + s[50:122] + "...", "   " + circunflexos(40)},
		{"sobram exatamente 72", s, 128,
			"..." + s[128:], "   " + circunflexos(40)},
		{"sobram 73", s, 127,
			"..." + s[127:199] + "...", "   " + circunflexos(40)},
		{"sobram menos de 72", s[:100], 40,
			"..." + s[40:100], "   " + circunflexos(40)},
		{"registro perto do fim da linha", s[:100], 95,
			"..." + s[95:100], "   " + circunflexos(5)},
		{"registro no último caractere", s[:100], 99,
			"..." + s[99:100], "   ^"},
		{"coluna depois do fim da linha", s[:100], 500,
			"...", "   ^"},
		{"coluna negativa", s[:100], -3,
			s[:72] + "...", circunflexos(40)},
		{"tabs no corte", "\t" + s[:40] + "\t{\t" + s[:90], 42,
			"...{\t" + s[:70] + "...", "   " + circunflexos(40)},
		{"tab no primeiro caractere do registro", s[:80] + "\tfim", 80,
			"...\tfim", "   " + circunflexos(4)},
		{"multibyte conta em runas", string(r[:72]), 0,
			string(r[:72]), circunflexos(40)},
		{"multibyte cortado no começo", acentos, 0,
			string(r[:72]) + "...", circunflexos(40)},
		{"multibyte cortado no meio", acentos, 33,
			"..." + string(r[33:105]) + "...", "   " + circunflexos(40)},
		{"multibyte perto do fim", acentos, 147,
			"..." + string(r[147:]), "   " + circunflexos(3)},
		{"CRLF numa linha de 72 sai como antes", s[:72] + "\r", 0,
			s[:72] + "\r", circunflexos(40)},
		{"CRLF numa linha de 73", s[:73] + "\r", 0,
			s[:72] + "...", circunflexos(40)},
		{"CRLF não vai no trecho final", s[:100] + "\r", 40,
			"..." + s[40:100], "   " + circunflexos(40)},
	} {
		t.Run(c.nome, func(t *testing.T) {
			fonteDeDados(t, "antes\n"+c.linha+"\ndepois\n")
			localizaEm(2, c.coluna)
			want := "falhou no registro 7\n  2 | " + c.mostrada + "\n      " + c.marca
			if got := ErroLinhaDados(7, "falhou no registro %d", 7).Error(); got != want {
				t.Errorf("ErroLinhaDados:\n got %q\nwant %q", got, want)
			}
			pos := PosicaoDados{Linha: 2, Coluna: c.coluna}
			if got := ErroPosicaoDados(pos, "falhou no registro %d", 7).Error(); got != want {
				t.Errorf("ErroPosicaoDados:\n got %q\nwant %q", got, want)
			}
			linha, _, _, _, ok := trechoDados(pos)
			if !ok || linha != "  2 | "+c.mostrada {
				t.Errorf("trechoDados: ok=%v linha=%q, want %q", ok, linha, "  2 | "+c.mostrada)
			}
		})
	}
}

func TestErroDeDadosMarcadorDeLinhaCurta(t *testing.T) {
	fonteDeDados(t, "    z|abc  \n\n")
	for _, c := range []struct {
		linha int
		want  string
	}{
		{1, "m\n  1 |     z|abc  \n      ^^^^^"},
		{2, "m\n  2 | \n      ^"},
	} {
		if got := ErroPosicaoDados(PosicaoDados{Linha: c.linha}, "m").Error(); got != c.want {
			t.Errorf("linha %d: got %q, want %q", c.linha, got, c.want)
		}
	}
}

func TestErroDeDadosRecuaOTrechoComALinhaDeVariosDigitos(t *testing.T) {
	s := sequencia(100)
	fonteDeDados(t, strings.Repeat("a\n", 11)+s+"\n")
	localizaEm(12, 10)
	want := "m\n  12 | ..." + s[10:82] + "...\n          " + circunflexos(40)
	if got := ErroLinhaDados(1, "m").Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestErroDeDadosSemLinhaMostraSoAMensagem(t *testing.T) {
	fonteDeDados(t, "a\nb\n")
	for _, linha := range []int{-1, 0, 3} {
		pos := PosicaoDados{Linha: linha}
		if got := ErroPosicaoDados(pos, "falhou").Error(); got != "falhou" {
			t.Errorf("linha %d: %q", linha, got)
		}
		if _, _, _, _, ok := trechoDados(pos); ok {
			t.Errorf("linha %d: trechoDados deveria falhar", linha)
		}
	}
}

func TestLocalizadorDeDados(t *testing.T) {
	fonteDeDados(t, "um\ndois\ntres\n")

	if got, want := ErroLinhaDados(2, "falhou").Error(), "falhou\n  2 | dois\n      ^^^^"; got != want {
		t.Errorf("sem localizador: got %q, want %q", got, want)
	}

	var pedidos []int
	DefLocalizadorDados(func(registro int) PosicaoDados {
		pedidos = append(pedidos, registro)
		return PosicaoDados{Linha: registro + 1}
	})
	if got, want := ErroLinhaDados(2, "falhou no registro %d", 2).Error(), "falhou no registro 2\n  3 | tres\n      ^^^^"; got != want {
		t.Errorf("com localizador: got %q, want %q", got, want)
	}
	if len(pedidos) != 1 || pedidos[0] != 2 {
		t.Errorf("registros pedidos ao localizador: %v", pedidos)
	}

	DefLocalizadorDados(nil)
	if got, want := ErroLinhaDados(1, "falhou").Error(), "falhou\n  1 | um\n      ^^"; got != want {
		t.Errorf("localizador nil: got %q, want %q", got, want)
	}

	localizaEm(3, 0)
	DefFonteDados("quatro\ncinco\n")
	if localizadorDados != nil {
		t.Error("DefFonteDados deveria descartar o localizador")
	}
	if got, want := ErroLinhaDados(2, "falhou").Error(), "falhou\n  2 | cinco\n      ^^^^^"; got != want {
		t.Errorf("depois de DefFonteDados: got %q, want %q", got, want)
	}
}

func TestErroDeDadosDoInterpretadorUsaOLocalizador(t *testing.T) {
	longa := "  " + sequencia(80) + "|x"
	for _, c := range []struct{ nome, tipo, valor, trecho string }{
		{"inteiro", "numeric", "x", "valor inválido em 'n' (registro 2)"},
		{"decimal", "float", "x", "valor inválido em 'n' (registro 2)"},
		{"hora", "time", "x", "valor inválido em 'n' (registro 2)"},
		{"data", "date", "x", "valor inválido em 'n' (registro 2)"},
		{"campos", "string", "", "registro 2 tem 1 campos, esperados 2"},
	} {
		t.Run(c.nome, func(t *testing.T) {
			fonteDeDados(t, "cabeçalho\nok\n"+longa+"\n")
			toks, err := Tokenize("input a, n ["+c.tipo+"]\nend\n", false)
			if err != nil {
				t.Fatal(err)
			}
			s, err := Parse(toks)
			if err != nil {
				t.Fatal(err)
			}
			DefLocalizadorDados(func(registro int) PosicaoDados {
				return PosicaoDados{Linha: registro + 1, Coluna: 2}
			})
			ruim := []string{"a", c.valor}
			if c.nome == "campos" {
				ruim = ruim[:1]
			}
			_, err = NovoInterp(s, [][]string{{"a", ""}, ruim}, false)
			if err == nil {
				t.Fatal("esperado um erro de dados")
			}
			want := "\n  3 | ..." + sequencia(72) + "...\n         " + circunflexos(40)
			if got := err.Error(); !strings.Contains(got, c.trecho) || !strings.HasSuffix(got, want) {
				t.Errorf("got %q, want conter %q e terminar com %q", got, c.trecho, want)
			}
		})
	}
}
