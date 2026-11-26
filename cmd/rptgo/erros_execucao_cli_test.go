package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Aceitação dos erros de execução pela linha de comando: o relatório para no primeiro erro, sem
// saída, e o stderr diz a seção, o trecho do script, a conta (na divisão por zero) e o registro
// lido.

// opcoesSaida devolve as opções de cada formato de saída.
func opcoesSaida(t *testing.T) []struct {
	nome   string
	opcoes []string
} {
	t.Helper()
	return []struct {
		nome   string
		opcoes []string
	}{
		{"text", nil},
		{"html", argsHTML(t)},
		{"csv", []string{"--format", "csv"}},
		{"json", []string{"--format", "json"}},
	}
}

// parteScriptCLI é a parte 'script' esperada: a linha do script que contém o trecho (que aparece
// uma vez só nele) com "^" sob o trecho inteiro.
func parteScriptCLI(t *testing.T, script, trecho string) string {
	t.Helper()
	if n := strings.Count(script, trecho); n != 1 {
		t.Fatalf("o trecho %q aparece %d vezes no script:\n%s", trecho, n, script)
	}
	antes, _, _ := strings.Cut(script, trecho)
	linha := strings.Count(antes, "\n") + 1
	prefixo := fmt.Sprintf("  %d | ", linha)
	coluna := len([]rune(antes[strings.LastIndexByte(antes, '\n')+1:]))
	return fmt.Sprintf("script, linha %d:\n%s%s\n%s%s", linha, prefixo, strings.Split(script, "\n")[linha-1],
		strings.Repeat(" ", len(prefixo)+coluna), strings.Repeat("^", len([]rune(trecho))))
}

// parteDadosCLI é a parte 'dados' esperada de um registro mostrado na linha dada.
func parteDadosCLI(linha, registro int, texto string) string {
	return fmt.Sprintf("dados, linha %d (registro %d):\n  %d | %s", linha, registro, linha, texto)
}

// erroCLI junta as partes de um erro de execução como ele sai no stderr.
func erroCLI(partes ...string) string { return "rptgo: " + strings.Join(partes, "\n") + "\n" }

// confereErroExato confere código 1, nada no stdout, o stderr exato e o arquivo de -o nem criado
// nem alterado.
func confereErroExato(t *testing.T, nome string, opcoes []string, script, dados, want string) {
	t.Helper()
	confereFalha(t, nome, opcoes, script, dados, func(errb string) bool {
		if errb != want {
			t.Logf("%s:\n--- got ---\n%s--- want ---\n%s", nome, errb, want)
		}
		return errb == want
	})
}

