package main

import (
	"fmt"
	"strings"
	"testing"
)

// Os erros de dados mostram a primeira linha física do registro, com o número dela, em qualquer
// formato de entrada.

const scriptDados = "input a, n [numeric]\nend\n"

// valorInvalido é a mensagem de um 'abc' na coluna numérica n do registro dado.
func valorInvalido(registro int) string {
	return fmt.Sprintf(`valor inválido em 'n' (registro %d): "abc" (strconv.ParseFloat: parsing "abc": invalid syntax)`, registro)
}

// camposErrados é a mensagem de um registro com a quantidade errada de campos.
func camposErrados(registro, campos int) string {
	return fmt.Sprintf("phase error no arquivo de entrada: registro %d tem %d campos, esperados 2", registro, campos)
}

// campoComposto é a mensagem de um campo do JSON que é objeto ou array.
func campoComposto(registro int) string {
	return fmt.Sprintf(`JSON: registro %d, campo "n" não é um valor simples (texto, número ou booleano)`, registro)
}

// itemNaoObjeto é a mensagem de um item de array do JSON que não é objeto.
func itemNaoObjeto(item int) string {
	return fmt.Sprintf("JSON: o item %d do array não é um objeto", item)
}

// linhaInteira é o stderr de um erro numa linha de até 72 caracteres, mostrada como está.
func linhaInteira(msg string, linha int, texto string) string {
	marca := strings.Repeat("^", max(1, min(40, len(strings.TrimSpace(texto)))))
	prefixo := fmt.Sprintf("  %d | ", linha)
	return "rptgo: " + msg + "\n" + prefixo + texto + "\n" + strings.Repeat(" ", len(prefixo)) + marca + "\n"
}

// linhaCortada é o stderr de um erro numa linha longa: antes e depois dizem de que lados houve
// corte, e marcas é a quantidade de "^".
func linhaCortada(msg string, linha int, antes bool, trecho string, depois bool, marcas int) string {
	prefixo := fmt.Sprintf("  %d | ", linha)
	esquerda, direita := "", ""
	if antes {
		esquerda = "..."
	}
	if depois {
		direita = "..."
	}
	return "rptgo: " + msg + "\n" + prefixo + esquerda + trecho + direita + "\n" +
		strings.Repeat(" ", len(prefixo)+len(esquerda)) + strings.Repeat("^", marcas) + "\n"
}

type casoDados struct {
	nome   string
	opcoes []string
	dados  string
	want   string
}

// confereLinhasDados roda cada caso lendo os dados do arquivo e da entrada padrão: código 1,
// nada no stdout e exatamente o stderr esperado.
func confereLinhasDados(t *testing.T, comuns []string, casos []casoDados) {
	t.Helper()
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			rp, dat := arquivosLinha(t, scriptDados, c.dados)
			opcoes := append(append([]string{}, comuns...), c.opcoes...)
			for _, fonte := range []struct{ nome, entrada, stdin string }{
				{"arquivo", dat, ""},
				{"stdin", "-", c.dados},
			} {
				args := append(append([]string{}, opcoes...), rp, fonte.entrada)
				if code, out, errb := executa(t, fonte.stdin, args...); code != 1 || out != "" || errb != c.want {
					t.Errorf("%s: code=%d stdout=%q\n--- got ---\n%q\n--- want ---\n%q", fonte.nome, code, out, errb, c.want)
				}
			}
		})
	}
}

