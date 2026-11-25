package main

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// nomePrograma é o nome pelo qual o binário foi chamado (rptgo, rpt, ...), usado no uso e
// nas mensagens de erro. main() ajusta a partir de os.Args[0].
var nomePrograma = "rptgo"

// Valores aceitos, compartilhados entre a validação de analisaArgs e o texto de ajuda.
var (
	valoresHeader   = []string{"auto", "sim", "nao"}
	valoresNewline  = []string{"auto", "lf", "crlf"}
	valoresFormato  = []string{"text", "csv", "json", "html"}
	valoresCSVQuote = []string{aspasMinimas, aspasTexto, aspasTodas}
	valoresEncoding = []string{"utf-8", "utf8", "latin-1", "latin1", "iso-8859-1", "iso8859-1", "cp850"}
)

func contem(lista []string, v string) bool {
	for _, x := range lista {
		if x == v {
			return true
		}
	}
	return false
}

// textoAjuda devolve a ajuda no formato usual de comandos Linux (Uso / Opções / Exemplos).
func textoAjuda(nome string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Uso: %s [OPÇÃO]... SCRIPT.rp [ENTRADA.txt | -]\n\n", nome)
	b.WriteString("Executa um script de relatório (.rp) sobre um arquivo de dados (texto com campos separados\n")
	b.WriteString("por '|', CSV ou JSON) e escreve o relatório paginado, PDF/HTML, CSV ou JSON.\n")
	b.WriteString("Sem ENTRADA.txt, ou com '-', lê os dados da entrada padrão.\n\n")
	b.WriteString("Opções:\n")
	b.WriteString("  -o, --saida, --output ARQUIVO  grava no ARQUIVO em vez da saída padrão\n")
	b.WriteString("      --format FORMATO           formato da saída: text (padrão), csv, json ou html;\n")
	b.WriteString("                                 csv, json e html não combinam com --form-feed\n")
	b.WriteString("      --encoding CODIFICAÇÃO     leitura e escrita: utf-8 (padrão), latin-1 ou cp850\n")
	b.WriteString("      --header MODO              1ª linha dos dados é cabeçalho: auto (padrão),\n")
	b.WriteString("                                 sim ou nao\n")
	b.WriteString("      --form-feed                separa páginas com \\f em vez de completar com linhas\n")
	b.WriteString("                                 em branco até o 'length' do script\n")
	b.WriteString("      --newline MODO             fim de linha: auto (padrão), lf ou crlf\n")
	b.WriteString("      --box                      troca caixas ASCII (+---+ e |) por caracteres de desenho\n")
	b.WriteString("                                 de linha; com --format text (utf-8 ou cp850) ou html\n")
	b.WriteString("      --strict                   comentários /* */ abrem e fecham na mesma linha; com\n")
	b.WriteString("                                 'before report', o 'header' não roda na 1ª página\n")
	b.WriteString("\nFonte dos dados (padrão: texto com campos separados por '|', ou pelo 'separator' do script):\n")
	b.WriteString("      --input FONTE              delimited (padrão), csv ou json. csv lê aspas do RFC 4180;\n")
	b.WriteString("                                 json lê um array de objetos (ou um objeto por linha) e liga\n")
	b.WriteString("                                 cada coluna do input à chave de mesmo nome\n")
	b.WriteString("      --input-delimiter CARACTERE  delimitador do CSV de entrada: ; , | tab ou auto (padrão: detecta\n")
	b.WriteString("                                 pelo número de colunas do input; aspas são sempre aceitas)\n")
	b.WriteString("      --input-decimal CARACTERE  separador decimal dos números na entrada: . (padrão), , ou auto\n")
	b.WriteString("                                 (com , lê 1.250,50 e R$ 1.250,50 como 1250.50; auto escolhe pelos\n")
	b.WriteString("                                 valores; em JSON vale só para valores entre aspas);\n")
	b.WriteString("                                 CSV do Excel brasileiro: --input csv --input-decimal auto\n")
	b.WriteString("                                 --encoding latin-1\n")
	b.WriteString("\nOpções de --format csv (padrão: RFC 4180, vírgula, aspas só quando necessárias):\n")
	b.WriteString("      --csv-delimiter CARACTERE  delimitador dos campos: ; | tab ... (padrão: vírgula)\n")
	b.WriteString("      --csv-quote MODO           minimal (padrão), strings (todo texto e o cabeçalho entre\n")
	b.WriteString("                                 aspas) ou all (todos os campos)\n")
	b.WriteString("      --csv-decimal CARACTERE    separador decimal dos números: . (padrão) ou ,\n")
	b.WriteString("      --csv-no-header            não escreve a linha de cabeçalho\n")
	b.WriteString("\nOpções de --format html (a saída é sempre UTF-8, sem linhas em branco de preenchimento):\n")
	b.WriteString("      --template ARQUIVO         modelo HTML (padrão: folhas A4 embutidas)\n")
	b.WriteString("      --var NOME=VALOR           define {{NOME}} no modelo; pode repetir\n")
	b.WriteString("      --logo ARQUIVO             imagem (png, jpg, gif, svg, webp) em {{Logo}}\n")
	b.WriteString("      --title TEXTO              {{Title}} (padrão: nome do script)\n")
	b.WriteString("      --lines N                  linhas de texto por página (padrão: length - margens\n")
	b.WriteString("                                 superior e inferior do script)\n")
	b.WriteString("  Modelo: {{#Pages}}...{{/Pages}} repete por página. Variáveis: Title, Script, Date, Time,\n")
	b.WriteString("  Columns, PageCount, Logo, as de --var e, dentro do bloco, PageNumber e Text.\n\n")
	b.WriteString("Outras opções:\n")
	b.WriteString("  -r, --check                    só verifica a sintaxe do script, sem gerar relatório\n")
	b.WriteString("  -h, --help                     mostra esta ajuda e sai\n\n")
	b.WriteString("Variáveis de ambiente: DATEFMT, CENTURY_CUTOFF, TIMEFMT, WIDTH, PAGELN, SEPARATOR,\n")
	b.WriteString("RADIXSEP, CURRSYM e RPT_NOW (fixa o relógio, ex.: 2026-11-24 17:47).\n\n")
	b.WriteString("Código de saída: 0 sucesso; 1 erro no script, nos dados ou de E/S; 2 uso inválido.\n\n")
	b.WriteString("Exemplos:\n")
	fmt.Fprintf(&b, "  %s relatorio.rp dados.txt > relatorio.txt\n", nome)
	fmt.Fprintf(&b, "  %s --format json relatorio.rp dados.txt\n", nome)
	fmt.Fprintf(&b, "  cat dados.txt | %s -o saida.txt relatorio.rp -\n", nome)
	fmt.Fprintf(&b, "  %s --format csv --csv-delimiter ';' --csv-quote strings --csv-decimal ',' relatorio.rp dados.txt\n", nome)
	fmt.Fprintf(&b, "  %s --format html --box --template modelo.html --logo logo.svg relatorio.rp dados.txt > saida.html\n", nome)
	return b.String()
}

