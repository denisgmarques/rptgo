package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"rptgo/internal/report"
)

// preparar converte um script e dados separados por '|' em Script e linhas já ordenadas.
func preparar(t *testing.T, script string, dados []string) (*report.Script, []report.Row) {
	t.Helper()
	toks, err := report.Tokenize(script, false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := report.Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	regs := make([][]string, len(dados))
	for i, d := range dados {
		regs[i] = strings.Split(d, "|")
	}
	it, err := report.NovoInterp(s, regs, false)
	if err != nil {
		t.Fatal(err)
	}
	return s, it.LinhasOrdenadas()
}

const scriptTipos = `input
  z      [string 20],
  a      [numeric],
  m      [float],
  nasc   [date 8],
  cad    [ldate 10],
  ok     [bool]
detail
  print z
end
`

var dadosTipos = []string{
	"um|1|1.5|05/03/85|2020-01-15|yes",
	"dois||2|31/12/29|2019-12-31|no",
	"tres|3||||xxx",
}

var reISO = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func TestEscreveCSVHeaderEContagem(t *testing.T) {
	s, rows := preparar(t, scriptTipos, dadosTipos)
	var buf bytes.Buffer
	if err := escreveCSV(&buf, s, rows, csvPadrao); err != nil {
		t.Fatal(err)
	}
	linhas := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(linhas) != len(dadosTipos)+1 {
		t.Fatalf("esperadas %d linhas, vieram %d:\n%s", len(dadosTipos)+1, len(linhas), buf.String())
	}
	if linhas[0] != "z,a,m,nasc,cad,ok" {
		t.Errorf("cabeçalho = %q", linhas[0])
	}
	if linhas[3] != "tres,3,,,," {
		t.Errorf("linha com nulos: %q", linhas[3])
	}
}

func TestEscreveCSVQuoting(t *testing.T) {
	const script = "input\n  a [string 30],\n  b [string 30]\ndetail\n  print a\nend\n"
	s, _ := preparar(t, script, []string{"x|y"})
	rows := []report.Row{
		{"com,virgula", `com "aspas"`},
		{"linha1\nlinha2", "simples"},
		{"", "fim"},
	}
	var buf bytes.Buffer
	if err := escreveCSV(&buf, s, rows, csvPadrao); err != nil {
		t.Fatal(err)
	}
	want := "a,b\n\"com,virgula\",\"com \"\"aspas\"\"\"\n\"linha1\nlinha2\",simples\n,fim\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
	lido, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(lido) != 4 || lido[1][0] != "com,virgula" || lido[1][1] != `com "aspas"` || lido[2][0] != "linha1\nlinha2" {
		t.Errorf("round-trip pelo encoding/csv falhou: %q", lido)
	}
}

func TestEscreveCSVDatasISO(t *testing.T) {
	// DATEFMT e CENTURY_CUTOFF diferentes do padrão não podem afetar a saída ISO.
	t.Setenv("DATEFMT", "MM/DD/YY")
	t.Setenv("CENTURY_CUTOFF", "50")
	s, rows := preparar(t, scriptTipos, []string{"um|1|1.5|03/05/85|2020-01-15|yes", "dois|2|2|12/31/29|12/31/2019|no"})
	var buf bytes.Buffer
	if err := escreveCSV(&buf, s, rows, csvPadrao); err != nil {
		t.Fatal(err)
	}
	lido, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range lido[1:] {
		for _, idx := range []int{3, 4} { // nasc (date) e cad (ldate)
			if !reISO.MatchString(r[idx]) {
				t.Errorf("registro %d, coluna %s: %q não é AAAA-MM-DD", i+1, lido[0][idx], r[idx])
			}
		}
	}
	if lido[1][3] != "1985-03-05" || lido[1][4] != "2020-01-15" || lido[2][3] != "2029-12-31" {
		t.Errorf("datas inesperadas: %v / %v", lido[1], lido[2])
	}
}

func TestEscreveCSVTiposEIgnoraLayout(t *testing.T) {
	s1, r1 := preparar(t, scriptTipos, dadosTipos)
	s2, r2 := preparar(t, strings.Replace(scriptTipos, "input\n", "width 40\nlength 10\ntop margin 5\ninput\n", 1), dadosTipos)
	var a, b bytes.Buffer
	if err := escreveCSV(&a, s1, r1, csvPadrao); err != nil {
		t.Fatal(err)
	}
	if err := escreveCSV(&b, s2, r2, csvPadrao); err != nil {
		t.Fatal(err)
	}
	if a.String() != b.String() {
		t.Errorf("layout alterou o CSV:\n%s\n---\n%s", a.String(), b.String())
	}
	lido, _ := csv.NewReader(&a).ReadAll()
	if got := strings.Join(lido[1], ","); got != "um,1,1.5,1985-03-05,2020-01-15,true" {
		t.Errorf("linha 1 = %q", got)
	}
	if got := strings.Join(lido[2], ","); got != "dois,,2,2029-12-31,2019-12-31,false" {
		t.Errorf("linha 2 = %q", got)
	}
	if got := strings.Join(lido[3], ","); got != "tres,3,,,," {
		t.Errorf("linha 3 = %q", got)
	}
}

func TestEscreveJSONEstrutura(t *testing.T) {
	s, rows := preparar(t, scriptTipos, dadosTipos)
	var buf bytes.Buffer
	if err := escreveJSON(&buf, s, rows); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(buf.Bytes()) {
		t.Fatalf("JSON inválido:\n%s", buf.String())
	}
	var objs []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &objs); err != nil {
		t.Fatal(err)
	}
	if len(objs) != len(dadosTipos) {
		t.Fatalf("esperados %d objetos, vieram %d", len(dadosTipos), len(objs))
	}
	for i, o := range objs {
		if len(o) != 6 {
			t.Errorf("objeto %d: %d chaves, esperadas 6: %v", i, len(o), o)
		}
		for _, k := range []string{"z", "a", "m", "nasc", "cad", "ok"} {
			if _, ok := o[k]; !ok {
				t.Errorf("objeto %d sem a chave %q", i, k)
			}
		}
	}
	// A ordem das chaves segue o 'input' (z, a, m...), não a ordem alfabética.
	linha := strings.Split(buf.String(), "\n")[1]
	ordem := []string{`"z":`, `"a":`, `"m":`, `"nasc":`, `"cad":`, `"ok":`}
	ult := -1
	for _, k := range ordem {
		p := strings.Index(linha, k)
		if p <= ult {
			t.Errorf("chave %s fora da ordem declarada em %s", k, linha)
		}
		ult = p
	}
}