// '/' e '%', com inteiros e decimais, em cada uma das posições de seção do relatório.
func TestMatrizDivisaoPorZeroNasSecoes(t *testing.T) {
	tipos := []struct {
		tipo, fracao, divisor, zero string
	}{
		{"numeric", "", "3", "0"},
		{"float", ".5", "2.5", "0.0"},
	}
	secoes := []struct {
		nome, secao string
		paginas     bool   // páginas de 3 linhas
		cabecalho   string // linha que abre a seção no script
		registro    int    // o registro com divisor zero, que é o que a seção lê
	}{
		{"detail", "detail", false, "detail", 2},
		{"before report", "before report", false, "before report", 1},
		{"before da chave", "before g", false, "before g", 3},
		{"after da chave", "after g", false, "after g", 2},
		{"after report", "after report", false, "after report", 4},
		{"header na página 1", "header", false, "header", 1},
		{"header numa página seguinte", "header", true, "header", 3},
		{"footer numa quebra de página", "footer", true, "footer", 3},
		// o registro 4 não quebra a página: quem o lê é o fechamento da última
		{"footer na última página", "footer", true, "footer", 4},
	}
	for _, op := range []string{"/", "%"} {
		for _, tp := range tipos {
			for _, s := range secoes {
				nome := op + " " + tp.tipo + " " + s.nome
				t.Run(nome, func(t *testing.T) {
					conta := "a " + op + " z"
					script := "input g [string], a [" + tp.tipo + "], z [" + tp.tipo + "]\n" +
						"left margin 0\ntop margin 0\nbottom margin 0\n"
					if s.paginas {
						script += "length 3\n"
					}
					script += "sort g\n" + s.cabecalho + "\n   print 'v', " + conta + "\n"
					if s.secao != "detail" {
						script += "detail\n   print g\n"
					}
					script += "end\n"

					var linhas []string
					for i, g := range []string{"x", "x", "y", "y"} {
						z := tp.divisor
						if i+1 == s.registro {
							z = tp.zero
						}
						linhas = append(linhas, fmt.Sprintf("%s|%d%s|%s", g, (i+1)*10, tp.fracao, z))
					}
					want := erroCLI(
						"divisão por zero na seção "+s.secao,
						parteScriptCLI(t, script, conta),
						fmt.Sprintf("conta: %s, com a = %d%s e z = 0", conta, s.registro*10, tp.fracao),
						parteDadosCLI(s.registro, s.registro, linhas[s.registro-1]))
					dados := strings.Join(linhas, "\n") + "\n"
					confereErroExato(t, "texto", nil, script, dados, want)
					confereErroExato(t, "html", argsHTML(t), script, dados, want)
				})
			}
		}
	}
}

// O divisor zero no argumento ou no 'where' de cada agregado.
func TestMatrizDivisaoPorZeroNosAgregados(t *testing.T) {
	const dados = "10|2\n20|0\n30|3\n"
	for _, f := range []string{"count", "total", "avg", "min", "max"} {
		for _, op := range []string{"/", "%"} {
			conta := "a " + op + " z"
			for _, c := range []struct{ nome, chamada string }{
				{"argumento", f + "(" + conta + ")"},
				{"where", f + "(a where " + conta + " > 0)"},
			} {
				script := "input a [numeric], z [numeric]\ndetail\n   print a\nafter report\n   print " + c.chamada + "\nend\n"
				confereErroExato(t, f+" "+op+" "+c.nome, nil, script, dados, erroCLI(
					"divisão por zero na seção after report",
					parteScriptCLI(t, script, conta),
					"conta: "+conta+", com a = 20 e z = 0",
					parteDadosCLI(2, 2, "20|0")))
			}
		}
	}
}

// O divisor zero numa chave do sort, direta ou nomeada, primeira ou não, com ou sem 'desc', em
// todos os formatos de saída (csv e json não rodam seção, mas ordenam).
func TestMatrizDivisaoPorZeroNoSort(t *testing.T) {
	const dados = "x|10|2\ny|20|0\nz|30|0\n"
	for _, op := range []string{"/", "%"} {
		conta := "a " + op + " z"
		for _, c := range []struct{ nome, sort, detalhe string }{
			{"direta", "sort " + conta, "g"},
			{"direta desc", "sort " + conta + " desc", "g"},
			{"nomeada no sort", "sort <k> " + conta, "k"},
			{"nomeada no sort desc", "sort <k> " + conta + " desc", "k"},
			{"nomeada declarada na seção", "sort u", "<u> " + conta},
			{"nomeada declarada na seção desc", "sort u desc", "<u> " + conta},
			{"segunda chave", "sort g, " + conta, "g"},
			{"segunda chave desc", "sort g desc, " + conta + " desc", "g"},
			{"terceira chave nomeada", "sort g, a, <k> " + conta + " desc", "g"},
		} {
			script := "input g [string], a [numeric], z [numeric]\n" + c.sort + "\ndetail\n   print " + c.detalhe + "\nend\n"
			want := erroCLI(
				"divisão por zero na seção sort",
				parteScriptCLI(t, script, conta),
				"conta: "+conta+", com a = 20 e z = 0",
				parteDadosCLI(2, 2, "y|20|0"))
			for _, saida := range opcoesSaida(t) {
				confereErroExato(t, c.nome+" "+op+" "+saida.nome, saida.opcoes, script, dados, want)
			}
		}
	}
}

