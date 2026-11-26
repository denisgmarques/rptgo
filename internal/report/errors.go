package report

import (
	"fmt"
	"strings"
)

// FonteScript e FonteDados guardam as linhas do script e do arquivo de dados em uso, só pra
// anexar contexto (a linha + um indicador) nas mensagens de erro - não participam da lógica.
var (
	FonteScript []string
	FonteDados  []string
)

func dividirLinhas(texto string) []string {
	ls := strings.Split(texto, "\n")
	if n := len(ls); n > 0 && ls[n-1] == "" {
		ls = ls[:n-1]
	}
	return ls
}

// DefFonteScript e DefFonteDados registram o texto de origem pra contexto de erro.
func DefFonteScript(texto string) { FonteScript = dividirLinhas(texto) }
func DefFonteDados(texto string)  { FonteDados = dividirLinhas(texto) }

func comLinha(msg string, linha int, fonte []string) string {
	if linha <= 0 || linha > len(fonte) {
		return msg
	}
	texto := fonte[linha-1]
	prefixo := fmt.Sprintf("  %d | ", linha)
	n := len(strings.TrimSpace(texto))
	if n > 40 {
		n = 40
	}
	if n < 1 {
		n = 1
	}
	marcador := strings.Repeat(" ", len(prefixo)) + strings.Repeat("^", n)
	return fmt.Sprintf("%s\n%s%s\n%s", msg, prefixo, texto, marcador)
}

// Error é o único tipo de erro do pacote: erros de sintaxe e de execução são tratados igual.
type Error struct{ msg string }

func (e *Error) Error() string { return e.msg }

// Erro cria um erro sem linha de origem associada (a maioria dos erros de execução/semântica).
func Erro(formato string, args ...any) error {
	return &Error{msg: fmt.Sprintf(formato, args...)}
}

// ErroLinhaScript cria um erro anexando a linha do SCRIPT (.rp) correspondente.
func ErroLinhaScript(linha int, formato string, args ...any) error {
	msg := fmt.Sprintf("linha %d: %s", linha, fmt.Sprintf(formato, args...))
	return &Error{msg: comLinha(msg, linha, FonteScript)}
}

// larguraTrecho é o máximo de caracteres de uma linha do script mostrados num erro.
const larguraTrecho = 72

// erroNaPalavra cria um erro de script com o marcador sob a palavra que começa na coluna col
// (em runas, a partir de 0) da linha e ocupa tam runas. De uma linha longa sai só o trecho em
// volta da palavra, com "..." nos lados cortados.
func erroNaPalavra(linha, col, tam int, formato string, args ...any) error {
	msg := fmt.Sprintf("linha %d: %s", linha, fmt.Sprintf(formato, args...))
	if linha <= 0 || linha > len(FonteScript) {
		return &Error{msg: msg}
	}
	texto := []rune(strings.TrimSuffix(FonteScript[linha-1], "\r"))
	n := len(texto)
	col = min(max(col, 0), n)
	ini, fim := 0, n
	if n > larguraTrecho {
		ini = col
		if tam < larguraTrecho { // centraliza a palavra no trecho
			ini = max(0, min(col-(larguraTrecho-tam)/2, n-larguraTrecho))
		}
		fim = min(ini+larguraTrecho, n)
	}
	antes, depois := "", ""
	if ini > 0 {
		antes = "..."
	}
	if fim < n {
		depois = "..."
	}
	prefixo := fmt.Sprintf("  %d | ", linha)
	var marcador strings.Builder
	marcador.WriteString(strings.Repeat(" ", len(prefixo)+len(antes)))
	// os tabs são repetidos para o marcador alinhar com qualquer largura de tabulação
	for _, c := range texto[ini:col] {
		if c == '\t' {
			marcador.WriteByte('\t')
		} else {
			marcador.WriteByte(' ')
		}
	}
	marcador.WriteString(strings.Repeat("^", max(1, min(tam, fim-col))))
	return &Error{msg: fmt.Sprintf("%s\n%s%s%s%s\n%s", msg, prefixo, antes, string(texto[ini:fim]), depois, marcador.String())}
}

// ErroLinhaDados cria um erro anexando a linha do ARQUIVO DE DADOS correspondente.
func ErroLinhaDados(linha int, formato string, args ...any) error {
	msg := fmt.Sprintf(formato, args...)
	return &Error{msg: comLinha(msg, linha, FonteDados)}
}