func TestEscreveJSONTipos(t *testing.T) {
	t.Setenv("DATEFMT", "MM/DD/YY")
	t.Setenv("CENTURY_CUTOFF", "50")
	s, rows := preparar(t, scriptTipos, []string{
		"um|1|1.5|03/05/85|2020-01-15|yes",
		"dois||2|12/31/29|12/31/2019|no",
		"tres|3||||xxx",
	})
	var buf bytes.Buffer
	if err := escreveJSON(&buf, s, rows); err != nil {
		t.Fatal(err)
	}
	cru := buf.String()
	// datas: string AAAA-MM-DD no JSON cru (nada de RFC 3339 com hora)
	if strings.Contains(cru, "T00:00") {
		t.Errorf("data com componente de hora: %s", cru)
	}
	for _, trecho := range []string{`"nasc":"1985-03-05"`, `"cad":"2020-01-15"`, `"nasc":"2029-12-31"`, `"cad":"2019-12-31"`} {
		if !strings.Contains(cru, trecho) {
			t.Errorf("faltou %s em %s", trecho, cru)
		}
	}
	var objs []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &objs); err != nil {
		t.Fatal(err)
	}
	if objs[0]["a"] != float64(1) || objs[0]["m"] != 1.5 || objs[0]["ok"] != true || objs[1]["ok"] != false {
		t.Errorf("tipos do primeiro/segundo objeto: %v / %v", objs[0], objs[1])
	}
	if _, ok := objs[0]["nasc"].(string); !ok || !reISO.MatchString(objs[0]["nasc"].(string)) {
		t.Errorf("nasc = %v", objs[0]["nasc"])
	}
	// nulos: a (2º), m, nasc, cad e ok (3º, valor inválido de bool) viram null
	if objs[1]["a"] != nil || objs[2]["m"] != nil || objs[2]["nasc"] != nil || objs[2]["cad"] != nil || objs[2]["ok"] != nil {
		t.Errorf("nulos esperados: %v / %v", objs[1], objs[2])
	}
	if !strings.Contains(cru, `"a":null`) {
		t.Errorf("faltou \"a\":null em %s", cru)
	}
}