// Tudo o que vale zero numa conta também para o relatório quando é o divisor.
func TestDivisorQueContaComoZero(t *testing.T) {
	for _, c := range []struct{ nome, tipo, valor, descricao string }{
		{"nulo", "numeric", "", "z = nulo (conta como 0)"},
		{"texto vazio", "string", "", "z = texto vazio (conta como 0)"},
		{"texto", "string", "abc", `z = texto "abc" (conta como 0)`},
		{"falso", "bool", "no", "z = falso (conta como 0)"},
		{"inteiro", "numeric", "0", "z = 0"},
		{"decimal", "float", "0.00", "z = 0"},
	} {
		for _, op := range []string{"/", "%"} {
			script := "input a [numeric], z [" + c.tipo + "]\ndetail\n   print a " + op + " z\nend\n"
			confereErroExato(t, c.nome+" "+op, nil, script, "7|"+c.valor+"\n", erroCLI(
				"divisão por zero na seção detail",
				parteScriptCLI(t, script, "a "+op+" z"),
				"conta: a "+op+" z, com a = 7 e "+c.descricao,
				parteDadosCLI(1, 1, "7|"+c.valor)))
		}
	}
}

// Os comandos dos itens B3 (no 'header' e no 'footer') e B4 do backlog agora falham.
func TestComandosDoBacklog(t *testing.T) {
	const dadosB3 = "1|1\n2|1\n3|1\n4|0\n5|0\n"
	for _, secao := range []string{"header", "footer"} {
		script := "input n [numeric], z [numeric] top margin 0 bottom margin 0 length 3 " +
			secao + " print 'h', 10 / z detail print n end"
		prefixo := "  1 | "
		marca := strings.Repeat(" ", len(prefixo))
		code, out, errb := executa(t, dadosB3, "-e", script)
		if code != 1 || out != "" {
			t.Errorf("B3 %s: code=%d stdout=%q", secao, code, out)
		}
		// o script tem uma linha longa: dela sai um trecho com a operação
		linhas := strings.Split(errb, "\n")
		if len(linhas) != 8 || linhas[0] != "rptgo: divisão por zero na seção "+secao || linhas[1] != "script, linha 1:" ||
			!strings.HasPrefix(linhas[2], prefixo+"...") || !strings.HasPrefix(linhas[3], marca) ||
			strings.TrimSpace(linhas[3]) != "^^^^^^" || linhas[4] != "conta: 10 / z, com z = 0" ||
			linhas[5] != "dados, linha 5 (registro 5):" || linhas[6] != "  5 | 5|0" || linhas[7] != "" {
			t.Errorf("B3 %s: stderr:\n%s", secao, errb)
			continue
		}
		if i := strings.IndexByte(linhas[3], '^'); i+6 > len(linhas[2]) || linhas[2][i:i+6] != "10 / z" {
			t.Errorf("B3 %s: marcador fora do lugar:\n%s", secao, errb)
		}
		// com uma linha de cabeçalho o registro continua o 5, agora na linha 6
		code, out, errb = executa(t, "n|z\n"+dadosB3, "-e", script)
		if code != 1 || out != "" || !strings.HasSuffix(errb, "\n"+parteDadosCLI(6, 5, "5|0")+"\n") {
			t.Errorf("B3 %s com cabeçalho: code=%d stdout=%q stderr=%q", secao, code, out, errb)
		}
	}

	const scriptB4 = "input n [numeric], z [numeric] sort n / z end"
	want := erroCLI("divisão por zero na seção sort", parteScriptCLI(t, scriptB4, "n / z"),
		"conta: n / z, com n = 1 e z = 0", parteDadosCLI(1, 1, "1|0"))
	if code, out, errb := executa(t, "1|0\n", "--format", "csv", "-e", scriptB4); code != 1 || out != "" || errb != want {
		t.Errorf("B4: code=%d stdout=%q\n--- got ---\n%s--- want ---\n%s", code, out, errb, want)
	}
}

