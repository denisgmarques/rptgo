package main

import "strings"

// Conversão de "caixas ASCII" (+---+ e |) em caracteres de desenho de linha. O filtro olha
// a página como uma grade de caracteres: cada '+' vira canto ou junção conforme os vizinhos
// (traço à esquerda/direita, barra acima/abaixo); '-' e '|' só viram linha quando fazem
// parte de uma (têm vizinho de linha), então "2026-06", "a - b" ou um '|' solto no texto
// não são alterados.

const (
	vizN = 1 << iota // vizinho de cima liga
	vizE
	vizS
	vizW
)

// caixaPorVizinhos mapeia a combinação de vizinhos de um '+' para o caractere de desenho.
var caixaPorVizinhos = map[int]rune{
	vizE | vizS:               '┌',
	vizW | vizS:               '┐',
	vizN | vizE:               '└',
	vizN | vizW:               '┘',
	vizN | vizS:               '│',
	vizE | vizW:               '─',
	vizN | vizE | vizS:        '├',
	vizN | vizS | vizW:        '┤',
	vizE | vizS | vizW:        '┬',
	vizN | vizE | vizW:        '┴',
	vizN | vizE | vizS | vizW: '┼',
}

func ehHoriz(r rune) bool { return r == '-' || r == '+' || r == '─' }
func ehVert(r rune) bool  { return r == '|' || r == '+' || r == '│' }

// converteCaixasLinhas converte as linhas de uma página; as linhas podem ter tamanhos
// diferentes (o que falta conta como espaço).
func converteCaixasLinhas(linhas []string) []string {
	grade := make([][]rune, len(linhas))
	for i, l := range linhas {
		grade[i] = []rune(l)
	}
	em := func(l, c int) rune {
		if l < 0 || l >= len(grade) || c < 0 || c >= len(grade[l]) {
			return ' '
		}
		return grade[l][c]
	}
	saida := make([]string, len(linhas))
	for l, linha := range grade {
		out := make([]rune, len(linha))
		for c, r := range linha {
			out[c] = r
			switch r {
			case '+':
				v := 0
				if ehVert(em(l-1, c)) {
					v |= vizN
				}
				if ehHoriz(em(l, c+1)) {
					v |= vizE
				}
				if ehVert(em(l+1, c)) {
					v |= vizS
				}
				if ehHoriz(em(l, c-1)) {
					v |= vizW
				}
				if novo, ok := caixaPorVizinhos[v]; ok {
					out[c] = novo
				}
			case '-':
				if ehHoriz(em(l, c-1)) || ehHoriz(em(l, c+1)) {
					out[c] = '─'
				}
			case '|':
				if ehVert(em(l-1, c)) || ehVert(em(l+1, c)) {
					out[c] = '│'
				}
			}
		}
		saida[l] = string(out)
	}
	return saida
}

// converteCaixas converte um texto inteiro; páginas separadas por \f (modo --form-feed) são
// convertidas separadamente, para uma caixa não "atravessar" a quebra de página.
func converteCaixas(texto string) string {
	paginas := strings.Split(texto, "\f")
	for i, p := range paginas {
		paginas[i] = strings.Join(converteCaixasLinhas(strings.Split(p, "\n")), "\n")
	}
	return strings.Join(paginas, "\f")
}
