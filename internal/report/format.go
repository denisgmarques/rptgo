package report

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// incrementarDigitos soma 1 a uma sequência de dígitos decimais (string), tratando o
// transporte (carry) corretamente, inclusive quando estoura o dígito mais à esquerda
// (ex.: "999" -> "1000"). Usado pra arredondar "para cima" sem depender de ponto flutuante.
func incrementarDigitos(s string) string {
	b := []byte(s)
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] < '9' {
			b[i]++
			return string(b)
		}
		b[i] = '0'
	}
	return "1" + string(b)
}

// arredondaMeioParaCima arredonda |v| pra `casas` decimais (meio-pra-cima), devolvendo os
// dígitos inteiros (sem zeros à esquerda) e os `casas` dígitos fracionários, separados do
// sinal. Opera sobre a representação decimal mais curta do float pra evitar ruído de ponto flutuante binário.
func arredondaMeioParaCima(v float64, casas int) (digInt string, digFrac string, neg bool) {
	neg = v < 0
	s := strconv.FormatFloat(v, 'f', -1, 64) // representação decimal mais curta que ida-e-volta
	s = strings.TrimPrefix(s, "-")
	ip, fp, temPonto := strings.Cut(s, ".")
	if !temPonto {
		fp = ""
	}
	var arredondar byte = '0'
	if len(fp) > casas {
		arredondar = fp[casas]
	}
	fracMantida := fp
	if len(fracMantida) < casas {
		fracMantida += strings.Repeat("0", casas-len(fracMantida))
	} else {
		fracMantida = fracMantida[:casas]
	}
	combinado := ip + fracMantida
	if arredondar >= '5' {
		combinado = incrementarDigitos(combinado)
	}
	if casas == 0 {
		return strings.TrimLeft(combinado, "0"), "", neg
	}
	divisao := len(combinado) - casas
	digInt = strings.TrimLeft(combinado[:divisao], "0")
	digFrac = combinado[divisao:]
	return digInt, digFrac, neg
}

var rePrintfNum = regexp.MustCompile(`^%(-?)(\d*)(?:\.(\d+))?l{0,2}([dfi])$`)
var reStrMask = regexp.MustCompile(`^(\d+)(r?)$`)

