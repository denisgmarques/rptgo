package report

import (
	"strings"
	"testing"
)

// Os erros do 'header' e do 'footer', em qualquer página, param o relatório: nenhum é descartado
// na quebra nem no fechamento da página.

// Os dois comandos do item B3 do backlog, sem e com uma linha de cabeçalho nos dados: o registro
// é o 5, que causa a quebra, na linha que o localizador indicar.
func TestErroDePaginacaoComandosB3(t *testing.T) {
	dados := []string{"1|1", "2|1", "3|1", "4|0", "5|0"}
	for _, secao := range []string{"header", "footer"} {
		script := "input n [numeric], z [numeric] top margin 0 bottom margin 0 length 3 " +
			secao + " print 'h', 10 / z detail print n end"
		for _, c := range []struct {
			nome      string
			cabecalho string
			parte     string
		}{
			{"sem cabeçalho", "", "dados, linha 5 (registro 5):\n  5 | 5|0"},
			{"com cabeçalho", "n|z\n", "dados, linha 6 (registro 5):\n  6 | 5|0"},
		} {
			t.Run(secao+" "+c.nome, func(t *testing.T) {
				it := interpDeExecucao(t, script, dados)
				if c.cabecalho != "" {
					fonteDeDados(t, c.cabecalho+strings.Join(dados, "\n")+"\n")
					DefLocalizadorDados(func(registro int) PosicaoDados { return PosicaoDados{Linha: registro + 1} })
				}
				got := erroDoRun(t, it)
				if inicio := "divisão por zero na seção " + secao + "\nscript, linha 1:\n"; !strings.HasPrefix(got, inicio) {
					t.Errorf("começo: got %q, want %q", got, inicio)
				}
				if fim := "\nconta: 10 / z, com z = 0\n" + c.parte; !strings.HasSuffix(got, fim) {
					t.Errorf("fim: got %q, want %q", got, fim)
				}
			})
		}
	}
}

// Um 'header' ou um 'footer' rodado numa quebra de página para o relatório como as outras
// seções, e o registro é o que ele leu: na quebra, o que a causou.
func TestErroNoHeaderENoFooterNaQuebraDePagina(t *testing.T) {
	const inicio = "input n [numeric], z [numeric]\ntop margin 0\nbottom margin 0\nlength 3\n"
	// com o header ou o footer ocupando uma das 3 linhas, o registro 5 causa a segunda quebra e é
	// o primeiro lido com divisor zero
	dados := []string{"1|1", "2|1", "3|1", "4|1", "5|0", "6|0"}
	const contaB3 = "10 / z, com z = 0"
	caso := func(nome, secoes, secao, conta string, registro int) casoDeExecucao {
		return casoDeExecucao{nome: nome, marcado: inicio + secoes + "\nend\n", dados: dados,
			mensagem: "divisão por zero na seção " + secao, conta: conta, registro: []int{registro}}
	}
	casos := []casoDeExecucao{
		caso("header numa página seguinte",
			"header\n   print 'h', «10 / z»\ndetail\n   print n", "header", contaB3, 5),
		caso("footer numa quebra de página",
			"footer\n   print 'f', «10 / z»\ndetail\n   print n", "footer", contaB3, 5),
		caso("resto no header",
			"header\n   print «n % z»\ndetail\n   print n", "header", "n % z, com n = 5 e z = 0", 5),
		caso("decimal no footer",
			"footer\n   print «1.5 % (z * 1.0)»\ndetail\n   print n", "footer",
			"1.5 % (z * 1.0), com (z * 1.0) = 0", 5),
		caso("footer no fechamento da última página",
			"footer\n   if pageno > 1 then print «n / (n - 6)»\ndetail\n   print n",
			"footer", "n / (n - 6), com n = 6 e (n - 6) = 0", 6),
		caso("set no header",
			"header\n   if pageno > 1 then set v to «1 / 0»\ndetail\n   print n", "header", "1 / 0", 4),
	}
	// cada forma de quebrar a página propaga o erro do footer e o do header da página nova; 'q'
	// ainda é nula na medida do footer e no header da primeira página, que não imprimem nada
	for _, q := range []struct {
		nome, detalhe string
		registro      int
	}{
		{"page", "   if n = 2 then page\n   print n", 2},
		{"need", "   if n = 2 then need 9\n   print n", 2},
		{"skip", "   if n = 2 then skip 9\n   print n", 2},
		{"página cheia", "   print n", 4},
		{"print sem quebra de linha antes do page", "   print n no newline\n   if n = 2 then page", 2},
		{"linha extra de uma máscara de texto", "   if n = 2 then print 'a b c d e' using '1' else print n", 2},
	} {
		for _, secao := range []string{"header", "footer"} {
			casos = append(casos, caso(secao+" com "+q.nome,
				secao+"\n   if q = 1 then print «1 / 0»\ndetail\n   set q to 1\n"+q.detalhe,
				secao, "1 / 0", q.registro))
		}
	}
	// o primeiro erro encerra o relatório: nem o header nem o detail seguintes rodam
	casos = append(casos, caso("só o primeiro erro",
		"header\n   if pageno > 1 then print n / 0\nfooter\n   if pageno = 1 and n > 1 then print «n / 0»\n"+
			"detail\n   if n = 2 then page\n   print n / (n - 2)",
		"footer", "n / 0, com n = 2", 2))
	// numa quebra dentro do detail, o agregado do header calcula sobre o grupo vazio dele
	casos = append(casos, caso("agregado herdado do detail",
		"header\n   if pageno > 1 then print «1 / count(*)»\ndetail\n   print n",
		"header", "1 / count(*), com count(*) = 0", 4))
	confereCasos(t, casos)
}

