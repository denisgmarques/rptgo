package report

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRowCompactoSaidaIdentica: com Row = []Value a saída de texto continua idêntica,
// byte a byte, à referência golden gravada antes da troca (RF-07).
func TestRowCompactoSaidaIdentica(t *testing.T) {
	for _, c := range []struct {
		nome, ref string
		formFeed  bool
	}{
		{"vendas", "vendas.txt", false},
		{"estoque", "estoque.txt", false},
		{"misto", "misto.txt", false},
		{"misto", "misto_formfeed.txt", true},
	} {
		want, err := os.ReadFile(filepath.Join("testdata", "golden", c.ref))
		if err != nil {
			t.Fatal(err)
		}
		if got := executaGolden(t, c.nome, c.formFeed); got != string(want) {
			t.Errorf("%s: saída mudou com o Row compacto", c.ref)
		}
	}
}

func TestRowPosicionalEIndice(t *testing.T) {
	const script = "input\n  a [numeric],\n  b [string 5],\n  c [bool]\ndetail\n  print b\nend\n"
	toks, _ := Tokenize(script, false)
	s, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	it, err := NovoInterp(s, [][]string{{"7", "xy", "yes"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	r := it.rows[0]
	if len(r) != 3 || r[0] != int64(7) || r[1] != "xy" || r[2] != true {
		t.Errorf("registro inesperado: %#v", r)
	}
	if it.colIdx["a"] != 0 || it.colIdx["b"] != 1 || it.colIdx["c"] != 2 {
		t.Errorf("colIdx inesperado: %v", it.colIdx)
	}
}

func TestLinhasOrdenadasSemEventos(t *testing.T) {
	const script = `input
  g [string 3],
  h [numeric],
  v [numeric]
sort <g> g, <h> h desc
before g
  set tocou to 1
after g
  set tocou to 2
before h
  set tocou to 3
after h
  set tocou to 4
detail
  set tocou to 5
  print v
end
`
	toks, err := Tokenize(script, false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	dados := [][]string{{"b", "1", "10"}, {"a", "1", "20"}, {"b", "2", "30"}, {"a", "3", "40"}, {"b", "2", "50"}}
	it, err := NovoInterp(s, dados, false)
	if err != nil {
		t.Fatal(err)
	}
	linhas := it.LinhasOrdenadas()
	var got []int64
	for _, r := range linhas {
		got = append(got, r[2].(int64))
	}
	// g crescente; dentro de cada g, h decrescente (estável para empates: 30 antes de 50)
	want := []int64{40, 20, 30, 50, 10}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if len(it.vars) != 0 {
		t.Errorf("variáveis alteradas por seções de evento: %v", it.vars)
	}
	if it.pageno != 0 {
		t.Errorf("pageno = %d, esperado 0", it.pageno)
	}
}
