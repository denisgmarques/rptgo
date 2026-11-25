package main

import (
	"path/filepath"
	"strings"
	"testing"

	"rptgo/internal/report"
)

func TestLeRegistrosCSV(t *testing.T) {
	casos := []struct {
		nome, texto string
		delim       rune
		cab         string
		want        [][]string
	}{
		{"com cabeçalho automático", "regiao,v\nSul,1\nNorte,2\n", ',', "auto", [][]string{{"Sul", "1"}, {"Norte", "2"}}},
		{"sem cabeçalho", "Sul,1\nNorte,2\n", ',', "auto", [][]string{{"Sul", "1"}, {"Norte", "2"}}},
		{"cabeçalho forçado", "a,b\nSul,1\n", ',', "sim", [][]string{{"Sul", "1"}}},
		{"cabeçalho recusado", "regiao,v\nSul,1\n", ',', "nao", [][]string{{"regiao", "v"}, {"Sul", "1"}}},
		{"aspas, delimitador e quebra", "regiao,v\n\"Sul, Norte\",\"diz \"\"oi\"\"\"\n\"a\nb\",3\n", ',', "auto",
			[][]string{{"Sul, Norte", `diz "oi"`}, {"a\nb", "3"}}},
		{"ponto e vírgula", "regiao;v\nSul;1,5\n", ';', "auto", [][]string{{"Sul", "1,5"}}},
		{"tab", "regiao\tv\nSul\t1\n", '\t', "auto", [][]string{{"Sul", "1"}}},
		{"BOM e CRLF", "\xEF\xBB\xBFregiao,v\r\nSul,1\r\n", ',', "auto", [][]string{{"Sul", "1"}}},
		{"vazio", "", ',', "auto", nil},
	}
	for _, c := range casos {
		got, err := leRegistrosCSV(c.texto, c.delim, "regiao", c.cab)
		if err != nil {
			t.Errorf("%s: %v", c.nome, err)
			continue
		}
		if len(got) != len(c.want) {
			t.Errorf("%s: got %q, want %q", c.nome, got, c.want)
			continue
		}
		for i := range got {
			if strings.Join(got[i], "\x00") != strings.Join(c.want[i], "\x00") {
				t.Errorf("%s: linha %d got %q, want %q", c.nome, i, got[i], c.want[i])
			}
		}
	}
}

func TestLeRegistrosJSON(t *testing.T) {
	cols := []string{"nome", "qtd", "preco", "ok", "nasc"}
	casos := []struct {
		nome, texto string
		want        [][]string
	}{
		{"array", `[{"nome":"Ana","qtd":3,"preco":1250.50,"ok":true,"nasc":"1985-03-15"},{"nome":"B","qtd":1,"preco":2,"ok":false,"nasc":null}]`,
			[][]string{{"Ana", "3", "1250.50", "true", "1985-03-15"}, {"B", "1", "2", "false", ""}}},
		{"objetos por linha (NDJSON)", "{\"nome\":\"A\",\"qtd\":1,\"preco\":1,\"ok\":true,\"nasc\":\"x\"}\n{\"nome\":\"B\",\"qtd\":2,\"preco\":2,\"ok\":true,\"nasc\":\"y\"}\n",
			[][]string{{"A", "1", "1", "true", "x"}, {"B", "2", "2", "true", "y"}}},
		{"um objeto só", `{"nome":"A","qtd":1,"preco":1,"ok":true,"nasc":"x"}`, [][]string{{"A", "1", "1", "true", "x"}}},
		{"chaves sem diferenciar maiúsculas e extras ignoradas", `[{"NOME":"A","Qtd":1,"PRECO":1,"ok":1,"nasc":"x","extra":"y"}]`,
			[][]string{{"A", "1", "1", "1", "x"}}},
		{"chave ausente em um registro vira vazio", `[{"nome":"A","qtd":1,"preco":1,"ok":true,"nasc":"x"},{"nome":"B","qtd":2,"preco":2,"ok":true}]`,
			[][]string{{"A", "1", "1", "true", "x"}, {"B", "2", "2", "true", ""}}},
		{"número grande e notação científica mantêm o texto", `[{"nome":"A","qtd":12345678901234,"preco":1.5e3,"ok":true,"nasc":"x"}]`,
			[][]string{{"A", "12345678901234", "1.5e3", "true", "x"}}},
		{"BOM", "\xEF\xBB\xBF" + `[{"nome":"A","qtd":1,"preco":1,"ok":true,"nasc":"x"}]`, [][]string{{"A", "1", "1", "true", "x"}}},
		{"array vazio", `[]`, [][]string{}},
	}
	for _, c := range casos {
		got, err := leRegistrosJSON(c.texto, cols)
		if err != nil {
			t.Errorf("%s: %v", c.nome, err)
			continue
		}
		if len(got) != len(c.want) {
			t.Errorf("%s: got %q, want %q", c.nome, got, c.want)
			continue
		}
		for i := range got {
			if strings.Join(got[i], "\x00") != strings.Join(c.want[i], "\x00") {
				t.Errorf("%s: registro %d got %q, want %q", c.nome, i, got[i], c.want[i])
			}
		}
	}
}

