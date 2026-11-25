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

// ErroLinhaDados cria um erro anexando a linha do ARQUIVO DE DADOS correspondente.
func ErroLinhaDados(linha int, formato string, args ...any) error {
	msg := fmt.Sprintf(formato, args...)
	return &Error{msg: comLinha(msg, linha, FonteDados)}
}
