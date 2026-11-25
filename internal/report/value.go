package report

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Value é um valor em tempo de execução: nil, int64, float64, string, time.Time (data) ou
// bool. Não existe um tipo "Value" de fato - é só um alias pra deixar as assinaturas claras
// sobre a intenção, já que o motor é dinamicamente tipado.
type Value = any

// verdade decide se um valor conta como verdadeiro num 'if'/'and'/'or' (nil, 0, "" e false
// são falsos).
func verdade(v Value) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case int64:
		return x != 0
	case float64:
		return x != 0
	case string:
		return x != ""
	default:
		return true
	}
}

// normaliza converte um par (a, b) onde um dos dois pode ser nil, substituindo o nil por um
// valor neutro do MESMO tipo do outro lado (string nula = "", data nula = data mínima,
// numérico nulo = 0), pra permitir a comparação.
func normaliza(a, b Value) (Value, Value) {
	if a == nil && b == nil {
		return a, b
	}
	if a == nil {
		switch b.(type) {
		case string:
			a = ""
		case time.Time:
			a = time.Time{}
		default:
			a = int64(0)
		}
	}
	if b == nil {
		switch a.(type) {
		case string:
			b = ""
		case time.Time:
			b = time.Time{}
		default:
			b = int64(0)
		}
	}
	return a, b
}

func boolParaInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// normalizaBool converte o texto (já sem espaços nas pontas e não vazio) lido de uma coluna
// bool: número float válido (0 = falso, outro = verdadeiro) ou texto cuja primeira letra,
// sem diferenciar maiúsculas, é N/F (falso) ou Y/T (verdadeiro). Qualquer outra coisa é nulo.
func normalizaBool(t string) Value {
	if f, err := strconv.ParseFloat(t, 64); err == nil {
		return f != 0
	}
	switch strings.ToUpper(t[:1]) {
	case "N", "F":
		return false
	case "Y", "T":
		return true
	}
	return nil
}

func paraFloat(v Value) float64 {
	switch x := v.(type) {
	case int64:
		return float64(x)
	case float64:
		return x
	case bool:
		if x {
			return 1
		}
		return 0
	case nil:
		return 0
	default:
		return 0
	}
}

func ehInteiro(v Value) bool {
	_, ok := v.(int64)
	return ok
}

func ehNumero(v Value) bool {
	switch v.(type) {
	case int64, float64:
		return true
	default:
		return false
	}
}

func ehData(v Value) bool {
	_, ok := v.(time.Time)
	return ok
}

// comparar implementa os operadores relacionais (= <> < > <= >=), com a mesma normalização
// de nulos e de tipo (se um lado é string e o outro não, os dois virem string).
func comparar(op string, a, b Value) (bool, error) {
	if x, ok := a.(bool); ok {
		a = boolParaInt(x)
	}
	if x, ok := b.(bool); ok {
		b = boolParaInt(x)
	}
	a, b = normaliza(a, b)
	_, aStr := a.(string)
	_, bStr := b.(string)
	if aStr != bStr {
		a, b = fmt.Sprint(a), fmt.Sprint(b)
	}
	switch x := a.(type) {
	case string:
		y := b.(string)
		return cmpOrdenado(op, x < y, x == y, x > y), nil
	case time.Time:
		y := b.(time.Time)
		return cmpOrdenado(op, x.Before(y), x.Equal(y), x.After(y)), nil
	default:
		fa, fb := paraFloat(a), paraFloat(b)
		return cmpOrdenado(op, fa < fb, fa == fb, fa > fb), nil
	}
}

func cmpOrdenado(op string, menor, igual, maior bool) bool {
	switch op {
	case "=":
		return igual
	case "<>":
		return !igual
	case "<":
		return menor
	case ">":
		return maior
	case "<=":
		return menor || igual
	case ">=":
		return maior || igual
	}
	return false
}

// aritm implementa + - * / sobre números, datas (data-data=dias, data±N=data) e
// concatenação de strings com '+' - mesmas regras.
func aritm(op string, a, b Value) (Value, error) {
	if x, ok := a.(bool); ok {
		a = boolParaInt(x)
	}
	if x, ok := b.(bool); ok {
		b = boolParaInt(x)
	}
	ad, aok := a.(time.Time)
	bd, bok := b.(time.Time)
	if aok || bok {
		switch {
		case op == "-" && aok && bok:
			return int64(ad.Sub(bd).Hours() / 24), nil
		case op == "+" && aok:
			return ad.AddDate(0, 0, int(paraFloat(b))), nil
		case op == "+" && bok:
			return bd.AddDate(0, 0, int(paraFloat(a))), nil
		case op == "-" && aok:
			return ad.AddDate(0, 0, -int(paraFloat(b))), nil
		}
		return nil, Erro("operação inválida com datas")
	}
	as, aIsStr := a.(string)
	bs, bIsStr := b.(string)
	if aIsStr && bIsStr && op == "+" {
		return as + bs, nil
	}
	// com inteiros puros, soma/subtração/multiplicação ficam em int64 (como int+int do Python)
	if ehInteiro(a) || a == nil {
		if ehInteiro(b) || b == nil {
			ai, bi := int64(0), int64(0)
			if a != nil {
				ai = a.(int64)
			}
			if b != nil {
				bi = b.(int64)
			}
			switch op {
			case "+":
				return ai + bi, nil
			case "-":
				return ai - bi, nil
			case "*":
				return ai * bi, nil
			case "/":
				if bi == 0 {
					return nil, Erro("divisão por zero")
				}
				return ai / bi, nil // divisão inteira: trunca a parte fracionária
			case "%":
				if bi == 0 {
					return nil, Erro("divisão por zero")
				}
				return ai % bi, nil
			}
		}
	}
	fa, fb := paraFloat(a), paraFloat(b)
	switch op {
	case "+":
		return fa + fb, nil
	case "-":
		return fa - fb, nil
	case "*":
		return fa * fb, nil
	case "/":
		if fb == 0 {
			return nil, Erro("divisão por zero")
		}
		return fa / fb, nil
	case "%":
		if fb == 0 {
			return nil, Erro("divisão por zero")
		}
		return math.Mod(fa, fb), nil
	}
	return nil, Erro("operador desconhecido: %s", op)
}