// FmtMascara formata um valor numérico (int64/float64) segundo uma máscara `using` do
// relatório: ou um formato %printf (ex. '%ld', '%7.2f'), ou a notação própria com '#', '&', '*', '+', '-', '$', '(', ')', ',' e '.'.
func FmtMascara(v Value, mask string) (string, error) {
	if v == nil {
		v = int64(0)
	}
	if strings.HasPrefix(mask, "%") {
		m := rePrintfNum.FindStringSubmatch(mask)
		if m == nil {
			return "", Erro("formato inválido: %q", mask)
		}
		esq, larg, prec, tipo := m[1], m[2], m[3], m[4]
		spec := "%" + esq + larg
		if prec != "" {
			spec += "." + prec
		}
		if tipo == "d" || tipo == "i" {
			return fmt.Sprintf(spec+"d", int64(paraFloat(v))), nil
		}
		return fmt.Sprintf(spec+"f", paraFloat(v)), nil
	}

	m := []byte(mask)
	visto := false
	for i, ch := range m {
		if ch == '&' {
			visto = true
		} else if visto && strings.ContainsRune("#$*+-(", rune(ch)) {
			m[i] = '&'
		}
	}
	mascara := string(m)
	ip, fp, _ := strings.Cut(mascara, ".")
	casas := 0
	for _, ch := range fp {
		if ch != ')' && ch != ',' {
			casas++
		}
	}
	digInt, digFrac, neg := arredondaMeioParaCima(paraFloat(v), casas)

	res := make([]byte, len(ip))
	restantes := []byte(digInt)
	sinalImpresso, moedaImpressa := false, false
	moeda := cfg("CURRSYM", "$")
	// separador de milhar (TRIADSEP, padrão vírgula): ocupa uma posição da máscara, então só o
	// primeiro byte conta; vazio vira espaço.
	triada := byte(',')
	if v := cfg("TRIADSEP", ","); v == "" {
		triada = ' '
	} else {
		triada = v[0]
	}
	for pos := len(ip) - 1; pos >= 0; pos-- {
		ch := ip[pos]
		switch {
		case strings.ContainsRune("#&*-+$(", rune(ch)):
			if len(restantes) > 0 {
				res[pos] = restantes[len(restantes)-1]
				restantes = restantes[:len(restantes)-1]
				continue
			}
			switch ch {
			case '&':
				res[pos] = '0'
			case '#':
				res[pos] = ' '
			case '*':
				res[pos] = '*'
			case '-':
				if neg && !sinalImpresso {
					res[pos], sinalImpresso = '-', true
				} else {
					res[pos] = ' '
				}
			case '+':
				if !sinalImpresso {
					if neg {
						res[pos] = '-'
					} else {
						res[pos] = '+'
					}
					sinalImpresso = true
				} else {
					res[pos] = ' '
				}
			case '$':
				if !moedaImpressa {
					// CURRSYM pode ter mais de 1 caractere; nesse caso o restante da máscara
					// cuida das outras posições (só o 1º caractere do símbolo é usado).
					if moeda == "" {
						res[pos] = ' '
					} else {
						res[pos] = moeda[0]
					}
					moedaImpressa = true
				} else {
					res[pos] = ' '
				}
			case '(':
				if neg && !sinalImpresso {
					res[pos], sinalImpresso = '(', true
				} else {
					res[pos] = ' '
				}
			}
		case ch == ',':
			if len(restantes) > 0 {
				res[pos] = triada
			} else {
				res[pos] = ' '
			}
		case ch == ')':
			if neg {
				res[pos] = ')'
			} else {
				res[pos] = ' '
			}
		default:
			res[pos] = ch
		}
	}
	var sb strings.Builder
	for i := len(restantes) - 1; i >= 0; i-- { // sobra: dígitos que não couberam na máscara
		sb.WriteByte(restantes[i])
	}
	sb.Write(res)
	if casas > 0 {
		sb.WriteString(cfg("RADIXSEP", "."))
		sb.WriteString(digFrac)
	} else if strings.Contains(mascara, ".") {
		sb.WriteString(cfg("RADIXSEP", "."))
	}
	for _, ch := range fp {
		if ch == ')' {
			if neg {
				sb.WriteByte(')')
			} else {
				sb.WriteByte(' ')
			}
		}
	}
	return sb.String(), nil
}

// QuebraTexto implementa a máscara `using 'Nr'` de colunas string/text: quebra o texto entre
// palavras em linhas de largura N (a opção 'r' de margem "esgarçada" não muda o algoritmo,
// que já é por palavra).
func QuebraTexto(txt string, larg int) []string {
	var linhas []string
	for _, par := range strings.Split(txt, "\n") {
		atual := ""
		for _, w := range strings.Split(par, " ") {
			for len(w) > larg {
				if atual != "" {
					linhas = append(linhas, atual)
					atual = ""
				}
				linhas = append(linhas, w[:larg])
				w = w[larg:]
			}
			switch {
			case atual == "":
				atual = w
			case len(atual)+1+len(w) <= larg:
				atual += " " + w
			default:
				linhas = append(linhas, atual)
				atual = w
			}
		}
		linhas = append(linhas, atual)
	}
	if len(linhas) == 0 {
		return []string{""}
	}
	return linhas
}

// ehMascaraString reconhece o formato 'N' ou 'Nr' usado em colunas string/text.
func ehMascaraString(mask string) (larg int, ok bool) {
	m := reStrMask.FindStringSubmatch(mask)
	if m == nil {
		return 0, false
	}
	n, _ := strconv.Atoi(m[1])
	return n, true
}
