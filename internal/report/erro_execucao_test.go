package report

import (
	"fmt"
	"strings"
	"testing"
)

// interpDeExecucao registra o script e os dados como fontes de erro (restauradas no fim do teste)
// e devolve o Interp pronto pra rodar. Os campos dos dados são separados por '|'.
func interpDeExecucao(t *testing.T, script string, dados []string) *Interp {
	t.Helper()
	antes := FonteScript
	t.Cleanup(func() { FonteScript = antes })
	DefFonteScript(script)
	fonteDeDados(t, strings.Join(dados, "\n")+"\n")
	regs := make([][]string, len(dados))
	for i, d := range dados {
		regs[i] = strings.Split(strings.TrimSuffix(d, "\r"), "|")
	}
	it, err := NovoInterp(scriptValido(t, script), regs, false)
	if err != nil {
		t.Fatal(err)
	}
	return it
}

// erroDeExecucao roda o relatório e devolve a mensagem do erro.
func erroDeExecucao(t *testing.T, script string, dados []string) string {
	t.Helper()
	return erroDoRun(t, interpDeExecucao(t, script, dados))
}

func erroDoRun(t *testing.T, it *Interp) string {
	t.Helper()
	saida, err := it.Run()
	if err == nil {
		t.Fatalf("esperado um erro, veio a saída %q", saida)
	}
	if saida != "" {
		t.Errorf("não deveria haver saída junto do erro: %q", saida)
	}
	return err.Error()
}

// trechoMarcado recebe um script com a construção que falha entre « » (numa linha só) e devolve o
// script sem as marcas e a parte 'script' esperada no erro.
func trechoMarcado(t *testing.T, marcado string) (script, parte string) {
	t.Helper()
	antes, resto, ok1 := strings.Cut(marcado, "«")
	trecho, depois, ok2 := strings.Cut(resto, "»")
	if !ok1 || !ok2 || strings.Contains(trecho, "\n") {
		t.Fatalf("caso sem « » numa linha só: %q", marcado)
	}
	script = antes + trecho + depois
	linha := strings.Count(antes, "\n") + 1
	fonte := strings.Split(script, "\n")[linha-1]
	prefixo := fmt.Sprintf("  %d | ", linha)
	pad := strings.Repeat(" ", len(prefixo))
	for _, c := range antes[strings.LastIndex(antes, "\n")+1:] {
		if c == '\t' {
			pad += "\t"
		} else {
			pad += " "
		}
	}
	return script, fmt.Sprintf("script, linha %d:\n%s%s\n%s%s", linha, prefixo, fonte, pad, circunflexos(len([]rune(trecho))))
}

// dadosDe é a parte 'dados' esperada de um registro cuja linha é o próprio número dele.
func dadosDe(registro int, linha string) string {
	return fmt.Sprintf("dados, linha %d (registro %d):\n  %d | %s", registro, registro, registro, linha)
}

// casoDeExecucao é um erro de execução esperado por inteiro: a mensagem com a seção, o trecho
// marcado no script, a conta (só na divisão por zero) e os registros lidos.
type casoDeExecucao struct {
	nome     string
	marcado  string
	dados    []string
	mensagem string
	conta    string
	registro []int
}

func confereCasos(t *testing.T, casos []casoDeExecucao) {
	t.Helper()
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			script, parte := trechoMarcado(t, c.marcado)
			want := []string{c.mensagem, parte}
			if c.conta != "" {
				want = append(want, "conta: "+c.conta)
			}
			for _, n := range c.registro {
				want = append(want, dadosDe(n, c.dados[n-1]))
			}
			if got, want := erroDeExecucao(t, script, c.dados), strings.Join(want, "\n"); got != want {
				t.Errorf("script:\n%s\n--- got ---\n%s\n--- want ---\n%s", script, got, want)
			}
		})
	}
}

