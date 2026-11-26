package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Estes testes fixam o comportamento anterior às mudanças nas mensagens de erro de execução: o
// que roda sem erro continua com a mesma saída (no stdout e no arquivo de -o), os erros de
// dados em linhas curtas de texto delimitado sem cabeçalho não mudam e data e hora com '/' ou
// '%' continuam sendo operações inválidas. As saídas esperadas foram capturadas do commit
// a3ae252.

// modeloRegressao é um modelo HTML mínimo: só o número de páginas e o texto de cada uma.
const modeloRegressao = "<p>{{PageCount}}</p>\n{{#Pages}}<pre>{{PageNumber}}\n{{Text}}</pre>\n{{/Pages}}"

// emLinhas junta as linhas de uma saída esperada, cada uma terminada por "\n".
func emLinhas(linhas ...string) string { return strings.Join(linhas, "\n") + "\n" }

// argsHTML devolve as opções da saída HTML com o modelo mínimo, seguidas das opções extras.
func argsHTML(t *testing.T, extras ...string) []string {
	t.Helper()
	modelo := filepath.Join(t.TempDir(), "m.html")
	writeFile(t, modelo, modeloRegressao)
	return append([]string{"--format", "html", "--template", modelo}, extras...)
}

// confereSucesso roda o script no stdout e com -o: a saída é a esperada nos dois, sem nada no
// stderr e com código 0.
func confereSucesso(t *testing.T, nome string, opcoes []string, script, dados, saida string) {
	t.Helper()
	rp, dat := arquivosLinha(t, script, dados)
	args := append(append([]string{}, opcoes...), rp, dat)
	if code, out, errb := executa(t, "", args...); code != 0 || out != saida || errb != "" {
		t.Errorf("%s: code=%d stderr=%q\n--- got ---\n%q\n--- want ---\n%q", nome, code, errb, out, saida)
	}
	destino := filepath.Join(t.TempDir(), "saida")
	code, out, errb := executa(t, "", append([]string{"-o", destino}, args...)...)
	if code != 0 || out != "" || errb != "" {
		t.Errorf("%s com -o: code=%d stdout=%q stderr=%q", nome, code, out, errb)
		return
	}
	if got := lerArquivo(t, destino); got != saida {
		t.Errorf("%s com -o:\n--- got ---\n%q\n--- want ---\n%q", nome, got, saida)
	}
}

// confereFalha roda um script que termina em erro: código 1, nada no stdout, o stderr aceito
// por confere e o arquivo de -o nem criado nem alterado.
func confereFalha(t *testing.T, nome string, opcoes []string, script, dados string, confere func(stderr string) bool) {
	t.Helper()
	rp, dat := arquivosLinha(t, script, dados)
	args := append(append([]string{}, opcoes...), rp, dat)
	if code, out, errb := executa(t, "", args...); code != 1 || out != "" || !confere(errb) {
		t.Errorf("%s: code=%d stdout=%q stderr=%q", nome, code, out, errb)
	}

	novo := filepath.Join(t.TempDir(), "novo")
	if code, out, errb := executa(t, "", append([]string{"-o", novo}, args...)...); code != 1 || out != "" || !confere(errb) {
		t.Errorf("%s com -o: code=%d stdout=%q stderr=%q", nome, code, out, errb)
	}
	if _, err := os.Stat(novo); !os.IsNotExist(err) {
		t.Errorf("%s: o arquivo de -o não devia existir (err=%v)", nome, err)
	}

	const anterior = "conteúdo anterior\r\n"
	existente := filepath.Join(t.TempDir(), "existente")
	writeFile(t, existente, anterior)
	if code, _, _ := executa(t, "", append([]string{"-o", existente}, args...)...); code != 1 {
		t.Errorf("%s com -o existente: code=%d", nome, code)
	}
	if got := lerArquivo(t, existente); got != anterior {
		t.Errorf("%s: o arquivo de -o existente mudou: %q", nome, got)
	}
}