func TestLinhaDoErroNoTextoDelimitado(t *testing.T) {
	longa := strings.Repeat("y", 40) + strings.Repeat("z", 40) + "|abc"
	confereLinhasDados(t, nil, []casoDados{
		{"sem cabeçalho", nil, "x|1\ny|abc\n",
			linhaInteira(valorInvalido(2), 2, "y|abc")},
		{"cabeçalho automático", nil, "a|n\nx|abc\n",
			linhaInteira(valorInvalido(1), 2, "x|abc")},
		{"cabeçalho forçado", []string{"--header", "sim"}, "nome|valor\nx|abc\n",
			linhaInteira(valorInvalido(1), 2, "x|abc")},
		{"cabeçalho recusado", []string{"--header", "nao"}, "a|n\nx|1\n",
			"rptgo: " + `valor inválido em 'n' (registro 1): "n" (strconv.ParseFloat: parsing "n": invalid syntax)` +
				"\n  1 | a|n\n      ^^^\n"},
		{"campos a menos com cabeçalho", nil, "a|n\nx|1\ny\n",
			linhaInteira(camposErrados(2, 1), 3, "y")},
		{"linha em branco é um registro", nil, "a|n\nx|1\n\ny|2\n",
			linhaInteira(camposErrados(2, 1), 3, "")},
		{"CRLF", nil, "a|n\r\nx|1\r\ny|abc\r\n",
			linhaInteira(valorInvalido(2), 3, "y|abc\r")},
		{"linha longa", nil, "a|n\n" + longa + "\n",
			linhaCortada(valorInvalido(1), 2, false, longa[:72], true, 40)},
		{"BOM na primeira linha sem cabeçalho", nil, bom + "x|abc\n",
			linhaInteira(valorInvalido(1), 1, bom+"x|abc")},
	})
}

func TestLinhaDoErroNoCSV(t *testing.T) {
	longa := strings.Repeat("y", 90) + ",abc"
	confereLinhasDados(t, []string{"--input", "csv"}, []casoDados{
		{"sem cabeçalho", nil, "x,1\ny,abc\n",
			linhaInteira(valorInvalido(2), 2, "y,abc")},
		{"cabeçalho automático", nil, "a,n\nx,abc\n",
			linhaInteira(valorInvalido(1), 2, "x,abc")},
		{"cabeçalho forçado", []string{"--header", "sim"}, "nome,valor\nx,abc\n",
			linhaInteira(valorInvalido(1), 2, "x,abc")},
		{"linhas em branco antes do registro", nil, "a,n\n\nx,1\n\n\ny,abc\n",
			linhaInteira(valorInvalido(2), 6, "y,abc")},
		{"linhas em branco antes do cabeçalho", nil, "\n\na,n\nx,abc\n",
			linhaInteira(valorInvalido(1), 4, "x,abc")},
		{"campo anterior com quebra de linha", nil, "a,n\n\"x\ny\",1\nz,abc\n",
			linhaInteira(valorInvalido(2), 4, "z,abc")},
		{"registro de várias linhas mostra a primeira", nil, "a,n\nx,1\n\"y\nz\",abc\nw,2\n",
			linhaInteira(valorInvalido(2), 3, `"y`)},
		{"cabeçalho de várias linhas", []string{"--header", "sim"}, "\"no\nme\",valor\nx,abc\n",
			linhaInteira(valorInvalido(1), 3, "x,abc")},
		{"campos demais", nil, "a,n\n\nx,1,2\n",
			linhaInteira(camposErrados(1, 3), 3, "x,1,2")},
		{"CRLF", nil, "a,n\r\n\r\n\"x\r\ny\",1\r\nz,abc\r\n",
			linhaInteira(valorInvalido(2), 5, "z,abc\r")},
		{"BOM", nil, bom + "a,n\nx,abc\n",
			linhaInteira(valorInvalido(1), 2, "x,abc")},
		{"outro delimitador", []string{"--input-delimiter", ";"}, "a;n\n\nx;abc\n",
			linhaInteira(valorInvalido(1), 3, "x;abc")},
		{"linha longa", nil, "a,n\n\n" + longa + "\n",
			linhaCortada(valorInvalido(1), 3, false, longa[:72], true, 40)},
	})
}

