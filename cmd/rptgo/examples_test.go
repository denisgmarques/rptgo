package main

import (
	"encoding/json"
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

func TestExemploCadastroClientes(t *testing.T) {
	script := dirExemplos + "cadastro-clientes/relatorio.rp"
	dados := dirExemplos + "cadastro-clientes/dados.txt"
	code, out, errb := executa(t, "", script, dados)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{"YES", "NO", "***", "Clientes ativos:"} {
		if !strings.Contains(out, want) {
			t.Errorf("texto sem %q:\n%s", want, out)
		}
	}
	code, out, errb = executa(t, "", "--format", "json", script, dados)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var objs []map[string]any
	if err := json.Unmarshal([]byte(out), &objs); err != nil {
		t.Fatalf("JSON inválido: %v\n%s", err, out)
	}
	if len(objs) != 5 || objs[0]["ativo"] != true || objs[1]["ativo"] != false || objs[3]["ativo"] != nil ||
		objs[2]["nascimento"] != nil || objs[0]["nascimento"] != "1985-03-15" || objs[0]["cadastro"] != "2020-01-15" {
		t.Errorf("objetos inesperados: %v", objs)
	}
}

// TestExemplosDeNegocio roda os exemplos em inglês nos três formatos: o texto traz os
// totais esperados e o JSON/CSV tem um registro por linha de dados.
func TestExemplosDeNegocio(t *testing.T) {
	casos := []struct {
		dir     string
		linhas  int
		texto   []string
		jsonKey string
	}{
		{"payroll-by-department", 11, []string{"PAYROLL BY DEPARTMENT", "Headcount:", "25,200.50", "ALL DEPARTMENTS", "51,301.50"}, "active"},
		{"invoice-aging", 8, []string{"OVERDUE 30+", "current", "Open balance:", "TOTAL OPEN", "39,291.50"}, "paid"},
		{"purchase-orders", 8, []string{"BULK", "Shipping:", "RUSH ORDER", "Customer total:", "ALL ORDERS"}, "rush"},
	}
	for _, c := range casos {
		t.Run(c.dir, func(t *testing.T) {
			script := dirExemplos + c.dir + "/report.rp"
			dados := dirExemplos + c.dir + "/data.txt"
			code, out, errb := executa(t, "", script, dados)
			if code != 0 {
				t.Fatalf("exit %d: %s", code, errb)
			}
			for _, want := range c.texto {
				if !strings.Contains(out, want) {
					t.Errorf("texto sem %q:\n%s", want, out)
				}
			}
			code, out, errb = executa(t, "", "--format", "json", script, dados)
			if code != 0 {
				t.Fatalf("json: exit %d: %s", code, errb)
			}
			var objs []map[string]any
			if err := json.Unmarshal([]byte(out), &objs); err != nil {
				t.Fatalf("JSON inválido: %v\n%s", err, out)
			}
			if len(objs) != c.linhas {
				t.Errorf("%d objetos, esperados %d", len(objs), c.linhas)
			}
			if _, ok := objs[0][c.jsonKey].(bool); !ok {
				t.Errorf("%s deveria ser boolean JSON: %v", c.jsonKey, objs[0])
			}
			code, out, errb = executa(t, "", "--format", "csv", script, dados)
			if code != 0 {
				t.Fatalf("csv: exit %d: %s", code, errb)
			}
			if n := len(strings.Split(strings.TrimRight(out, "\n"), "\n")); n != c.linhas+1 {
				t.Errorf("csv com %d linhas, esperadas %d", n, c.linhas+1)
			}
		})
	}
}