func TestLeRegistrosJSONErros(t *testing.T) {
	cols := []string{"nome", "qtd"}
	casos := []struct{ nome, texto, quer string }{
		{"JSON inválido", `[{"nome":`, "JSON inválido"},
		{"item que não é objeto", `[{"nome":"A","qtd":1}, 5]`, "item 2 do array não é um objeto"},
		{"valor que não é objeto nem array", `"texto"`, "esperado um array de objetos"},
		{"campo aninhado", `[{"nome":{"a":1},"qtd":1}]`, `campo "nome" não é um valor simples`},
		{"campo array", `[{"nome":"A","qtd":[1,2]}]`, `campo "qtd" não é um valor simples`},
		{"coluna ausente em todos", `[{"nome":"A"},{"nome":"B"}]`, `a coluna "qtd" do input não aparece`},
	}
	for _, c := range casos {
		_, err := leRegistrosJSON(c.texto, cols)
		if err == nil || !strings.Contains(err.Error(), c.quer) {
			t.Errorf("%s: err = %v, quer %q", c.nome, err, c.quer)
		}
	}
}

func TestExecutarEntradaCSVEJSONMesmoRelatorio(t *testing.T) {
	// o mesmo script sobre o mesmo dado em três fontes dá exatamente o mesmo relatório
	rp := filepath.Join("../../examples/relatorio-vendas-regiao", "relatorio.rp")
	dir := "../../examples/relatorio-vendas-regiao/"
	code, txt, errb := executa(t, "", rp, dir+"dados.txt")
	if code != 0 {
		t.Fatalf("texto: exit %d: %s", code, errb)
	}
	code, csv, errb := executa(t, "", "--input", "csv", rp, dir+"dados.csv")
	if code != 0 || csv != txt {
		t.Errorf("csv: exit %d stderr=%q, relatório %s do texto", code, errb, igualDiferente(csv == txt))
	}
	code, js, errb := executa(t, "", "--input", "json", rp, dir+"dados.json")
	if code != 0 || js != txt {
		t.Errorf("json: exit %d stderr=%q, relatório %s do texto", code, errb, igualDiferente(js == txt))
	}
	// pela entrada padrão também
	code, js2, _ := executa(t, lerArquivo(t, dir+"dados.json"), "--input", "json", rp, "-")
	if code != 0 || js2 != txt {
		t.Errorf("json por stdin: exit %d", code)
	}
}

func igualDiferente(igual bool) string {
	if igual {
		return "igual ao"
	}
	return "diferente do"
}

func TestExecutarEntradaCSVOpcoes(t *testing.T) {
	dir := t.TempDir()
	rp := filepath.Join(dir, "s.rp")
	writeFile(t, rp, "input\n  nome [string 20],\n  valor [amount 10]\nsort nome\ndetail\n  print nome in col 1, valor in col 25 using '###,##&.&&'\nend\n")
	dat := filepath.Join(dir, "d.csv")
	writeFile(t, dat, "nome;valor\n\"Silva; Ana\";1250.5\nBeto;3\n")
	code, out, errb := executa(t, "", "--input", "csv", "--input-delimiter", ";", rp, dat)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{"Silva; Ana", "1,250.50", "Beto"} {
		if !strings.Contains(out, want) {
			t.Errorf("saída sem %q:\n%s", want, out)
		}
	}
	// dado com número de campos diferente do input: erro de dados com a linha
	writeFile(t, dat, "nome;valor\nA;1;extra\n")
	if code, _, errb := executa(t, "", "--input", "csv", "--input-delimiter", ";", rp, dat); code != 1 || !strings.Contains(errb, "phase error") {
		t.Errorf("campos a mais: code=%d %q", code, errb)
	}
	// erros de fonte são código 1
	writeFile(t, dat, "[1,2]")
	if code, _, errb := executa(t, "", "--input", "json", rp, dat); code != 1 || !strings.Contains(errb, "rptgo: JSON") {
		t.Errorf("json inválido: code=%d %q", code, errb)
	}
}

