package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"rptgo/internal/report"
)

// formatoISO é a data ISO-8601 sem hora, igual em CSV e JSON para date e ldate. Ignora de
// propósito DATEFMT/CENTURY_CUTOFF e a diferença de largura de ano do modo texto.
const formatoISO = "2006-01-02"

// Modos de aspas do CSV (--csv-quote).
const (
	aspasMinimas = "minimal" // só quando o campo precisa (RFC 4180): padrão
	aspasTexto   = "strings" // todo valor de texto (e o cabeçalho) entre aspas; números, datas e bool não
	aspasTodas   = "all"     // todos os campos entre aspas
)

// csvOpcoes são as opções de apresentação do CSV. O padrão (csvPadrao) é o do RFC 4180:
// vírgula, aspas só quando necessárias, ponto decimal e cabeçalho.
type csvOpcoes struct {
	Delim        rune   // delimitador de campos
	Aspas        string // aspasMinimas, aspasTexto ou aspasTodas
	Decimal      rune   // separador decimal dos números com fração: '.' ou ','
	SemCabecalho bool
}

var csvPadrao = csvOpcoes{Delim: ',', Aspas: aspasMinimas, Decimal: '.'}

// valorCSV devolve o texto de um valor convertido para saída CSV e se ele é "texto" (para o modo
// de aspas strings): nulo vira vazio, data vira AAAA-MM-DD, números usam a menor representação
// decimal que preserva o valor (com o separador decimal escolhido), e bool vira true/false.
// Nunca usa máscaras 'using' nem a formatação de datas do modo texto.
func valorCSV(v report.Value, decimal rune) (txt string, ehTexto bool, err error) {
	switch x := v.(type) {
	case nil:
		return "", false, nil
	case time.Time:
		return x.Format(formatoISO), false, nil
	case report.Hora:
		return x.ISO(), false, nil
	case int64:
		return strconv.FormatInt(x, 10), false, nil
	case float64:
		t := strconv.FormatFloat(x, 'f', -1, 64)
		if decimal != '.' {
			t = strings.Replace(t, ".", string(decimal), 1)
		}
		return t, false, nil
	case bool:
		return strconv.FormatBool(x), false, nil
	case string:
		return x, true, nil
	}
	return "", false, fmt.Errorf("tipo de valor inesperado: %T", v)
}

// campoCSV aplica as aspas do RFC 4180 (aspas internas dobradas). Entre aspas quando forcar,
// ou quando o campo tem o delimitador, aspas, quebra de linha ou começa com branco.
func campoCSV(f string, delim rune, forcar bool) string {
	precisa := forcar || strings.ContainsRune(f, delim) || strings.ContainsAny(f, "\"\r\n") ||
		(f != "" && (f[0] == ' ' || f[0] == '\t'))
	if !precisa {
		return f
	}
	return `"` + strings.ReplaceAll(f, `"`, `""`) + `"`
}

// escreveCSV grava um cabeçalho com os nomes das colunas do 'input' (a menos que
// opt.SemCabecalho) e uma linha por registro, na ordem recebida. O delimitador, as aspas e o
// separador decimal vêm de opt; o separador do arquivo de entrada não conta. Só lê s.Input:
// largura, comprimento e margens do script não têm efeito aqui.
func escreveCSV(w io.Writer, s *report.Script, rows []report.Row, opt csvOpcoes) error {
	var sb strings.Builder
	linha := func(campos []string, forcar []bool) {
		if len(campos) == 1 && campos[0] == "" {
			sb.WriteString(`""`) // um registro de um só campo vazio não pode virar linha em branco
		} else {
			for i, c := range campos {
				if i > 0 {
					sb.WriteRune(opt.Delim)
				}
				sb.WriteString(campoCSV(c, opt.Delim, forcar[i]))
			}
		}
		sb.WriteString("\n")
	}
	n := len(s.Input)
	campos := make([]string, n)
	forcar := make([]bool, n)
	if !opt.SemCabecalho {
		for j, c := range s.Input {
			campos[j] = c.Name
			forcar[j] = opt.Aspas != aspasMinimas // o cabeçalho é texto
		}
		linha(campos, forcar)
	}
	for _, row := range rows {
		for j := range s.Input {
			t, ehTexto, err := valorCSV(row[j], opt.Decimal)
			if err != nil {
				return err
			}
			campos[j] = t
			forcar[j] = opt.Aspas == aspasTodas || (opt.Aspas == aspasTexto && ehTexto)
		}
		linha(campos, forcar)
	}
	_, err := io.WriteString(w, sb.String())
	return err
}
