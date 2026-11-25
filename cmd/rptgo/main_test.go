package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const dirGolden = "../../internal/report/testdata/golden"

// executa roda executar() com o relógio fixo e sem variáveis de configuração herdadas.
func executa(t *testing.T, stdin string, args ...string) (int, string, string) {
	t.Helper()
	t.Setenv("RPT_NOW", "2026-11-24 17:47")
	for _, k := range []string{"DATEFMT", "CENTURY_CUTOFF", "TIMEFMT", "WIDTH", "PAGELN", "SEPARATOR", "RADIXSEP", "CURRSYM"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	var out, errb bytes.Buffer
	code := executar(args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

func lerGolden(t *testing.T, nome string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dirGolden, nome))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestExecutarPadraoAtual fixa a saída de texto de executar() para combinações de flags
// existentes (T01); as mesmas referências valem para --format text depois (RF-03).
func TestExecutarPadraoAtual(t *testing.T) {
	script := filepath.Join(dirGolden, "misto.rp")
	dados := filepath.Join(dirGolden, "misto.dat")
	dadosComCabecalho := "id|nome|nasc|cadast|ativo|saldo\n" + lerGolden(t, "misto.dat")
	casos := []struct {
		nome  string
		stdin string
		args  []string
		want  string
	}{
		{"padrao", "", []string{script, dados}, lerGolden(t, "misto.txt")},
		{"stdin", lerGolden(t, "misto.dat"), []string{script, "-"}, lerGolden(t, "misto.txt")},
		{"form-feed", "", []string{"--form-feed", script, dados}, lerGolden(t, "misto_formfeed.txt")},
		{"newline crlf", "", []string{"--newline", "crlf", script, dados}, strings.ReplaceAll(lerGolden(t, "misto.txt"), "\n", "\r\n")},
		{"header sim", dadosComCabecalho, []string{"--header", "sim", script, "-"}, lerGolden(t, "misto.txt")},
		{"header auto", dadosComCabecalho, []string{script, "-"}, lerGolden(t, "misto.txt")},
		{"vendas", "", []string{filepath.Join(dirGolden, "vendas.rp"), filepath.Join(dirGolden, "vendas.dat")}, lerGolden(t, "vendas.txt")},
		{"estoque", "", []string{filepath.Join(dirGolden, "estoque.rp"), filepath.Join(dirGolden, "estoque.dat")}, lerGolden(t, "estoque.txt")},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			code, out, errb := executa(t, c.stdin, c.args...)
			if code != 0 {
				t.Fatalf("exit %d, stderr: %s", code, errb)
			}
			if out != c.want {
				t.Errorf("saída diferente da referência\n--- got ---\n%q\n--- want ---\n%q", out, c.want)
			}
		})
	}
}

func TestAnalisaArgsFormat(t *testing.T) {
	casos := []struct {
		nome    string
		args    []string
		want    string
		wantErr bool
	}{
		{"padrão", []string{"s.rp"}, "text", false},
		{"text", []string{"--format", "text", "s.rp"}, "text", false},
		{"csv", []string{"--format", "csv", "s.rp"}, "csv", false},
		{"json depois do posicional", []string{"s.rp", "d.txt", "--format", "json"}, "json", false},
		{"valor inválido", []string{"--format", "xml", "s.rp"}, "", true},
		{"maiúsculas não valem", []string{"--format", "CSV", "s.rp"}, "", true},
		{"sem valor", []string{"--format"}, "", true},
	}
	for _, c := range casos {
		op, _, err := analisaArgs(c.args)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: err = %v", c.nome, err)
			continue
		}
		if !c.wantErr && op.formato != c.want {
			t.Errorf("%s: formato = %q, want %q", c.nome, op.formato, c.want)
		}
	}
	code, out, errb := executa(t, "", "--format", "xml", "s.rp")
	if code != 2 || out != "" || !strings.HasPrefix(errb, "rptgo: ") {
		t.Errorf("--format xml: code=%d stdout=%q stderr=%q", code, out, errb)
	}
}