// Cada linha da tabela de RF-06: a mensagem de sempre, a seção, a construção marcada e o registro.
func TestErroDeExecucaoMarcaAConstrucao(t *testing.T) {
	const entrada = "input n [numeric], d [date], h [time], s\n"
	dados := []string{"1|2026-01-01|10:00|ab", "2|2026-02-01|11:00|cd"}
	detalhe := func(comando string) string {
		return entrada + "detail\n   if n = 2 then " + comando + "\nend\n"
	}
	casos := []struct{ nome, comando, mensagem string }{
		{"nome desconhecido", "print n, «zz»", "nome desconhecido: 'zz'"},
		{"nome desconhecido no if", "if «zz» = 1 then print n", "nome desconhecido: 'zz'"},
		{"nome desconhecido no set", "set v to 1 + «zz»", "nome desconhecido: 'zz'"},
		{"dow", "print n + «dow»(n)", "a função dow() espera uma data"},
		{"mdy", "print «mdy»(s, 0)", "a função mdy() espera uma data"},
		{"função desconhecida", "print n, «foo»(n)", "'foo' é um nome de função desconhecido"},
		{"substr", "print s, n «substr»[1-2]", "substr só pode ser usado com texto"},
		{"data com número", "if «d = n» then print n", "não é possível comparar uma data com um número"},
		{"número com data", "if n > 1 and «(n + 1) < d» then print n", "não é possível comparar uma data com um número"},
		{"hora com outro tipo", "if «h >= n» then print n", "não é possível comparar uma hora com outro tipo"},
		{"concatenação sem texto", "print s, «s /+ n»", "o operador /+ só pode ser usado com texto"},
		{"operação com datas", "print «d * 2»", "operação inválida com datas"},
		{"divisão de data", "print «d / 0»", "operação inválida com datas"},
		{"operação com horas", "print «h + h»", "operação inválida com horas"},
		{"resto de hora", "print «h % 0»", "operação inválida com horas"},
		{"coluna", "print n col «n - 2»", "número de coluna inválido: 0"},
		{"coluna com 'in col'", "print n in col «(0)» using '99'", "número de coluna inválido: 0"},
		{"máscara de texto", "print s using «'0'»", `máscara de texto inválida: "0" (a largura começa em 1)`},
		{"formato numérico", "print n using «'%x'»", `formato inválido: "%x"`},
		{"máscara calculada", "print n col 2 using «'%' + 'q'»", `formato inválido: "%q"`},
	}
	var todos []casoDeExecucao
	for _, c := range casos {
		todos = append(todos, casoDeExecucao{nome: c.nome, marcado: detalhe(c.comando), dados: dados,
			mensagem: c.mensagem + " na seção detail", registro: []int{2}})
	}
	// em detail um agregado calcula sobre um grupo vazio: fora de um grupo ficam header e footer
	for _, f := range []string{"count", "total", "avg", "min", "max"} {
		todos = append(todos, casoDeExecucao{nome: f + " fora de um grupo",
			marcado: entrada + "header\n   print n, «" + f + "»(n)\nend\n", dados: dados,
			mensagem: "a função " + f + "() só pode ser usada em 'after sort_key' e 'after report' na seção header",
			registro: []int{1}})
	}
	todos = append(todos, casoDeExecucao{
		nome:     "expressão nomeada que se refere a si mesma",
		marcado:  entrada + "detail\n   print a\nafter report\n   print <a> «a» + 1\nend\n",
		dados:    dados,
		mensagem: "a expressão nomeada 'a' refere-se a si mesma na seção detail",
		registro: []int{1},
	})
	confereCasos(t, todos)
}

// A seção é a que está rodando e o registro é o que ela lê.
func TestDivisaoPorZeroEmCadaSecao(t *testing.T) {
	const entrada = "input g [string], a [numeric], z [numeric]\nsort g\n"
	// o divisor zera no registro 2, que é o último do grupo 'x', e no 3, que abre o grupo 'y'
	dados := []string{"x|10|1", "x|20|0", "y|30|0", "y|40|2"}
	caso := func(nome, secoes, secao, conta string, registro int) casoDeExecucao {
		return casoDeExecucao{nome: nome, marcado: entrada + secoes + "\nend\n", dados: dados,
			mensagem: "divisão por zero na seção " + secao, conta: conta, registro: []int{registro}}
	}
	zeroNoPrimeiro := []string{"x|10|0", "x|20|1"}
	primeiro := func(nome, secoes, secao string) casoDeExecucao {
		c := caso(nome, secoes, secao, "a / z, com a = 10 e z = 0", 1)
		c.dados = zeroNoPrimeiro
		return c
	}
	confereCasos(t, []casoDeExecucao{
		caso("detail", "detail\n   print «a / z»", "detail", "a / z, com a = 20 e z = 0", 2),
		caso("resto", "detail\n   print «a % z»", "detail", "a % z, com a = 20 e z = 0", 2),
		caso("before da chave", "before g\n   print «a / z»", "before g", "a / z, com a = 30 e z = 0", 3),
		caso("after da chave", "after g\n   print «a / z»", "after g", "a / z, com a = 20 e z = 0", 2),
		caso("after report", "after report\n   print «a / (z - 2)»", "after report",
			"a / (z - 2), com a = 40 e (z - 2) = 0", 4),
		primeiro("before report", "before report\n   print «a / z»", "before report"),
		primeiro("header na primeira página", "header\n   print «a / z»", "header"),
		primeiro("header depois do before report", "before report\n   print 'x'\nheader\n   print «a / z»", "header"),
		primeiro("footer na medida da altura", "footer\n   print «a / z»", "footer"),
		primeiro("dentro de if e begin", "detail\n   if a > 0 then begin\n      set v to «a / z»\n   end", "detail"),
	})
}