func TestEscreveJSONEscapeEVazio(t *testing.T) {
	const script = "input\n  a [string 30]\ndetail\n  print a\nend\n"
	s, _ := preparar(t, script, []string{"x"})
	var buf bytes.Buffer
	if err := escreveJSON(&buf, s, []report.Row{{"aspas \" e \\ e\nquebra"}}); err != nil {
		t.Fatal(err)
	}
	var objs []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &objs); err != nil || objs[0]["a"] != "aspas \" e \\ e\nquebra" {
		t.Errorf("escape falhou: %v %v", objs, err)
	}
	buf.Reset()
	if err := escreveJSON(&buf, s, nil); err != nil || strings.TrimSpace(buf.String()) != "[]" {
		t.Errorf("vazio = %q (%v)", buf.String(), err)
	}
}

func TestEscreveCSVOpcoes(t *testing.T) {
	const script = "input\n  nome [string 30],\n  qtd [numeric],\n  preco [amount 10],\n  dt [date],\n  ok [bool]\ndetail\n  print nome\nend\n"
	s, _ := preparar(t, script, []string{"x|1|1|01/01/2020|1"})
	rows, _ := func() ([]report.Row, error) {
		_, r := preparar(t, script, []string{"Ana; Maria|3|1234.5|15/03/1985|yes", "Plain|10|0.5||no", "|7||31/12/2020|"})
		return r, nil
	}()
	casos := []struct {
		nome string
		opt  csvOpcoes
		want string
	}{
		{"padrão", csvPadrao,
			"nome,qtd,preco,dt,ok\nAna; Maria,3,1234.5,1985-03-15,true\nPlain,10,0.5,,false\n,7,,2020-12-31,\n"},
		{"ponto e vírgula", csvOpcoes{Delim: ';', Aspas: aspasMinimas, Decimal: '.'},
			"nome;qtd;preco;dt;ok\n\"Ana; Maria\";3;1234.5;1985-03-15;true\nPlain;10;0.5;;false\n;7;;2020-12-31;\n"},
		{"texto entre aspas", csvOpcoes{Delim: ';', Aspas: aspasTexto, Decimal: '.'},
			"\"nome\";\"qtd\";\"preco\";\"dt\";\"ok\"\n\"Ana; Maria\";3;1234.5;1985-03-15;true\n\"Plain\";10;0.5;;false\n\"\";7;;2020-12-31;\n"},
		{"tudo entre aspas", csvOpcoes{Delim: ',', Aspas: aspasTodas, Decimal: '.'},
			"\"nome\",\"qtd\",\"preco\",\"dt\",\"ok\"\n\"Ana; Maria\",\"3\",\"1234.5\",\"1985-03-15\",\"true\"\n\"Plain\",\"10\",\"0.5\",\"\",\"false\"\n\"\",\"7\",\"\",\"2020-12-31\",\"\"\n"},
		{"decimal com vírgula", csvOpcoes{Delim: ';', Aspas: aspasTexto, Decimal: ','},
			"\"nome\";\"qtd\";\"preco\";\"dt\";\"ok\"\n\"Ana; Maria\";3;1234,5;1985-03-15;true\n\"Plain\";10;0,5;;false\n\"\";7;;2020-12-31;\n"},
		{"sem cabeçalho", csvOpcoes{Delim: '|', Aspas: aspasMinimas, Decimal: '.', SemCabecalho: true},
			"Ana; Maria|3|1234.5|1985-03-15|true\nPlain|10|0.5||false\n|7||2020-12-31|\n"},
		{"tab", csvOpcoes{Delim: '\t', Aspas: aspasMinimas, Decimal: '.', SemCabecalho: true},
			"Ana; Maria\t3\t1234.5\t1985-03-15\ttrue\nPlain\t10\t0.5\t\tfalse\n\t7\t\t2020-12-31\t\n"},
	}
	for _, c := range casos {
		var buf bytes.Buffer
		if err := escreveCSV(&buf, s, rows, c.opt); err != nil {
			t.Errorf("%s: %v", c.nome, err)
			continue
		}
		if buf.String() != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.nome, buf.String(), c.want)
		}
	}
}