// erroUso escreve a mensagem de erro de uso e a dica padrão, como os comandos Linux.
func erroUso(w io.Writer, err error) {
	fmt.Fprintf(w, "%s: %v\n", nomePrograma, err)
	fmt.Fprintf(w, "Tente '%s --help' para mais informações.\n", nomePrograma)
}

// delimitadorCSV interpreta o valor de --csv-delimiter: um caractere (por exemplo ; ou |) ou um
// dos nomes comma, semicolon, tab e pipe.
func delimitadorCSV(v string) (rune, error) {
	nomes := map[string]rune{"comma": ',', "semicolon": ';', "tab": '\t', "pipe": '|', "\\t": '\t'}
	if r, ok := nomes[v]; ok {
		return r, nil
	}
	rs := []rune(v)
	if len(rs) != 1 || rs[0] == '"' || rs[0] == '\r' || rs[0] == '\n' || rs[0] == utf8.RuneError {
		return 0, fmt.Errorf("valor inválido para --csv-delimiter: %q (use um caractere, como ; ou |, ou comma, semicolon, tab, pipe)", v)
	}
	return rs[0], nil
}

// csvOpcoesValidas junta as opções --csv-* e confere se combinam entre si.
func (op opcoes) csvOpcoesValidas() (csvOpcoes, error) {
	c := csvPadrao
	if op.csvDelim != "" {
		d, err := delimitadorCSV(op.csvDelim)
		if err != nil {
			return c, err
		}
		c.Delim = d
	}
	if op.csvQuote != "" {
		c.Aspas = op.csvQuote
	}
	if op.csvDec != "" {
		c.Decimal = []rune(op.csvDec)[0]
	}
	c.SemCabecalho = op.csvSemCab
	if c.Delim == c.Decimal {
		return c, fmt.Errorf("--csv-delimiter e --csv-decimal não podem ser o mesmo caractere (%q)", string(c.Delim))
	}
	return c, nil
}

// csvOpcoes devolve as opções já validadas por analisaArgs.
func (op opcoes) csvOpcoes() csvOpcoes {
	c, _ := op.csvOpcoesValidas()
	return c
}