func TestDivisaoPorZeroOperandos(t *testing.T) {
	const entrada = "input a [numeric], z [numeric], f [float], s, b [bool], d [date]\n"
	// 2026-01-04 é um domingo: dow() devolve 0
	const registro = "7||0||0|2026-01-04"
	caso := func(nome, antes, expr, conta string) casoDeExecucao {
		return casoDeExecucao{nome: nome, marcado: entrada + "detail\n" + antes + "   print «" + expr + "»\nend\n",
			dados: []string{registro}, mensagem: "divisão por zero na seção detail", conta: conta, registro: []int{1}}
	}
	casos := []casoDeExecucao{
		caso("só constantes", "", "1 / 0", "1 / 0"),
		caso("constantes com sinal", "", "-5 % -0", "-5 % -0"),
		caso("texto e nulo constantes", "", "'x' / null",
			`'x' / null, com 'x' = texto "x" (conta como 0) e null = nulo (conta como 0)`),
		caso("texto vazio constante", "", "10 / ''", "10 / '', com '' = texto vazio (conta como 0)"),
		caso("nulo constante no resto", "", "10 % null", "10 % null, com null = nulo (conta como 0)"),
		caso("nulo constante com sinal", "", "10 / +null", "10 / +null, com +null = nulo (conta como 0)"),
		caso("texto vazio constante no dividendo", "", "'' / 0", "'' / 0, com '' = texto vazio (conta como 0)"),
		caso("repetição vazia", "", "10 / 0[x]", "10 / 0[x], com 0[x] = texto vazio (conta como 0)"),
		caso("constante decimal", "", "1.5 / 0.0", "1.5 / 0.0"),
		caso("constante entre parênteses", "", "1 / (0)", "1 / (0), com (0) = 0"),
		caso("só o divisor é coluna", "", "10 / f", "10 / f, com f = 0"),
		caso("nulo", "", "a / z", "a / z, com a = 7 e z = nulo (conta como 0)"),
		caso("texto vazio", "", "a % s", "a % s, com a = 7 e s = texto vazio (conta como 0)"),
		caso("texto", "   set t to 'abc \"x\"'\n", "a / t", `a / t, com a = 7 e t = texto "abc \"x\"" (conta como 0)`),
		caso("falso", "", "a / b", "a / b, com a = 7 e b = falso (conta como 0)"),
		caso("verdadeiro no dividendo", "", "(a > 1) / (a < 1)",
			"(a > 1) / (a < 1), com (a > 1) = 1 e (a < 1) = 0"),
		{nome: "verdadeiro", marcado: entrada + "detail\n   print «b / f»\nend\n", dados: []string{"7||0||1|"},
			mensagem: "divisão por zero na seção detail",
			conta:    "b / f, com b = verdadeiro (conta como 1) e f = 0", registro: []int{1}},
		caso("variável", "   set v to a - 7\n", "a / v", "a / v, com a = 7 e v = 0"),
		caso("número grande sem expoente", "   set v to 100000000.0 * 100000000.0 * 10000\n", "v / f",
			"v / f, com v = 100000000000000000000 e f = 0"),
		caso("decimal", "", "a * 0.5 / f", "a * 0.5 / f, com a * 0.5 = 3.5 e f = 0"),
		caso("expressão nomeada", "   print <k> f * 2\n", "a / k", "a / k, com a = 7 e k = 0"),
		caso("função", "", "a / dow(d)", "a / dow(d), com a = 7 e dow(d) = 0"),
		caso("negação", "", "a / -f", "a / -f, com a = 7 e -f = 0"),
		caso("pageno", "", "a % (pageno - 1)", "a % (pageno - 1), com a = 7 e (pageno - 1) = 0"),
		caso("substr", "", "a / s substr[1-1]", "a / s substr[1-1], com a = 7 e s substr[1-1] = texto vazio (conta como 0)"),
		caso("espaços como escritos", "", "a/(z  *  2)", "a/(z * 2), com a = 7 e (z * 2) = 0"),
	}
	confereCasos(t, casos)
}

