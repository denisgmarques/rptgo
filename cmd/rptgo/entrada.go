package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"rptgo/internal/report"
)

// Fontes de dados de entrada (--input): texto delimitado pelo separador do script (padrão),
// CSV (RFC 4180, com aspas) e JSON (array de objetos ou um objeto por linha). Todas viram a
// mesma lista de registros de campos em texto, que o motor converte conforme o 'input' do
// script; por isso datas, números e bool seguem as mesmas regras em qualquer fonte.
const (
	entradaTexto = "delimited"
	entradaCSV   = "csv"
	entradaJSON  = "json"
)

var valoresInput = []string{entradaTexto, entradaCSV, entradaJSON}

// leRegistrosCSV lê CSV com o delimitador dado. Campos entre aspas podem ter o delimitador,
// aspas dobradas e quebras de linha. O cabeçalho segue --header (sim | nao | auto, que compara
// o primeiro campo com o nome da primeira coluna do input).
func leRegistrosCSV(texto string, delim rune, primeiraColuna, cabecalho string) ([][]string, error) {
	r := csv.NewReader(strings.NewReader(strings.TrimPrefix(texto, "\xEF\xBB\xBF")))
	r.Comma = delim
	r.FieldsPerRecord = -1 // a conferência de colunas é do motor ("phase error")
	r.LazyQuotes = true
	regs, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV inválido: %v", err)
	}
	if len(regs) > 0 {
		primeira := strings.ToLower(strings.TrimSpace(regs[0][0]))
		if cabecalho == "sim" || (cabecalho == "auto" && primeira == primeiraColuna) {
			regs = regs[1:]
		}
	}
	return regs, nil
}

// leRegistrosJSON lê um array JSON de objetos, ou objetos soltos (um por linha ou em
// sequência). Cada coluna do input busca a chave de mesmo nome (primeiro exata, depois sem
// diferenciar maiúsculas); chave ausente ou null vira campo vazio (nulo). Valores simples
// (texto, número, true/false) viram o texto do campo; objeto ou array dentro do registro é erro.
// Uma coluna do input que não aparece em nenhum registro é erro, pois quase sempre é erro de
// digitação no nome.
func leRegistrosJSON(texto string, colunas []string) ([][]string, error) {
	cols := make([]report.InputCol, len(colunas))
	for i, n := range colunas {
		cols[i] = report.InputCol{Name: n, Type: "string"}
	}
	return leRegistrosJSONDec(texto, cols, false)
}

// leRegistrosJSONDec é leRegistrosJSON com o tipo das colunas do input e, se decimalVirgula,
// com os valores que vêm como texto JSON ("R$ 54.390,00", "1.250,50") em colunas numéricas
// convertidos para o formato interno (54390.00). Números JSON de verdade nunca são alterados.
func leRegistrosJSONDec(texto string, cols []report.InputCol, decimalVirgula bool) ([][]string, error) {
	colunas := make([]string, len(cols))
	for i, c := range cols {
		colunas[i] = c.Name
	}
	dec := json.NewDecoder(strings.NewReader(strings.TrimPrefix(texto, "\xEF\xBB\xBF")))
	dec.UseNumber()
	var objs []map[string]any
	// primeiro valor: array de objetos ou um objeto
	for {
		var v any
		if err := dec.Decode(&v); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("JSON inválido: %v", err)
		}
		switch x := v.(type) {
		case map[string]any:
			objs = append(objs, x)
		case []any:
			for i, it := range x {
				o, ok := it.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("JSON: o item %d do array não é um objeto", i+1)
				}
				objs = append(objs, o)
			}
		default:
			return nil, fmt.Errorf("JSON: esperado um array de objetos ou objetos, veio %T", v)
		}
	}
	vista := make([]bool, len(colunas))
	regs := make([][]string, len(objs))
	for i, o := range objs {
		reg := make([]string, len(colunas))
		for j, nome := range colunas {
			v, ok := o[nome]
			if !ok {
				for k, val := range o {
					if strings.EqualFold(k, nome) {
						v, ok = val, true
						break
					}
				}
			}
			if !ok {
				continue
			}
			vista[j] = true
			switch x := v.(type) {
			case nil:
			case string:
				if decimalVirgula && tiposNumericos[cols[j].Type] {
					x = limpaNumeroBR(x)
				}
				reg[j] = x
			case json.Number:
				reg[j] = x.String()
			case bool:
				if x {
					reg[j] = "true"
				} else {
					reg[j] = "false"
				}
			default:
				return nil, fmt.Errorf("JSON: registro %d, campo %q não é um valor simples (texto, número ou booleano)", i+1, nome)
			}
		}
		regs[i] = reg
	}
	if len(regs) > 0 {
		for j, nome := range colunas {
			if !vista[j] {
				return nil, fmt.Errorf("JSON: a coluna %q do input não aparece em nenhum registro", nome)
			}
		}
	}
	return regs, nil
}

// textoFonteJSON devolve o texto que serve de contexto para mensagens de erro (uma linha por
// registro), no mesmo formato dos outros tipos de fonte.
func textoFonteJSON(texto string) string { return string(bytes.TrimSpace([]byte(texto))) }

var tiposNumericos = map[string]bool{"numeric": true, "integer": true, "long": true, "int": true,
	"float": true, "amount": true, "double": true, "real": true, "huge": true}