func TestFormatFormFeedRejeitado(t *testing.T) {
	script := filepath.Join(dirGolden, "vendas.rp")
	dados := filepath.Join(dirGolden, "vendas.dat")
	saida := filepath.Join(t.TempDir(), "saida.out")
	for _, fmtOpt := range []string{"csv", "json"} {
		for _, args := range [][]string{
			{"--format", fmtOpt, "--form-feed", script, dados},
			{"--form-feed", "--format", fmtOpt, script, dados},
			{script, dados, "--form-feed", "--format", fmtOpt, "-o", saida},
		} {
			code, out, errb := executa(t, "", args...)
			if code != 2 || out != "" || !strings.HasPrefix(errb, "rptgo: ") {
				t.Errorf("%v: code=%d stdout=%q stderr=%q", args, code, out, errb)
			}
		}
	}
	if _, err := os.Stat(saida); err == nil {
		t.Error("o arquivo -o não deveria ter sido criado")
	}
	// --format text com --form-feed continua válido
	if code, _, errb := executa(t, "", "--format", "text", "--form-feed", script, dados); code != 0 {
		t.Errorf("--format text --form-feed: code=%d stderr=%q", code, errb)
	}
}

func TestExecutarFormatoTextoInalterado(t *testing.T) {
	for _, c := range []struct{ nome, ref string }{
		{"vendas", "vendas.txt"}, {"estoque", "estoque.txt"}, {"misto", "misto.txt"},
	} {
		script := filepath.Join(dirGolden, c.nome+".rp")
		dados := filepath.Join(dirGolden, c.nome+".dat")
		want := lerGolden(t, c.ref)
		for _, args := range [][]string{{script, dados}, {"--format", "text", script, dados}} {
			code, out, errb := executa(t, "", args...)
			if code != 0 || out != want {
				t.Errorf("%v: code=%d stderr=%q, saída diferente da referência", args, code, errb)
			}
		}
	}
}

