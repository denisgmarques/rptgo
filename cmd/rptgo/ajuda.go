package main

import (
	"fmt"
	"io"
	"strings"
)

// nomePrograma é o nome pelo qual o binário foi chamado (rptgo, rpt, ...), usado no uso e
// nas mensagens de erro. main() ajusta a partir de os.Args[0].
var nomePrograma = "rptgo"

// Valores aceitos, compartilhados entre a validação de analisaArgs e o texto de ajuda.
var (
	valoresHeader   = []string{"auto", "sim", "nao"}
	valoresNewline  = []string{"auto", "lf", "crlf"}
	valoresFormato  = []string{"text", "csv", "json", "html"}
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
	b.WriteString("Executa um script de relatório (.rp) sobre um arquivo de dados com campos separados\n")
	b.WriteString("por '|' (um registro por linha) e escreve o relatório paginado, ou CSV/JSON.\n")
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
	b.WriteString("      --strict                   exige que comentários /* */ abram e fechem na\n")
	b.WriteString("                                 mesma linha\n")
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
	fmt.Fprintf(&b, "  %s --format html --box --template modelo.html --logo logo.svg relatorio.rp dados.txt > saida.html\n", nome)
	return b.String()
}

// erroUso escreve a mensagem de erro de uso e a dica padrão, como os comandos Linux.
func erroUso(w io.Writer, err error) {
	fmt.Fprintf(w, "%s: %v\n", nomePrograma, err)
	fmt.Fprintf(w, "Tente '%s --help' para mais informações.\n", nomePrograma)
}
