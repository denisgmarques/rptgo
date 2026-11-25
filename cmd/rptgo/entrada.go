package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
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
			f := strings.ReplaceAll(strings.TrimSpace(reg[j]), ".", "")
			reg[j] = strings.Replace(f, ",", ".", 1)
		}
	}
}