func TestDivisaoPorZeroMarcaSoAOperacaoQueFalhou(t *testing.T) {
	const entrada = "input a [numeric], b [numeric], c [numeric], d [numeric]\n"
	caso := func(nome, print, dados, conta string) casoDeExecucao {
		return casoDeExecucao{nome: nome, marcado: entrada + "detail\n\tprint " + print + "\nend\n", dados: []string{dados},
			mensagem: "divisão por zero na seção detail", conta: conta, registro: []int{1}}
	}
	confereCasos(t, []casoDeExecucao{
		caso("segunda da linha", "a / b, «c / d»", "1|2|3|0", "c / d, com c = 3 e d = 0"),
		caso("primeira da linha", "«a / b», c / d", "1|0|3|0", "a / b, com a = 1 e b = 0"),
		caso("de dentro", "a / («b / c»)", "5|1|0|0", "b / c, com b = 1 e c = 0"),
		caso("de fora", "«a / (b / c)»", "5|0|1|0", "a / (b / c), com a = 5 e (b / c) = 0"),
		caso("parênteses à esquerda", "'ç', «(a + 1) / (d)»", "5|0|1|0", "(a + 1) / (d), com (a + 1) = 6 e (d) = 0"),
		caso("cadeia", "«a / b / d»", "8|2|0|0", "a / b / d, com a / b = 4 e d = 0"),
	})
}

func TestDivisaoPorZeroEmVariasLinhas(t *testing.T) {
	const entrada = "input a [numeric], z [numeric]\ndetail\n"
	for _, c := range []struct{ nome, corpo, script, conta string }{
		// só o último 'a' é o operando esquerdo da divisão
		{"operador na primeira linha", "   print a + 1 + a /\n      (z)\n",
			"script, linha 3:\n  3 |    print a + 1 + a /\n                       ^^^",
			"a / (z), com a = 5 e (z) = 0"},
		{"operador na segunda linha", "   print (a +\n      1) % z + 1\n",
			"script, linha 4:\n  4 |       1) % z + 1\n            ^^^^^^",
			"(a + 1) % z, com (a + 1) = 6 e z = 0"},
		{"operador sozinho na linha", "   print a\n      /\n      z\n",
			"script, linha 4:\n  4 |       /\n            ^",
			"a / z, com a = 5 e z = 0"},
		{"comentário no meio", "   print a /* ç */ / /* d\n */ z\n",
			"script, linha 3:\n  3 |    print a /* ç */ / /* d\n               ^^^^^^^^^^^",
			"a / z, com a = 5 e z = 0"},
	} {
		t.Run(c.nome, func(t *testing.T) {
			got := erroDeExecucao(t, entrada+c.corpo+"end\n", []string{"5|0"})
			want := "divisão por zero na seção detail\n" + c.script + "\nconta: " + c.conta + "\n" + dadosDe(1, "5|0")
			if got != want {
				t.Errorf("--- got ---\n%s\n--- want ---\n%s", got, want)
			}
		})
	}
}