// Um erro depois da primeira página não deixa saída parcial, e um erro no sort não deixa saída
// em csv nem em json.
func TestErroNaoDeixaSaidaParcial(t *testing.T) {
	const paginas = "input n [numeric], z [numeric]\nleft margin 0\ntop margin 0\nbottom margin 0\nlength 2\n" +
		"detail\n   print n / z\nend\n"
	// o registro com divisor zero fica na terceira página
	const dados = "1|1\n2|1\n3|1\n4|1\n5|0\n6|0\n"
	want := erroCLI("divisão por zero na seção detail", parteScriptCLI(t, paginas, "n / z"),
		"conta: n / z, com n = 5 e z = 0", parteDadosCLI(5, 5, "5|0"))
	confereErroExato(t, "texto", nil, paginas, dados, want)
	confereErroExato(t, "texto com form-feed", []string{"--form-feed"}, paginas, dados, want)
	confereErroExato(t, "html", argsHTML(t), paginas, dados, want)

	const ordenado = "input n [numeric], z [numeric]\nsort n / z\nend\n"
	want = erroCLI("divisão por zero na seção sort", parteScriptCLI(t, ordenado, "n / z"),
		"conta: n / z, com n = 5 e z = 0", parteDadosCLI(5, 5, "5|0"))
	confereErroExato(t, "csv", []string{"--format", "csv"}, ordenado, dados, want)
	confereErroExato(t, "json", []string{"--format", "json"}, ordenado, dados, want)
}

// Com vários registros e seções que falhariam, sai um erro só: o primeiro na ordem de execução
// e, nas chaves do sort, o do primeiro registro da entrada.
func TestSoOPrimeiroErroERelatado(t *testing.T) {
	const secoes = "input n [numeric], z [numeric]\nbefore report\n   print 'x'\nheader\n   print 'h'\n" +
		"detail\n   print n / z\nafter report\n   print 1 / 0\nfooter\n   if pageno > 9 then print zz\nend\n"
	confereErroExato(t, "seções", nil, secoes, "1|1\n2|0\n3|0\n", erroCLI(
		"divisão por zero na seção detail", parteScriptCLI(t, secoes, "n / z"),
		"conta: n / z, com n = 2 e z = 0", parteDadosCLI(2, 2, "2|0")))

	const chaves = "input n [numeric], z [numeric]\nsort n, <k> n / z\ndetail\n   print 1 / 0\nend\n"
	for _, saida := range opcoesSaida(t) {
		confereErroExato(t, "chaves "+saida.nome, saida.opcoes, chaves, "9|1\n8|0\n1|0\n", erroCLI(
			"divisão por zero na seção sort", parteScriptCLI(t, chaves, "n / z"),
			"conta: n / z, com n = 8 e z = 0", parteDadosCLI(2, 2, "8|0")))
	}
}

// Uma chave que fica nula sem mensagem não esconde a divisão por zero de outra chave.
func TestChaveSilenciosaEChaveComDivisao(t *testing.T) {
	const entrada = "input d [ldate], n [numeric], z [numeric]\n"
	const dados = "2026-03-02|3|1\n2026-01-01|1|0\n"
	for _, chaves := range []string{"d substr[1-2], n / z", "n / z, d substr[1-2]", "d substr[1-2] desc, <k> n / z desc"} {
		script := entrada + "sort " + chaves + "\ndetail\n   print n\nend\n"
		want := erroCLI("divisão por zero na seção sort", parteScriptCLI(t, script, "n / z"),
			"conta: n / z, com n = 1 e z = 0", parteDadosCLI(2, 2, "2026-01-01|1|0"))
		for _, saida := range opcoesSaida(t) {
			confereErroExato(t, chaves+" "+saida.nome, saida.opcoes, script, dados, want)
		}
	}
}

