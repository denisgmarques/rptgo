package report

import (
	"strings"
	"testing"
)

// rodar executa um script sobre registros separados por '|' e devolve as linhas não vazias
// da saída (sem espaços nas pontas).
func rodar(t *testing.T, script string, dados []string) ([]string, error) {
	t.Helper()
	toks, err := Tokenize(script, false)
	if err != nil {
		return nil, err
	}
	s, err := Parse(toks)
	if err != nil {
		return nil, err
	}
	regs := make([][]string, len(dados))
	for i, d := range dados {
		regs[i] = strings.Split(d, "|")
	}
	it, err := NovoInterp(s, regs, false)
	if err != nil {
		return nil, err
	}
	texto, err := it.Run()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, l := range strings.Split(texto, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out, nil
}

func igual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBoolNormalizacao(t *testing.T) {
	casos := []struct{ entrada, saida string }{
		{"0", "NO"}, {"0.0", "NO"}, {"0.0e6", "NO"}, {"FALSE", "NO"}, {"NO", "NO"},
		{"false", "NO"}, {"no", "NO"}, {"Nope", "NO"}, {"fx", "NO"}, {"N", "NO"}, {"f", "NO"},
		{"1", "YES"}, {"999", "YES"}, {"9.6", "YES"}, {"1.0", "YES"}, {"1.0e6", "YES"},
		{"-3", "YES"}, {"TRUE", "YES"}, {"YES", "YES"}, {"true", "YES"}, {"yes", "YES"},
		{"Yep", "YES"}, {"tx", "YES"}, {"Y", "YES"}, {"t", "YES"},
		{"5xx", "***"}, {"xxx", "***"}, {"0.x", "***"}, {"*", "***"},
		{"", "***"}, {"   ", "***"},
	}
	const script = "input\n  id [numeric],\n  f [bool]\ndetail\n  print f\nend\n"
	for _, c := range casos {
		got, err := rodar(t, script, []string{"1|" + c.entrada})
		if err != nil {
			t.Errorf("entrada %q: erro inesperado: %v", c.entrada, err)
			continue
		}
		if !igual(got, []string{c.saida}) {
			t.Errorf("entrada %q: got %q, want %q", c.entrada, got, c.saida)
		}
	}
}

func TestBoolIsNullEIf(t *testing.T) {
	const script = `input
  id [numeric],
  f [bool]
detail
  if f is null then print 'nulo'
  else if f then print 'sim'
  else print 'nao'
end
`
	casos := []struct{ f, want string }{{"1", "sim"}, {"NO", "nao"}, {"xxx", "nulo"}, {"", "nulo"}}
	for _, c := range casos {
		got, err := rodar(t, script, []string{"1|" + c.f})
		if err != nil || !igual(got, []string{c.want}) {
			t.Errorf("f=%q: got %q err=%v, want %q", c.f, got, err, c.want)
		}
	}
}

func TestLogicaEmAritmetica(t *testing.T) {
	casos := []struct{ expr, dado, want string }{
		{"(n > 1) + 1", "5|1", "2"},
		{"(n > 1) + 1", "0|1", "1"},
		{"(n > 1) * 10", "5|1", "10"},
		{"f + 1", "5|1", "2"},
		{"f + 1", "5|0", "1"},
		{"n + (f and n > 1)", "5|YES", "6"},
		{"n + (f and n > 1)", "5|NO", "5"},
		{"f is null", "5|xxx", "1"},
		{"f is not null", "5|xxx", "0"},
	}
	for _, c := range casos {
		script := "input\n  n [numeric],\n  f [bool]\ndetail\n  print " + c.expr + "\nend\n"
		got, err := rodar(t, script, []string{c.dado})
		if err != nil || !igual(got, []string{c.want}) {
			t.Errorf("%s com %q: got %q err=%v, want %q", c.expr, c.dado, got, err, c.want)
		}
	}
}

func TestCondicaoAritmeticaEhErro(t *testing.T) {
	casos := []struct {
		nome, corpo string
		erro        bool
	}{
		{"coluna numérica no if", "if n then print 'x'", true},
		{"conta no if", "if n + 1 then print 'x'", true},
		{"constante no if", "if 1 then print 'x'", true},
		{"texto no if", "if s then print 'x'", true},
		{"and com aritmética", "if n > 1 and n then print 'x'", true},
		{"or com aritmética", "if n or f then print 'x'", true},
		{"not aritmético", "if not n then print 'x'", true},
		{"1 and 1 and not 0", "if 1 and 1 and not 0 then print 'x'", true},
		{"aritmética dentro de lógica no if", "if (n + 1) and f then print 'x'", true},
		{"comparação", "if n > 1 then print 'x'", false},
		{"coluna bool", "if f then print 'x'", false},
		{"not bool", "if not f then print 'x'", false},
		{"and/or de lógicas", "if f and n > 1 or s = 'a' then print 'x'", false},
		{"is null", "if n is not null then print 'x'", false},
		{"variável de set é aceita", "set v to n > 1\n  if v then print 'x'", false},
	}
	for _, c := range casos {
		script := "input\n  n [numeric],\n  s [string 5],\n  f [bool]\ndetail\n  " + c.corpo + "\nend\n"
		_, err := rodar(t, script, []string{"5|a|1"})
		if c.erro {
			if err == nil || !strings.Contains(err.Error(), "lógica") {
				t.Errorf("%s: esperava erro mencionando 'lógica', veio %v", c.nome, err)
			}
		} else if err != nil {
			t.Errorf("%s: erro inesperado: %v", c.nome, err)
		}
	}
}

func TestWhereExigeLogico(t *testing.T) {
	base := "input\n  n [numeric],\n  f [bool]\nafter report\n  print count(* where %s)\nend\n"
	_, err := rodar(t, strings.Replace(base, "%s", "n", 1), []string{"5|1"})
	if err == nil || !strings.Contains(err.Error(), "where") {
		t.Errorf("where aritmético deveria falhar, veio %v", err)
	}
	for _, w := range []string{"f", "n > 1", "f and n > 1"} {
		got, err := rodar(t, strings.Replace(base, "%s", w, 1), []string{"5|1", "0|0", "9|yes"})
		want := map[string]string{"f": "2", "n > 1": "2", "f and n > 1": "2"}[w]
		if err != nil || !igual(got, []string{want}) {
			t.Errorf("where %s: got %q err=%v, want %s", w, got, err, want)
		}
	}
}

func TestPrecedenciaNotAndOr(t *testing.T) {
	// a, b, c são colunas bool; a saída é 1/0 da expressão lógica.
	casos := []struct{ expr, dado, want string }{
		{"a or b and c", "1|0|0", "1"}, // a or (b and c); (a or b) and c daria 0
		{"a or b and c", "0|1|0", "0"}, // b and c = 0
		{"not a and b", "0|0|0", "0"},  // (not a) and b; not (a and b) daria 1
		{"not a and b", "0|1|0", "1"},
		{"not a or b", "1|1|0", "1"},       // (not a) or b
		{"not a or b", "1|0|0", "0"},       // not (a or b) daria 0 aqui também, mas a=1,b=0 => 0
		{"not a or b and c", "1|1|0", "0"}, // (not a) or (b and c)
		{"not a or b and c", "0|0|0", "1"},
		{"a and b or c", "0|0|1", "1"}, // (a and b) or c; a and (b or c) daria 0
		{"not not a", "1|0|0", "1"},
	}
	for _, c := range casos {
		script := "input\n  a [bool],\n  b [bool],\n  c [bool]\ndetail\n  print " + c.expr + "\nend\n"
		got, err := rodar(t, script, []string{c.dado})
		if err != nil || !igual(got, []string{c.want}) {
			t.Errorf("%s com %s: got %q err=%v, want %s", c.expr, c.dado, got, err, c.want)
		}
	}
}

func TestModuloAritmetico(t *testing.T) {
	casos := []struct{ expr, dado, want string }{
		{"n % 3", "10|1", "1"},
		{"n % 3 + 1", "10|1", "2"},
		{"2 + n % 3 * 2", "10|1", "4"}, // % e * antes de +, da esquerda para a direita
		{"(n % 2 = 0) + 1", "10|1", "2"},
	}
	for _, c := range casos {
		script := "input\n  n [numeric],\n  f [bool]\ndetail\n  print " + c.expr + "\nend\n"
		got, err := rodar(t, script, []string{c.dado})
		if err != nil || !igual(got, []string{c.want}) {
			t.Errorf("%s com %q: got %q err=%v, want %q", c.expr, c.dado, got, err, c.want)
		}
	}
	if _, err := rodar(t, "input\n  n [numeric]\ndetail\n  print n % 0\nend\n", []string{"5"}); err == nil {
		t.Error("n %% 0 deveria dar erro de divisão por zero")
	}
}

func TestDivisaoInteiraTrunca(t *testing.T) {
	casos := []struct{ expr, want string }{
		{"7 / 2", "3"},
		{"-7 / 2", "-3"},
		{"7 / 2 * 2", "6"},
		{"7 / 2.0", "3.50"},
		{"n / 2", "3"}, // n = 7, coluna numeric
		{"p / 2", "3.50"},
	}
	for _, c := range casos {
		script := "input\n  n [numeric],\n  p [float]\ndetail\n  print " + c.expr + "\nend\n"
		got, err := rodar(t, script, []string{"7|7"})
		if err != nil || !igual(got, []string{c.want}) {
			t.Errorf("%s: got %q err=%v, want %q", c.expr, got, err, c.want)
		}
	}
}

func TestWhereEmTodosOsAgregados(t *testing.T) {
	dados := []string{"70", "80", "90"}
	casos := []struct{ expr, want string }{
		{"count(* where p > 77.00)", "2"},
		{"count(p where p > 77.00)", "2"},
		{"avg(p where p > 77.00)", "85.00"},
		{"total(p where p > 77.00)", "170.00"},
		{"min(p where p > 77.00)", "80.00"},
		{"max(p where p < 85.00)", "80.00"},
		{"total(p where p)", "erro"}, // where aritmético continua sendo erro
	}
	for _, c := range casos {
		script := "input\n  p [float]\nafter report\n  print " + c.expr + "\nend\n"
		got, err := rodar(t, script, dados)
		if c.want == "erro" {
			if err == nil {
				t.Errorf("%s: esperava erro", c.expr)
			}
			continue
		}
		if err != nil || !igual(got, []string{c.want}) {
			t.Errorf("%s: got %q err=%v, want %q", c.expr, got, err, c.want)
		}
	}
}