// De uma linha longa do script sai um trecho com o operador, com o marcador alinhado mesmo com
// tabs e acentos.
func TestDivisaoPorZeroEmLinhaLongaDoScript(t *testing.T) {
	for _, antes := range []int{0, 50, 120} {
		linha := "\tprint '" + strings.Repeat("é", antes) + "',\ta / z, '" + strings.Repeat("ã", 90) + "'"
		got := erroDeExecucao(t, "input a [numeric], z [numeric]\ndetail\n"+linha+"\nend\n", []string{"5|0"})
		linhas := strings.Split(got, "\n")
		if len(linhas) != 7 || linhas[1] != "script, linha 3:" || linhas[4] != "conta: a / z, com a = 5 e z = 0" {
			t.Fatalf("antes=%d:\n%s", antes, got)
		}
		fonte, marca := []rune(linhas[2]), linhas[3]
		trecho := strings.TrimPrefix(linhas[2], "  3 | ")
		if n := len([]rune(strings.TrimSuffix(strings.TrimPrefix(trecho, "..."), "..."))); n != larguraTrecho {
			t.Errorf("antes=%d: %d runas mostradas", antes, n)
		}
		if !strings.HasSuffix(trecho, "...") || strings.HasPrefix(trecho, "...") != (antes > 0) {
			t.Errorf("antes=%d: cortes: %q", antes, trecho)
		}
		i := strings.IndexByte(marca, '^')
		if i < 0 || marca[i:] != "^^^^^" || i+5 > len(fonte) || string(fonte[i:i+5]) != "a / z" {
			t.Fatalf("antes=%d: marcador fora do lugar:\n%s\n%s", antes, string(fonte), marca)
		}
		for j, c := range marca[:i] {
			if (c == '\t') != (fonte[j] == '\t') {
				t.Errorf("antes=%d: tab desalinhado na posição %d:\n%q\n%q", antes, j, string(fonte), marca)
			}
		}
	}
}

// Numa operação de uma expressão nomeada, a linha é a da declaração que foi avaliada.
func TestDivisaoPorZeroEmExpressaoNomeada(t *testing.T) {
	const entrada = "input a [numeric], z [numeric]\n"
	confereCasos(t, []casoDeExecucao{
		{"declarada em outra seção", entrada + "detail\n   print u\nafter report\n   print <u> «a / z»\nend\n",
			[]string{"5|0"}, "divisão por zero na seção detail", "a / z, com a = 5 e z = 0", []int{1}},
		{"declaração repetida, usada pelo nome", entrada + "detail\n   print u\nafter report\n   print <u> «a / z»\n   print <u> a / z\nend\n",
			[]string{"5|0"}, "divisão por zero na seção detail", "a / z, com a = 5 e z = 0", []int{1}},
		{"declaração repetida, avaliada onde está", entrada + "before report\n   if 1 = 0 then print <u> a / z\ndetail\n   print <u> «a / z»\nend\n",
			[]string{"5|0"}, "divisão por zero na seção detail", "a / z, com a = 5 e z = 0", []int{1}},
		{"declarada no sort", entrada + "sort <u> «a / z»\ndetail\n   print u\nend\n",
			[]string{"5|0"}, "divisão por zero na seção sort", "a / z, com a = 5 e z = 0", []int{1}},
	})
}

// Num agregado o registro é o do grupo que está sendo calculado; com agregados como operandos, o
// que a seção lê.
func TestDivisaoPorZeroEmAgregados(t *testing.T) {
	const entrada = "input g [string], a [numeric], z [numeric]\nsort g\n"
	dados := []string{"x|10|1", "x|20|0", "x|30|1", "y|40|0"}
	var casos []casoDeExecucao
	for _, f := range []string{"count", "total", "avg", "min", "max"} {
		casos = append(casos, casoDeExecucao{f + " no argumento",
			entrada + "after g\n   print " + f + "(«a / z»)\nend\n", dados,
			"divisão por zero na seção after g", "a / z, com a = 20 e z = 0", []int{2}})
		casos = append(casos, casoDeExecucao{f + " no where",
			entrada + "after report\n   print " + f + "(a where «a / z» > 0)\nend\n", dados,
			"divisão por zero na seção after report", "a / z, com a = 20 e z = 0", []int{2}})
	}
	casos = append(casos,
		casoDeExecucao{"count(*) no where", entrada + "after report\n   print count(* where «a % z» = 0)\nend\n", dados,
			"divisão por zero na seção after report", "a % z, com a = 20 e z = 0", []int{2}},
		casoDeExecucao{"operandos agregados no after da chave", entrada + "after g\n   print «total(a) / total(z - z)»\nend\n", dados,
			"divisão por zero na seção after g", "total(a) / total(z - z), com total(a) = 60 e total(z - z) = 0", []int{3}},
		casoDeExecucao{"operandos agregados no after report", entrada + "after report\n   print «count(*) % total(z where a > 100)»\nend\n", dados,
			"divisão por zero na seção after report",
			"count(*) % total(z where a > 100), com count(*) = 4 e total(z where a > 100) = 0", []int{4}},
		casoDeExecucao{"depois de um agregado volta o registro da seção", entrada + "after g\n   print total(a), «a / (a - 30)»\nend\n", dados,
			"divisão por zero na seção after g", "a / (a - 30), com a = 30 e (a - 30) = 0", []int{3}},
	)
	confereCasos(t, casos)
}