// Um agregado em 'header' ou 'footer' calcula quando a quebra de página acontece dentro de um
// 'after', que é quem define o grupo. Fora dele (primeira página e fechamento da última) o
// script não avalia o agregado.
func TestRegressaoAgregadosNaQuebraDePagina(t *testing.T) {
	const inicio = `input g [string], v [numeric]
left margin 0
top margin 0
bottom margin 0
length 6
sort g

header
   if pageno > 1 then print 'cab', count(*), total(v)
footer
   if emgrupo = 1 then print 'rod', count(*), total(v) else print 'rod -'
detail
   print g, v
`
	const afterChave = inicio + `after g
   set emgrupo to 1
   print 'g1', g
   print 'g2'
   print 'g3'
   print 'g4'
   print 'g5'
   print 'g6'
   set emgrupo to 0
end
`
	const afterReport = inicio + `after report
   set emgrupo to 1
   print 'r1'
   print 'r2'
   print 'r3'
   print 'r4'
   print 'r5'
   print 'r6'
   print 'r7'
   set emgrupo to 0
end
`
	const dados = "a|1\na|2\nb|5\n"

	confereSucesso(t, "after de chave, texto", nil, afterChave, dados, emLinhas(
		"a 1", "a 2", "g1 a", "g2", "g3", "rod 2 3",
		"cab 2 3", "g4", "g5", "g6", "b 5", "rod 1 5",
		"cab 1 5", "g1 b", "g2", "g3", "g4", "rod 1 5",
		"cab 1 5", "g5", "g6", "", "", "rod -"))
	confereSucesso(t, "after de chave, html", argsHTML(t), afterChave, dados, emLinhas(
		"<p>4</p>",
		"<pre>1", "a 1", "a 2", "g1 a", "g2", "g3", "rod 2 3</pre>",
		"<pre>2", "cab 2 3", "g4", "g5", "g6", "b 5", "rod 1 5</pre>",
		"<pre>3", "cab 1 5", "g1 b", "g2", "g3", "g4", "rod 1 5</pre>",
		"<pre>4", "cab 1 5", "g5", "g6", "", "", "rod -</pre>"))

	confereSucesso(t, "after report, texto", nil, afterReport, dados, emLinhas(
		"a 1", "a 2", "b 5", "r1", "r2", "rod 3 8",
		"cab 3 8", "r3", "r4", "r5", "r6", "rod 3 8",
		"cab 3 8", "r7", "", "", "", "rod -"))
	confereSucesso(t, "after report, html", argsHTML(t), afterReport, dados, emLinhas(
		"<p>3</p>",
		"<pre>1", "a 1", "a 2", "b 5", "r1", "r2", "rod 3 8</pre>",
		"<pre>2", "cab 3 8", "r3", "r4", "r5", "r6", "rod 3 8</pre>",
		"<pre>3", "cab 3 8", "r7", "", "", "", "rod -</pre>"))
}

