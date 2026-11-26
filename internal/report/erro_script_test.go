package report

import (
	"fmt"
	"strings"
	"testing"
)

// erroDoScript devolve a mensagem de erro da leitura do script ("" se ele for válido).
func erroDoScript(t *testing.T, script string, estrito bool) string {
	t.Helper()
	antes := FonteScript
	t.Cleanup(func() { FonteScript = antes })
	DefFonteScript(script)
	toks, err := Tokenize(script, estrito)
	if err == nil {
		_, err = Parse(toks)
	}
	if err == nil {
		return ""
	}
	return err.Error()
}

// conferePalavra recebe um script com a palavra errada entre « » e confere que o erro mostra a
// linha dela, com o marcador exatamente embaixo.
func conferePalavra(t *testing.T, marcado, trecho string, estrito bool) {
	t.Helper()
	antes, resto, ok1 := strings.Cut(marcado, "«")
	palavra, depois, ok2 := strings.Cut(resto, "»")
	if !ok1 || !ok2 {
		t.Fatalf("caso sem « »: %q", marcado)
	}
	script := antes + palavra + depois
	linha := strings.Count(antes, "\n") + 1
	fonte := strings.TrimSuffix(strings.Split(script, "\n")[linha-1], "\r")
	naLinha := antes[strings.LastIndex(antes, "\n")+1:]
	prefixo := fmt.Sprintf("  %d | ", linha)
	pad := strings.Repeat(" ", len(prefixo))
	for _, c := range naLinha {
		if c == '\t' {
			pad += "\t"
		} else {
			pad += " "
		}
	}
	want := "\n" + prefixo + fonte + "\n" + pad + strings.Repeat("^", len([]rune(palavra)))
	got := erroDoScript(t, script, estrito)
	if !strings.HasPrefix(got, fmt.Sprintf("linha %d: ", linha)) || !strings.Contains(got, trecho) || !strings.HasSuffix(got, want) {
		t.Errorf("%q:\n%s\n--- esperado conter %q e terminar com ---%s", script, got, trecho, want)
	}
}

func TestErroDeScriptMarcaAPalavra(t *testing.T) {
	for _, c := range []struct{ script, trecho string }{
		// léxico
		{"input a\ndetail\n   print a «@» a\nend", "caractere não reconhecido"},
		{"input a\ndetail\n  print «'abc def»   ", "string não terminada"},
		{"input a\ndetail\n  print «'abc»\n'\nend", "string deve estar numa única linha"},
		{"input a «/* abc»  \nend", "comentário não terminado"},
		// parser
		{"input a\ndetail\n print a\n«detail»\n print a\nend", "só pode aparecer uma vez"},
		{"input a\ndetail\n print <x> a, <«x»> a + 1\nend", "já foi dado a outra expressão"},
		{"input a\ndetail\n print <«1»> a\nend", "nome esperado entre < e >"},
		{"input a\ndetail\n print a substr[«x»-2]\nend", "substr espera"},
		{"input a\ndetail\n print a substr[1-«2.5»]\nend", "substr espera"},
		{"input a\ndetail\n print a «substr»[3-1]\nend", "substr [3-1] inválido"},
		{"input a\ndetail\n print «1/123/2000»\nend", "linha 3"},
		{"input a\nwidth «abc»\nend", "constante inteira"},
		{"input a\n  «banana»\nend", "instrução inválida fora de seção: 'banana'"},
		{"input a\ndetail\n print a, «,»\nend", "expressão inválida: encontrado ','"},
		{"input «1»\nend", "nome de coluna esperado"},
		{"input a, «5»\n detail\nend", "nome de coluna esperado"},
		{"input a\nbefore «5»\n print a\nend", "nome da seção esperado"},
		{"input a\ndetail\n set «5»\n print a\nend", "nome de variável esperado"},
		{"input a\ndetail\n page, no «rodape»\n print a\nend", "'footer' ou 'header' esperado"},
		{"input a\ndetail\n if a = '1' «print» a\nend", "esperado 'then'"},
		// fim do script: aponta a última palavra, mesmo com linhas e comentários depois
		{"input a\ndetail\n print «a»\n\n/* fim */\n", "falta o 'end' que termina o script: chegou ao fim do script depois de 'a'"},
		{"input a\ndetail\n print a «+»\n\n", "expressão inválida: chegou ao fim do script depois de '+'"},
		{"input a\ndetail\n if a = '1' then begin\n  if a = '2' then begin\n print «a»\n", "aberto na linha 4"},
		{"input a [«numerc»]\nend", "tipo desconhecido 'numerc' na coluna 'a'"},
		{"input Valor [«Numerc» 10]\nend", "tipo desconhecido 'Numerc' na coluna 'Valor'"},
		{"input a [«10»]\nend", "tipo desconhecido '10'"},
		{"input a [«]»\nend", "tipo desconhecido ']'"},
		{"input a «[»\n", "falta o tipo da coluna 'a'"},
		// sort
		{"input a\nsort «zz»\nend", "nome desconhecido no sort: 'zz'"},
		{"input a\nsort a, <k> a + «Zz» desc\nend", "nome desconhecido no sort: 'Zz'"},
		{"input a\nsort <k> dow(«zz»)\nend", "nome desconhecido no sort: 'zz'"},
		{"input a\nsort k\ndetail\n print <k> «zz»\nend", "nome desconhecido no sort: 'zz'"},
		{"input a\nsort <x> «x»\nend", "nome desconhecido no sort: 'x'"},
		{"input a\nsort <p> «q», <q> p\ndetail\n print <z> 1\nend", ""},
		{"input a\nsort «foo»(a)\nend", "função desconhecida no sort: 'foo'"},
		{"input a\nsort «today»()\nend", "função desconhecida no sort: 'today'"},
		{"input a\nsort «count»(*)\nend", "funções de agregação não podem ser usadas no sort: 'count'"},
		{"input a\nsort «total»(a)\nend", "'total'"},
		{"input a\nsort «avg»(a)\nend", "'avg'"},
		{"input a\nsort «min»(a)\nend", "'min'"},
		{"input a\nsort k\nafter report\n print <k> «max»(a)\nend", "'max'"},
	} {
		if c.trecho == "" { // nomes que se referem uns aos outros: aceitos como antes, sem laço
			if got := erroDoScript(t, strings.NewReplacer("«", "", "»", "").Replace(c.script), false); got != "" {
				t.Errorf("%q: %s", c.script, got)
			}
			continue
		}
		conferePalavra(t, c.script, c.trecho, false)
	}
	conferePalavra(t, "input a\n  «/* um»\ndois */\nend", "na mesma linha", true)
}

