package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRenderizaTemplate(t *testing.T) {
	globais := map[string]string{"Title": "A & B <x>", "PageCount": "2", "Company": `Acme "Inc"`}
	tpl := "<h1>{{Title}}</h1>\n{{#Pages}}[{{ PageNumber }}/{{PageCount}}:{{Text}}]{{/Pages}}\n{{Company}}"
	got, err := renderizaTemplate(tpl, globais, []string{"um <b>", "dois & tres"})
	if err != nil {
		t.Fatal(err)
	}
	want := "<h1>A &amp; B &lt;x&gt;</h1>\n[1/2:um &lt;b&gt;][2/2:dois &amp; tres]\nAcme &#34;Inc&#34;"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderizaTemplateErros(t *testing.T) {
	g := map[string]string{"Title": "x"}
	casos := []struct{ nome, tpl, quer string }{
		{"variável desconhecida", "ok\n{{#Pages}}\n{{Nada}}\n{{/Pages}}", "linha 3: variável desconhecida {{Nada}}"},
		{"Text fora do bloco", "{{Text}}{{#Pages}}{{/Pages}}", "só vale dentro de {{#Pages}}"},
		{"sem bloco", "{{Title}}", "falta o bloco {{#Pages}}"},
		{"bloco aberto", "{{#Pages}}x", "sem {{/Pages}}"},
		{"fim sem início", "x{{/Pages}}", "sem {{#Pages}}"},
		{"bloco repetido", "{{#Pages}}{{/Pages}}{{#Pages}}{{/Pages}}", "repetido ou aninhado"},
		{"chave aberta", "{{Title", "sem '}}'"},
		{"marcador inválido", "{{ a b }}", "marcador inválido"},
	}
	for _, c := range casos {
		_, err := renderizaTemplate(c.tpl, g, []string{"p"})
		if err == nil || !strings.Contains(err.Error(), c.quer) {
			t.Errorf("%s: err = %v, quer %q", c.nome, err, c.quer)
		}
	}
}

func TestHTMLPadrao(t *testing.T) {
	code, out, errb := executa(t, "", "--format", "html", "--title", "Sales <2026>",
		filepath.Join(dirGolden, "vendas.rp"), filepath.Join(dirGolden, "vendas.dat"))
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{"<!DOCTYPE html>", `<meta charset="utf-8">`, "<title>Sales &lt;2026&gt;</title>",
		"VENDAS POR REGIAO", "Total Sul", "7,192.00", "2026-11-24 17:47", "Page 1 of 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("HTML sem %q", want)
		}
	}
	// sem preenchimento: nada de \f, sem margem esquerda nem linhas em branco de topo
	if strings.Contains(out, "\f") {
		t.Error("HTML não deveria ter \\f")
	}
	if !strings.Contains(out, "<pre>                     VENDAS POR REGIAO\n") {
		t.Errorf("o texto devia começar direto no <pre>, sem linhas de margem:\n%s", out)
	}
	if strings.Contains(out, "<img") {
		t.Error("sem --logo não deveria haver <img>")
	}
	if strings.Contains(out, "{{") {
		t.Error("sobrou marcador sem substituir")
	}
}