// Os outros erros de execução do 'header' e do 'footer' também param o relatório.
func TestOutrosErrosNoHeaderENoFooter(t *testing.T) {
	const inicio = "input n [numeric]\ntop margin 0\nbottom margin 0\nlength 3\n"
	dados := []string{"1", "2", "3", "4"}
	confereCasos(t, []casoDeExecucao{
		{"nome desconhecido no header da quebra",
			inicio + "header\n   if pageno > 1 then print «zz»\ndetail\n   print n\nend\n", dados,
			"nome desconhecido: 'zz' na seção header", "", []int{4}},
		{"dow no footer do fechamento",
			inicio + "footer\n   if pageno > 1 then print «dow»(n)\ndetail\n   print n\nend\n", dados,
			"a função dow() espera uma data na seção footer", "", []int{4}},
		{"agregado no footer do fechamento",
			inicio + "footer\n   if pageno > 1 then print «total»(n)\ndetail\n   print n\nend\n", dados,
			"a função total() só pode ser usada em 'after sort_key' e 'after report' na seção footer", "", []int{4}},
	})
}

// A quebra de página propaga o erro do header e o do footer nas saídas paginadas, com ou sem
// --form-feed, --lines e --strict.
func TestErroNaQuebraDePaginaNasSaidas(t *testing.T) {
	dados := []string{"1", "2", "3", "4"}
	for _, secao := range []string{"header", "footer"} {
		script := "input n [numeric]\ntop margin 0\nbottom margin 0\nlength 3\n" +
			secao + "\n   if q = 1 then print 1 / 0\nbefore report\n   print 'x'\n" +
			"detail\n   set q to 1\n   print n\nend\n"
		inicio := "divisão por zero na seção " + secao + "\n"
		for _, formFeed := range []bool{false, true} {
			for _, strict := range []bool{false, true} {
				it := interpDeExecucao(t, script, dados)
				it.FormFeed, it.preenche, it.Strict = formFeed, !formFeed, strict
				if got := erroDoRun(t, it); !strings.HasPrefix(got, inicio) {
					t.Errorf("%s form-feed=%v strict=%v: %q", secao, formFeed, strict, got)
				}
			}
		}
		// --lines troca o número de linhas por página da saída HTML
		for _, linhas := range []int{0, 2} {
			paginas, err := interpDeExecucao(t, script, dados).Paginas(linhas)
			if paginas != nil || err == nil || !strings.HasPrefix(err.Error(), inicio) {
				t.Errorf("%s Paginas(%d): %v %v", secao, linhas, paginas, err)
			}
		}
	}
}

// Depois do primeiro erro nada mais roda: nem o 'header' depois de um 'footer' que falhou, nem os
// comandos que vêm depois da quebra, e só um erro é relatado.
func TestErroDePaginacaoNadaRodaDepoisDoPrimeiroErro(t *testing.T) {
	const entrada = "input n [numeric]\ntop margin 0\nbottom margin 0\nlength 3\n"
	for _, c := range []struct{ nome, quebra string }{
		{"page", "page"},
		{"need", "need 9"},
		{"skip", "skip 9"},
		{"página cheia", "begin print 'a' print 'b' print 'c' end"},
	} {
		t.Run(c.nome, func(t *testing.T) {
			it := interpDeExecucao(t, entrada+
				"header\n   if q = 1 then begin set cabecalho to 1 print zz end\n"+
				"footer\n   if q = 1 then begin print 1 / 0 set rodape to 1 end\n"+
				"detail\n   set q to 1\n   if n = 2 then begin\n      "+c.quebra+"\n      set depois to 1\n   end\n"+
				"   print n\n   set ultimo to n\n"+
				"after report\n   set fim to 1\nend\n", []string{"1", "2", "3"})
			got := erroDoRun(t, it)
			if !strings.HasPrefix(got, "divisão por zero na seção footer\n") {
				t.Errorf("erro: %q", got)
			}
			if n := strings.Count(got, " na seção "); n != 1 {
				t.Errorf("esperado um só erro, vieram %d: %q", n, got)
			}
			for _, v := range []string{"cabecalho", "rodape", "depois", "fim"} {
				if _, rodou := it.vars[v]; rodou {
					t.Errorf("o comando 'set %s' não deveria ter rodado", v)
				}
			}
			if it.vars["ultimo"] != int64(1) {
				t.Errorf("só o detail do primeiro registro deveria ter terminado: %v", it.vars["ultimo"])
			}
		})
	}
}

// O que fica pendente no fim do relatório e a última página também propagam o erro: a linha
// aberta com 'no newline' que não cabe mais na página a quebra só no fechamento.
func TestErroDePaginacaoNoFechamentoDoRelatorio(t *testing.T) {
	const inicio = "input n [numeric]\ntop margin 0\nbottom margin 0\nlength 3\n"
	dados := []string{"1", "2"}
	const fim = "\ndetail\n   print n\nafter report\n   set q to 1\n   print 'x'\n   print 'y' no newline\nend\n"
	confereCasos(t, []casoDeExecucao{
		{"header da quebra da linha pendente", inicio + "header\n   if q = 1 then print «1 / 0»" + fim, dados,
			"divisão por zero na seção header", "1 / 0", []int{2}},
		{"footer da quebra da linha pendente", inicio + "footer\n   if q = 1 then print «1 / 0»" + fim, dados,
			"divisão por zero na seção footer", "1 / 0", []int{2}},
	})
}
