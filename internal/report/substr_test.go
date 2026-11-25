package report

import (
	"strings"
	"testing"
)

// Casos da tabela de exemplos do manual (seção "The Substring Operator"). Os resultados são
// impressos entre colchetes para deixar visíveis os brancos e o texto vazio.
func TestSubstrTabelaDoManual(t *testing.T) {
	casos := []struct{ expr, want string }{
		{"'AEIOU' substr [1-3]", "[AEI]"},
		{"'AEIOU' substr [3-5]", "[IOU]"},
		{"'AEIOU' substr [3-9]", "[IOU]"}, // fim além do tamanho é cortado
		{"'AEIOU' substr [6-99]", "[]"},   // início além do tamanho: texto vazio
		{"'AEIOU' substr [2-2]", "[E]"},   // um caractere
		{"'AEIOU' substr [1-6]", "[AEIOU]"},
		{"'AEIOU ' substr [6-107]", "[ ]"}, // o branco final é um caractere
	}
	for _, c := range casos {
		script := "input\n  n [numeric]\ndetail\n  print '[' + (" + c.expr + ") + ']'\nend\n"
		got, err := rodarRaw(t, script, []string{"1"})
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.expr, got, c.want)
		}
	}
}

// rodarRaw devolve a primeira linha da saída sem cortar os brancos do fim.
func rodarRaw(t *testing.T, script string, dados []string) (string, error) {
	t.Helper()
	toks, err := Tokenize(script, false)
	if err != nil {
		return "", err
	}
	s, err := Parse(toks)
	if err != nil {
		return "", err
	}
	regs := make([][]string, len(dados))
	for i, d := range dados {
		regs[i] = strings.Split(d, "|")
	}
	it, err := NovoInterp(s, regs, false)
	if err != nil {
		return "", err
	}
	texto, err := it.Run()
	if err != nil {
		return "", err
	}
	for _, l := range strings.Split(texto, "\n") {
		if strings.TrimSpace(l) != "" {
			return strings.TrimPrefix(l, strings.Repeat(" ", s.Left)), nil
		}
	}
	return "", nil
}

func TestSubstrEmColunasEVariaveis(t *testing.T) {
	const script = `input
  cod   [string 10],
  ativo [string 5]
detail
  set pref to cod substr [1-2]
  print pref in col 1, cod substr [3-4] in col 6, ativo substr [1-1] in col 12
end
`
	got, err := rodar(t, script, []string{"ABCDEFG|sim", "XY|", "Z|nao"})
	if err != nil {
		t.Fatal(err)
	}
	// sem sort a ordem de entrada é mantida; fora do tamanho vira texto vazio
	want := []string{"AB   CD    s", "XY", "Z          n"}
	if !igual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSubstrPrecedenciaEComparacao(t *testing.T) {
	// substr liga ao operando mais próximo e pode ser comparado
	casos := []struct{ expr, want string }{
		{"if cod substr [1-1] = 'A' then print 'sim' else print 'nao'", "sim"},
		{"if cod substr [2-3] = 'BC' and cod substr [4-4] = 'D' then print 'sim' else print 'nao'", "sim"},
		{"if (cod substr [1-1]) + (cod substr [4-4]) = 'AD' then print 'sim' else print 'nao'", "sim"},
		{"if cod substr [9-9] = '' then print 'vazio' else print 'cheio'", "vazio"},
		{"if cod substr [1-2] substr [2-2] = 'B' then print 'sim' else print 'nao'", "sim"}, // encadeado
	}
	for _, c := range casos {
		script := "input\n  cod [string 10]\ndetail\n  " + c.expr + "\nend\n"
		got, err := rodar(t, script, []string{"ABCD"})
		if err != nil || !igual(got, []string{c.want}) {
			t.Errorf("%s: got %q err=%v, want %q", c.expr, got, err, c.want)
		}
	}
}

func TestSubstrErros(t *testing.T) {
	casos := []struct{ nome, script, quer string }{
		{"início zero", "input\n  c [string 5]\ndetail\n  print c substr [0-2]\nend\n", "inválido"},
		{"fim menor que início", "input\n  c [string 5]\ndetail\n  print c substr [3-2]\nend\n", "inválido"},
		{"sem colchetes", "input\n  c [string 5]\ndetail\n  print c substr 1\nend\n", ""},
		{"valor não inteiro", "input\n  c [string 5]\ndetail\n  print c substr [1.5-2]\nend\n", "inteiros"},
		{"em coluna numérica", "input\n  n [numeric]\ndetail\n  print n substr [1-1]\nend\n", "só pode ser usado com texto"},
	}
	for _, c := range casos {
		_, err := rodar(t, c.script, []string{"123"})
		if err == nil || !strings.Contains(err.Error(), c.quer) {
			t.Errorf("%s: err = %v, quer %q", c.nome, err, c.quer)
		}
	}
}

func TestSubstrNulo(t *testing.T) {
	got, err := rodar(t, "input\n  c [string 5]\ndetail\n  if c substr [1-1] is null then print 'nulo' else print 'ok'\nend\n", []string{""})
	if err != nil || !igual(got, []string{"nulo"}) {
		t.Errorf("got %q err=%v", got, err)
	}
}

// Concatenação indireta /+ (manual, "The Indirect Concatenation Operator").
func TestConcatenacaoIndireta(t *testing.T) {
	casos := []struct{ expr, want string }{
		{"'ALPHA   ' /+ 'BET   '", "[ALPHABET   ]"},
		{"'ALPHA   ' /+ '' /+ 'BET'", "[ALPHABET]"},
		{"'ALPHA   ' + 'BET'", "[ALPHA   BET]"}, // + direto mantém os brancos
		{"'A' /+ 'B' /+ 'C'", "[ABC]"},
	}
	for _, c := range casos {
		script := "input\n  n [numeric]\ndetail\n  print '[' + (" + c.expr + ") + ']'\nend\n"
		got, err := rodarRaw(t, script, []string{"1"})
		if err != nil || got != c.want {
			t.Errorf("%s: got %q err=%v, want %q", c.expr, got, err, c.want)
		}
	}
	if _, err := rodar(t, "input\n  n [numeric]\ndetail\n  print n /+ 'x'\nend\n", []string{"1"}); err == nil {
		t.Error("/+ com número deveria dar erro")
	}
	// divisão seguida de mais unário continua funcionando com espaço
	got, err := rodar(t, "input\n  n [numeric]\ndetail\n  print 10 / +5\nend\n", []string{"1"})
	if err != nil || !igual(got, []string{"2"}) {
		t.Errorf("10 / +5: %q %v", got, err)
	}
}