// Um erro numa chave do sort que não é de tipo errado nem divisão por zero deixa a chave nula,
// sem mensagem: 'substr' sobre data e '/' ou '%' sobre data e hora.
func TestRegressaoChaveDeSortComErroSilencioso(t *testing.T) {
	const datas = "input d [ldate], n [numeric]\nleft margin 0\ntop margin 0\nbottom margin 0\nlength 5\n"
	const dadosDatas = "2026-03-02|3\n2026-01-01|1\n2026-02-05|2\n"

	// só a chave nula: fica a ordem da entrada
	so := datas + "sort d substr[1-2]\ndetail\n   print d, n\nend\n"
	confereSucesso(t, "substr em data, texto", nil, so, dadosDatas, emLinhas(
		"02/03/2026 3", "01/01/2026 1", "05/02/2026 2", "", ""))
	confereSucesso(t, "substr em data, csv", []string{"--format", "csv"}, so, dadosDatas, emLinhas(
		"d,n", "2026-03-02,3", "2026-01-01,1", "2026-02-05,2"))
	confereSucesso(t, "substr em data, json", []string{"--format", "json"}, so, dadosDatas, emLinhas(
		"[", `{"d":"2026-03-02","n":3},`, `{"d":"2026-01-01","n":1},`, `{"d":"2026-02-05","n":2}`, "]"))
	confereSucesso(t, "substr em data, html", argsHTML(t), so, dadosDatas, emLinhas(
		"<p>1</p>", "<pre>1", "02/03/2026 3", "01/01/2026 1", "05/02/2026 2</pre>"))

	// a chave seguinte continua ordenando
	depois := datas + "sort d substr[1-2], n\ndetail\n   print d, n\nend\n"
	confereSucesso(t, "substr em data e outra chave, texto", nil, depois, dadosDatas, emLinhas(
		"01/01/2026 1", "05/02/2026 2", "02/03/2026 3", "", ""))
	confereSucesso(t, "substr em data e outra chave, csv", []string{"--format", "csv"}, depois, dadosDatas, emLinhas(
		"d,n", "2026-01-01,1", "2026-02-05,2", "2026-03-02,3"))
	confereSucesso(t, "substr em data e outra chave, json", []string{"--format", "json"}, depois, dadosDatas, emLinhas(
		"[", `{"d":"2026-01-01","n":1},`, `{"d":"2026-02-05","n":2},`, `{"d":"2026-03-02","n":3}`, "]"))
	confereSucesso(t, "substr em data e outra chave, html", argsHTML(t), depois, dadosDatas, emLinhas(
		"<p>1</p>", "<pre>1", "01/01/2026 1", "05/02/2026 2", "02/03/2026 3</pre>"))

	invalidas := "input d [date], h [time], n [numeric]\nleft margin 0\ntop margin 0\nbottom margin 0\nlength 3\n" +
		"sort <k> d / n, <j> h % n\ndetail\n   print d, h, n\nend\n"
	const dadosInvalidas = "2026-01-02|10:30|0\n2026-01-01|09:00|2\n"
	confereSucesso(t, "operação inválida na chave, texto", nil, invalidas, dadosInvalidas, emLinhas(
		"02/01/26 10:30 0", "01/01/26 09:00 2", ""))
	confereSucesso(t, "operação inválida na chave, csv", []string{"--format", "csv"}, invalidas, dadosInvalidas, emLinhas(
		"d,h,n", "2026-01-02,10:30:00,0", "2026-01-01,09:00:00,2"))
}

// Uma divisão que não é avaliada não gera erro, mesmo com divisor zero.
func TestRegressaoDivisaoNaoAvaliada(t *testing.T) {
	const inicio = `input g [string], n [numeric], z [numeric]
left margin 0
top margin 0
bottom margin 0
length 8
sort g
`
	// o segundo registro tem divisor zero e o terceiro, divisor nulo
	const dados = "a|7|2\na|5|0\nb|9|\n"

	confereSucesso(t, "ramo do 'if' não executado", nil, inicio+`detail
   if z <> 0 then print g, n / z, n % z else print g, 'sem divisor'
   if z = 0 then print 'zero' else print 10 / z
end
`, dados, emLinhas("a 3 1", "5", "a sem divisor", "zero", "b sem divisor", "zero", "", ""))

	confereSucesso(t, "'and' e 'or' em curto-circuito", nil, inicio+`detail
   if z <> 0 and n / z > 1 then print g, 'and sim' else print g, 'and nao'
   if z = 0 or n % z > 0 then print g, 'or sim' else print g, 'or nao'
end
`, dados, emLinhas("a and sim", "a or sim", "a and nao", "a or sim", "b and nao", "b or sim", "", ""))

	confereSucesso(t, "registro excluído pelo 'where'", nil, inicio+`detail
   print g, n
after g
   print count(n / z where z <> 0), total(n / z where z <> 0), avg(n % z where z <> 0)
after report
   print min(n / z where z <> 0), max(n % z where z <> 0), count(* where z <> 0 and n / z > 0)
end
`, dados, emLinhas("a 7", "a 5", "1 3 1.00", "b 9", "0 0 0", "3 1 1", "", ""))

	// com --strict e 'before report', um relatório de uma página não roda o 'header'
	semCabecalho := inicio + `before report
   print 'inicio'
header
   print 1 / 0
detail
   print g, n
end
`
	confereSucesso(t, "seção que não roda, texto", []string{"--strict"}, semCabecalho, dados, emLinhas(
		"inicio", "a 7", "a 5", "b 9", "", "", "", ""))
	confereSucesso(t, "seção que não roda, html", argsHTML(t, "--strict"), semCabecalho, dados, emLinhas(
		"<p>1</p>", "<pre>1", "inicio", "a 7", "a 5", "b 9</pre>"))

	// csv e json não rodam nenhuma seção
	todas := inicio
	for _, secao := range []string{"header", "footer", "detail", "before report", "before g", "after g", "after report"} {
		todas += secao + "\n   print n / z, n % 0, 1 / 0\n"
	}
	todas += "end\n"
	confereSucesso(t, "todas as seções, csv", []string{"--format", "csv"}, todas, dados, emLinhas(
		"g,n,z", "a,7,2", "a,5,0", "b,9,"))
	confereSucesso(t, "todas as seções, json", []string{"--format", "json"}, todas, dados, emLinhas(
		"[", `{"g":"a","n":7,"z":2},`, `{"g":"a","n":5,"z":0},`, `{"g":"b","n":9,"z":null}`, "]"))
}