// Com o sort reordenando os registros, o número e a linha continuam os da entrada.
func TestErroDeExecucaoDepoisDoSort(t *testing.T) {
	const entrada = "input k [numeric], z [numeric]\nsort k desc\n"
	dados := []string{"1|5", "3|0", "2|5"}
	confereCasos(t, []casoDeExecucao{
		{"detail", entrada + "detail\n   print «k / z»\nend\n", dados,
			"divisão por zero na seção detail", "k / z, com k = 3 e z = 0", []int{2}},
		// o footer é medido antes de tudo, no primeiro registro ordenado
		{"medida do footer", entrada + "header\n   print 1 / 0\nfooter\n   print «1 / z»\nend\n", dados,
			"divisão por zero na seção footer", "1 / z, com z = 0", []int{2}},
		{"after report lê o último", entrada + "after report\n   print «k / (k - 1)»\nend\n", dados,
			"divisão por zero na seção after report", "k / (k - 1), com k = 1 e (k - 1) = 0", []int{1}},
	})
}

// A medida da altura do footer usa pageno 1 e o primeiro registro ordenado.
func TestErroNaMedidaDoFooter(t *testing.T) {
	confereCasos(t, []casoDeExecucao{
		{"valores da medida", "input a [numeric]\nsort a\nfooter\n   print «a / (pageno - 1)»\nend\n",
			[]string{"9", "4"}, "divisão por zero na seção footer",
			"a / (pageno - 1), com a = 4 e (pageno - 1) = 0", []int{2}},
		{"agregado", "input a [numeric]\nfooter\n   print «total»(a)\ndetail\n   print a\nend\n",
			[]string{"9"}, "a função total() só pode ser usada em 'after sort_key' e 'after report' na seção footer",
			"", []int{1}},
	})
}