// ajustaDecimalEntrada converte, nas colunas numéricas, o formato com vírgula decimal e ponto de
// milhar (1.250,50) para o formato interno (1250.50). É o formato do CSV exportado pelo Excel
// com configuração regional brasileira; colunas de texto e de data não são tocadas.
func ajustaDecimalEntrada(regs [][]string, cols []report.InputCol) {
	for _, reg := range regs {
		for j := range reg {
			if j >= len(cols) || !tiposNumericos[cols[j].Type] {
				continue
			}
			reg[j] = limpaNumeroBR(reg[j])
		}
	}
}

var (
	reMoedaIni  = regexp.MustCompile(`(?i)^\s*([+-]?)\s*(R\$|US\$|BRL|USD|EUR|[$€£¥])?\s*`)
	reMoedaFim  = regexp.MustCompile(`(?i)\s*(BRL|USD|EUR|[$€£¥])?\s*$`)
	reSoNumeros = regexp.MustCompile(`^[+-]?[0-9.,]+$`)
)

// tiraMoeda remove um símbolo de moeda no começo ou no fim ("R$ 54.390,00" vira "54.390,00").
// Se o que sobra não for só um número, devolve o texto original, para o erro de valor inválido
// continuar aparecendo em vez de ser escondido.
func tiraMoeda(s string) string {
	t := reMoedaFim.ReplaceAllString(reMoedaIni.ReplaceAllString(s, "$1"), "")
	if !reSoNumeros.MatchString(t) {
		return s
	}
	return t
}

// limpaNumeroBR converte um número com vírgula decimal, ponto de milhar e símbolo de moeda
// opcional ("R$ 1.250,50") para o formato interno (1250.50).
func limpaNumeroBR(s string) string {
	t := tiraMoeda(s)
	if t == s {
		t = strings.TrimSpace(s)
	}
	t = strings.ReplaceAll(t, ".", "")
	return strings.Replace(t, ",", ".", 1)
}

// detectaDelimitadorCSV escolhe o delimitador de um CSV olhando o começo do arquivo: para cada
// candidato (; , tab |) lê até 30 registros com as regras de aspas do CSV e conta quantos têm
// exatamente ncols campos (o número de colunas do input do script). Vence o candidato com mais
// registros certos; se nenhum acerta, vence o que dá mais registros com o mesmo número de campos
// (>= 2) do primeiro, e o motor reporta a diferença de colunas. Sem pistas, vírgula.
// Aspas não precisam ser detectadas: o leitor de CSV trata campos com e sem aspas no mesmo arquivo.
func detectaDelimitadorCSV(texto string, ncols int) rune {
	const limite = 64 << 10
	amostra := strings.TrimPrefix(texto, "\xEF\xBB\xBF")
	if len(amostra) > limite {
		amostra = amostra[:limite]
		if i := strings.LastIndex(amostra, "\n"); i > 0 {
			amostra = amostra[:i]
		}
	}
	candidatos := []rune{';', ',', '\t', '|'}
	melhor, melhorScore := ',', 0
	for pass := 0; pass < 2 && melhorScore == 0; pass++ {
		for _, d := range candidatos {
			r := csv.NewReader(strings.NewReader(amostra))
			r.Comma = d
			r.FieldsPerRecord = -1
			r.LazyQuotes = true
			score, primeiro := 0, -1
			for i := 0; i < 30; i++ {
				reg, err := r.Read()
				if err != nil {
					break
				}
				if primeiro < 0 {
					primeiro = len(reg)
				}
				if pass == 0 && ncols >= 2 && len(reg) == ncols {
					score++
				}
				if pass == 1 && primeiro >= 2 && len(reg) == primeiro {
					score++
				}
			}
			if score > melhorScore {
				melhor, melhorScore = d, score
			}
		}
	}
	return melhor
}

var (
	reDecVirgula = regexp.MustCompile(`^-?(\d{1,3}(\.\d{3})+|\d+),\d+$`)
	reDecPonto   = regexp.MustCompile(`^-?\d+\.\d+$`)
	reMilhar     = regexp.MustCompile(`^-?\d{1,3}(\.\d{3})+$`)
)

// temDecimalVirgula diz se alguma coluna numérica tem um valor na forma 1250,50 / 1.250,50 (com
// ou sem símbolo de moeda). No JSON isso só pode vir de um valor entre aspas, então basta um para
// escolher a vírgula, mesmo que outros valores sejam números JSON de verdade.
func temDecimalVirgula(regs [][]string, cols []report.InputCol) bool {
	for _, reg := range regs {
		for j, f := range reg {
			if j < len(cols) && tiposNumericos[cols[j].Type] && reDecVirgula.MatchString(tiraMoeda(strings.TrimSpace(f))) {
				return true
			}
		}
	}
	return false
}

// detectaDecimalEntrada decide o separador decimal olhando as colunas numéricas: vírgula se algum
// valor tem a forma 1250,50 ou 1.250,50 e nenhum tem a forma 1250.50 (um ponto com 3 casas depois,
// como em 1.250, é ambíguo e não conta); caso contrário, ponto. Com delimitador vírgula, a vírgula
// não pode ser decimal.
func detectaDecimalEntrada(regs [][]string, cols []report.InputCol, delim rune) rune {
	if delim == ',' {
		return '.'
	}
	virgula, ponto := 0, 0
	for _, reg := range regs {
		for j, f := range reg {
			if j >= len(cols) || !tiposNumericos[cols[j].Type] {
				continue
			}
			f = tiraMoeda(strings.TrimSpace(f))
			switch {
			case reDecVirgula.MatchString(f):
				virgula++
			case reDecPonto.MatchString(f) && !reMilhar.MatchString(f):
				ponto++
			}
		}
	}
	if virgula > 0 && ponto == 0 {
		return ','
	}
	return '.'
}
