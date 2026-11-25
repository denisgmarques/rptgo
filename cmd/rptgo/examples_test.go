package main

import (
	"regexp"
	"strings"
	"testing"
)

const dirExemplos = "../../examples/"

func TestExemploVendasRegiao(t *testing.T) {
	code, out, errb := executa(t, "", dirExemplos+"relatorio-vendas-regiao/relatorio.rp", dirExemplos+"relatorio-vendas-regiao/dados.txt")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{"Total Leste", "Total Norte", "Total Sul", "TOTAL GERAL", "7,192.00", "3,661.35"} {
		if !strings.Contains(out, want) {
			t.Errorf("saída sem %q:\n%s", want, out)
		}
	}
}

func TestExemploEstoqueCategoria(t *testing.T) {
	script := dirExemplos + "relatorio-estoque-categoria/relatorio.rp"
	code, out, errb := executa(t, "", script, dirExemplos+"relatorio-estoque-categoria/dados.txt")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{"Produtos:", "Preco medio:", "Itens na categoria:", "TOTAL EM ESTOQUE", "9,149.00", "16.75"} {
		if !strings.Contains(out, want) {
			t.Errorf("saída sem %q:\n%s", want, out)
		}
	}
	// o script declara 2+ nomes no sort, cada um com before e after, e usa count/total/avg
	src := lerArquivo(t, script)
	sort := regexp.MustCompile(`(?m)^sort (.+)$`).FindStringSubmatch(src)
	if sort == nil {
		t.Fatal("script sem sort")
	}
	nomes := regexp.MustCompile(`<(\w+)>`).FindAllStringSubmatch(sort[1], -1)
	if len(nomes) < 2 {
		t.Fatalf("esperados 2+ nomes no sort, vieram %d", len(nomes))
	}
	for _, n := range nomes {
		for _, ev := range []string{"before ", "after "} {
			if !regexp.MustCompile(`(?m)^` + ev + n[1] + `$`).MatchString(src) {
				t.Errorf("sem seção '%s%s'", ev, n[1])
			}
		}
	}
	for _, f := range []string{"count(", "total(", "avg("} {
		if !strings.Contains(src, f) {
			t.Errorf("script não usa %s", f)
		}
	}
}