// O mesmo script dá o mesmo stderr, byte a byte, num arquivo .rp e no -e.
func TestErroDeExecucaoIgualNoArquivoENoE(t *testing.T) {
	for _, c := range []struct{ nome, script, dados string }{
		{"detail", "input a [numeric], z [numeric]\ndetail\n\tprint 'São João', a / z\nend\n", "1|0\n"},
		{"várias linhas", "input a [numeric], z [numeric]\ndetail\n   print (a +\n      1) % z\nend\n", "1|0\n"},
		{"linha longa", "input a [numeric], z [numeric]\ndetail\n   print '" + strings.Repeat("é", 90) + "', a / z\nend\n", "1|0\n"},
		{"header", "input a [numeric], z [numeric]\nheader\n   print a / z\nend\n", "1|0\n"},
		{"sort", "input a [numeric], z [numeric]\nsort a / z\nend\n", "1|0\n"},
		{"outro erro", "input a [numeric], z [numeric]\ndetail\n   print dow(a)\nend\n", "1|0\n"},
		{"comparação no sort", "input a [numeric], d [date]\nsort index(a - 1, a, d)\nend\n",
			"1|2026-01-01\n2|2026-02-01\n"},
		{"fim de linha CRLF", "input a [numeric], z [numeric]\r\ndetail\r\n   print a / z\r\nend\r\n", "1|0\n"},
	} {
		t.Run(c.nome, func(t *testing.T) {
			rp, dat := arquivosLinha(t, c.script, c.dados)
			code, out, doArquivo := executa(t, "", rp, dat)
			if code != 1 || out != "" || !strings.HasPrefix(doArquivo, "rptgo: ") || !strings.Contains(doArquivo, "\nscript, linha ") {
				t.Fatalf("arquivo: code=%d stdout=%q stderr=%q", code, out, doArquivo)
			}
			for _, opcao := range []string{"-e", "--script"} {
				if code, out, errb := executa(t, "", opcao, c.script, dat); code != 1 || out != "" || errb != doArquivo {
					t.Errorf("%s: code=%d stdout=%q\n--- got ---\n%q\n--- want ---\n%q", opcao, code, out, errb, doArquivo)
				}
			}
		})
	}
}

