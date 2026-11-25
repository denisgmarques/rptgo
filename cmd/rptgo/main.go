// Command rptgo interpreta um script de relatório (seções input/sort/eventos, comandos
// print/skip/page/need/set/if, máscaras de coluna) e os dados de entrada (registros com
// campos separados por um caractere, por padrão '|'), produzindo o texto final paginado.
//
// Uso:
//
//	rptgo [opções] script.rp [entrada.txt | -]  > relatorio.txt
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"rptgo/internal/report"
)

type opcoes struct {
	script   string
	entrada  string
	saida    string
	encoding string
	header   string // auto | sim | nao
	formFeed bool
	newline  string // auto | lf | crlf
	strict   bool
	check    bool
	formato  string // text | csv | json
	box      bool
}

func main() {
	if len(os.Args) > 0 && os.Args[0] != "" {
		nomePrograma = filepath.Base(os.Args[0])
	}
	os.Exit(executar(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func executar(args []string, entradaPadrao io.Reader, saidaPadrao, erroPadrao io.Writer) int {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			fmt.Fprint(saidaPadrao, textoAjuda(nomePrograma))
			return 0
		}
	}
	op, posicionais, err := analisaArgs(args)
	if err != nil {
		erroUso(erroPadrao, err)
		return 2
	}
	if len(posicionais) < 1 {
		fmt.Fprintf(erroPadrao, "%s: falta o arquivo de script\n\n", nomePrograma)
		fmt.Fprint(erroPadrao, textoAjuda(nomePrograma))
		return 2
	}
	op.script = posicionais[0]
	op.entrada = "-"
	if len(posicionais) > 1 {
		op.entrada = posicionais[1]
	}

	textoScriptBytes, err := os.ReadFile(op.script)
	if err != nil {
		fmt.Fprintln(erroPadrao, nomePrograma+":", err)
		return 1
	}
	textoScript := decodificar(textoScriptBytes, op.encoding)
	report.DefFonteScript(textoScript)
	toks, err := report.Tokenize(textoScript, op.strict)
	if err != nil {
		fmt.Fprintln(erroPadrao, nomePrograma+":", err)
		return 1
	}
	script, err := report.Parse(toks)
	if err != nil {
		fmt.Fprintln(erroPadrao, nomePrograma+":", err)
		return 1
	}
	if op.check {
		fmt.Fprintln(saidaPadrao, "No syntax errors were found in the report script.")
		return 0
	}

	var leitorDados io.Reader = entradaPadrao
	if op.entrada != "-" {
		f, err := os.Open(op.entrada)
		if err != nil {
			fmt.Fprintln(erroPadrao, nomePrograma+":", err)
			return 1
		}
		defer f.Close()
		leitorDados = f
	}
	dadosBytes, err := io.ReadAll(leitorDados)
	if err != nil {
		fmt.Fprintln(erroPadrao, nomePrograma+":", err)
		return 1
	}
	dados := decodificar(dadosBytes, op.encoding)
	report.DefFonteDados(dados)

	registros := leRegistros(dados, script.Separator, script.Input[0].Name, op.header, len(script.Input))

	it, err := report.NovoInterp(script, registros, op.formFeed)
	if err != nil {
		fmt.Fprintln(erroPadrao, nomePrograma+":", err)
		return 1
	}
	var texto string
	switch op.formato {
	case "csv", "json":
		// CSV/JSON não executam nenhuma seção de evento: só 'input' e 'sort' contam.
		var buf bytes.Buffer
		escreve := escreveCSV
		if op.formato == "json" {
			escreve = escreveJSON
		}
		if err := escreve(&buf, script, it.LinhasOrdenadas()); err != nil {
			fmt.Fprintln(erroPadrao, nomePrograma+":", err)
			return 1
		}
		texto = buf.String()
	default:
		texto, err = it.Run()
		if err != nil {
			fmt.Fprintln(erroPadrao, nomePrograma+":", err)
			return 1
		}
	}

	if op.box {
		texto = converteCaixas(texto)
	}
	crlf := op.newline == "crlf" || (op.newline == "auto" && runtime.GOOS == "windows")
	if crlf {
		texto = strings.ReplaceAll(texto, "\n", "\r\n")
	}

	saidaBytes := codificar(texto, op.encoding)
	if op.saida != "" {
		return escreveArquivo(op.saida, saidaBytes, erroPadrao)
	}
	saidaPadrao.Write(saidaBytes)
	return 0
}

// decodificar/codificar dão suporte a --encoding: utf-8 (padrão, usado ao pé da letra),
// latin-1/iso-8859-1 (mapeamento direto byte<->rune) e cp850 (tabela em cp850.go, comum em
// impressoras matriciais). Qualquer outro valor é tratado como utf-8.
func decodificar(b []byte, enc string) string {
	switch strings.ToLower(enc) {
	case "cp850":
		r := make([]rune, len(b))
		for i, c := range b {
			r[i] = cp850ParaRune[c]
		}
		return string(r)
	case "latin-1", "latin1", "iso-8859-1", "iso8859-1":
		r := make([]rune, len(b))
		for i, c := range b {
			r[i] = rune(c)
		}
		return string(r)
	default:
		return string(b)
	}
}

func codificar(s string, enc string) []byte {
	switch strings.ToLower(enc) {
	case "cp850":
		b := make([]byte, 0, len(s))
		for _, r := range s {
			c, ok := cp850ParaByte[r]
			if !ok {
				c = '?'
			}
			b = append(b, c)
		}
		return b
	case "latin-1", "latin1", "iso-8859-1", "iso8859-1":
		b := make([]byte, 0, len(s))
		for _, r := range s {
			if r > 255 {
				r = '?'
			}
			b = append(b, byte(r))
		}
		return b
	default:
		return []byte(s)
	}
}

func escreveArquivo(caminho string, texto []byte, erroPadrao io.Writer) int {
	f, err := os.Create(caminho)
	if err != nil {
		fmt.Fprintln(erroPadrao, nomePrograma+":", err)
		return 1
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	if _, err := w.Write(texto); err != nil {
		fmt.Fprintln(erroPadrao, nomePrograma+":", err)
		return 1
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintln(erroPadrao, nomePrograma+":", err)
		return 1
	}
	return 0
}

// analisaArgs faz um parsing manual simples (em vez do pacote flag) pra aceitar opções e
// posicionais misturados em qualquer ordem.
func analisaArgs(args []string) (opcoes, []string, error) {
	op := opcoes{encoding: "utf-8", header: "auto", newline: "auto", formato: "text"}
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		next := func() (string, error) {
			i++
			if i >= len(args) {
				return "", fmt.Errorf("falta valor para %s", a)
			}
			return args[i], nil
		}
		switch a {
		case "-o", "--saida", "--output":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			op.saida = v
		case "--encoding":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			if !contem(valoresEncoding, strings.ToLower(v)) {
				return op, nil, fmt.Errorf("valor inválido para --encoding: %s (use utf-8, latin-1 ou cp850)", v)
			}
			op.encoding = v
		case "--header":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			if !contem(valoresHeader, v) {
				return op, nil, fmt.Errorf("valor inválido para --header: %s (use auto, sim ou nao)", v)
			}
			op.header = v
		case "--form-feed":
			op.formFeed = true
		case "--newline":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			if !contem(valoresNewline, v) {
				return op, nil, fmt.Errorf("valor inválido para --newline: %s (use auto, lf ou crlf)", v)
			}
			op.newline = v
		case "--format":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			if !contem(valoresFormato, v) {
				return op, nil, fmt.Errorf("valor inválido para --format: %s (use text, csv ou json)", v)
			}
			op.formato = v
		case "--box":
			op.box = true
		case "--strict":
			op.strict = true
		case "--check", "-r":
			op.check = true
		default:
			if strings.HasPrefix(a, "-") && a != "-" {
				return op, nil, fmt.Errorf("opção desconhecida: %s", a)
			}
			pos = append(pos, a)
		}
	}
	if op.box && op.formato != "text" {
		return op, nil, fmt.Errorf("--box só vale com --format text")
	}
	if op.box && contem([]string{"latin-1", "latin1", "iso-8859-1", "iso8859-1"}, strings.ToLower(op.encoding)) {
		return op, nil, fmt.Errorf("--box não funciona com --encoding %s (sem caracteres de desenho de caixa); use utf-8 ou cp850", op.encoding)
	}
	if op.formato != "text" && op.formFeed {
		return op, nil, fmt.Errorf("--format %s não pode ser combinado com --form-feed", op.formato)
	}
	return op, pos, nil
}

// leRegistros separa o texto de entrada em registros (um por linha, campos pelo separador
// configurado). Uma linha totalmente vazia conta como um registro de 1 campo vazio (gera
// "phase error" se não bater com o input).
func leRegistros(texto, sep, primeiraColuna, cabecalho string, ncols int) [][]string {
	linhas := strings.Split(texto, "\n")
	if n := len(linhas); n > 0 && linhas[n-1] == "" {
		linhas = linhas[:n-1]
	}
	if len(linhas) > 0 {
		primeiraPalavra := strings.ToLower(strings.TrimSpace(strings.SplitN(linhas[0], sep, 2)[0]))
		if cabecalho == "sim" || (cabecalho == "auto" && primeiraPalavra == primeiraColuna) {
			linhas = linhas[1:]
		}
	}
	regs := make([][]string, 0, len(linhas))
	for _, ln := range linhas {
		ln = strings.TrimSuffix(ln, "\r")
		campos := strings.Split(ln, sep)
		if len(campos) == ncols+1 && campos[len(campos)-1] == "" {
			campos = campos[:len(campos)-1]
		}
		regs = append(regs, campos)
	}
	return regs
}
