package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	scriptLinha = "input nome, valor [float]\nleft margin 0\nsort nome\ndetail\n   print nome, valor\nend\n"
	dadosLinha  = "Beto|20\nAna|10.5\n"
)

// arquivosLinha grava o script e os dados num diretório temporário.
func arquivosLinha(t *testing.T, script, dados string) (rp, dat string) {
	t.Helper()
	dir := t.TempDir()
	rp, dat = filepath.Join(dir, "s.rp"), filepath.Join(dir, "d.txt")
	writeFile(t, rp, script)
	writeFile(t, dat, dados)
	return rp, dat
}

// -e e --script dão a mesma saída (e o mesmo erro) que um arquivo .rp com o mesmo texto.
func TestScriptNaLinhaIgualAoArquivo(t *testing.T) {
	modelo := filepath.Join(t.TempDir(), "m.html")
	writeFile(t, modelo, "<h1>{{Title}}</h1>{{#Pages}}<pre>{{Text}}</pre>{{/Pages}}")
	casos := []struct {
		nome, script string
		opcoes       []string
	}{
		{"texto", scriptLinha, nil},
		{"csv", scriptLinha, []string{"--format", "csv"}},
		{"json", scriptLinha, []string{"--format", "json"}},
		{"html", scriptLinha, []string{"--format", "html", "--title", "T", "--template", modelo}},
		{"strict", "/* c */\n" + scriptLinha, []string{"--strict"}},
		{"comentario longo com strict", "/* a\nb */\n" + scriptLinha, []string{"--strict"}},
		{"erro de sintaxe na linha 2", "input nome\nbanana\nend\n", nil},
		{"erro na execução", "input nome\ndetail\n print outra\nend\n", nil},
		{"vazio", "", nil},
	}
	for _, c := range casos {
		rp, dat := arquivosLinha(t, c.script, dadosLinha)
		codigo, saida, erro := executa(t, "", append(c.opcoes, rp, dat)...)
		for _, opcao := range []string{"-e", "--script"} {
			code, out, errb := executa(t, "", append(c.opcoes, opcao, c.script, dat)...)
			if code != codigo || out != saida || errb != erro {
				t.Errorf("%s %s: code=%d/%d\n--- linha ---\n%s%s\n--- arquivo ---\n%s%s", c.nome, opcao, code, codigo, out, errb, saida, erro)
			}
		}
		destino := filepath.Join(t.TempDir(), "saida")
		if code, _, _ := executa(t, "", append(c.opcoes, "-o", destino, "-e", c.script, dat)...); code == 0 && lerArquivo(t, destino) != saida {
			t.Errorf("%s: o arquivo de -o difere", c.nome)
		}
	}
}

// Um erro de execução sai byte a byte igual do arquivo .rp e do -e.
func TestErroDeExecucaoIgualNoArquivoENaLinha(t *testing.T) {
	const entrada = "input nome, valor [numeric]\n"
	longa := "\tprint '" + strings.Repeat("é", 90) + "',\tvalor / (valor - valor)"
	for _, c := range []struct{ nome, script, secao string }{
		{"detail", entrada + "detail\n   print nome, valor / 0\nend\n", "detail"},
		{"linha longa com tabs e acentos", entrada + "detail\n" + longa + "\nend\n", "detail"},
		{"várias linhas com CRLF", "input nome, valor [numeric]\r\ndetail\r\n   print valor %\r\n      ''\r\nend\r\n", "detail"},
		{"sem quebra de linha no fim", entrada + "sort valor / null\nend", "sort"},
	} {
		rp, dat := arquivosLinha(t, c.script, dadosLinha)
		code, out, want := executa(t, "", rp, dat)
		if code != 1 || out != "" || !strings.HasPrefix(want, "rptgo: divisão por zero na seção "+c.secao+"\nscript, linha ") {
			t.Fatalf("%s: code=%d stdout=%q stderr=%q", c.nome, code, out, want)
		}
		for _, opcao := range []string{"-e", "--script"} {
			if code, out, errb := executa(t, "", opcao, c.script, dat); code != 1 || out != "" || errb != want {
				t.Errorf("%s %s: code=%d stdout=%q\n--- linha ---\n%q\n--- arquivo ---\n%q", c.nome, opcao, code, out, errb, want)
			}
		}
	}
}

// O texto de -e nunca é tratado como caminho de arquivo.
func TestScriptNaLinhaNaoLeArquivo(t *testing.T) {
	rp, dat := arquivosLinha(t, scriptLinha, dadosLinha)
	if code, _, errb := executa(t, "", "-e", rp, dat); code != 1 || errb == "" {
		t.Errorf("code=%d stderr=%q", code, errb)
	}
}

func TestScriptNaLinhaFonteDosDados(t *testing.T) {
	_, dat := arquivosLinha(t, scriptLinha, dadosLinha)
	_, want, _ := executa(t, "", "-e", scriptLinha, dat)
	for _, c := range []struct {
		stdin string
		args  []string
	}{
		{dadosLinha, []string{"-e", scriptLinha}},
		{dadosLinha, []string{"-e", scriptLinha, "-"}},
		{"", []string{dat, "-e", scriptLinha}},
	} {
		if code, out, errb := executa(t, c.stdin, c.args...); code != 0 || out != want {
			t.Errorf("%q: code=%d stderr=%q\n%s", c.args, code, errb, out)
		}
	}
	code, _, errb := executa(t, "", "-e", scriptLinha, dat+".nao-existe")
	if code != 1 || !strings.HasPrefix(errb, "rptgo: ") {
		t.Errorf("dados inexistentes: code=%d stderr=%q", code, errb)
	}
}

