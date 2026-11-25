package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"rptgo/internal/report"
)

// formatoISO é a data ISO-8601 sem hora, igual em CSV e JSON para date e ldate. Ignora de
// propósito DATEFMT/CENTURY_CUTOFF e a diferença de largura de ano do modo texto.
const formatoISO = "2006-01-02"

// valorTexto devolve o texto de um valor convertido para saída CSV: nulo vira vazio, data
// vira AAAA-MM-DD, números usam a menor representação decimal que preserva o valor, e bool
// vira true/false. Nunca usa máscaras 'using' nem a formatação de datas do modo texto.
func valorTexto(v report.Value) (string, error) {
	switch x := v.(type) {
	case nil:
		return "", nil
	case time.Time:
		return x.Format(formatoISO), nil
	case int64:
		return strconv.FormatInt(x, 10), nil
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(x), nil
	case string:
		return x, nil
	}
	return "", fmt.Errorf("tipo de valor inesperado: %T", v)
}

// escreveCSV grava um cabeçalho com os nomes das colunas do 'input' e uma linha por
// registro, na ordem recebida. Usa vírgula como delimitador (não o separador do arquivo de
// entrada) e as aspas do RFC 4180. Só lê s.Input: largura, comprimento e margens do script
// não têm efeito aqui.
func escreveCSV(w io.Writer, s *report.Script, rows []report.Row) error {
	cw := csv.NewWriter(w)
	cab := make([]string, len(s.Input))
	for j, c := range s.Input {
		cab[j] = c.Name
	}
	if err := cw.Write(cab); err != nil {
		return err
	}
	campos := make([]string, len(s.Input))
	for _, row := range rows {
		for j := range s.Input {
			t, err := valorTexto(row[j])
			if err != nil {
				return err
			}
			campos[j] = t
		}
		if err := cw.Write(campos); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
