package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func lerArquivo(t *testing.T, caminho string) string {
	t.Helper()
	b, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// O README usa marcadores em comentários HTML (invisíveis na página) antes de blocos de código:
//
//	<!-- file: caminho -->   o bloco seguinte é idêntico ao arquivo
//	<!-- run -->             o bloco seguinte é um comando ./rptgo ...
//	<!-- expect -->          o bloco seguinte é a saída do último <!-- run -->
//	<!-- run-only -->        o bloco seguinte é um comando que deve terminar com código 0
//
// Assim os scripts, os dados e as saídas mostrados no README nunca ficam diferentes do que o
// programa faz (RF-12).
type blocoReadme struct {
	marcador string // "file", "run", "expect", "run-only"
	arg      string // caminho (para "file")
	texto    string
	linha    int
}

// paginasMarcadas são os .md cujos blocos de código são conferidos: o README, a página de cada
// exemplo e a página da saída HTML.
func paginasMarcadas(t *testing.T) []string {
	t.Helper()
	paginas := []string{"../../README.md", "../../SAIDA-HTML.md"}
	exemplos, err := filepath.Glob("../../examples/*/README.md")
	if err != nil {
		t.Fatal(err)
	}
	return append(paginas, exemplos...)
}

func blocosMarcados(t *testing.T, caminho string) []blocoReadme {
	t.Helper()
	linhas := strings.Split(lerArquivo(t, caminho), "\n")
	var blocos []blocoReadme
	pend, arg := "", ""
	for i := 0; i < len(linhas); i++ {
		l := strings.TrimSpace(linhas[i])
		switch {
		case strings.HasPrefix(l, "<!-- file: ") && strings.HasSuffix(l, " -->"):
			pend, arg = "file", strings.TrimSuffix(strings.TrimPrefix(l, "<!-- file: "), " -->")
		case l == "<!-- run -->":
			pend, arg = "run", ""
		case l == "<!-- expect -->":
			pend, arg = "expect", ""
		case l == "<!-- run-only -->":
			pend, arg = "run-only", ""
		case strings.HasPrefix(l, "```") && pend != "":
			ini := i
			var corpo []string
			for i++; i < len(linhas) && !strings.HasPrefix(strings.TrimSpace(linhas[i]), "```"); i++ {
				corpo = append(corpo, linhas[i])
			}
			blocos = append(blocos, blocoReadme{pend, arg, strings.Join(corpo, "\n"), ini + 1})
			pend, arg = "", ""
		}
	}
	return blocos
}

// argsDoComando transforma "./rptgo --opt "a b" x \ continuação > saida" nos argumentos do
// programa: junta linhas continuadas com '\', respeita aspas simples e duplas e descarta o redirecionamento.
func argsDoComando(t *testing.T, cmd string) []string {
	t.Helper()
	cmd = strings.ReplaceAll(cmd, "\\\n", " ")
	var args []string
	var atual strings.Builder
	temAtual := false
	var aspas rune // aspa que está aberta (' ou "), 0 se nenhuma
	for _, r := range cmd {
		switch {
		case (r == '"' || r == '\'') && (aspas == 0 || aspas == r):
			if aspas == 0 {
				aspas = r
			} else {
				aspas = 0
			}
			temAtual = true
		case (r == ' ' || r == '\n' || r == '\t') && aspas == 0:
			if temAtual {
				args = append(args, atual.String())
				atual.Reset()
				temAtual = false
			}
		default:
			atual.WriteRune(r)
			temAtual = true
		}
	}
	if temAtual {
		args = append(args, atual.String())
	}
	if len(args) == 0 || args[0] != "./rptgo" {
		t.Fatalf("o comando deveria começar com ./rptgo: %q", cmd)
	}
	args = args[1:]
	for i, a := range args {
		if a == ">" { // redirecionamento do shell: não faz parte dos argumentos
			args = args[:i]
			break
		}
	}
	for i, a := range args {
		if strings.HasPrefix(a, "examples/") {
			args[i] = "../../" + a
		}
	}
	return args
}

func TestReadmeBlocosConferem(t *testing.T) {
	var files, runs, expects, onlys int
	for _, pagina := range paginasMarcadas(t) {
		nome := strings.TrimPrefix(pagina, "../../")
		var ultimoCmd []string
		for _, b := range blocosMarcados(t, pagina) {
			onde := nome + ":" + itoa(b.linha)
			switch b.marcador {
			case "file":
				files++
				if want := strings.TrimRight(lerArquivo(t, "../../"+b.arg), "\n"); b.texto != want {
					t.Errorf("%s: o bloco difere do arquivo %s\n--- página ---\n%s\n--- arquivo ---\n%s", onde, b.arg, b.texto, want)
				}
			case "run":
				runs++
				ultimoCmd = argsDoComando(t, b.texto)
			case "expect":
				expects++
				if ultimoCmd == nil {
					t.Fatalf("%s: <!-- expect --> sem <!-- run --> antes", onde)
				}
				code, out, errb := executa(t, "", ultimoCmd...)
				if code != 0 {
					t.Fatalf("%s: %v saiu com %d: %s", onde, ultimoCmd, code, errb)
				}
				if got := strings.TrimRight(out, "\n"); got != b.texto {
					t.Errorf("%s: a saída documentada mudou para %v\n--- gerada ---\n%s\n--- documentada ---\n%s", onde, ultimoCmd, got, b.texto)
				}
				ultimoCmd = nil
			case "run-only":
				onlys++
				args := argsDoComando(t, b.texto)
				if code, _, errb := executa(t, "", args...); code != 0 {
					t.Errorf("%s: %v saiu com %d: %s", onde, args, code, errb)
				}
			}
		}
	}
	// o README e as páginas são a vitrine do projeto: precisam ter os quatro tipos de bloco
	if files < 15 || runs < 12 || expects != runs || onlys < 4 {
		t.Errorf("poucos blocos verificados: file=%d run=%d expect=%d run-only=%d", files, runs, expects, onlys)
	}
}

func TestReadmeImagensExistem(t *testing.T) {
	var todo strings.Builder
	for _, pagina := range paginasMarcadas(t) {
		todo.WriteString(lerArquivo(t, pagina))
	}
	for _, img := range []string{"docs/images/invoice.png", "docs/images/purchase-orders.png", "docs/images/payroll.png"} {
		if !strings.Contains(todo.String(), img) {
			t.Errorf("nenhuma página referencia %s", img)
		}
		if b, err := os.ReadFile("../../" + img); err != nil || len(b) < 1000 || string(b[1:4]) != "PNG" {
			t.Errorf("%s ausente ou inválida: %v", img, err)
		}
	}
	// o README mostra a fatura no topo e aponta para a referência e para os exemplos
	readme := lerArquivo(t, "../../README.md")
	for _, want := range []string{"](docs/images/invoice.png)", "](REFERENCIA.md)", "](#exemplos)"} {
		if !strings.Contains(readme, want) {
			t.Errorf("o README não contém %s", want)
		}
	}
}

// ---------------------------------------------------------------------------------------------
// Exemplos autocontidos (REFERENCIA.md): cada exemplo é um script, os dados e a saída esperada,
// todos dentro do próprio .md, e o teste roda o script sobre os dados e compara a saída.
//
//	<!-- script -->          o bloco seguinte é um script .rp
//	<!-- data -->            o bloco seguinte são os dados (um registro por linha)
//	<!-- output: OPÇÕES -->  o bloco seguinte é a saída esperada; OPÇÕES são argumentos do rptgo
//	                         (--format csv ...) e/ou VAR=valor para o ambiente
//	<!-- error: OPÇÕES -->   como output, mas o bloco é a mensagem de erro esperada (stderr)
//
// A saída é comparada sem os brancos do fim das linhas, sem as linhas em branco do começo e do
// fim e sem a indentação comum (a margem esquerda do script), para o texto ficar enxuto.

func normalizaDidatico(s string) string {
	linhas := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i, l := range linhas {
		linhas[i] = strings.TrimRight(l, " \t")
	}
	for len(linhas) > 0 && linhas[0] == "" {
		linhas = linhas[1:]
	}
	for len(linhas) > 0 && linhas[len(linhas)-1] == "" {
		linhas = linhas[:len(linhas)-1]
	}
	min := -1
	for _, l := range linhas {
		if l == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, " "))
		if min < 0 || n < min {
			min = n
		}
	}
	if min > 0 {
		for i, l := range linhas {
			if len(l) >= min {
				linhas[i] = l[min:]
			}
		}
	}
	return strings.Join(linhas, "\n")
}

