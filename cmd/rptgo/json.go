package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"rptgo/internal/report"
)

// valorJSON codifica um valor convertido como JSON: nulo vira null, data vira string
// AAAA-MM-DD, números e bool saem como número/boolean JSON.
func valorJSON(v report.Value) ([]byte, error) {
	switch x := v.(type) {
	case nil:
		return []byte("null"), nil
	case time.Time:
		return json.Marshal(x.Format(formatoISO))
	case report.Hora:
		return json.Marshal(x.ISO())
	case int64, float64, bool, string:
		return json.Marshal(x)
	}
	return nil, fmt.Errorf("tipo de valor inesperado: %T", v)
}

// escreveJSON grava um array JSON com um objeto por registro (um por linha), com as chaves
// na ordem declarada no 'input'. Os objetos são montados à mão: serializar um
// map[string]any com json.Marshal reordenaria as chaves em ordem alfabética e escreveria
// time.Time em RFC 3339, em vez de AAAA-MM-DD.
func escreveJSON(w io.Writer, s *report.Script, rows []report.Row) error {
	nomes := make([][]byte, len(s.Input))
	for j, c := range s.Input {
		b, err := json.Marshal(c.Name)
		if err != nil {
			return err
		}
		nomes[j] = b
	}
	if len(rows) == 0 {
		_, err := io.WriteString(w, "[]\n")
		return err
	}
	var buf []byte
	buf = append(buf, "[\n"...)
	for i, row := range rows {
		buf = append(buf, '{')
		for j := range s.Input {
			if j > 0 {
				buf = append(buf, ',')
			}
			v, err := valorJSON(row[j])
			if err != nil {
				return err
			}
			buf = append(buf, nomes[j]...)
			buf = append(buf, ':')
			buf = append(buf, v...)
		}
		buf = append(buf, '}')
		if i < len(rows)-1 {
			buf = append(buf, ',')
		}
		buf = append(buf, '\n')
	}
	buf = append(buf, "]\n"...)
	_, err := w.Write(buf)
	return err
}