func TestErroDeExecucaoUsaOLocalizadorDeDados(t *testing.T) {
	longa := "2|0|" + sequencia(100)
	it := interpDeExecucao(t, "input a [numeric], z [numeric], s\ndetail\n   print a / z\nend\n", []string{"1|1|x", longa})
	fonteDeDados(t, "a|z|s\r\n1|1|x\r\n\r\n"+longa+"\r\n")
	DefLocalizadorDados(func(registro int) PosicaoDados { return PosicaoDados{Linha: registro * 2} })
	want := "divisão por zero na seção detail\n" +
		"script, linha 3:\n  3 |    print a / z\n               ^^^^^\n" +
		"conta: a / z, com a = 2 e z = 0\n" +
		"dados, linha 4 (registro 2):\n  4 | " + longa[:72] + "..."
	if got := erroDoRun(t, it); got != want {
		t.Errorf("--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

// Sem o texto ou sem a posição, só a linha ou a parte que falta é omitida.
func TestErroDeExecucaoSemFontes(t *testing.T) {
	const script = "input z [numeric]\ndetail\n   print 1 / z\nend\n"
	it := interpDeExecucao(t, script, []string{"0"})
	FonteScript, FonteDados = nil, nil
	want := "divisão por zero na seção detail\nscript, linha 3:\nconta: 1 / z, com z = 0\ndados, linha 1 (registro 1):"
	if got := erroDoRun(t, it); got != want {
		t.Errorf("sem textos: got %q, want %q", got, want)
	}

	it = interpDeExecucao(t, script, []string{"0"})
	DefLocalizadorDados(func(int) PosicaoDados { return PosicaoDados{} })
	if got := erroDoRun(t, it); !strings.HasSuffix(got, "com z = 0\ndados (registro 1):") {
		t.Errorf("registro não localizado: %q", got)
	}

	// uma árvore sem posições (montada fora do Parse) fica sem a parte do script
	it = interpDeExecucao(t, script, []string{"0"})
	it.S.Events[EventKey{Kind: "detail"}] = []Stmt{PrintStmt{Items: []PrintItem{
		{Expr: BinExpr{Op: "/", L: ConstExpr{Val: int64(1)}, R: ColExpr{Name: "z"}}}}}}
	want = "divisão por zero na seção detail\ndados, linha 1 (registro 1):\n  1 | 0"
	if got := erroDoRun(t, it); got != want {
		t.Errorf("sem posições: got %q, want %q", got, want)
	}
}

// Os erros que não vêm da avaliação de uma seção ou de uma chave continuam só com a mensagem.
func TestErroDePreparacaoNaoMuda(t *testing.T) {
	it := interpDeExecucao(t, "input a\ndetail\n print a\nend\n", nil)
	it.rows = nil
	if got := erroDoRun(t, it); got != "não há registros de entrada" {
		t.Errorf("sem registros: %q", got)
	}
	_, err := NovoInterp(scriptValido(t, "input a\nbefore g\n print a\nend\n"), [][]string{{"1"}}, false)
	if err == nil || err.Error() != "'g' na seção before não é uma expressão de sort existente" {
		t.Errorf("evento sem chave: %v", err)
	}
}

func TestErroNaChaveDoSort(t *testing.T) {
	const entrada = "input n [numeric], d [date], z [numeric]\n"
	dados := []string{"1|2026-01-01|1", "2|2026-02-01|0", "3|2026-03-01|0"}
	confereCasos(t, []casoDeExecucao{
		{"divisão direta", entrada + "sort «n / z»\ndetail\n   print n\nend\n", dados,
			"divisão por zero na seção sort", "n / z, com n = 2 e z = 0", []int{2}},
		{"segunda chave com desc", entrada + "sort n desc, «n % z» desc\ndetail\n   print n\nend\n", dados,
			"divisão por zero na seção sort", "n % z, com n = 2 e z = 0", []int{2}},
		{"expressão nomeada declarada em outro lugar", entrada + "sort u\ndetail\n   print <u> «n / z»\nend\n", dados,
			"divisão por zero na seção sort", "n / z, com n = 2 e z = 0", []int{2}},
		{"pageno é 0 durante o sort", entrada + "sort «n / pageno»\ndetail\n   print n\nend\n", dados,
			"divisão por zero na seção sort", "n / pageno, com n = 1 e pageno = 0", []int{1}},
		{"variável é nula durante o sort", entrada + "sort «n / v»\ndetail\n   set v to 1\nend\n", dados,
			"divisão por zero na seção sort", "n / v, com n = 1 e v = nulo (conta como 0)", []int{1}},
		// a chave também é calculada durante a execução, já com pageno 1: no sort ele valia 0, o
		// que inverteu a ordem
		{"só durante a execução", entrada + "sort «n / (pageno - 1)»\ndetail\n   print n\nend\n", dados,
			"divisão por zero na seção sort", "n / (pageno - 1), com n = 3 e (pageno - 1) = 0", []int{3}},
		{"tipo errado", entrada + "sort «dow»(n)\ndetail\n   print n\nend\n", dados,
			"no sort: a função dow() espera uma data na seção sort", "", []int{1}},
		{"tipo errado em outro registro", entrada + "sort n, <k> index(z, d, n) = «mdy»(index(z, n, d), 0)\ndetail\n   print n\nend\n",
			[]string{"1|2026-01-01|1", "2|2026-02-01|0"},
			"no sort: a função mdy() espera uma data na seção sort", "", []int{2}},
		{"comparação dentro da chave", entrada + "sort «d > z»\ndetail\n   print n\nend\n", dados,
			"no sort: não é possível comparar uma data com um número na seção sort", "", []int{1}},
		{"chave nula não esconde a divisão", entrada + "sort d substr[1-2], «n / z»\ndetail\n   print n\nend\n", dados,
			"divisão por zero na seção sort", "n / z, com n = 2 e z = 0", []int{2}},
		{"o primeiro registro da entrada que falha", entrada + "sort «1 / z», n\ndetail\n   print 1 / 0\nend\n",
			[]string{"9||1", "8||1", "7||0", "1||0"},
			"divisão por zero na seção sort", "1 / z, com z = 0", []int{3}},
	})
}

// Duas chaves que não se comparam: a chave marcada desde o começo e os dois registros, o de
// número menor primeiro, nos dois sentidos.
func TestErroDeComparacaoNoSort(t *testing.T) {
	const entrada = "input n [numeric], d [date], h [time]\n"
	// com n = 1 a chave é um número; com n = 2, uma data
	var casos []casoDeExecucao
	for _, dados := range [][]string{
		{"1|2026-01-01|10:00", "2|2026-02-01|11:00"},
		{"2|2026-02-01|11:00", "1|2026-01-01|10:00"},
	} {
		for _, sentido := range []string{"", " desc"} {
			casos = append(casos, casoDeExecucao{"data e número" + sentido + " " + dados[0][:1],
				entrada + "sort «index(n - 1, n, d)»" + sentido + "\ndetail\n   print n\nend\n", dados,
				"no sort: não é possível comparar uma data com um número na seção sort", "", []int{1, 2}})
		}
	}
	casos = append(casos,
		casoDeExecucao{"hora na segunda chave", entrada + "sort d, <k> «(index(n - 1, n, h))» desc\ndetail\n   print n\nend\n",
			[]string{"1|2026-01-01|10:00", "2|2026-01-01|11:00"},
			"no sort: não é possível comparar uma hora com outro tipo na seção sort", "", []int{1, 2}},
	)
	confereCasos(t, casos)

	t.Run("chave em várias linhas", func(t *testing.T) {
		got := erroDeExecucao(t, entrada+"sort index(n - 1,\n   n, d)\ndetail\n   print n\nend\n",
			[]string{"1|2026-01-01|", "2|2026-02-01|"})
		want := "no sort: não é possível comparar uma data com um número na seção sort\n" +
			"script, linha 2:\n  2 | sort index(n - 1,\n           ^^^^^^^^^^^^\n" +
			dadosDe(1, "1|2026-01-01|") + "\n" + dadosDe(2, "2|2026-02-01|")
		if got != want {
			t.Errorf("--- got ---\n%s\n--- want ---\n%s", got, want)
		}
	})

	t.Run("registros reordenados por um sort anterior", func(t *testing.T) {
		it := interpDeExecucao(t, entrada+"sort index(n - 1, n, d)\nend\n",
			[]string{"2|2026-02-01|", "3||", "1|2026-01-01|"})
		it.rows[0], it.rows[2] = it.rows[2], it.rows[0]
		it.numeros = []int{3, 2, 1}
		_, err := it.LinhasOrdenadas()
		if err == nil || !strings.HasSuffix(err.Error(), dadosDe(1, "2|2026-02-01|")+"\n"+dadosDe(3, "1|2026-01-01|")) {
			t.Errorf("erro: %v", err)
		}
	})
}

// As três saídas do interpretador montam o mesmo erro.
func TestErroDeExecucaoEmTodasAsSaidas(t *testing.T) {
	const script = "input z [numeric]\nsort 1 / z\ndetail\n   print z\nend\n"
	want := "divisão por zero na seção sort\nscript, linha 2:\n  2 | sort 1 / z\n           ^^^^^\n" +
		"conta: 1 / z, com z = 0\n" + dadosDe(2, "0")
	dados := []string{"1", "0"}
	if got := erroDeExecucao(t, script, dados); got != want {
		t.Errorf("Run: %q", got)
	}
	if linhas, err := interpDeExecucao(t, script, dados).LinhasOrdenadas(); linhas != nil || err == nil || err.Error() != want {
		t.Errorf("LinhasOrdenadas: %v %v", linhas, err)
	}
	if paginas, err := interpDeExecucao(t, script, dados).Paginas(0); paginas != nil || err == nil || err.Error() != want {
		t.Errorf("Paginas: %v %v", paginas, err)
	}
}

// Um erro descartado numa chave do sort deixa a chave nula e nunca é montado.
func TestErroSilenciosoNoSortNaoEMontado(t *testing.T) {
	it := interpDeExecucao(t, "input d [date], n [numeric]\nsort d substr[1-2], n\ndetail\n   print n\nend\n",
		[]string{"2026-01-01|2", "2026-01-02|1"})
	DefLocalizadorDados(func(registro int) PosicaoDados {
		t.Errorf("o registro %d não deveria ser procurado", registro)
		return PosicaoDados{}
	})
	saida, err := it.Run()
	if err != nil || strings.Join(strings.Fields(saida), " ") != "1 2" {
		t.Errorf("saída %q, erro %v", saida, err)
	}
}
