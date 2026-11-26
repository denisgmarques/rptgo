package main

import (
	"path/filepath"
	"testing"
)

// Estes testes fixam o comportamento anterior às mudanças nas mensagens de erro de script: o que
// é válido continua com a mesma saída, e os erros de dados e de tipo lógico não mudam.

func TestRegressaoScriptsDoRepositorio(t *testing.T) {
	var scripts []string
	for _, padrao := range []string{"../../examples/*/*.rp", "../../internal/report/testdata/golden/*.rp"} {
		achados, err := filepath.Glob(padrao)
		if err != nil {
			t.Fatal(err)
		}
		scripts = append(scripts, achados...)
	}
	if len(scripts) < 10 {
		t.Fatalf("poucos scripts encontrados: %v", scripts)
	}
	for _, s := range scripts {
		if code, out, errb := executa(t, "", "--check", s); code != 0 || out != "Script sem erros de sintaxe.\n" || errb != "" {
			t.Errorf("%s: code=%d stdout=%q stderr=%q", s, code, out, errb)
		}
	}
}

func TestRegressaoScriptsValidos(t *testing.T) {
	for _, c := range []struct{ nome, script, dados, saida string }{
		{"colunas com nome de palavra-chave",
			"input date, end, sort, detail, width\nleft margin 0\nlength 20\ntop margin 0\nbottom margin 0\ndetail\n   print date, end, sort, detail, width\nend\n",
			"a|b|c|d|e\n",
			"a b c d e\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n"},
		{"ultima coluna depois de virgula",
			"input nome, end\nleft margin 0\nlength 20\ntop margin 0\nbottom margin 0\ndetail\n   print nome, end\nend\n",
			"a|b\n",
			"a b\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n"},
		{"coluna sort e chave",
			"input nome, sort\nleft margin 0\nlength 20\ntop margin 0\nbottom margin 0\nsort sort\ndetail\n   print nome, sort\nend\n",
			"a|2\nb|1\n",
			"b 1\na 2\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n"},
		{"sort por coluna",
			"input n [numeric]\nleft margin 0\nlength 20\ntop margin 0\nbottom margin 0\nsort n desc\ndetail\n print n\nend\n",
			"1\n3\n2\n",
			"3\n2\n1\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n"},
		{"sort com expressao nomeada",
			"input data [ldate]\nleft margin 0\nlength 20\ntop margin 0\nbottom margin 0\nsort <mes> mdy(data, 0)\nbefore mes\n print 'mes', mes\ndetail\n print data\nend\n",
			"2026-03-01\n2026-01-05\n",
			"mes 1\n05/01/2026\nmes 3\n01/03/2026\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n"},
		{"sort por nome declarado num print",
			"input n [numeric]\nleft margin 0\nlength 20\ntop margin 0\nbottom margin 0\nsort dobro\ndetail\n print <dobro> n * -1\nend\n",
			"1\n3\n2\n",
			"-3\n-2\n-1\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n"},
		{"sort por variavel",
			"input n [numeric]\nleft margin 0\nlength 20\ntop margin 0\nbottom margin 0\nsort v\nbefore report\n set v to 1\nbefore v\n print 'antes'\ndetail\n print n\n set v to n\nafter v\n print 'depois'\nend\n",
			"2\n1\n",
			"antes\n2\ndepois\nantes\n1\ndepois\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n"},
		{"apelido da propria coluna",
			"input g [string]\nleft margin 0\nlength 20\ntop margin 0\nbottom margin 0\nsort <g> g\ndetail\n print g\nend\n",
			"b\na\n",
			"a\nb\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n"},
		{"sort com embutidas e funcoes",
			"input d [ldate], n [numeric]\nleft margin 0\nlength 20\ntop margin 0\nbottom margin 0\nsort today, hour, pageno, null, dow(d), mdy(d, 1), index(0, n), d substr[1-2]\ndetail\n print d, n\nend\n",
			"2026-03-02|2\n2026-01-01|1\n",
			"02/03/2026 2\n01/01/2026 1\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n"},
		{"agregado em print nomeado fora do sort",
			"input n [numeric]\nleft margin 0\nlength 20\ntop margin 0\nbottom margin 0\nsort n\ndetail\n print n\nafter report\n print <soma> total(n)\nend\n",
			"2\n1\n",
			"1\n2\n3\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n"},
	} {
		rp, dat := arquivosLinha(t, c.script, c.dados)
		if code, out, errb := executa(t, "", rp, dat); code != 0 || out != c.saida || errb != "" {
			t.Errorf("%s: code=%d stderr=%q saída=%q", c.nome, code, errb, out)
		}
		destino := filepath.Join(t.TempDir(), "saida")
		if code, _, errb := executa(t, "", "-o", destino, rp, dat); code != 0 || lerArquivo(t, destino) != c.saida {
			t.Errorf("%s com -o: code=%d stderr=%q", c.nome, code, errb)
		}
		if code, _, errb := executa(t, "", "--check", rp); code != 0 {
			t.Errorf("%s com --check: code=%d stderr=%q", c.nome, code, errb)
		}
	}
}

func TestRegressaoErrosSemMudanca(t *testing.T) {
	for _, c := range []struct{ nome, script, dados, erro string }{
		{"valor invalido em linha longa",
			"input a, n [numeric]\nend\n",
			"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx|abc\n",
			"rptgo: valor inválido em 'n' (registro 1): \"abc\" (strconv.ParseFloat: parsing \"abc\": invalid syntax)\n  1 | xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx|abc\n      ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n"},
		{"linha indentada",
			"input a, n [numeric]\nend\n",
			"    z|abc\n",
			"rptgo: valor inválido em 'n' (registro 1): \"abc\" (strconv.ParseFloat: parsing \"abc\": invalid syntax)\n  1 |     z|abc\n      ^^^^^\n"},
		{"linha com tabs",
			"input a, n [numeric]\nend\n",
			"\tz\t|abc\n",
			"rptgo: valor inválido em 'n' (registro 1): \"abc\" (strconv.ParseFloat: parsing \"abc\": invalid syntax)\n  1 | \tz\t|abc\n      ^^^^^^\n"},
		{"campos demais",
			"input a, b\nend\n",
			"    xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx|b|c\n",
			"rptgo: phase error no arquivo de entrada: registro 1 tem 3 campos, esperados 2\n  1 |     xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx|b|c\n      ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n"},
		{"condicao aritmetica",
			"input n [numeric]\ndetail\n   if n + 1 then print n\nend\n",
			"1\n",
			"rptgo: linha 3: a condição do 'if' precisa ser uma expressão lógica (comparação, 'is null', and/or/not ou coluna bool), não aritmética\n  3 |    if n + 1 then print n\n      ^^^^^^^^^^^^^^^^^^^^^\n"},
	} {
		rp, dat := arquivosLinha(t, c.script, c.dados)
		if code, _, errb := executa(t, "", rp, dat); code != 1 || errb != c.erro {
			t.Errorf("%s: code=%d stderr=%q, want %q", c.nome, code, errb, c.erro)
		}
	}
}