func TestHTMLPaginasELinhas(t *testing.T) {
	args := []string{"--format", "html", "--lines", "10", filepath.Join(dirGolden, "estoque.rp"), filepath.Join(dirGolden, "estoque.dat")}
	code, out, errb := executa(t, "", args...)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	n := strings.Count(out, `<section class="sheet">`)
	if n < 3 {
		t.Fatalf("com --lines 10 esperava várias páginas, vieram %d", n)
	}
	if !strings.Contains(out, "Page 1 of "+itoa(n)) || !strings.Contains(out, "Page "+itoa(n)+" of "+itoa(n)) {
		t.Errorf("numeração de páginas inconsistente com %d páginas", n)
	}
	// nenhuma página passa de 10 linhas de texto
	for _, p := range strings.Split(out, "<pre>")[1:] {
		texto := p[:strings.Index(p, "</pre>")]
		if l := strings.Count(texto, "\n") + 1; l > 10 {
			t.Errorf("página com %d linhas, máximo 10:\n%s", l, texto)
		}
	}
	// sem --lines usa length - margens do script (60 - 2 - 2 = 56): o estoque cabe em 1 página
	_, out2, _ := executa(t, "", "--format", "html", filepath.Join(dirGolden, "estoque.rp"), filepath.Join(dirGolden, "estoque.dat"))
	if c := strings.Count(out2, `<section class="sheet">`); c != 1 {
		t.Errorf("sem --lines: %d páginas, esperada 1", c)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestHTMLTemplateVarLogoEBox(t *testing.T) {
	dir := t.TempDir()
	rp := filepath.Join(dir, "s.rp")
	dat := filepath.Join(dir, "d.txt")
	tpl := filepath.Join(dir, "t.html")
	logo := filepath.Join(dir, "logo.svg")
	writeFile(t, rp, "input\n  nome [string 10],\n  v [numeric]\nbefore report\n  print '+------------+------+'\n  print '| Nome       | Valor|'\n  print '+------------+------+'\ndetail\n  print '|' in col 1, nome in col 3, '|' in col 14, v in col 16 using '####', '|' in col 21\nafter report\n  print '+------------+------+'\nend\n")
	writeFile(t, dat, "Acme|10\nBeta|200\n")
	writeFile(t, tpl, "<h1>{{Company}} - {{Title}} ({{Script}})</h1><img src=\"{{Logo}}\">\n{{#Pages}}<pre class=\"p{{PageNumber}}\">{{Text}}</pre>{{/Pages}}\n<p>{{PageCount}} / {{Columns}} / {{Date}} {{Time}}</p>")
	writeFile(t, logo, "<svg xmlns='http://www.w3.org/2000/svg'/>")
	code, out, errb := executa(t, "", "--format", "html", "--box", "--template", tpl, "--var", "Company=Acme & Co", "--logo", logo, rp, dat)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, want := range []string{"<h1>Acme &amp; Co - s (s.rp)</h1>", `<img src="data:image/svg+xml;base64,`,
		`<pre class="p1">┌────────────┬──────┐`, "│ Nome       │ Valor│", "├────────────┼──────┤", "└────────────┴──────┘",
		"<p>1 / 21 / 2026-11-24 17:47</p>"} {
		if !strings.Contains(out, want) {
			t.Errorf("saída sem %q:\n%s", want, out)
		}
	}
	// sem --box os caracteres ASCII continuam
	_, out2, _ := executa(t, "", "--format", "html", "--template", tpl, "--var", "Company=x", rp, dat)
	if !strings.Contains(out2, "+------------+------+") || strings.Contains(out2, "┌") {
		t.Errorf("sem --box:\n%s", out2)
	}
}

func TestHTMLSaidaSempreUTF8(t *testing.T) {
	// entrada em cp850; o HTML sai em UTF-8 (não em cp850), pois declara <meta charset="utf-8">
	dir := t.TempDir()
	rp := filepath.Join(dir, "s.rp")
	dat := filepath.Join(dir, "d.txt")
	os.WriteFile(rp, codificar("input\n  nome [string 12]\ndetail\n  print nome\nend\n", "cp850"), 0o644)
	os.WriteFile(dat, codificar("José\n", "cp850"), 0o644)
	_, out, errb := executa(t, "", "--format", "html", "--encoding", "cp850", rp, dat)
	if !strings.Contains(out, "José") {
		t.Errorf("esperava UTF-8 'José' no HTML: %q (stderr %q)", out, errb)
	}
}

func TestHTMLValidacoes(t *testing.T) {
	rp := filepath.Join(dirGolden, "vendas.rp")
	dat := filepath.Join(dirGolden, "vendas.dat")
	// opções de HTML fora do --format html, e combinações inválidas: uso inválido (código 2)
	for _, args := range [][]string{
		{"--template", "t.html", rp, dat},
		{"--logo", "l.png", rp, dat},
		{"--title", "x", "--format", "csv", rp, dat},
		{"--var", "A=1", "--format", "json", rp, dat},
		{"--lines", "10", rp, dat},
		{"--format", "html", "--form-feed", rp, dat},
		{"--format", "html", "--var", "semigual", rp, dat},
		{"--format", "html", "--var", "1x=2", rp, dat},
		{"--format", "html", "--lines", "0", rp, dat},
		{"--format", "html", "--lines", "abc", rp, dat},
	} {
		code, out, errb := executa(t, "", args...)
		if code != 2 || out != "" || !strings.HasPrefix(errb, "rptgo: ") {
			t.Errorf("%v: code=%d stdout=%q stderr=%q", args, code, out, errb)
		}
	}
	// erros de execução (arquivo ou modelo): código 1
	for _, args := range [][]string{
		{"--format", "html", "--template", "/nao/existe.html", rp, dat},
		{"--format", "html", "--logo", "/nao/existe.png", rp, dat},
		{"--format", "html", "--logo", "logo.bmp", rp, dat},
		{"--format", "html", "--var", "Title=x", rp, dat},
	} {
		code, out, errb := executa(t, "", args...)
		if code != 1 || out != "" || !strings.HasPrefix(errb, "rptgo: ") {
			t.Errorf("%v: code=%d stdout=%q stderr=%q", args, code, out, errb)
		}
	}
}
