package main

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"rptgo/internal/report"
)

// Onde cada registro começa no texto de dados, por formato de entrada, para a linha mostrada nos
// erros. Os leitores não guardam posições: a busca relê o texto e só acontece quando um erro vai
// ser mostrado. Os registros são numerados a partir de 1, sem contar o cabeçalho pulado. Um
// registro que não é achado fica na posição zero, e o erro sai sem a linha de dados.

// localizadorTexto acha os registros do texto delimitado: um por linha, depois do cabeçalho.
func localizadorTexto(texto, sep, primeiraColuna, cabecalho string) func(int) report.PosicaoDados {
	return func(registro int) report.PosicaoDados {
		primeira, _, _ := strings.Cut(texto, "\n")
		if ehCabecalhoTexto(primeira, sep, primeiraColuna, cabecalho) {
			return report.PosicaoDados{Linha: registro + 1}
		}
		return report.PosicaoDados{Linha: registro}
	}
}

// localizadorCSV acha os registros do CSV, que pode ter linhas em branco e campos entre aspas com
// quebras de linha: vale a primeira linha física do registro.
func localizadorCSV(texto string, delim rune, primeiraColuna, cabecalho string) func(int) report.PosicaoDados {
	return func(registro int) report.PosicaoDados {
		r := novoLeitorCSV(texto, delim)
		alvo := registro // número do registro contando o cabeçalho
		for n := 1; ; n++ {
			reg, err := r.Read()
			if err != nil {
				return report.PosicaoDados{}
			}
			if n == 1 && ehCabecalhoCSV(reg, primeiraColuna, cabecalho) {
				alvo++
			}
			if n == alvo {
				linha, _ := r.FieldPos(0)
				return report.PosicaoDados{Linha: linha}
			}
		}
	}
}

// localizadorJSON acha os registros do JSON.
func localizadorJSON(texto string) func(int) report.PosicaoDados {
	return func(registro int) report.PosicaoDados { return posicaoJSON(texto, registro-1) }
}

// posicaoJSON acha onde começa o item de índice item (a partir de 0), contando os itens de todos
// os valores do texto, com os arrays abertos.
func posicaoJSON(texto string, item int) report.PosicaoDados {
	corpo := strings.TrimPrefix(texto, bom)
	dec := json.NewDecoder(strings.NewReader(corpo))
	// proximo é onde começa o próximo valor: pula os brancos e a vírgula que o separa do anterior
	proximo := func() int {
		i := int(dec.InputOffset())
		for i < len(corpo) && strings.IndexByte(" \t\r\n,", corpo[i]) >= 0 {
			i++
		}
		return i
	}
	var bruto json.RawMessage
	n := 0
	for {
		ini := proximo()
		if ini >= len(corpo) {
			return report.PosicaoDados{}
		}
		if corpo[ini] != '[' {
			if err := dec.Decode(&bruto); err != nil {
				return report.PosicaoDados{}
			}
			if n == item {
				return posicaoNoTexto(texto, corpo, ini)
			}
			n++
			continue
		}
		if _, err := dec.Token(); err != nil { // '['
			return report.PosicaoDados{}
		}
		for dec.More() {
			ini = proximo()
			if err := dec.Decode(&bruto); err != nil {
				return report.PosicaoDados{}
			}
			if n == item {
				return posicaoNoTexto(texto, corpo, ini)
			}
			n++
		}
		if _, err := dec.Token(); err != nil { // ']'
			return report.PosicaoDados{}
		}
	}
}

// posicaoNoTexto converte o byte i de corpo (o texto sem o BOM) em linha e coluna do texto.
func posicaoNoTexto(texto, corpo string, i int) report.PosicaoDados {
	antes := corpo[:i]
	pos := report.PosicaoDados{
		Linha:  strings.Count(antes, "\n") + 1,
		Coluna: utf8.RuneCountInString(antes[strings.LastIndexByte(antes, '\n')+1:]),
	}
	if pos.Linha == 1 && len(corpo) < len(texto) {
		pos.Coluna++ // o BOM faz parte da primeira linha mostrada
	}
	return pos
}
