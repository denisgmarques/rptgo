package report

import (
	"fmt"
	"strings"
	"unicode/utf8"
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

// DefFonteScript e DefFonteDados registram o texto de origem pra contexto de erro. Um texto de
// dados novo descarta o localizador do anterior.
func DefFonteScript(texto string) { FonteScript = dividirLinhas(texto) }
func DefFonteDados(texto string) {
	FonteDados = dividirLinhas(texto)
	localizadorDados = nil
}

// PosicaoDados é onde um registro começa no texto de dados: a linha física (a partir de 1) e a
// coluna, em runas (a partir de 0).
type PosicaoDados struct {
	Linha, Coluna int
}

// localizadorDados acha o começo de um registro; só é chamado quando um erro vai ser mostrado.
var localizadorDados func(registro int) PosicaoDados

// DefLocalizadorDados registra como achar o começo de cada registro (numerado a partir de 1) no
// texto passado a DefFonteDados, e por isso deve ser chamada depois dela. Com nil, a linha é o
// próprio número do registro, na coluna 0.
func DefLocalizadorDados(localiza func(registro int) PosicaoDados) { localizadorDados = localiza }

func posicaoDados(registro int) PosicaoDados {
	if localizadorDados == nil {
		return PosicaoDados{Linha: registro}
	}
	return localizadorDados(registro)
}

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
type Error struct {
	msg    string
	deTipo bool // valor de tipo errado para a operação: para o relatório mesmo numa chave do sort
}

func (e *Error) Error() string { return e.msg }

// Erro cria um erro sem linha de origem associada (a maioria dos erros de execução/semântica).
func Erro(formato string, args ...any) error {
	return &Error{msg: fmt.Sprintf(formato, args...)}
}

// erroDeTipo cria um erro de valor com o tipo errado para a operação.
func erroDeTipo(formato string, args ...any) error {
	return &Error{msg: fmt.Sprintf(formato, args...), deTipo: true}
}

// ErroLinhaScript cria um erro anexando a linha do SCRIPT (.rp) correspondente.
func ErroLinhaScript(linha int, formato string, args ...any) error {
	msg := fmt.Sprintf("linha %d: %s", linha, fmt.Sprintf(formato, args...))
	return &Error{msg: comLinha(msg, linha, FonteScript)}
}

// larguraTrecho é o máximo de caracteres de uma linha (do script ou dos dados) mostrados num erro
// que corta as linhas longas.
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

// larguraMarcadorDados é o máximo de "^" sob uma linha de dados.
const larguraMarcadorDados = 40

// trechoDados monta a linha de dados como aparece num erro, "  N | texto", sem o marcador. Uma
// linha de até larguraTrecho caracteres (sem contar o fim de linha) sai inteira e cortada é false.
// De uma linha maior saem no máximo larguraTrecho caracteres a partir da coluna do registro, com
// "..." nos lados cortados: inicio é a posição, na linha montada, do primeiro caractere do
// registro, e tam é quantos caracteres da linha original foram mostrados. ok é false quando a
// linha não existe no texto de dados.
func trechoDados(pos PosicaoDados) (linha string, inicio, tam int, cortada, ok bool) {
	if pos.Linha <= 0 || pos.Linha > len(FonteDados) {
		return "", 0, 0, false, false
	}
	texto := FonteDados[pos.Linha-1]
	prefixo := fmt.Sprintf("  %d | ", pos.Linha)
	semFim := strings.TrimSuffix(texto, "\r")
	if utf8.RuneCountInString(semFim) <= larguraTrecho {
		return prefixo + texto, len(prefixo), 0, false, true
	}
	runas := []rune(semFim)
	ini := min(max(pos.Coluna, 0), len(runas))
	fim := min(ini+larguraTrecho, len(runas))
	antes, depois := "", ""
	if ini > 0 {
		antes = "..."
	}
	if fim < len(runas) {
		depois = "..."
	}
	return prefixo + antes + string(runas[ini:fim]) + depois, len(prefixo) + len(antes), fim - ini, true, true
}

// ErroLinhaDados cria um erro anexando a linha do ARQUIVO DE DADOS em que começa o registro de
// número registro (a partir de 1).
func ErroLinhaDados(registro int, formato string, args ...any) error {
	return ErroPosicaoDados(posicaoDados(registro), formato, args...)
}

// ErroPosicaoDados cria um erro de dados na posição dada. Numa linha longa, o marcador fica sob o
// primeiro caractere do registro e não passa do fim do trecho mostrado.
func ErroPosicaoDados(pos PosicaoDados, formato string, args ...any) error {
	msg := fmt.Sprintf(formato, args...)
	linha, inicio, tam, cortada, ok := trechoDados(pos)
	switch {
	case !ok:
		return &Error{msg: msg}
	case !cortada:
		return &Error{msg: comLinha(msg, pos.Linha, FonteDados)}
	}
	marcador := strings.Repeat(" ", inicio) + strings.Repeat("^", max(1, min(larguraMarcadorDados, tam)))
	return &Error{msg: fmt.Sprintf("%s\n%s\n%s", msg, linha, marcador)}
}
