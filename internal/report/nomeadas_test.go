package report

import (
	"strings"
	"testing"
)

// Expressões nomeadas (manual, "Named Expressions"): "<nome> expressão" no sort ou num print;
// depois de nomeado, o nome vale nas expressões seguintes como se fosse uma coluna.

const entradaNomeada = "input\n  item [string 8],\n  price [amount 10],\n  discount [amount 10]\n"

func espacosUnicos(linhas []string) []string {
	out := make([]string, len(linhas))
	for i, l := range linhas {
		out[i] = strings.Join(strings.Fields(l), " ")
	}
	return out
}

func TestNomeadaExemploDoManual(t *testing.T) {
	// sort item, <cost> price - discount / print item, <amount> cost + (cost * 0.06) ...
	script := entradaNomeada + `sort item, <cost> price - discount
detail
  print item, <amount> cost + (cost * 0.06) in col 30
  print 'CHECK AMOUNT LESS PRICE IS:', amount - price
end
`
	got, err := rodar(t, script, []string{"pen|10.00|1.50", "book|20.00|5.00"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"book 15.90", "CHECK AMOUNT LESS PRICE IS: -4.10", "pen 9.01", "CHECK AMOUNT LESS PRICE IS: -0.99"}
	if g := espacosUnicos(got); !igual(g, want) {
		t.Errorf("got %q, want %q", g, want)
	}
}

func TestNomeadaCasos(t *testing.T) {
	casos := []struct {
		nome, corpo string
		dados       []string
		want        []string
	}{
		{"nome do sort usado no print", "sort item, <cost> price - discount\ndetail\n  print item, cost in col 20\n",
			[]string{"pen|10.00|1.50"}, []string{"pen 8.50"}},
		{"nome do print usado em outro print", "detail\n  print <liq> price - discount\n  print liq * 2\n",
			[]string{"a|10.00|4.00"}, []string{"6.00", "12.00"}},
		{"apelido usado em condição lógica", "detail\n  print <caro> price > 15\n  if caro then print 'caro' else print 'barato'\n",
			[]string{"a|10.00|0", "b|20.00|0"}, []string{"0", "barato", "1", "caro"}},
		{"apelido em agregado", "sort item, <cost> price - discount\nafter report\n  print total(cost), count(* where cost > 10)\n",
			[]string{"a|10.00|1.00", "b|20.00|2.00", "c|30.00|5.00"}, []string{"52.00 2"}},
		{"apelido encadeado", "sort item, <a> price - discount, <b> a * 2\ndetail\n  print b\n",
			[]string{"x|10.00|4.00"}, []string{"12.00"}},
		{"mesmo apelido e mesma expressão repetidos", "detail\n  print <liq> price - discount\n  print <liq> price - discount\n",
			[]string{"a|10.00|4.00"}, []string{"6.00", "6.00"}},
		{"apelido trivial <g> g continua valendo", "sort <item> item\nbefore item\n  print 'abre'\ndetail\n  print item\n",
			[]string{"a|1|1"}, []string{"abre", "a"}},
	}
	for _, c := range casos {
		got, err := rodar(t, entradaNomeada+c.corpo+"end\n", c.dados)
		if err != nil {
			t.Errorf("%s: %v", c.nome, err)
			continue
		}
		if g := espacosUnicos(got); !igual(g, c.want) {
			t.Errorf("%s: got %q, want %q", c.nome, g, c.want)
		}
	}
}

func TestNomeadaErros(t *testing.T) {
	casos := []struct{ nome, corpo, quer string }{
		{"mesmo nome, outra expressão", "detail\n  print <x> price\n  print <x> discount\n", "já foi dado a outra expressão"},
		{"nome de coluna", "detail\n  print <price> discount\n", "já é uma coluna do input"},
		{"nome de variável", "detail\n  set v to 1\n  print <v> price\n", "já é uma variável de 'set'"},
		{"recursivo", "detail\n  print <a> b\n  print <b> a\n  print a\n", "refere-se a si mesma"},
		{"sem nome entre <>", "detail\n  print <> price\n", "expressão inválida"},
		{"sem >", "detail\n  print <x price\n", "esperado"},
	}
	for _, c := range casos {
		_, err := rodar(t, entradaNomeada+c.corpo+"end\n", []string{"a|1|1"})
		if err == nil || !strings.Contains(err.Error(), c.quer) {
			t.Errorf("%s: err = %v, quer %q", c.nome, err, c.quer)
		}
	}
}

func TestNomeadaTipoLogicoVsAritmetico(t *testing.T) {
	// o apelido tem o tipo da expressão que nomeia: aritmético não serve de condição
	_, err := rodar(t, entradaNomeada+"detail\n  print <liq> price - discount\n  if liq then print 'x'\n"+"end\n", []string{"a|1|1"})
	if err == nil || !strings.Contains(err.Error(), "lógica") {
		t.Errorf("apelido aritmético no if: err = %v", err)
	}
	// e um apelido lógico serve
	if _, err := rodar(t, entradaNomeada+"detail\n  print <ok> price > discount\n  if ok and not (discount > 5) then print 'x'\n"+"end\n", []string{"a|1|1"}); err != nil {
		t.Errorf("apelido lógico: %v", err)
	}
}
