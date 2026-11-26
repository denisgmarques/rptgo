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
		case Hora:
			a = Hora(0)
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
		case Hora:
			b = Hora(0)
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
	// Só números finitos e sem '_': "nan" e "inf" são texto (nan começa com N, logo falso;
	// inf não começa com Y/T/N/F, logo nulo) e "1_0" não é um número válido.
	if f, err := strconv.ParseFloat(t, 64); err == nil && !math.IsNaN(f) && !math.IsInf(f, 0) && !strings.Contains(t, "_") {
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

// textoOuNulo devolve o texto de v (nulo vale "") e false se v não for texto.
func textoOuNulo(v Value) (string, bool) {
	switch x := v.(type) {
	case nil:
		return "", true
	case string:
		return x, true
	}
	return "", false
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
// de nulos e de tipo (se um lado é string e o outro não, os dois viram string).
func comparar(op string, a, b Value) (bool, error) {
	if x, ok := a.(bool); ok {
		a = boolParaInt(x)
	}
	if x, ok := b.(bool); ok {
		b = boolParaInt(x)
	}
	a, b = normaliza(a, b)
	ha, aH := a.(Hora)
	hb, bH := b.(Hora)
	if aH || bH {
		if !aH || !bH {
			return false, Erro("não é possível comparar uma hora com outro tipo")
		}
		return cmpOrdenado(op, ha < hb, ha == hb, ha > hb), nil
	}
	_, aStr := a.(string)
	_, bStr := b.(string)
	if aStr != bStr {
		a, b = fmt.Sprint(a), fmt.Sprint(b)
	}
	if ehData(a) != ehData(b) {
		return false, erroDeTipo("não é possível comparar uma data com um número")
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
// concatenação de strings com '+'.
func aritm(op string, a, b Value) (Value, error) {
	if op == "/+" { // concatenação indireta: tira os brancos finais do lado esquerdo
		as, aok := textoOuNulo(a)
		bs, bok := textoOuNulo(b)
		if !aok || !bok {
			return nil, Erro("o operador /+ só pode ser usado com texto")
		}
		return strings.TrimRight(as, " ") + bs, nil
	}
	if x, ok := a.(bool); ok {
		a = boolParaInt(x)
	}
	if x, ok := b.(bool); ok {
		b = boolParaInt(x)
	}
	if ah, bh, ok := horasEmOperacao(a, b); ok {
		return aritmHora(op, a, b, ah, bh)
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
					return nil, erroDivisaoPorZero()
				}
				return ai / bi, nil // divisão inteira: trunca a parte fracionária
			case "%":
				if bi == 0 {
					return nil, erroDivisaoPorZero()
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
			return nil, erroDivisaoPorZero()
		}
		return fa / fb, nil
	case "%":
		if fb == 0 {
			return nil, erroDivisaoPorZero()
		}
		return math.Mod(fa, fb), nil
	}
	return nil, Erro("operador desconhecido: %s", op)
}

// horasEmOperacao diz se algum dos operandos é uma Hora.
func horasEmOperacao(a, b Value) (Hora, Hora, bool) {
	ha, aok := a.(Hora)
	hb, bok := b.(Hora)
	return ha, hb, aok || bok
}

// aritmHora implementa a aritmética de horas do dia: hora - hora dá os minutos entre elas;
// hora +/- número soma ou subtrai minutos (dando a volta em 24h, pois é uma hora do dia);
// somar duas horas não existe.
func aritmHora(op string, a, b Value, ha, hb Hora) (Value, error) {
	_, aH := a.(Hora)
	_, bH := b.(Hora)
	minutos := func(v Value) (float64, bool) {
		switch v.(type) {
		case nil, int64, float64:
			return paraFloat(v), true
		}
		return 0, false
	}
	soma := func(h Hora, min float64) Value {
		s := (int64(h) + int64(min*60)) % 86400
		if s < 0 {
			s += 86400
		}
		return Hora(s)
	}
	switch {
	case aH && bH && op == "-":
		return (int64(ha) - int64(hb)) / 60, nil
	case aH && !bH && (op == "+" || op == "-"):
		if m, ok := minutos(b); ok {
			if op == "-" {
				m = -m
			}
			return soma(ha, m), nil
		}
	case bH && !aH && op == "+":
		if m, ok := minutos(a); ok {
			return soma(hb, m), nil
		}
	}
	return nil, Erro("operação inválida com horas")
}