// A parte 'dados' mostra a primeira linha física do registro, em qualquer formato de entrada, e
// corta as linhas longas. Vale lendo do arquivo e da entrada padrão.
func TestParteDadosPorFormatoDeEntrada(t *testing.T) {
	const script = "input a, z [numeric]\ndetail\n   print a, 10 / z\nend\n"
	longo := strings.Repeat("ã", 90)
	antes := "[" + strings.Repeat(`{"a":"x","z":1},`, 6)
	objeto := `{"a":"` + longo + `","z":0}`
	csvLongo := longo + ",0"
	delimLongo := longo + "|0"
	cortada := func(linha, registro int, texto string) string { return parteDadosCLI(linha, registro, texto) }
	for _, c := range []struct {
		nome   string
		opcoes []string
		dados  string
		parte  string
	}{
		{"delimitado", nil, "x|1\ny|0\n", parteDadosCLI(2, 2, "y|0")},
		{"delimitado com cabeçalho automático", nil, "a|z\nx|1\ny|0\n", parteDadosCLI(3, 2, "y|0")},
		{"delimitado com cabeçalho forçado", []string{"--header", "sim"}, "n|v\ny|0\n", parteDadosCLI(2, 1, "y|0")},
		{"delimitado com CRLF", nil, "x|1\r\ny|0\r\n", parteDadosCLI(2, 2, "y|0")},
		{"delimitado longo", nil, "x|1\n" + delimLongo + "\n", cortada(2, 2, string([]rune(delimLongo)[:72])+"...")},
		{"csv com linhas em branco e campo de várias linhas", []string{"--input", "csv"},
			"a,z\n\n\"x\ny\",1\n\n\"w\nv\",0\n", parteDadosCLI(6, 2, `"w`)},
		{"csv longo", []string{"--input", "csv"}, "a,z\n" + csvLongo + "\n",
			cortada(2, 1, string([]rune(csvLongo)[:72])+"...")},
		{"json em várias linhas", []string{"--input", "json"},
			"\n\n[\n  {\"a\":\"x\",\"z\":1},\n  {\n    \"a\": \"y\",\n    \"z\": 0\n  }\n]\n", parteDadosCLI(5, 2, "  {")},
		{"json com um objeto por linha", []string{"--input", "json"},
			"{\"a\":\"x\",\"z\":1}\n{\"a\":\"y\",\"z\":0}\n", parteDadosCLI(2, 2, `{"a":"y","z":0}`)},
		{"json compacto longo", []string{"--input", "json"}, antes + objeto + "]\n",
			cortada(1, 7, "..."+string([]rune(objeto)[:72])+"...")},
		{"json compacto com até 72 no resto", []string{"--input", "json"}, antes + `{"a":"y","z":0}]` + "\n",
			cortada(1, 7, `...{"a":"y","z":0}]`)},
		{"latin-1", []string{"--encoding", "latin-1"}, string(codificar("João|1\nJosé|0\n", "latin-1")),
			parteDadosCLI(2, 2, "José|0")},
		{"cp850", []string{"--encoding", "cp850"}, string(codificar("João|1\nJosé|0\n", "cp850")),
			parteDadosCLI(2, 2, "José|0")},
	} {
		t.Run(c.nome, func(t *testing.T) {
			want := erroCLI("divisão por zero na seção detail", parteScriptCLI(t, script, "10 / z"),
				"conta: 10 / z, com z = 0", c.parte)
			// o stderr sai na codificação do texto interno (UTF-8)
			confereErroExato(t, "arquivo", c.opcoes, script, c.dados, want)
			rp, _ := arquivosLinha(t, script, "")
			args := append(append([]string{}, c.opcoes...), rp, "-")
			if code, out, errb := executa(t, c.dados, args...); code != 1 || out != "" || errb != want {
				t.Errorf("stdin: code=%d stdout=%q\n--- got ---\n%q\n--- want ---\n%q", code, out, errb, want)
			}
		})
	}
}

// Com o sort, o número e a linha são os da entrada; a medida da altura do 'footer' lê o primeiro
// registro ordenado, e um erro de comparação mostra os dois registros na ordem da entrada.
func TestParteDadosDepoisDoSort(t *testing.T) {
	const medida = "input k [numeric], z [numeric]\nsort k desc\nfooter\n   print k / z\ndetail\n   print 1 / 0\nend\n"
	confereErroExato(t, "medida do footer com cabeçalho", nil, medida, "k|z\n1|5\n3|0\n2|5\n", erroCLI(
		"divisão por zero na seção footer", parteScriptCLI(t, medida, "k / z"),
		"conta: k / z, com k = 3 e z = 0", parteDadosCLI(3, 2, "3|0")))

	const comparacao = "input n [numeric], d [date]\nsort index(n - 1, n, d) desc\nend\n"
	for _, c := range []struct {
		nome              string
		opcoes            []string
		dados             string
		primeiro, segundo string
	}{
		{"delimitado", nil, "n|d\n2|2026-02-01\n1|2026-01-01\n",
			parteDadosCLI(2, 1, "2|2026-02-01"), parteDadosCLI(3, 2, "1|2026-01-01")},
		{"csv", []string{"--input", "csv"}, "n,d\n\n2,2026-02-01\n\n1,2026-01-01\n",
			parteDadosCLI(3, 1, "2,2026-02-01"), parteDadosCLI(5, 2, "1,2026-01-01")},
		{"json", []string{"--input", "json"},
			"[\n{\"n\":2,\"d\":\"2026-02-01\"},\n{\"n\":1,\"d\":\"2026-01-01\"}\n]\n",
			parteDadosCLI(2, 1, `{"n":2,"d":"2026-02-01"},`), parteDadosCLI(3, 2, `{"n":1,"d":"2026-01-01"}`)},
	} {
		want := erroCLI("no sort: não é possível comparar uma data com um número na seção sort",
			parteScriptCLI(t, comparacao, "index(n - 1, n, d)"), c.primeiro, c.segundo)
		for _, saida := range opcoesSaida(t) {
			opcoes := append(append([]string{}, c.opcoes...), saida.opcoes...)
			confereErroExato(t, c.nome+" "+saida.nome, opcoes, comparacao, c.dados, want)
		}
	}
}

