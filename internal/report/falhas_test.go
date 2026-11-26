package report

import (
	"strings"
	"testing"
	"time"
)

// Chamadas com argumentos errados são erro de script, apontando o nome da função.
func TestNumeroDeArgumentos(t *testing.T) {
	for _, c := range []struct{ expr, erro string }{
		{"«dow»()", "a função dow() espera 1 argumento e recebeu 0"},
		{"«dow»(d, 1)", "recebeu 2"},
		{"«dow»(*)", "não aceita '*'"},
		{"«mdy»(d)", "a função mdy() espera 2 argumentos e recebeu 1"},
		{"«mdy»()", "recebeu 0"},
		{"«mdy»(d, 1, 2)", "recebeu 3"},
		{"«index»()", "pelo menos 1 argumento"},
		{"«count»()", "a função count() espera 1 argumento e recebeu 0"},
		{"«total»()", "recebeu 0"},
		{"«avg»(d, d)", "recebeu 2"},
		{"«min»()", "recebeu 0"},
		{"«max»()", "recebeu 0"},
	} {
		conferePalavra(t, "input d [date]\nafter report\n print "+c.expr+"\nend", c.erro, false)
	}
	for _, e := range []string{"count(*)", "count(* where d is null)", "total(d)", "index(0)", "index(1, d, d)", "dow(d)", "mdy(d, 0)"} {
		if got := erroDoScript(t, "input d [date]\nafter report\n print "+e+"\nend", false); got != "" {
			t.Errorf("%s: %s", e, got)
		}
	}
}

// Valores de tipo errado param o relatório com uma mensagem, sem derrubar o programa.
func TestTiposErradosNaoDerrubam(t *testing.T) {
	const entrada = "input n [numeric], d [date], s\n"
	dados := []string{"2|2026-01-01|ab", "1|2026-02-01|cd"}
	for _, c := range []struct{ corpo, erro string }{
		{"detail\n print dow(n)", "a função dow() espera uma data"},
		{"detail\n print mdy(s, 0)", "a função mdy() espera uma data"},
		{"detail\n if d = 1 then print n", "comparar uma data com um número"},
		{"detail\n if n < d then print n", "comparar uma data com um número"},
		{"sort dow(n)\ndetail\n print n", "no sort: a função dow() espera uma data"},
		{"sort index(n - 1, d, n)\ndetail\n print n", "no sort: não é possível comparar uma data"},
		{"detail\n print s using '0'", "máscara de texto inválida"},
	} {
		fim := make(chan error, 1)
		go func() {
			_, err := rodarRaw(t, entrada+c.corpo+"\nend\n", dados)
			fim <- err
		}()
		select {
		case err := <-fim:
			if err == nil || !strings.Contains(err.Error(), c.erro) {
				t.Errorf("%q: %v", c.corpo, err)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("%q: não terminou", c.corpo)
		}
	}
}
