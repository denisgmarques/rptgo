package report

import (
	"strings"
	"testing"
)

// Expressões nomeadas: "<nome> expressão" no sort ou num print;
// depois de nomeado, o nome vale nas expressões seguintes como se fosse uma coluna.

const entradaNomeada = "input\n  produto [string 8],\n  preco [amount 10],\n  desconto [amount 10]\n"

func espacosUnicos(linhas []string) []string {
	out := make([]string, len(linhas))
	for i, l := range linhas {
		out[i] = strings.Join(strings.Fields(l), " ")
	}
	return out
}

func TestNomeadaSortEPrint(t *testing.T) {
	// sort produto, <custo> preco - desconto / print produto, <final> custo + (custo * 0.10) ...
	script := entradaNomeada + `sort produto, <custo> preco - desconto
detail
  print produto, <final> custo + (custo * 0.10) in col 30
  print 'DIFERENCA:', preco - final
end
`
	got, err := rodar(t, script, []string{"lapis|10.00|1.50", "caderno|20.00|5.00"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"caderno 16.50", "DIFERENCA: 3.50", "lapis 9.35", "DIFERENCA: 0.65"}
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
		{"nome do sort usado no print", "sort produto, <custo> preco - desconto\ndetail\n  print produto, custo in col 20\n",
			[]string{"pen|10.00|1.50"}, []string{"pen 8.50"}},
		{"nome do print usado em outro print", "detail\n  print <liq> preco - desconto\n  print liq * 2\n",
			[]string{"a|10.00|4.00"}, []string{"6.00", "12.00"}},
		{"apelido usado em condição lógica", "detail\n  print <caro> preco > 15\n  if caro then print 'caro' else print 'barato'\n",
			[]string{"a|10.00|0", "b|20.00|0"}, []string{"0", "barato", "1", "caro"}},
		{"apelido em agregado", "sort produto, <custo> preco - desconto\nafter report\n  print total(custo), count(* where custo > 10)\n",
			[]string{"a|10.00|1.00", "b|20.00|2.00", "c|30.00|5.00"}, []string{"52.00 2"}},
		{"apelido encadeado", "sort produto, <a> preco - desconto, <b> a * 2\ndetail\n  print b\n",
			[]string{"x|10.00|4.00"}, []string{"12.00"}},
		{"mesmo apelido e mesma expressão repetidos", "detail\n  print <liq> preco - desconto\n  print <liq> preco - desconto\n",
			[]string{"a|10.00|4.00"}, []string{"6.00", "6.00"}},
		{"apelido trivial <g> g continua valendo", "sort <produto> produto\nbefore produto\n  print 'abre'\ndetail\n  print produto\n",
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
		{"mesmo nome, outra expressão", "detail\n  print <x> preco\n  print <x> desconto\n", "já foi dado a outra expressão"},
		{"nome de coluna", "detail\n  print <preco> desconto\n", "já é uma coluna do input"},
		{"nome de variável", "detail\n  set v to 1\n  print <v> preco\n", "já é uma variável de 'set'"},
		{"recursivo", "detail\n  print <a> b\n  print <b> a\n  print a\n", "refere-se a si mesma"},
		{"sem nome entre <>", "detail\n  print <> preco\n", "expressão inválida"},
		{"sem >", "detail\n  print <x preco\n", "esperado"},
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
	_, err := rodar(t, entradaNomeada+"detail\n  print <liq> preco - desconto\n  if liq then print 'x'\n"+"end\n", []string{"a|1|1"})
	if err == nil || !strings.Contains(err.Error(), "lógica") {
		t.Errorf("apelido aritmético no if: err = %v", err)
	}
	// e um apelido lógico serve
	if _, err := rodar(t, entradaNomeada+"detail\n  print <ok> preco > desconto\n  if ok and not (desconto > 5) then print 'x'\n"+"end\n", []string{"a|1|1"}); err != nil {
		t.Errorf("apelido lógico: %v", err)
	}
}