// Outros erros de execução, com a seção, o trecho e o registro, sem a conta.
func TestOutrosErrosDeExecucaoNaLinhaDeComando(t *testing.T) {
	const entrada = "input n [numeric], d [date], s\ndetail\n   "
	for _, c := range []struct{ nome, comando, trecho, mensagem string }{
		{"nome desconhecido", "print zz", "zz", "nome desconhecido: 'zz'"},
		{"dow", "print dow(n)", "dow", "a função dow() espera uma data"},
		{"substr", "print n substr[1-2]", "substr", "substr só pode ser usado com texto"},
		{"comparação", "if d = n then print n", "d = n", "não é possível comparar uma data com um número"},
		{"concatenação", "print s /+ n", "s /+ n", "o operador /+ só pode ser usado com texto"},
		{"data", "print d / 0", "d / 0", "operação inválida com datas"},
		{"coluna", "print n col (n - 1)", "(n - 1)", "número de coluna inválido: 0"},
		{"máscara", "print n using '%x'", "'%x'", `formato inválido: "%x"`},
	} {
		script := entrada + c.comando + "\nend\n"
		confereErroExato(t, c.nome, nil, script, "1|2026-01-01|ab\n", erroCLI(
			c.mensagem+" na seção detail", parteScriptCLI(t, script, c.trecho),
			parteDadosCLI(1, 1, "1|2026-01-01|ab")))
	}
}