// leitorVigiado registra se a entrada padrão foi lida.
type leitorVigiado struct{ lido bool }

func (l *leitorVigiado) Read([]byte) (int, error) {
	l.lido = true
	return 0, os.ErrClosed
}

func TestScriptNaLinhaErrosDeUso(t *testing.T) {
	s := scriptLinha
	for _, args := range [][]string{
		{"-e"},
		{"--script"},
		{"-e", s, "-e", s + " "},
		{"--script", s, "--script", s + " "},
		{"-e", s, "--script", s + " "},
		{"-e", s, "-e", s},
		{"-e", s, "x", "y"},
		{"-e", s, "script.rp", "dados.txt"},
	} {
		for _, prefixo := range [][]string{nil, {"--check"}} {
			destino := filepath.Join(t.TempDir(), "saida")
			todos := append(append([]string{"-o", destino}, prefixo...), args...)
			entrada := &leitorVigiado{}
			var out, errb bytes.Buffer
			code := executar(todos, entrada, &out, &errb)
			if code != 2 || out.Len() != 0 || !strings.HasPrefix(errb.String(), "rptgo: ") ||
				!strings.Contains(errb.String(), "Tente 'rptgo --help' para mais informações.") {
				t.Errorf("%q: code=%d stdout=%q stderr=%q", todos, code, out.String(), errb.String())
			}
			if entrada.lido {
				t.Errorf("%q: leu a entrada padrão", todos)
			}
			if _, err := os.Stat(destino); err == nil {
				t.Errorf("%q: criou o arquivo de -o", todos)
			}
		}
	}
}

func TestScriptNaLinhaCheck(t *testing.T) {
	for _, opcao := range []string{"--check", "-r"} {
		for _, args := range [][]string{{opcao, "-e", scriptLinha}, {opcao, "-e", scriptLinha, "nao-existe.txt"}} {
			entrada := &leitorVigiado{}
			var out, errb bytes.Buffer
			if code := executar(args, entrada, &out, &errb); code != 0 || out.String() != "Script sem erros de sintaxe.\n" || entrada.lido {
				t.Errorf("%q: code=%d lido=%v stdout=%q stderr=%q", args, code, entrada.lido, out.String(), errb.String())
			}
		}
		if code, _, errb := executa(t, "", opcao, "-e", "input a\nbanana\nend"); code != 1 || errb == "" {
			t.Errorf("%s com script inválido: code=%d stderr=%q", opcao, code, errb)
		}
	}
}

// --encoding vale para os dados e a saída; o texto de -e é sempre UTF-8.
func TestScriptNaLinhaCodificacao(t *testing.T) {
	script := "input nome\nleft margin 0\ndetail\n print 'São', nome\nend\n"
	for enc, byteA := range map[string]byte{"latin-1": 0xE3, "cp850": 0xC6} {
		dados := string(codificar("João\n", enc))
		rp, dat := arquivosLinha(t, string(codificar(script, enc)), dados)
		_, want, _ := executa(t, "", "--encoding", enc, rp, dat)
		code, out, errb := executa(t, "", "--encoding", enc, "-e", script, dat)
		if code != 0 || out != want || !strings.Contains(out, string([]byte{'S', byteA, 'o', ' ', 'J', 'o', byteA})) {
			t.Errorf("%s: code=%d stderr=%q saída=%q", enc, code, errb, out)
		}
		if _, html, _ := executa(t, "", "--encoding", enc, "--format", "html", "-e", script, dat); !strings.Contains(html, "São João") {
			t.Errorf("%s: o HTML deveria continuar em UTF-8", enc)
		}
	}
	if _, out, _ := executa(t, "Ana\n", "-e", script); !strings.Contains(out, "São Ana") {
		t.Errorf("utf-8: %q", out)
	}
}

func TestScriptNaLinhaHTML(t *testing.T) {
	antes := nomePrograma
	nomePrograma = "rpt"
	defer func() { nomePrograma = antes }()
	rp, dat := arquivosLinha(t, scriptLinha, dadosLinha)
	modelo := filepath.Join(t.TempDir(), "m.html")
	writeFile(t, modelo, "{{Title}}|[{{Script}}]{{#Pages}}{{/Pages}}")
	for _, c := range []struct {
		want string
		args []string
	}{
		{"rptgo|[]", []string{"-e", scriptLinha, dat}},
		{"X|[]", []string{"--title", "X", "-e", scriptLinha, dat}},
		{"s|[s.rp]", []string{rp, dat}},
	} {
		args := append([]string{"--format", "html", "--template", modelo}, c.args...)
		if code, out, errb := executa(t, "", args...); code != 0 || out != c.want {
			t.Errorf("%q: code=%d stderr=%q saída=%q, want %q", args, code, errb, out, c.want)
		}
	}
	if _, out, _ := executa(t, "", "--format", "html", "-e", scriptLinha, dat); !strings.Contains(out, "<h1>rptgo</h1>") {
		t.Errorf("modelo padrão sem o título rptgo:\n%s", out)
	}
}
