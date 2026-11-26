package report

import (
	"regexp"
	"strconv"
	"strings"
)

// TokKind identifica o tipo de um token léxico.
type TokKind int

const (
	TokNum TokKind = iota
	TokStr
	TokHora // constante de hora sem aspas, ex. 14:30
	TokData // constante de data sem aspas, ex. 02/08/2026 (duas barras distingue de divisão)
	TokRep  // caractere repetido, ex. 27[-]
	TokID
	TokOp
	TokEOF
)

// Token é uma unidade léxica com a posição de origem, usada tanto pelo parser quanto
// para montar mensagens de erro com contexto.
type Token struct {
	Kind    TokKind
	Text    string // texto bruto (identificador, operador, string decodificada, etc.)
	Num     float64
	IsInt   bool
	RepN    int    // para TokRep: quantidade de repetições
	RepChar string // para TokRep: o caractere repetido
	Line    int
	Col     int    // posição na linha, em runas, a partir de 0
	Src     string // o trecho como foi escrito no script (com aspas e maiúsculas)
}

var reNum = regexp.MustCompile(`^\d+(\.\d*)?|^\.\d+`)
var reDataLit = regexp.MustCompile(`^\d{1,4}/\d{1,4}/\d{1,4}`)
var reHoraLit = regexp.MustCompile(`^\d{1,2}:\d{2}`)
var reID = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*`)

var ops2 = map[string]bool{"<>": true, "^=": true, "!=": true, "<=": true, ">=": true, "/+": true}

// Tokenize converte o texto do script em uma lista de tokens.
//
// comentarioUmaLinha=true exige que o comentário /* */ abra e feche na mesma linha; por padrão,
// comentários de várias linhas são aceitos (mais prático pra documentar os scripts).
// indexRunas acha a posição (em runas, não em bytes) da primeira ocorrência de sub dentro
// de r. Existe só pra nunca precisar misturar índice de byte com índice de runa.
func indexRunas(r []rune, sub string) int {
	subR := []rune(sub)
	if len(subR) == 0 {
		return 0
	}
	for i := 0; i+len(subR) <= len(r); i++ {
		ok := true
		for j, c := range subR {
			if r[i+j] != c {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

func Tokenize(src string, comentarioUmaLinha bool) ([]Token, error) {
	var toks []Token
	runes := []rune(src)
	n := len(runes)
	i, linha := 0, 1
	inicioLinha, contado := 0, 0

	restante := func(pos int) string { return string(runes[pos:]) }
	// posiciona atualiza a linha atual até a posição dada, contando também as quebras de
	// linha que ficam dentro de comentários e strings.
	posiciona := func(ate int) {
		for ; contado < ate && contado < n; contado++ {
			if runes[contado] == '\n' {
				linha++
				inicioLinha = contado + 1
			}
		}
	}
	// emite acrescenta o token que vai da posição atual até fim.
	emite := func(t Token, fim int) {
		t.Line, t.Col, t.Src = linha, i-inicioLinha, string(runes[i:fim])
		toks = append(toks, t)
		i = fim
	}
	erro := func(tam int, formato string, args ...any) error {
		return erroNaPalavra(linha, i-inicioLinha, tam, formato, args...)
	}
	// restoDaLinha é o tamanho do trecho da posição atual até o último caractere visível da linha.
	restoDaLinha := func() int {
		f := i
		for f < n && runes[f] != '\n' {
			f++
		}
		for f > i && strings.ContainsRune(" \t\r\f", runes[f-1]) {
			f--
		}
		return f - i
	}

	for i < n {
		posiciona(i)
		c := runes[i]
		switch {
		case c == '\n' || c == ' ' || c == '\t' || c == '\r' || c == '\f':
			i++
		case c == '/' && i+1 < n && runes[i+1] == '*':
			// busca "*/" direto no slice de runas (nunca por índice de byte via strings.Index -
			// um comentário com acento tem bytes multibyte, e misturar índice de byte com
			// índice de runa corrompe a posição; já bateu nisso uma vez, por isso o comentário).
			if comentarioUmaLinha {
				fimLinha := i
				for fimLinha < n && runes[fimLinha] != '\n' {
					fimLinha++
				}
				j := indexRunas(runes[i+2:fimLinha], "*/")
				if j < 0 {
					return nil, erro(restoDaLinha(), "comentário precisa abrir e fechar na mesma linha (--strict)")
				}
				i = i + 2 + j + 2
			} else {
				j := indexRunas(runes[i+2:], "*/")
				if j < 0 {
					return nil, erro(restoDaLinha(), "comentário não terminado")
				}
				i = i + 2 + j + 2
			}
		case c == '\'' || c == '"':
			aspas := c
			j := i + 1
			var buf []rune
			for j < n && runes[j] != aspas {
				if runes[j] == '\n' {
					return nil, erro(restoDaLinha(), "string deve estar numa única linha")
				}
				if runes[j] == '\\' && j+1 < n {
					buf = append(buf, runes[j+1])
					j += 2
					continue
				}
				buf = append(buf, runes[j])
				j++
			}
			if j >= n {
				return nil, erro(restoDaLinha(), "string não terminada")
			}
			emite(Token{Kind: TokStr, Text: string(buf)}, j+1)
		case c >= '0' && c <= '9' && reHoraLit.MatchString(restante(i)):
			m := reHoraLit.FindString(restante(i))
			emite(Token{Kind: TokHora, Text: m}, i+len([]rune(m)))
		case c >= '0' && c <= '9' && reDataLit.MatchString(restante(i)):
			m := reDataLit.FindString(restante(i))
			emite(Token{Kind: TokData, Text: m}, i+len([]rune(m)))
		case (c >= '0' && c <= '9') || (c == '.' && i+1 < n && runes[i+1] >= '0' && runes[i+1] <= '9'):
			m := reNum.FindString(restante(i))
			j := i + len([]rune(m))
			if !strings.Contains(m, ".") && j+2 < n && runes[j] == '[' && runes[j+2] == ']' {
				// repetição de caractere: 27[-]
				qtd, _ := strconv.Atoi(m)
				emite(Token{Kind: TokRep, RepN: qtd, RepChar: string(runes[j+1])}, j+3)
				continue
			}
			isInt := !strings.Contains(m, ".")
			var num float64
			if isInt {
				v, _ := strconv.ParseInt(m, 10, 64)
				num = float64(v)
			} else {
				num, _ = strconv.ParseFloat(m, 64)
			}
			emite(Token{Kind: TokNum, Text: m, Num: num, IsInt: isInt}, j)
		case c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z'):
			m := reID.FindString(restante(i))
			emite(Token{Kind: TokID, Text: strings.ToLower(m)}, i+len([]rune(m)))
		default:
			dois := ""
			if i+1 < n {
				dois = string(runes[i : i+2])
			}
			if ops2[dois] {
				emite(Token{Kind: TokOp, Text: dois}, i+2)
			} else if strings.ContainsRune("+-*/%=<>(),[]^", c) {
				emite(Token{Kind: TokOp, Text: string(c)}, i+1)
			} else {
				return nil, erro(1, "caractere não reconhecido: %s", strconv.QuoteRune(c))
			}
		}
	}
	posiciona(n)
	toks = append(toks, Token{Kind: TokEOF, Line: linha})
	return toks, nil
}