// Erros no 'header' e no 'footer' param o relatório em qualquer forma de quebrar a página, com
// --form-feed, --lines e --strict.
func TestErroDePaginacaoNasOpcoes(t *testing.T) {
	const inicio = "input n [numeric], z [numeric]\nleft margin 0\ntop margin 0\nbottom margin 0\nlength 3\n"
	const dados = "1|1\n2|1\n3|0\n4|0\n"
	for _, secao := range []string{"header", "footer"} {
		for _, q := range []struct {
			nome, detalhe string
			registro      int // o registro que causa a quebra
		}{
			{"página cheia", "   print n", 4},
			{"page", "   if n = 3 then page\n   print n", 3},
			{"need", "   if n = 3 then need 9\n   print n", 3},
			{"skip", "   if n = 3 then skip 9\n   print n", 3},
		} {
			// 'q' só deixa de ser nula no detail: a medida do footer e o primeiro header não
			// dividem; 'z' é zero a partir do registro 3
			script := inicio + secao + "\n   if q = 1 and n > 2 then print 10 / z\ndetail\n   set q to 1\n" + q.detalhe + "\nend\n"
			want := erroCLI("divisão por zero na seção "+secao, parteScriptCLI(t, script, "10 / z"),
				"conta: 10 / z, com z = 0", parteDadosCLI(q.registro, q.registro, fmt.Sprintf("%d|0", q.registro)))
			for _, saida := range []struct {
				nome   string
				opcoes []string
			}{
				{"texto", nil},
				{"form-feed", []string{"--form-feed"}},
				{"strict", []string{"--strict"}},
				{"strict e form-feed", []string{"--strict", "--form-feed"}},
				{"html", argsHTML(t)},
				{"lines", argsHTML(t, "--lines", "3")},
				{"lines e strict", argsHTML(t, "--lines", "3", "--strict")},
			} {
				confereErroExato(t, secao+" "+q.nome+" "+saida.nome, saida.opcoes, script, dados, want)
			}
		}
	}

	// --lines muda onde a página quebra, e com isso o registro que o header lê
	linhas := "input n [numeric]\nheader\n   if pageno > 1 then print 1 / 0\ndetail\n   print n\nend\n"
	for _, c := range []struct {
		paginas  string
		registro int
	}{{"2", 3}, {"4", 5}} {
		confereErroExato(t, "--lines "+c.paginas, argsHTML(t, "--lines", c.paginas), linhas, "1\n2\n3\n4\n5\n", erroCLI(
			"divisão por zero na seção header", parteScriptCLI(t, linhas, "1 / 0"), "conta: 1 / 0",
			parteDadosCLI(c.registro, c.registro, fmt.Sprint(c.registro))))
	}

	// com --strict e 'before report' a primeira página não roda o header
	estrito := inicio + "before report\n   print 'x'\nheader\n   print 10 / z\ndetail\n   print n\nend\n"
	for _, c := range []struct {
		opcoes   []string
		dados    string
		registro int
		linha    string
	}{
		{nil, "1|0\n2|1\n3|0\n", 1, "1|0"},
		{[]string{"--strict"}, "1|0\n2|1\n3|0\n", 3, "3|0"},
	} {
		confereErroExato(t, "before report "+strings.Join(c.opcoes, " "), c.opcoes, estrito, c.dados, erroCLI(
			"divisão por zero na seção header", parteScriptCLI(t, estrito, "10 / z"),
			"conta: 10 / z, com z = 0", parteDadosCLI(c.registro, c.registro, c.linha)))
	}
}

// Um agregado no 'footer' que só é avaliado no fechamento da última página: antes a linha se
// perdia com código 0.
func TestAgregadoNoFooterDaUltimaPagina(t *testing.T) {
	const script = "input n [numeric]\nleft margin 0\ntop margin 0\nbottom margin 0\nlength 3\n" +
		"footer\n   if pageno > 1 then print total(n)\ndetail\n   print n\nend\n"
	want := erroCLI("a função total() só pode ser usada em 'after sort_key' e 'after report' na seção footer",
		parteScriptCLI(t, script, "total"), parteDadosCLI(4, 4, "4"))
	confereErroExato(t, "texto", nil, script, "1\n2\n3\n4\n", want)
	confereErroExato(t, "html", argsHTML(t), script, "1\n2\n3\n4\n", want)

	// com uma página só o agregado não é avaliado, e o relatório sai
	rp, dat := arquivosLinha(t, script, "1\n2\n")
	if code, out, errb := executa(t, "", rp, dat); code != 0 || out != "1\n2\n\n" || errb != "" {
		t.Errorf("uma página: code=%d stdout=%q stderr=%q", code, out, errb)
	}
}

// O módulo continua sem dependências externas.
func TestModuloSemDependencias(t *testing.T) {
	raiz := filepath.Join("..", "..")
	for linha := range strings.SplitSeq(lerArquivo(t, filepath.Join(raiz, "go.mod")), "\n") {
		if campos := strings.Fields(linha); len(campos) > 0 && campos[0] == "require" {
			t.Errorf("go.mod não deve ter 'require': %q", linha)
		}
	}
	if _, err := os.Stat(filepath.Join(raiz, "go.sum")); !os.IsNotExist(err) {
		t.Errorf("não deve existir go.sum (err=%v)", err)
	}
}