func TestExecutarFormatoCSV(t *testing.T) {
	code, out, errb := executa(t, "", "--format", "csv",
		filepath.Join(dirGolden, "vendas.rp"), filepath.Join(dirGolden, "vendas.dat"))
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	want := "regiao,vendedor,valor\n" +
		"Leste,Diego Alves,450.25\nLeste,Fabio Neri,780\n" +
		"Norte,Bruno Lima,980\nNorte,Elisa Prado,1320.4\n" +
		"Sul,Ana Souza,1250.5\nSul,Carla Dias,2100.75\nSul,Gina Rocha,310.1\n"
	if out != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestExecutarFormatoJSON(t *testing.T) {
	code, out, errb := executa(t, "", "--format", "json",
		filepath.Join(dirGolden, "vendas.rp"), filepath.Join(dirGolden, "vendas.dat"))
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var objs []map[string]any
	if err := json.Unmarshal([]byte(out), &objs); err != nil {
		t.Fatalf("JSON inválido: %v\n%s", err, out)
	}
	if len(objs) != 7 || objs[0]["regiao"] != "Leste" || objs[0]["valor"] != 450.25 {
		t.Errorf("objetos inesperados: %v", objs)
	}
}

// TestExecutarFormatoOrdemESort: as seções de evento (com efeitos colaterais) não rodam em
// csv/json, e a ordem vem da cláusula sort. Layout (width/length/margens) é ignorado.
func TestExecutarFormatoOrdemESort(t *testing.T) {
	const script = `input
  g [string 3],
  n [numeric],
  v [numeric]
width 20
length 8
top margin 3
sort <g> g desc, <n> n
before report
  print 'CABECALHO-RELATORIO'
header
  print 'CABECALHO-PAGINA'
footer
  print 'RODAPE-PAGINA'
before g
  set contador to 1
after g
  print 'TOTAL-GRUPO', total(v)
after report
  print 'TOTAL-GERAL', total(v)
detail
  set contador to contador + 1
  print 'DETALHE', v
end
`
	dir := t.TempDir()
	rp := filepath.Join(dir, "s.rp")
	dat := filepath.Join(dir, "d.txt")
	if err := os.WriteFile(rp, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dat, []byte("a|2|10\nb|9|20\na|1|30\nb|3|40\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errb := executa(t, "", "--format", "csv", rp, dat)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if want := "g,n,v\nb,3,40\nb,9,20\na,1,30\na,2,10\n"; out != want {
		t.Errorf("csv:\n%s\nwant:\n%s", out, want)
	}
	code, out, errb = executa(t, "", "--format", "json", rp, dat)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	var objs []map[string]any
	if err := json.Unmarshal([]byte(out), &objs); err != nil {
		t.Fatal(err)
	}
	var ordem []float64
	for _, o := range objs {
		ordem = append(ordem, o["v"].(float64))
	}
	if fmt.Sprint(ordem) != "[40 20 30 10]" {
		t.Errorf("ordem = %v", ordem)
	}
	for _, proibido := range []string{"CABECALHO", "RODAPE", "TOTAL-", "DETALHE"} {
		if strings.Contains(out, proibido) {
			t.Errorf("conteúdo de evento vazou na saída: %q", proibido)
		}
	}
	// o mesmo script em modo texto continua produzindo o relatório normal
	code, out, _ = executa(t, "", rp, dat)
	if code != 0 || !strings.Contains(out, "TOTAL-GERAL") {
		t.Errorf("modo texto quebrou: code=%d", code)
	}
}

func TestExecutarNewlineEEncodingEmCSV(t *testing.T) {
	dir := t.TempDir()
	rp := filepath.Join(dir, "s.rp")
	dat := filepath.Join(dir, "d.txt")
	os.WriteFile(rp, []byte("input\n  nome [string 10]\ndetail\n  print nome\nend\n"), 0o644)
	os.WriteFile(dat, []byte(decodificarParaTeste("Jos\u00e9\n", "cp850")), 0o644)
	_, out, _ := executa(t, "", "--format", "csv", "--newline", "crlf", "--encoding", "cp850", rp, dat)
	if out != "nome\r\nJos\x82\r\n" { // 0x82 = é em CP850
		t.Errorf("got %q", out)
	}
}

// decodificarParaTeste codifica texto Unicode em bytes (como string) na página indicada.
func decodificarParaTeste(s, enc string) string { return string(codificar(s, enc)) }

func TestEncodingCP850(t *testing.T) {
	// round-trip de todos os 256 bytes por decodificar -> codificar
	todos := make([]byte, 256)
	for i := range todos {
		todos[i] = byte(i)
	}
	if got := codificar(decodificar(todos, "cp850"), "cp850"); !bytes.Equal(got, todos) {
		t.Error("round-trip completo falhou")
	}
	// case-insensitive, como os outros valores de --encoding
	if decodificar([]byte{0x82}, "CP850") != "é" || decodificar([]byte{0x82}, "Cp850") != "é" {
		t.Error("cp850 deveria ser case-insensitive")
	}
	// rune sem correspondência vira '?'
	if got := codificar("a€b", "cp850"); string(got) != "a?b" {
		t.Errorf("fallback = %q", got)
	}
	// ponta a ponta: script e dados em CP850, saída em CP850
	dir := t.TempDir()
	rp := filepath.Join(dir, "s.rp")
	dat := filepath.Join(dir, "d.txt")
	script := "input\n  nome [string 12],\n  cidade [string 12]\ndetail\n  print 'Ol\u00e1 ', nome in col 6, cidade in col 20\nend\n"
	os.WriteFile(rp, codificar(script, "cp850"), 0o644)
	os.WriteFile(dat, codificar("Jo\u00e3o|S\u00e3o Paulo\nZo\u00eb|\u00c1vila\n", "cp850"), 0o644)
	code, out, errb := executa(t, "", "--newline", "lf", "--encoding", "cp850", rp, dat)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	texto := decodificar([]byte(out), "cp850")
	for _, want := range []string{"Olá", "João", "São Paulo", "Zoë", "Ávila"} {
		if !strings.Contains(texto, want) {
			t.Errorf("saída decodificada sem %q:\n%s", want, texto)
		}
	}
	// os mesmos bytes lidos como utf-8 não produziriam os acentos
	code, out, _ = executa(t, "", "--newline", "lf", rp, dat)
	if code == 0 && strings.Contains(out, "João") {
		t.Error("sem --encoding cp850 os acentos não deveriam sair corretos")
	}
}

func writeFile(t *testing.T, caminho, conteudo string) {
	t.Helper()
	if err := os.WriteFile(caminho, []byte(conteudo), 0o644); err != nil {
		t.Fatal(err)
	}
}
