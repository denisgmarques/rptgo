package main

import (
	"os"
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

func blocosMarcados(t *testing.T) []blocoReadme {
	t.Helper()
	linhas := strings.Split(lerArquivo(t, "../../README.md"), "\n")
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
	blocos := blocosMarcados(t)
	var files, runs, expects, onlys int
	var ultimoCmd []string
	for _, b := range blocos {
		onde := "README.md:" + itoa(b.linha)
		switch b.marcador {
		case "file":
			files++
			if want := strings.TrimRight(lerArquivo(t, "../../"+b.arg), "\n"); b.texto != want {
				t.Errorf("%s: o bloco difere do arquivo %s\n--- README ---\n%s\n--- arquivo ---\n%s", onde, b.arg, b.texto, want)
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
				t.Errorf("%s: a saída documentada mudou para %v\n--- gerada ---\n%s\n--- README ---\n%s", onde, ultimoCmd, got, b.texto)
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
	// o README é a vitrine do projeto: precisa ter os quatro tipos de bloco
	if files < 6 || runs < 5 || expects != runs || onlys < 2 {
		t.Errorf("poucos blocos verificados: file=%d run=%d expect=%d run-only=%d", files, runs, expects, onlys)
	}
}

func TestReadmeImagensExistem(t *testing.T) {
	readme := lerArquivo(t, "../../README.md")
	for _, img := range []string{"docs/images/invoice.png", "docs/images/purchase-orders.png"} {
		if !strings.Contains(readme, "]("+img+")") {
			t.Errorf("o README não referencia %s", img)
		}
		if b, err := os.ReadFile("../../" + img); err != nil || len(b) < 1000 || string(b[1:4]) != "PNG" {
			t.Errorf("%s ausente ou inválida: %v", img, err)
		}
	}
}