// Um relatório sem erro, com divisões, sort e quebras de página, sob --form-feed, --lines e
// --strict.
func TestRegressaoRelatorioComDivisoes(t *testing.T) {
	const script = `input g [string], n [numeric], q [numeric]
left margin 0
top margin 0
bottom margin 0
length 7
sort g, <unit> n / q desc

before report
   print 'inicio', 100 / 4
header
   print 'pag', pageno, n / q
footer
   print 'fim', pageno, 10 % 3
before g
   print 'grupo', g
detail
   print n, q, unit, n % q
after g
   print 'soma', total(n / q), avg(n % q)
after report
   print 'geral', total(n) / total(q)
end
`
	const dados = "b|9|2\na|7|2\na|20|3\nb|8|4\n"

	confereSucesso(t, "texto", nil, script, dados, emLinhas(
		"inicio 25", "pag 1 6", "grupo a", "20 3 6 2", "7 2 3 1", "soma 9 1.50", "fim 1 1",
		"pag 2 4", "grupo b", "9 2 4 1", "8 4 2 0", "soma 6 0.50", "geral 4", "fim 2 1"))
	confereSucesso(t, "form-feed", []string{"--form-feed"}, script, dados, emLinhas(
		"inicio 25", "pag 1 6", "grupo a", "20 3 6 2", "7 2 3 1", "soma 9 1.50", "fim 1 1",
		"\fpag 2 4", "grupo b", "9 2 4 1", "8 4 2 0", "soma 6 0.50", "geral 4", "fim 2 1"))
	confereSucesso(t, "strict", []string{"--strict"}, script, dados, emLinhas(
		"inicio 25", "grupo a", "20 3 6 2", "7 2 3 1", "soma 9 1.50", "grupo b", "fim 1 1",
		"pag 2 4", "9 2 4 1", "8 4 2 0", "soma 6 0.50", "geral 4", "", "fim 2 1"))
	confereSucesso(t, "strict e form-feed", []string{"--strict", "--form-feed"}, script, dados, emLinhas(
		"inicio 25", "grupo a", "20 3 6 2", "7 2 3 1", "soma 9 1.50", "grupo b", "fim 1 1",
		"\fpag 2 4", "9 2 4 1", "8 4 2 0", "soma 6 0.50", "geral 4", "", "fim 2 1"))

	confereSucesso(t, "html", argsHTML(t), script, dados, emLinhas(
		"<p>2</p>",
		"<pre>1", "inicio 25", "pag 1 6", "grupo a", "20 3 6 2", "7 2 3 1", "soma 9 1.50", "fim 1 1</pre>",
		"<pre>2", "pag 2 4", "grupo b", "9 2 4 1", "8 4 2 0", "soma 6 0.50", "geral 4", "fim 2 1</pre>"))
	confereSucesso(t, "lines", argsHTML(t, "--lines", "5"), script, dados, emLinhas(
		"<p>4</p>",
		"<pre>1", "inicio 25", "pag 1 6", "grupo a", "20 3 6 2", "fim 1 1</pre>",
		"<pre>2", "pag 2 3", "7 2 3 1", "soma 9 1.50", "grupo b", "fim 2 1</pre>",
		"<pre>3", "pag 3 4", "9 2 4 1", "8 4 2 0", "soma 6 0.50", "fim 3 1</pre>",
		"<pre>4", "pag 4 2", "geral 4", "", "", "fim 4 1</pre>"))
	confereSucesso(t, "lines e strict", argsHTML(t, "--lines", "5", "--strict"), script, dados, emLinhas(
		"<p>3</p>",
		"<pre>1", "inicio 25", "grupo a", "20 3 6 2", "7 2 3 1", "fim 1 1</pre>",
		"<pre>2", "pag 2 3", "soma 9 1.50", "grupo b", "9 2 4 1", "fim 2 1</pre>",
		"<pre>3", "pag 3 2", "8 4 2 0", "soma 6 0.50", "geral 4", "fim 3 1</pre>"))
}

