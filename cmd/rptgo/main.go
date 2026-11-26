// Command rptgo interpreta um script de relatório (seções input/sort/eventos, comandos
// print/skip/page/need/set/if, máscaras de coluna) e os dados de entrada (registros com
// campos separados por um caractere, por padrão '|'), produzindo o texto final paginado.
//
// Uso:
//
//	rptgo [opções] script.rp [entrada.txt | -]  > relatorio.txt
//	rptgo [opções] -e 'SCRIPT' [entrada.txt | -]
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"rptgo/internal/report"
)

type opcoes struct {
	script         string
	textoScript    string // -e, --script: o script na própria linha de comando
	temTextoScript bool   // distingue -e "" da ausência da opção
	entrada        string
	saida          string
	encoding       string
	header         string // auto | sim | nao
	formFeed       bool
	newline        string // auto | lf | crlf
	strict         bool
	check          bool
	formato        string // text | csv | json
	box            bool
	input          string   // --input delimited|csv|json
	inputDelim     string   // --input-delimiter
	inputDec       string   // --input-decimal
	csvDelim       string   // --csv-delimiter
	csvQuote       string   // --csv-quote
	csvDec         string   // --csv-decimal
	csvSemCab      bool     // --csv-no-header
	template       string   // --template (html)
	vars           []string // --var NOME=VALOR (html)
	logo           string   // --logo (html)
	titulo         string   // --title (html)
	linhas         int      // --lines (html)
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
	// com -e o script já veio na linha de comando (em UTF-8, sem passar por --encoding) e o
	// único posicional são os dados; sem -e o primeiro posicional é o arquivo do script.
	textoScript := op.textoScript
	if !op.temTextoScript {
		if len(posicionais) < 1 {
			fmt.Fprintf(erroPadrao, "%s: falta o arquivo de script\n\n", nomePrograma)
			fmt.Fprint(erroPadrao, textoAjuda(nomePrograma))
			return 2
		}
		op.script, posicionais = posicionais[0], posicionais[1:]
		textoScriptBytes, err := os.ReadFile(op.script)
		if err != nil {
			fmt.Fprintln(erroPadrao, nomePrograma+":", err)
			return 1
		}
		textoScript = decodificar(textoScriptBytes, op.encoding)
	}
	op.entrada = "-"
	if len(posicionais) > 0 {
		op.entrada = posicionais[0]
	}
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
		fmt.Fprintln(saidaPadrao, "Script sem erros de sintaxe.")
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

	var registros [][]string
	var delimCSV rune
	switch op.input {
	case entradaCSV:
		if op.inputDelim == "" || op.inputDelim == "auto" {
			delimCSV = detectaDelimitadorCSV(dados, len(script.Input))
		} else {
			delimCSV, _ = delimitadorCSV(op.inputDelim)
		}
		delim := delimCSV
		registros, err = leRegistrosCSV(dados, delim, strings.ToLower(script.Input[0].Name), op.header)
	case entradaJSON:
		registros, err = leRegistrosJSONDec(dados, script.Input, op.inputDec == ",")
		if err == nil && op.inputDec == "auto" && temDecimalVirgula(registros, script.Input) {
			registros, err = leRegistrosJSONDec(dados, script.Input, true) // valores como "R$ 1.250,50"
		}
		report.DefFonteDados(textoFonteJSON(dados))
	default:
		registros = leRegistros(dados, script.Separator, script.Input[0].Name, op.header, len(script.Input))
	}
	if err != nil {
		fmt.Fprintln(erroPadrao, nomePrograma+":", err)
		return 1
	}
	decimal := op.inputDec
	if op.input == entradaJSON {
		decimal = "" // o JSON já foi convertido na leitura (só os valores entre aspas)
	}
	if decimal == "auto" {
		decimal = string(detectaDecimalEntrada(registros, script.Input, delimCSV))
	}
	if decimal == "," {
		if op.input == entradaCSV && delimCSV == ',' {
			fmt.Fprintln(erroPadrao, nomePrograma+": o decimal \",\" não pode ser usado com o delimitador vírgula; use --input-delimiter ';'")
			return 1
		}
		ajustaDecimalEntrada(registros, script.Input)
	}

	it, err := report.NovoInterp(script, registros, op.formFeed)
	if it != nil {
		it.Strict = op.strict
	}
	if err != nil {
		fmt.Fprintln(erroPadrao, nomePrograma+":", err)
		return 1
	}
	var texto string
	switch op.formato {
	case "csv", "json":
		// CSV/JSON não executam nenhuma seção de evento: só 'input' e 'sort' contam.
		var buf bytes.Buffer
		escreve := func(w io.Writer, s *report.Script, rows []report.Row) error {
			return escreveCSV(w, s, rows, op.csvOpcoes())
		}
		if op.formato == "json" {
			escreve = escreveJSON
		}
		if err := escreve(&buf, script, it.LinhasOrdenadas()); err != nil {
			fmt.Fprintln(erroPadrao, nomePrograma+":", err)
			return 1
		}
		texto = buf.String()
	case "html":
		paginas, err := it.Paginas(op.linhas)
		if err != nil {
			fmt.Fprintln(erroPadrao, nomePrograma+":", err)
			return 1
		}
		if op.box {
			for i := range paginas {
				paginas[i] = converteCaixasLinhas(paginas[i])
			}
		}
		texto, err = montaHTML(op, paginas)
		if err != nil {
			fmt.Fprintln(erroPadrao, nomePrograma+":", err)
			return 1
		}
	default:
		texto, err = it.Run()
		if err != nil {
			fmt.Fprintln(erroPadrao, nomePrograma+":", err)
			return 1
		}
	}

	if op.box && op.formato == "text" {
		texto = converteCaixas(texto)
	}
	crlf := op.newline == "crlf" || (op.newline == "auto" && runtime.GOOS == "windows")
	if crlf {
		texto = strings.ReplaceAll(texto, "\n", "\r\n")
	}

	saidaBytes := codificar(texto, op.encoding)
	if op.formato == "html" {
		saidaBytes = []byte(texto) // o HTML é sempre UTF-8 (declarado no <meta charset>)
	}
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
		case "-e", "--script":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			if op.temTextoScript {
				return op, nil, fmt.Errorf("-e/--script só pode ser usado uma vez")
			}
			op.textoScript, op.temTextoScript = v, true
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
				return op, nil, fmt.Errorf("valor inválido para --format: %s (use text, csv, json ou html)", v)
			}
			op.formato = v
		case "--box":
			op.box = true
		case "--input":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			if !contem(valoresInput, v) {
				return op, nil, fmt.Errorf("valor inválido para --input: %s (use delimited, csv ou json)", v)
			}
			op.input = v
		case "--input-delimiter":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			if v != "auto" {
				if _, err := delimitadorCSV(v); err != nil {
					return op, nil, fmt.Errorf("--input-delimiter: %v", err)
				}
			}
			op.inputDelim = v
		case "--input-decimal":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			if v != "." && v != "," && v != "auto" {
				return op, nil, fmt.Errorf("valor inválido para --input-decimal: %s (use ., , ou auto)", v)
			}
			op.inputDec = v
		case "--csv-delimiter":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			if _, err := delimitadorCSV(v); err != nil {
				return op, nil, err
			}
			op.csvDelim = v
		case "--csv-quote":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			if !contem(valoresCSVQuote, v) {
				return op, nil, fmt.Errorf("valor inválido para --csv-quote: %s (use minimal, strings ou all)", v)
			}
			op.csvQuote = v
		case "--csv-decimal":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			if v != "." && v != "," {
				return op, nil, fmt.Errorf("valor inválido para --csv-decimal: %s (use . ou ,)", v)
			}
			op.csvDec = v
		case "--csv-no-header":
			op.csvSemCab = true
		case "--template":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			op.template = v
		case "--var":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			if k, _, ok := strings.Cut(v, "="); !ok || !reNomeVar.MatchString(k) {
				return op, nil, fmt.Errorf("valor inválido para --var: %s (use NOME=VALOR, com NOME só de letras, números e _)", v)
			}
			op.vars = append(op.vars, v)
		case "--logo":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			op.logo = v
		case "--title":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			op.titulo = v
		case "--lines":
			v, err := next()
			if err != nil {
				return op, nil, err
			}
			n, convErr := strconv.Atoi(v)
			if convErr != nil || n < 1 {
				return op, nil, fmt.Errorf("valor inválido para --lines: %s (use um número inteiro maior que zero)", v)
			}
			op.linhas = n
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
	if op.temTextoScript && len(pos) > 1 {
		return op, nil, fmt.Errorf("com -e/--script só cabe o arquivo de dados; sobrou: %s", strings.Join(pos[1:], " "))
	}
	if op.box && op.formato != "text" && op.formato != "html" {
		return op, nil, fmt.Errorf("--box só vale com --format text ou html")
	}
	if op.inputDec == "," && op.input == entradaCSV && (op.inputDelim == "," || op.inputDelim == "comma") {
		return op, nil, fmt.Errorf("--input-decimal \",\" e o delimitador do CSV de entrada não podem ser a vírgula; use --input-delimiter ';'")
	}
	if op.inputDelim != "" && op.input != entradaCSV {
		return op, nil, fmt.Errorf("--input-delimiter só vale com --input csv")
	}
	if op.formato != "csv" {
		for opt, usada := range map[string]bool{"--csv-delimiter": op.csvDelim != "", "--csv-quote": op.csvQuote != "",
			"--csv-decimal": op.csvDec != "", "--csv-no-header": op.csvSemCab} {
			if usada {
				return op, nil, fmt.Errorf("%s só vale com --format csv", opt)
			}
		}
	} else if _, err := op.csvOpcoesValidas(); err != nil {
		return op, nil, err
	}
	if op.formato != "html" {
		for opt, usada := range map[string]bool{"--template": op.template != "", "--var": len(op.vars) > 0,
			"--logo": op.logo != "", "--title": op.titulo != "", "--lines": op.linhas > 0} {
			if usada {
				return op, nil, fmt.Errorf("%s só vale com --format html", opt)
			}
		}
	}
	if op.box && op.formato == "text" && contem([]string{"latin-1", "latin1", "iso-8859-1", "iso8859-1"}, strings.ToLower(op.encoding)) {
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

// valorOu devolve v, ou padrao quando v é vazio.
func valorOu(v, padrao string) string {
	if v == "" {
		return padrao
	}
	return v
}
