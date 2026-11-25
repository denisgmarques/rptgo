package report

import (
	"fmt"
	"math/rand"
	"strings"
)

// Gerador sintético e reproduzível (semente fixa) de registros com o esquema do exemplo
// examples/relatorio-vendas-regiao (regiao, vendedor, valor). Fica num _test.go de
// propósito: é usado só pelos benchmarks/testes e não entra no binário.
const (
	semente200k = 20261124
	total200k   = 200000
)

const scriptVendas200k = `input
   regiao    [string 10],
   vendedor  [string 14],
   valor     [amount 10]
sort regiao
detail
   print regiao in col 1, vendedor in col 14, valor in col 30
end
`

var regioes200k = []string{"Norte", "Sul", "Leste", "Oeste", "Centro"}

// geraRegistros200k devolve n registros já separados em campos, sempre os mesmos para a
// mesma semente.
func geraRegistros200k(n int) [][]string {
	r := rand.New(rand.NewSource(semente200k))
	out := make([][]string, n)
	for i := range out {
		out[i] = []string{
			regioes200k[r.Intn(len(regioes200k))],
			fmt.Sprintf("Vendedor %04d", r.Intn(5000)),
			fmt.Sprintf("%d.%02d", r.Intn(100000), r.Intn(100)),
		}
	}
	return out
}

func scriptVendas200kParseado() (*Script, error) {
	toks, err := Tokenize(strings.TrimSpace(scriptVendas200k)+"\n", false)
	if err != nil {
		return nil, err
	}
	return Parse(toks)
}