func TestLinhaDoErroNoJSON(t *testing.T) {
	const bom1, ruim = `{"a":"x","n":1}`, `{"a":"y","n":"abc"}`
	enchimento := strings.Repeat("y", 90)
	longo := `{"a":"` + enchimento + `","n":"abc"}`
	curto := `{"a":"y","n":"abc"}]`
	composto := `{"a":"y","n":{"v":1}}`
	compostoLongo := `{"a":"` + enchimento + `","n":[1]}`
	itemLongo := `"` + enchimento + `"`
	// prefixo empurra o item para depois da coluna 72 de um arquivo compacto
	prefixo := "[" + strings.Repeat(bom1+",", 6)

	confereLinhasDados(t, []string{"--input", "json"}, []casoDados{
		{"array em várias linhas", nil,
			"[\n  " + bom1 + ",\n  {\n    \"a\": \"y\",\n    \"n\": \"abc\"\n  }\n]\n",
			linhaInteira(valorInvalido(2), 3, "  {")},
		{"vírgula no começo da linha", nil, "[\n" + bom1 + "\n,\n" + ruim + "\n]\n",
			linhaInteira(valorInvalido(2), 4, ruim)},
		{"um objeto por linha", nil, bom1 + "\n" + ruim + "\n" + bom1 + "\n",
			linhaInteira(valorInvalido(2), 2, ruim)},
		{"linhas em branco no começo", nil, "\n\n  [\n   " + bom1 + ",\n   " + ruim + "\n  ]\n",
			linhaInteira(valorInvalido(2), 5, "   "+ruim)},
		{"vários valores", nil, "[" + bom1 + "," + bom1 + "]\n" + bom1 + "\n[]\n[\n  " + ruim + "\n]\n",
			linhaInteira(valorInvalido(4), 5, "  "+ruim)},
		{"CRLF", nil, "[\r\n" + bom1 + ",\r\n" + ruim + "\r\n]\r\n",
			linhaInteira(valorInvalido(2), 3, ruim+"\r")},
		{"BOM em linha curta", nil, bom + "[" + curto + "\n",
			linhaInteira(valorInvalido(1), 1, bom+"["+curto)},
		{"decimal automático relê o texto", []string{"--input-decimal", "auto"},
			"\n[\n" + `{"a":"x","n":"1,5"},` + "\n" + ruim + "\n]\n",
			linhaInteira(valorInvalido(2), 4, ruim)},

		{"compacto longo com até 72 no resto da linha", nil, prefixo + ruim + "]\n",
			linhaCortada(valorInvalido(7), 1, true, ruim+"]", false, len(ruim)+1)},
		{"compacto longo com mais de 72 no resto", nil, prefixo + longo + "]\n",
			linhaCortada(valorInvalido(7), 1, true, longo[:72], true, 40)},
		{"compacto longo no primeiro item", nil, "[" + longo + "]\n",
			linhaCortada(valorInvalido(1), 1, true, longo[:72], true, 40)},
		{"BOM em linha longa começa no objeto", nil, bom + "[" + longo + "]\n",
			linhaCortada(valorInvalido(1), 1, true, longo[:72], true, 40)},
		{"BOM não desloca a segunda linha", nil, bom + "[\n " + longo + "]\n",
			linhaCortada(valorInvalido(1), 2, true, longo[:72], true, 40)},
		{"coluna contada em caracteres", nil, `[{"a":"ááááé","n":1},` + longo + "]\n",
			linhaCortada(valorInvalido(2), 1, true, longo[:72], true, 40)},

		{"campo composto", nil, "[\n" + bom1 + ",\n" + composto + "\n]\n",
			linhaInteira(campoComposto(2), 3, composto)},
		{"campo composto depois de outros valores", nil, "\n" + bom1 + "\n\n[\n {\n  \"n\": [2]\n }\n]\n",
			linhaInteira(campoComposto(2), 5, " {")},
		{"campo composto em linha longa", nil, prefixo + compostoLongo + "]\n",
			linhaCortada(campoComposto(7), 1, true, compostoLongo[:72], true, 40)},
		{"item que não é objeto", nil, "[\n" + bom1 + ",\n  5\n]\n",
			linhaInteira(itemNaoObjeto(2), 3, "  5")},
		{"item que não é objeto num segundo array", nil, "[" + bom1 + "]\n[" + bom1 + ",\n null]\n",
			linhaInteira(itemNaoObjeto(2), 3, " null]")},
		{"item que não é objeto em linha longa", nil, prefixo + itemLongo + "]\n",
			linhaCortada(itemNaoObjeto(7), 1, true, itemLongo[:72], true, 40)},
		{"item que não é objeto no fim da linha longa", nil, prefixo + "[1]]\n",
			linhaCortada(itemNaoObjeto(7), 1, true, "[1]]", false, 4)},
	})
}