type exemploDidatico struct {
	script, dados string
	temDados      bool
}

func TestReferenciaExemplos(t *testing.T) {
	linhas := strings.Split(lerArquivo(t, "../../REFERENCIA.md"), "\n")
	var atual exemploDidatico
	marcador, arg := "", ""
	verificados := 0
	for i := 0; i < len(linhas); i++ {
		l := strings.TrimSpace(linhas[i])
		if strings.HasPrefix(l, "<!-- ") && strings.HasSuffix(l, " -->") {
			corpo := strings.TrimSuffix(strings.TrimPrefix(l, "<!-- "), " -->")
			nome, resto, _ := strings.Cut(corpo, ":")
			switch nome {
			case "script", "data", "output", "error":
				marcador, arg = nome, strings.TrimSpace(resto)
			}
			continue
		}
		if !strings.HasPrefix(l, "```") || marcador == "" {
			continue
		}
		ini := i + 1
		var corpo []string
		for i++; i < len(linhas) && !strings.HasPrefix(strings.TrimSpace(linhas[i]), "```"); i++ {
			corpo = append(corpo, linhas[i])
		}
		texto := strings.Join(corpo, "\n")
		onde := "REFERENCIA.md:" + itoa(ini)
		switch marcador {
		case "script":
			atual = exemploDidatico{script: texto}
		case "data":
			atual.dados, atual.temDados = texto+"\n", true
		case "output", "error":
			dir := t.TempDir()
			rp := dir + "/s.rp"
			writeFile(t, rp, atual.script+"\n")
			var args []string
			envExtra = nil
			for _, tok := range strings.Fields(arg) {
				if k, _, ok := strings.Cut(tok, "="); ok && k == strings.ToUpper(k) && !strings.HasPrefix(tok, "-") {
					envExtra = append(envExtra, tok)
				} else {
					args = append(args, tok)
				}
			}
			args = append(args, rp)
			if atual.temDados {
				dat := dir + "/d.txt"
				writeFile(t, dat, atual.dados)
				args = append(args, dat)
			} else {
				args = append(args, "-")
			}
			code, out, errb := executa(t, "", args...)
			envExtra = nil
			got := out
			if marcador == "error" {
				got = errb
				if code == 0 {
					t.Errorf("%s: esperava erro, saiu com 0", onde)
				}
			} else if code != 0 {
				t.Errorf("%s: saiu com %d: %s", onde, code, errb)
			}
			if normalizaDidatico(got) != normalizaDidatico(texto) {
				t.Errorf("%s: a saída mudou\n--- gerada ---\n%s\n--- documentada ---\n%s", onde, normalizaDidatico(got), normalizaDidatico(texto))
			}
			verificados++
		}
		marcador = ""
	}
	if verificados < 40 {
		t.Errorf("a referência deveria ter pelo menos 40 exemplos verificados, tem %d", verificados)
	}
}