func TestEntradaValidacoes(t *testing.T) {
	rp := filepath.Join(dirGolden, "vendas.rp")
	dat := filepath.Join(dirGolden, "vendas.dat")
	for _, args := range [][]string{
		{"--input", "xml", rp, dat},
		{"--input"},
		{"--input-delimiter", ";", rp, dat},                    // só com --input csv
		{"--input", "json", "--input-delimiter", ";", rp, dat}, // idem
		{"--input", "csv", "--input-delimiter", "ab", rp, dat}, // delimitador inválido
	} {
		code, out, errb := executa(t, "", args...)
		if code != 2 || out != "" || !strings.HasPrefix(errb, "rptgo: ") {
			t.Errorf("%v: code=%d stdout=%q stderr=%q", args, code, out, errb)
		}
	}
	if code, _, errb := executa(t, "", "--input", "delimited", rp, dat); code != 0 {
		t.Errorf("--input delimited: code=%d %q", code, errb)
	}
}

func TestAjustaDecimalEntrada(t *testing.T) {
	cols := []report.InputCol{{Name: "t", Type: "string"}, {Name: "a", Type: "amount"}, {Name: "n", Type: "numeric"}, {Name: "d", Type: "date"}}
	regs := [][]string{{"1.5", "1.250,50", " 3 ", "15/03/1985"}, {"x,y", "-0,5", "1.000", ""}}
	ajustaDecimalEntrada(regs, cols)
	want := [][]string{{"1.5", "1250.50", "3", "15/03/1985"}, {"x,y", "-0.5", "1000", ""}}
	for i := range regs {
		if strings.Join(regs[i], "|") != strings.Join(want[i], "|") {
			t.Errorf("registro %d: got %q, want %q", i, regs[i], want[i])
		}
	}
}

// CSV como o Excel brasileiro exporta: Windows-1252, ';', CRLF e decimal com vírgula.
func TestExecutarCSVDoExcelBrasileiro(t *testing.T) {
	dir := t.TempDir()
	rp := filepath.Join(dir, "s.rp")
	writeFile(t, rp, "input\n  nome [string 20],\n  nasc [date 10],\n  valor [amount 10]\nsort nome\ndetail\n  print nome in col 1, nasc in col 22, valor in col 34 using '###,##&.&&'\nend\n")
	dat := filepath.Join(dir, "excel.csv")
	writeFile(t, dat, "nome;nasc;valor\r\nJos\xe9 \"Z\";15/03/1985;1.250,50\r\nAna;02/11/1990;980,5\r\n")
	code, out, errb := executa(t, "", "--input", "csv", "--input-delimiter", ";", "--input-decimal", ",", "--encoding", "latin-1", rp, dat)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{"Ana", "02/11/90", "980.50", "Jos\xe9 \"Z\"", "15/03/85", "1,250.50"} {
		if !strings.Contains(out, want) {
			t.Errorf("saída sem %q:\n%q", want, out)
		}
	}
	// sem --input-decimal "1.250,50" não é número: erro de dados
	if code, _, errb := executa(t, "", "--input", "csv", "--input-delimiter", ";", "--encoding", "latin-1", rp, dat); code != 1 || !strings.Contains(errb, "valor inválido") {
		t.Errorf("sem --input-decimal: code=%d %q", code, errb)
	}
}

// O JSON gerado por --format json é aceito como fonte: ida e volta sem perder nada.
func TestJSONGeradoVoltaComoEntrada(t *testing.T) {
	dir := "../../examples/relatorio-vendas-regiao/"
	rp := dir + "relatorio.rp"
	_, original, _ := executa(t, "", rp, dir+"dados.txt")
	for _, fmtSaida := range []string{"json", "csv"} {
		code, exportado, errb := executa(t, "", "--format", fmtSaida, rp, dir+"dados.txt")
		if code != 0 {
			t.Fatalf("%s: exit %d: %s", fmtSaida, code, errb)
		}
		code, volta, errb := executa(t, exportado, "--input", fmtSaida, rp, "-")
		if code != 0 || volta != original {
			t.Errorf("%s: exit %d stderr=%q, relatório %s do original", fmtSaida, code, errb, igualDiferente(volta == original))
		}
	}
}

func TestInputDecimalValidacoes(t *testing.T) {
	rp := filepath.Join(dirGolden, "vendas.rp")
	dat := filepath.Join(dirGolden, "vendas.dat")
	for _, args := range [][]string{
		{"--input-decimal", ";", rp, dat},
		{"--input", "json", "--input-decimal", ",", rp, dat},
		{"--input", "csv", "--input-decimal", ",", rp, dat}, // delimitador padrão do CSV é a vírgula
	} {
		code, out, errb := executa(t, "", args...)
		if code != 2 || out != "" || !strings.HasPrefix(errb, "rptgo: ") {
			t.Errorf("%v: code=%d stdout=%q stderr=%q", args, code, out, errb)
		}
	}
	if code, _, errb := executa(t, "", "--input-decimal", ",", rp, dat); code != 0 {
		t.Errorf("--input-decimal com texto delimitado: code=%d %q", code, errb)
	}
}
