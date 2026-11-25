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

// Token é uma unidade léxica com a linha de origem, usada tanto pelo parser quanto
// para montar mensagens de erro com contexto.
type Token struct {
	Kind    TokKind
	Text    string // texto bruto (identificador, operador, string decodificada, etc.)
	Num     float64
	IsInt   bool
	RepN    int    // para TokRep: quantidade de repetições
	RepChar string // para TokRep: o caractere repetido
	Line    int
}

var reNum = regexp.MustCompile(`^\d+(\.\d*)?|^\.\d+`)
var reDataLit = regexp.MustCompile(`^\d{1,4}/\d{1,4}/\d{1,4}`)
var reHoraLit = regexp.MustCompile(`^\d{1,2}:\d{2}`)
var reID = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*`)

var ops2 = map[string]bool{"<>": true, "^=": true, "!=": true, "<=": true, ">=": true, "/+": true}

// Tokenize converte o texto do script em uma lista de tokens.
//
// comentarioUmaLinha=true valida um detalhe em que este programa é permissivo: por padrão aceita comentário de /* */ de várias linhas (mais prático pra
// documentar nossos próprios scripts); com essa opção, exige que abra e feche na mesma linha,
// como no modo estrito.
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

	restante := func(pos int) string { return string(runes[pos:]) }

	for i < n {
		c := runes[i]
		switch {
		case c == '\n':
			linha++
			i++
		case c == ' ' || c == '\t' || c == '\r' || c == '\f':
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
					return nil, ErroLinhaScript(linha, "comentário precisa abrir e fechar na mesma linha (--strict)")
				}
				i = i + 2 + j + 2
			} else {
				j := indexRunas(runes[i+2:], "*/")
				if j < 0 {
					return nil, ErroLinhaScript(linha, "comentário não terminado")
				}
				for _, rc := range runes[i : i+2+j] {
					if rc == '\n' {
						linha++
					}
				}
				i = i + 2 + j + 2
			}
		case c == '\'' || c == '"':
			aspas := c
			j := i + 1
			var buf []rune
			for j < n && runes[j] != aspas {
				if runes[j] == '\n' {
					return nil, ErroLinhaScript(linha, "string deve estar numa única linha")
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
				return nil, ErroLinhaScript(linha, "string não terminada")
			}
			toks = append(toks, Token{Kind: TokStr, Text: string(buf), Line: linha})
			i = j + 1
		case c >= '0' && c <= '9' && reHoraLit.MatchString(restante(i)):
			m := reHoraLit.FindString(restante(i))
			toks = append(toks, Token{Kind: TokHora, Text: m, Line: linha})
			i += len([]rune(m))
		case c >= '0' && c <= '9' && reDataLit.MatchString(restante(i)):
			m := reDataLit.FindString(restante(i))
			toks = append(toks, Token{Kind: TokData, Text: m, Line: linha})
			i += len([]rune(m))
		case (c >= '0' && c <= '9') || (c == '.' && i+1 < n && runes[i+1] >= '0' && runes[i+1] <= '9'):
			m := reNum.FindString(restante(i))
			j := i + len([]rune(m))
			if !strings.Contains(m, ".") && j+2 < n && runes[j] == '[' && runes[j+2] == ']' {
				// repetição de caractere: 27[-]
				qtd, _ := strconv.Atoi(m)
				toks = append(toks, Token{Kind: TokRep, RepN: qtd, RepChar: string(runes[j+1]), Line: linha})
				i = j + 3
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
			toks = append(toks, Token{Kind: TokNum, Text: m, Num: num, IsInt: isInt, Line: linha})
			i = j
		case c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z'):
			m := reID.FindString(restante(i))
			toks = append(toks, Token{Kind: TokID, Text: strings.ToLower(m), Line: linha})
			i += len([]rune(m))
		default:
			dois := ""
			if i+1 < n {
				dois = string(runes[i : i+2])
			}
			if ops2[dois] {
				toks = append(toks, Token{Kind: TokOp, Text: dois, Line: linha})
				i += 2
			} else if strings.ContainsRune("+-*/%=<>(),[]^", c) {
				toks = append(toks, Token{Kind: TokOp, Text: string(c), Line: linha})
				i++
			} else {
				return nil, ErroLinhaScript(linha, "caractere não reconhecido: %s", strconv.QuoteRune(c))
			}
		}
	}
	toks = append(toks, Token{Kind: TokEOF, Line: linha})
	return toks, nil
}

