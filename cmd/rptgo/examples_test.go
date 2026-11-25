package main

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

const dirExemplos = "../../examples/"

func TestExemploVendasRegiao(t *testing.T) {
	dir := dirExemplos + "relatorio-vendas-regiao/"
	code, out, errb := executa(t, "", dir+"relatorio.rp", dir+"dados.txt")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{"REGIAO Leste", "REGIAO Norte", "REGIAO Sul", "Subtotal do mes",
		"TOTAL Leste", "5,585.30", "TOTAL Norte", "8,031.95", "TOTAL Sul", "8,852.00", "TOTAL GERAL", "22,469.25", "21  vendas"} {
		if !strings.Contains(out, want) {
			t.Errorf("saída sem %q:\n%s", want, out)
		}
	}
	// dentro de cada região os meses vêm em ordem cronológica, inclusive na virada de ano, e as
	// vendas de um mês ficam juntas sob o seu "Mes MM/AAAA"
	for _, regiao := range []string{"REGIAO Leste", "REGIAO Norte", "REGIAO Sul"} {
		i := strings.Index(out, regiao)
		bloco := out[i:]
		if j := strings.Index(bloco, "TOTAL "); j > 0 {
			bloco = bloco[:j]
		}
		p12, p01, p02 := strings.Index(bloco, "Mes 12/2025"), strings.Index(bloco, "Mes 01/2026"), strings.Index(bloco, "Mes 02/2026")
		if !(p12 >= 0 && p12 < p01 && p01 < p02) {
			t.Errorf("%s: meses fora de ordem (%d, %d, %d):\n%s", regiao, p12, p01, p02, bloco)
		}
		if n := strings.Count(bloco, "Subtotal do mes"); n != 3 {
			t.Errorf("%s: %d subtotais, esperados 3", regiao, n)
		}
	}
	// os subtotais por mês de uma região: Leste dez/jan/fev
	for _, want := range []string{"1,230.25", "1,760.40", "2,594.65"} {
		if !strings.Contains(out, want) {
			t.Errorf("faltou o subtotal %s", want)
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
	// chaves do sort separadas por vírgula; o nome é o da coluna (ou o <nome> explícito)
	var nomes []string
	for _, chave := range strings.Split(sort[1], ",") {
		chave = strings.TrimSpace(chave)
		if m := regexp.MustCompile(`^<(\w+)>`).FindStringSubmatch(chave); m != nil {
			nomes = append(nomes, m[1])
		} else {
			nomes = append(nomes, strings.Fields(chave)[0])
		}
	}
	if len(nomes) < 2 {
		t.Fatalf("esperados 2+ nomes no sort, vieram %d", len(nomes))
	}
	for _, n := range nomes {
		for _, ev := range []string{"before ", "after "} {
			if !regexp.MustCompile(`(?m)^` + ev + n + `$`).MatchString(src) {
				t.Errorf("sem seção '%s%s'", ev, n)
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

func TestExemploInvoiceHTML(t *testing.T) {
	dir := dirExemplos + "invoice-html/"
	code, out, errb := executa(t, "", "--format", "html", "--box", "--template", dir+"template.html", "--logo", dir+"logo.svg",
		"--var", "Company=Northwind Ltd", "--var", "Invoice=2026-0142", "--var", "BillTo=Acme Corp", "--var", "Footer=due in 30 days",
		dir+"report.rp", dir+"data.txt")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{"No. 2026-0142", "Acme Corp", "Northwind Ltd &middot; due in 30 days", "data:image/svg+xml;base64,",
		"┌────────────────────────────┬───────┬────────────┬────────────┐", "1,885.88", "Page 1 of 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("HTML sem %q", want)
		}
	}
	// faltando uma variável do modelo, o erro diz qual e em que linha
	code, _, errb = executa(t, "", "--format", "html", "--template", dir+"template.html", "--logo", dir+"logo.svg", dir+"report.rp", dir+"data.txt")
	if code != 1 || !strings.Contains(errb, "variável desconhecida {{Invoice}}") && !strings.Contains(errb, "variável desconhecida {{Company}}") {
		t.Errorf("esperava erro de variável desconhecida: code=%d %q", code, errb)
	}
}

func TestExemploFipe(t *testing.T) {
	dir := dirExemplos + "fipe-veiculos/"
	code, out, errb := executa(t, "", "--input", "json", dir+"relatorio.rp", dir+"dados-exemplo.json")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	// marcas em ordem, modelos em ordem dentro da marca e anos em ordem dentro do modelo
	ordem := []string{"MARCA Aurora", "Cometa 1.0 Flex 12V 4p Mec.", "R$ 61.870,00", "R$ 67.240,00", "R$ 72.915,00",
		"Cometa 1.6 Flex 16V 4p Aut.", "Duna 2.0 Turbo 16V 5p Aut.", "Total de versoes de Aurora",
		"MARCA Brisa", "Onda 1.0 Flex 8V 4p", "Total de versoes de Brisa", "MARCA Cedro", "Pampa 2.8 Diesel 4x4 CD Mec."}
	pos := -1
	for _, want := range ordem {
		i := strings.Index(out[pos+1:], want)
		if i < 0 {
			t.Fatalf("faltou %q, ou fora de ordem, depois da posição %d:\n%s", want, pos, out)
		}
		pos += 1 + i
	}
	for _, want := range []string{"Referencia: novembro de 2026", "990001-1", "R$ 219.870,00"} {
		if !strings.Contains(out, want) {
			t.Errorf("saída sem %q", want)
		}
	}
	if n := strings.Count(out, "Versoes listadas:"); n != 6 {
		t.Errorf("%d modelos, esperados 6", n)
	}
	// o script auxiliar: só os códigos dos anos, um por linha, sem layout
	code, anos, errb := executa(t, `[{"code":"1985-1","name":"1985 Gasolina"},{"code":"1986-1","name":"1986 Gasolina"}]`,
		"--input", "json", "--format", "csv", "--csv-no-header", dir+"anos.rp", "-")
	if code != 0 || anos != "1985-1\n1986-1\n" {
		t.Errorf("anos.rp: code=%d %q %q", code, anos, errb)
	}
}