func TestLinhaDoErroNasCodificacoes(t *testing.T) {
	for _, enc := range []string{"utf-8", "latin-1", "cp850"} {
		cod := func(s string) string { return string(codificar(s, enc)) }
		longo := `{"a":"` + strings.Repeat("ã", 90) + `","n":"abc"}`
		for _, formato := range []struct {
			nome   string
			opcoes []string
			casos  []casoDados
		}{
			{"delimited", nil, []casoDados{
				{"curta", nil, cod("a|n\nJoão|1\nJosé|abc\n"),
					linhaInteira(valorInvalido(2), 3, "José|abc")},
				{"longa", nil, cod(strings.Repeat("é", 80) + "|abc\n"),
					linhaCortada(valorInvalido(1), 1, false, strings.Repeat("é", 72), true, 40)},
			}},
			{"csv", []string{"--input", "csv"}, []casoDados{
				{"curta", nil, cod("a,n\n\"São\nJoão\",1\nJosé,abc\n"),
					linhaInteira(valorInvalido(2), 4, "José,abc")},
			}},
			{"json", []string{"--input", "json"}, []casoDados{
				{"curta", nil, cod("[\n" + `{"a":"João","n":1},` + "\n" + `{"a":"José","n":"abc"}` + "\n]\n"),
					linhaInteira(valorInvalido(2), 3, `{"a":"José","n":"abc"}`)},
				{"longa", nil, cod(`[{"a":"ção","n":1},` + longo + "]\n"),
					linhaCortada(valorInvalido(2), 1, true, string([]rune(longo)[:72]), true, 40)},
			}},
		} {
			t.Run(enc+"/"+formato.nome, func(t *testing.T) {
				confereLinhasDados(t, append([]string{"--encoding", enc}, formato.opcoes...), formato.casos)
			})
		}
	}
}

// O localizador do JSON acha o '{' de cada objeto, seja qual for a disposição do texto.
func TestPosicaoDeCadaItemDoJSON(t *testing.T) {
	texto := bom + "[{\"a\":\"}{,[\"}, {\"b\":{\"c\":[1,{}]}}\n ,\n\t{}\n]{\"d\":1}{}\n\n  [ ] [\n{}]"
	localiza := localizadorJSON(texto)
	for i, want := range [][2]int{{1, 2}, {1, 16}, {3, 1}, {4, 1}, {4, 8}, {7, 0}} {
		pos := localiza(i + 1)
		if pos.Linha != want[0] || pos.Coluna != want[1] {
			t.Errorf("registro %d: linha %d coluna %d, want %v", i+1, pos.Linha, pos.Coluna, want)
			continue
		}
		linha := []rune(strings.Split(texto, "\n")[pos.Linha-1])
		if linha[pos.Coluna] != '{' {
			t.Errorf("registro %d: a posição não é um '{': %q", i+1, string(linha[pos.Coluna:]))
		}
	}
	if pos := localiza(7); pos.Linha != 0 {
		t.Errorf("registro inexistente: %+v", pos)
	}
}