// Erros de dados em linhas de até 72 caracteres, em texto delimitado sem cabeçalho.
func TestRegressaoErroDeDadosEmLinhaCurta(t *testing.T) {
	const script = "input nome, valor [numeric]\ndetail\n   print nome, valor\nend\n"
	invalido := func(registro, valor string) string {
		return "rptgo: valor inválido em 'valor' (registro " + registro + "): \"" + valor +
			"\" (strconv.ParseFloat: parsing \"" + valor + "\": invalid syntax)\n"
	}
	fase := func(registro, campos string) string {
		return "rptgo: phase error no arquivo de entrada: registro " + registro + " tem " + campos +
			" campos, esperados 2\n"
	}
	// 66 letras acentuadas mais "|abcde": 72 caracteres, mais que 72 bytes
	limite := strings.Repeat("ã", 66) + "|abcde"
	marcadorLongo := "      " + strings.Repeat("^", 40) + "\n"

	for _, c := range []struct {
		nome   string
		opcoes []string
		dados  string
		erro   string
	}{
		{"valor inválido no primeiro registro", nil, "Ana|abc\n",
			invalido("1", "abc") + "  1 | Ana|abc\n      ^^^^^^^\n"},
		{"valor inválido no meio, linha indentada", nil, "Ana|1\nBeto|2\n  Carla |3x\nDora|4\n",
			invalido("3", "3x") + "  3 |   Carla |3x\n      ^^^^^^^^^\n"},
		{"CRLF", nil, "Ana|1\r\nBeto|x\r\n",
			invalido("2", "x") + "  2 | Beto|x\r\n      ^^^^^^\n"},
		{"sem quebra de linha no fim", nil, "Ana|1\nBeto|x",
			invalido("2", "x") + "  2 | Beto|x\n      ^^^^^^\n"},
		{"72 caracteres", nil, limite + "\n",
			invalido("1", "abcde") + "  1 | " + limite + "\n" + marcadorLongo},
		{"72 caracteres com CRLF", nil, limite + "\r\n",
			invalido("1", "abcde") + "  1 | " + limite + "\r\n" + marcadorLongo},
		{"--header nao com o nome da coluna", []string{"--header", "nao"}, "nome|valor\n",
			invalido("1", "valor") + "  1 | nome|valor\n      ^^^^^^^^^^\n"},
		{"campos de menos", nil, "Ana|1\nBeto\n",
			fase("2", "1") + "  2 | Beto\n      ^^^^\n"},
		{"campos demais", nil, "Ana|1|2|3\n",
			fase("1", "4") + "  1 | Ana|1|2|3\n      ^^^^^^^^^\n"},
		{"linha vazia", nil, "Ana|1\n\nBeto|2\n",
			fase("2", "1") + "  2 | \n      ^\n"},
	} {
		confereFalha(t, c.nome, c.opcoes, script, c.dados, func(errb string) bool { return errb == c.erro })
	}
}

// Data e hora com '/' ou '%' são operações inválidas, mesmo com zero do outro lado. Só o
// trecho da mensagem é conferido.
func TestRegressaoDivisaoComDataEHora(t *testing.T) {
	const comDatas = "operação inválida com datas"
	const comHoras = "operação inválida com horas"
	for _, c := range []struct{ expr, mensagem string }{
		{"d / n", comDatas},
		{"d % n", comDatas},
		{"n / d", comDatas},
		{"n % d", comDatas},
		{"d / 2", comDatas},
		{"h / n", comHoras},
		{"h % n", comHoras},
		{"n / h", comHoras},
		{"n % h", comHoras},
		{"h / 2", comHoras},
	} {
		script := "input d [date], h [time], n [numeric]\ndetail\n   print " + c.expr + "\nend\n"
		confereFalha(t, c.expr, nil, script, "2026-01-02|10:30|0\n", func(errb string) bool {
			return strings.HasPrefix(errb, "rptgo: ") && strings.Contains(errb, c.mensagem)
		})
	}
}