func TestEscreveCSVAspasEEscapesComDelimitadorNoTexto(t *testing.T) {
	const script = "input\n  a [string 30]\ndetail\n  print a\nend\n"
	s, _ := preparar(t, script, []string{"x"})
	rows := []report.Row{{`diz "oi"; tchau`}, {" começa com branco"}, {"linha1\nlinha2"}, {""}}
	var buf bytes.Buffer
	if err := escreveCSV(&buf, s, rows, csvOpcoes{Delim: ';', Aspas: aspasMinimas, Decimal: '.'}); err != nil {
		t.Fatal(err)
	}
	want := "a\n\"diz \"\"oi\"\"; tchau\"\n\" começa com branco\"\n\"linha1\nlinha2\"\n\"\"\n"
	if buf.String() != want {
		t.Errorf("got %q want %q", buf.String(), want)
	}
	// o que escrevemos volta idêntico pelo encoding/csv (com o mesmo delimitador)
	r := csv.NewReader(strings.NewReader(buf.String()))
	r.Comma = ';'
	lido, err := r.ReadAll()
	if err != nil || len(lido) != 5 || lido[1][0] != `diz "oi"; tchau` || lido[3][0] != "linha1\nlinha2" || lido[4][0] != "" {
		t.Errorf("round-trip: %q %v", lido, err)
	}
}

func TestHoraEmCSVEJSONEEntradas(t *testing.T) {
	dir := t.TempDir()
	rp := dir + "/s.rp"
	writeFile(t, rp, "input\n  nome [string 10],\n  saida [time],\n  volta [time]\ndetail\n  print nome, saida, volta\nend\n")
	writeFile(t, dir+"/d.txt", "Ana|8:05|17:30:15\nBeto|23:59|\n")
	_, csvOut, _ := executa(t, "", "--format", "csv", rp, dir+"/d.txt")
	if want := "nome,saida,volta\nAna,08:05:00,17:30:15\nBeto,23:59:00,\n"; csvOut != want {
		t.Errorf("csv:\n%q\nwant %q", csvOut, want)
	}
	_, jsonOut, _ := executa(t, "", "--format", "json", rp, dir+"/d.txt")
	if !strings.Contains(jsonOut, `{"nome":"Ana","saida":"08:05:00","volta":"17:30:15"}`) || !strings.Contains(jsonOut, `"volta":null`) {
		t.Errorf("json: %s", jsonOut)
	}
	// o que foi exportado volta como fonte e dá o mesmo relatório
	_, original, _ := executa(t, "", rp, dir+"/d.txt")
	for _, f := range []string{"csv", "json"} {
		_, exp, _ := executa(t, "", "--format", f, rp, dir+"/d.txt")
		code, volta, errb := executa(t, exp, "--input", f, rp, "-")
		if code != 0 || volta != original {
			t.Errorf("%s: ida e volta diferente (exit %d %s)", f, code, errb)
		}
	}
}