func TestTiposValidosNoInput(t *testing.T) {
	for tipo := range tiposInt {
		if !strings.Contains(" "+strings.Join(tiposColuna, " ")+" ", " "+tipo+" ") {
			t.Errorf("%s fora de tiposColuna", tipo)
		}
	}
	for tipo := range tiposFlt {
		if !strings.Contains(" "+strings.Join(tiposColuna, " ")+" ", " "+tipo+" ") {
			t.Errorf("%s fora de tiposColuna", tipo)
		}
	}
	for _, tipo := range tiposColuna {
		for _, s := range []string{tipo, strings.ToUpper(tipo), tipo + " 10"} {
			if got := erroDoScript(t, "input a ["+s+"], b\nend", false); got != "" {
				t.Errorf("[%s]: %s", s, got)
			}
		}
	}
	got := erroDoScript(t, "input a [x]\nend", false)
	if !strings.Contains(got, "tipos válidos: "+strings.Join(tiposColuna, ", ")) {
		t.Errorf("sem a lista de tipos: %s", got)
	}
}

func TestDicaDaVirgulaSobrando(t *testing.T) {
	const dica = "foi lido como nome de coluna do input; pode haver uma vírgula sobrando na linha 1"
	for _, s := range []string{
		"input a,\nend", "input a, end", "input a,\nsort a\nend", "input a,\nwidth 80\nend",
		"input a,\nleft margin 0\nend", "input a,\nbefore report\nend", "input a,\ndetail\n print a\nend",
	} {
		if got := erroDoScript(t, s, false); !strings.Contains(got, dica) {
			t.Errorf("%q sem a dica: %s", s, got)
		}
	}
	for _, s := range []string{
		"input a, width\ndetail\n print a +\nend", "input a, date\ndetail\n print date +\nend",
		"input a, detail [numeric]\nbanana", "input end", "input a, sort\n @", "input a, end, b\nbanana\nend",
	} {
		if got := erroDoScript(t, s, false); got == "" || strings.Contains(got, dica) {
			t.Errorf("%q: %q", s, got)
		}
	}
}

func TestScriptSemPalavras(t *testing.T) {
	for _, s := range []string{"", "  \n", "/* nada */\n"} {
		got := erroDoScript(t, s, false)
		if !strings.Contains(got, "o script não tem nenhuma palavra") || strings.Contains(got, "|") {
			t.Errorf("%q: %q", s, got)
		}
	}
}

func TestErroEmLinhaLonga(t *testing.T) {
	enche := func(n int) string { return strings.Repeat("a ", n)[:n] }
	for _, c := range []struct {
		nome, linha string
		col, tam    int
		trecho      string
	}{
		{"72 cabe inteira", enche(71) + "@", 71, 1, enche(71) + "@"},
		{"73 corta o começo", enche(72) + "@", 72, 1, "..." + enche(72)[1:] + "@"},
		{"no começo", "@" + enche(100), 0, 1, "@" + enche(100)[:71] + "..."},
		{"no meio", enche(100) + "@" + enche(100), 100, 1, "..." + enche(100)[65:] + "@" + enche(100)[:36] + "..."},
		{"maior que o trecho", "abc " + strings.Repeat("x", 90), 4, 90, "..." + strings.Repeat("x", 72) + "..."},
		{"CRLF", enche(71) + "@\r", 71, 1, enche(71) + "@"},
		{"tabs e acentos", "\tSão ç\t" + enche(40) + "@", 47, 1, "\tSão ç\t" + enche(40) + "@"},
		{"acentos cortados", strings.Repeat("ã\t", 60) + "@", 120, 1, "..." + strings.Repeat("ã\t", 60)[len("ã\t")*24+len("ã"):] + "@"},
	} {
		antes := FonteScript
		DefFonteScript(c.linha)
		got := strings.Split(erroNaPalavra(1, c.col, c.tam, "x").Error(), "\n")
		FonteScript = antes
		if len(got) != 3 || got[1] != "  1 | "+c.trecho {
			t.Errorf("%s: %q\nwant %q", c.nome, got, c.trecho)
			continue
		}
		impressa, marcador := []rune(got[1]), []rune(got[2])
		ini := strings.Index(got[2], "^")
		primeira := []rune(c.linha)[c.col]
		if ini < 0 || impressa[len([]rune(got[2][:ini]))] != primeira {
			t.Errorf("%s: o marcador não está sob %q:\n%s\n%s", c.nome, primeira, got[1], got[2])
		}
		for i, r := range marcador { // tabs nas mesmas posições
			if r != '^' && (r == '\t') != (impressa[i] == '\t') {
				t.Errorf("%s: tab desalinhado na posição %d", c.nome, i)
			}
		}
		if n := strings.Count(got[2], "^"); n < 1 || n > min(c.tam, 72) {
			t.Errorf("%s: %d marcadores", c.nome, n)
		}
	}
}
