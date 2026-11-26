package report

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func scriptValido(t *testing.T, texto string) *Script {
	t.Helper()
	toks, err := Tokenize(texto, false)
	if err != nil {
		t.Fatal(err)
	}
	s, err := Parse(toks)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// escreve mostra o trecho de um nó: as palavras de ini a fim, a palavra do foco e a linha dela.
func escreve(s *Script, p posicao) string {
	if p == (posicao{}) {
		return "(sem posição)"
	}
	var palavras []string
	for _, tk := range s.toks[p.ini : p.fim+1] {
		palavras = append(palavras, tk.Src)
	}
	foco := s.toks[p.foco]
	return fmt.Sprintf("%s @ %s linha %d", strings.Join(palavras, " "), foco.Src, foco.Line)
}

// nos lista, em pré-ordem, os nós com posição de uma expressão.
func nos(s *Script, e Expr) []string {
	anota := func(tipo string, p posicao, filhos ...Expr) []string {
		out := []string{tipo + ": " + escreve(s, p)}
		for _, f := range filhos {
			out = append(out, nos(s, f)...)
		}
		return out
	}
	junta := func(filhos ...Expr) (out []string) {
		for _, f := range filhos {
			out = append(out, nos(s, f)...)
		}
		return out
	}
	switch x := e.(type) {
	case ColExpr:
		return anota("col", x.pos)
	case CmpExpr:
		return anota("cmp", x.pos, x.L, x.R)
	case BinExpr:
		return anota("bin", x.pos, x.L, x.R)
	case SubstrExpr:
		return anota("substr", x.pos, x.X)
	case CallExpr:
		return anota("call", x.pos, append(append([]Expr{}, x.Args...), x.Where)...)
	case NegExpr:
		return junta(x.X)
	case NotExpr:
		return junta(x.X)
	case IsNullExpr:
		return junta(x.X)
	case AndExpr:
		return junta(x.L, x.R)
	case OrExpr:
		return junta(x.L, x.R)
	}
	return nil
}

func itensDoDetail(t *testing.T, s *Script) []PrintItem {
	t.Helper()
	corpo := s.Events[EventKey{Kind: "detail"}]
	if len(corpo) != 1 {
		t.Fatalf("detail inesperado: %#v", corpo)
	}
	return corpo[0].(PrintStmt).Items
}

func TestPosicaoDasOperacoes(t *testing.T) {
	for _, c := range []struct {
		nome, expr string
		want       []string
	}{
		{"duas divisões na linha", "a / b, c / d", []string{
			"bin: a / b @ / linha 3", "col: a @ a linha 3", "col: b @ b linha 3",
			"bin: c / d @ / linha 3", "col: c @ c linha 3", "col: d @ d linha 3"}},
		{"aninhada", "a / (b / c)", []string{
			"bin: a / ( b / c ) @ / linha 3", "col: a @ a linha 3",
			"bin: b / c @ / linha 3", "col: b @ b linha 3", "col: c @ c linha 3"}},
		{"parênteses à esquerda", "(a + 1) / z", []string{
			"bin: ( a + 1 ) / z @ / linha 3", "bin: a + 1 @ + linha 3", "col: a @ a linha 3", "col: z @ z linha 3"}},
		{"parênteses dos dois lados", "((a)) % (z)", []string{
			"bin: ( ( a ) ) % ( z ) @ % linha 3", "col: a @ a linha 3", "col: z @ z linha 3"}},
		{"cadeia associativa à esquerda", "a / b / c", []string{
			"bin: a / b / c @ / linha 3", "bin: a / b @ / linha 3",
			"col: a @ a linha 3", "col: b @ b linha 3", "col: c @ c linha 3"}},
		{"precedência", "a + b * c", []string{
			"bin: a + b * c @ + linha 3", "col: a @ a linha 3",
			"bin: b * c @ * linha 3", "col: b @ b linha 3", "col: c @ c linha 3"}},
		{"sinais", "-a / +b", []string{
			"bin: - a / + b @ / linha 3", "col: a @ a linha 3", "col: b @ b linha 3"}},
		{"várias linhas", "(a +\n 1)\n /\n z", []string{
			"bin: ( a + 1 ) / z @ / linha 5", "bin: a + 1 @ + linha 3", "col: a @ a linha 3", "col: z @ z linha 6"}},
		{"soma de datas", "a /+ 1", []string{"bin: a /+ 1 @ /+ linha 3", "col: a @ a linha 3"}},
		{"comparação", "(a / b) >= c + 1", []string{
			"cmp: ( a / b ) >= c + 1 @ >= linha 3", "bin: a / b @ / linha 3", "col: a @ a linha 3",
			"col: b @ b linha 3", "bin: c + 1 @ + linha 3", "col: c @ c linha 3"}},
		{"comparação depois de is null", "a is null = b", []string{
			"cmp: a is null = b @ = linha 3", "col: a @ a linha 3", "col: b @ b linha 3"}},
		{"lógicos", "not a = 1 and b <> 2", []string{
			"cmp: a = 1 @ = linha 3", "col: a @ a linha 3", "cmp: b <> 2 @ <> linha 3", "col: b @ b linha 3"}},
		{"função marca o nome", "mdy(a, 1) / z", []string{
			"bin: mdy ( a , 1 ) / z @ / linha 3", "call: mdy @ mdy linha 3", "col: a @ a linha 3", "col: z @ z linha 3"}},
		{"agregado com where", "count(* where a / b > 0)", []string{
			"call: count @ count linha 3", "cmp: a / b > 0 @ > linha 3", "bin: a / b @ / linha 3",
			"col: a @ a linha 3", "col: b @ b linha 3"}},
		{"substr marca a palavra", "(a) SubStr[1-2] / z", []string{
			"bin: ( a ) SubStr [ 1 - 2 ] / z @ / linha 3", "substr: SubStr @ SubStr linha 3",
			"col: a @ a linha 3", "col: z @ z linha 3"}},
	} {
		t.Run(c.nome, func(t *testing.T) {
			s := scriptValido(t, "input a, b, c, d, z\ndetail\n print "+c.expr+"\nend\n")
			var got []string
			for _, item := range itensDoDetail(t, s) {
				got = append(got, nos(s, item.Expr)...)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(c.want, "\n"))
			}
		})
	}
}

func TestPosicaoDeColunaMascaraEChave(t *testing.T) {
	s := scriptValido(t, `input a, b
sort <x> (a + 1) / b desc,
     a,
     <y>
     dow(b)
detail
 print a col 1 + b using 'xx', b in col (a) centered using
   a substr[1-2]
 print a
end
`)
	confere := func(nome string, got posicao, want string) {
		t.Helper()
		if txt := escreve(s, got); txt != want {
			t.Errorf("%s: got %q, want %q", nome, txt, want)
		}
	}
	confere("chave com apelido", s.Sort[0].pos, "( a + 1 ) / b @ ( linha 2")
	confere("chave simples", s.Sort[1].pos, "a @ a linha 3")
	confere("chave na linha de baixo do apelido", s.Sort[2].pos, "dow ( b ) @ dow linha 5")

	corpo := s.Events[EventKey{Kind: "detail"}]
	itens := corpo[0].(PrintStmt).Items
	confere("coluna", itens[0].posCol, "1 + b @ 1 linha 7")
	confere("máscara", itens[0].posMascara, "'xx' @ 'xx' linha 7")
	confere("coluna com 'in col'", itens[1].posCol, "( a ) @ ( linha 7")
	confere("máscara na linha de baixo", itens[1].posMascara, "a substr [ 1 - 2 ] @ a linha 8")
	semOpcoes := corpo[1].(PrintStmt).Items[0]
	confere("sem coluna", semOpcoes.posCol, "(sem posição)")
	confere("sem máscara", semOpcoes.posMascara, "(sem posição)")
}

// Todos os tipos de nó numa expressão só.
const expressaoComTudo = `not (-a / (b % 2) >= mdy(c substr[1-2], 1) and a is not null
  or count(* where a = 1) > total(a where b < 2) or index(0, a, 'x') = today or hour = pageno)`

func TestNomeadaIgualEmLinhasDiferentes(t *testing.T) {
	// o sort não aceita agregados: as três declarações ficam em prints
	s := scriptValido(t, "input a, b, c\ndetail\n print <x> "+expressaoComTudo+
		"\n print <x> "+expressaoComTudo+"\n print <x>\n\t("+expressaoComTudo+")\nend\n")
	if len(s.Nomeadas) != 1 {
		t.Fatalf("nomeadas: %v", s.Nomeadas)
	}
	// a primeira declaração (linhas 3 e 4) é a que fica
	if p := s.Nomeadas["x"].(NotExpr).X.(OrExpr).L.(OrExpr).L.(OrExpr).R.(CmpExpr).pos; s.toks[p.foco].Line != 4 {
		t.Errorf("declaração guardada: %s", escreve(s, p))
	}
}

func TestNomeadaDiferenteContinuaErro(t *testing.T) {
	for _, outra := range []string{
		strings.Replace(expressaoComTudo, "% 2", "% 3", 1),
		strings.Replace(expressaoComTudo, "substr[1-2]", "substr[1-3]", 1),
		strings.Replace(expressaoComTudo, "b < 2", "c < 2", 1),
		strings.Replace(expressaoComTudo, "a = 1", "a <> 1", 1),
		strings.Replace(expressaoComTudo, "'x'", "'x', 'y'", 1),
		strings.Replace(expressaoComTudo, "-a", "a", 1),
	} {
		got := erroDoScript(t, "input a, b, c\ndetail\n print <x> "+expressaoComTudo+"\n print <x> "+outra+"\nend\n", false)
		if !strings.Contains(got, "já foi dado a outra expressão") {
			t.Errorf("%q: erro = %q", outra, got)
		}
	}
}

func TestNomeadaIgualNoSortENoPrint(t *testing.T) {
	const e = "mdy(a, 1) / (b substr[1-2]) >= index(0, -c % 2)"
	s := scriptValido(t, "input a, b, c\nsort <x> "+e+"\ndetail\n print a,\n  <x> ("+e+")\nend\n")
	if got := escreve(s, s.Nomeadas["x"].(CmpExpr).pos); !strings.HasSuffix(got, "@ >= linha 2") {
		t.Errorf("declaração guardada: %s", got)
	}
}

// semPosicao zera as posições de todos os nós, sem mexer na expressão original.
func TestSemPosicaoZeraTodosOsNos(t *testing.T) {
	s := scriptValido(t, "input a, b, c\ndetail\n print "+expressaoComTudo+"\nend\n")
	e := itensDoDetail(t, s)[0].Expr
	antes := nos(s, e)
	limpa := nos(s, semPosicao(e))
	if len(antes) < 15 || len(limpa) != len(antes) {
		t.Fatalf("nós: %d antes, %d depois", len(antes), len(limpa))
	}
	for i := range antes {
		if strings.Contains(antes[i], "(sem posição)") {
			t.Errorf("nó sem posição: %s", antes[i])
		}
		if !strings.HasSuffix(limpa[i], "(sem posição)") {
			t.Errorf("nó com posição: %s", limpa[i])
		}
	}
	if depois := nos(s, e); !reflect.DeepEqual(depois, antes) {
		t.Errorf("a expressão original mudou:\n%s", strings.Join(depois, "\n"))
	}
}

func TestTrechoDeLinhaLongaDoScript(t *testing.T) {
	s := sequencia(200)
	brancos := func(n int) string { return strings.Repeat(" ", n) }
	for _, c := range []struct {
		nome            string
		linha           string
		col, tam, foco  int
		mostrada, marca string
	}{
		{"linha curta", "  print a / b", 8, 5, 10,
			"  print a / b", brancos(8) + circunflexos(5)},
		{"trecho curto centralizado", s, 100, 10, 105,
			"..." + s[69:141] + "...", brancos(3+31) + circunflexos(10)},
		{"trecho curto no fim", s, 195, 5, 197,
			"..." + s[128:], brancos(3+67) + circunflexos(5)},
		{"trecho de 71 contém o operador no fim", s, 50, 71, 119,
			"..." + s[50:122] + "...", brancos(3) + circunflexos(71)},
		{"trecho longo começa nele", s, 50, 100, 60,
			"..." + s[50:122] + "...", brancos(3) + circunflexos(72)},
		{"operador na última coluna do corte", s, 50, 100, 121,
			"..." + s[50:122] + "...", brancos(3) + circunflexos(72)},
		{"operador logo depois do corte", s, 50, 100, 122,
			"..." + s[86:158] + "...", brancos(3) + circunflexos(64)},
		{"operador longe do começo", s, 10, 180, 150,
			"..." + s[114:186] + "...", brancos(3) + circunflexos(72)},
		{"operador perto do fim da linha", s, 0, 200, 195,
			"..." + s[128:], brancos(3) + circunflexos(72)},
		{"trecho longo que acaba dentro do corte", s, 0, 90, 80,
			"..." + s[44:116] + "...", brancos(3) + circunflexos(46)},
		{"tabs antes do marcador", "\ta\t" + s, 3, 3, 4,
			"\ta\t" + s[:69] + "...", "\t \t" + circunflexos(3)},
		{"CRLF não vai no trecho", "a / b\r", 0, 5, 2,
			"a / b", circunflexos(5)},
	} {
		t.Run(c.nome, func(t *testing.T) {
			antes := FonteScript
			t.Cleanup(func() { FonteScript = antes })
			DefFonteScript(c.linha + "\n")
			fonte, marca, ok := trechoScript(1, c.col, c.tam, c.foco)
			wantMarca := brancos(len("  1 | ")) + c.marca
			if !ok || fonte != "  1 | "+c.mostrada || marca != wantMarca {
				t.Errorf("ok=%v\n got %q\n     %q\nwant %q\n     %q", ok, fonte, marca, "  1 | "+c.mostrada, wantMarca)
			}
		})
	}
}

// Numa operação longa com tabs e acentos, o operador está sempre no que é mostrado.
func TestTrechoDoScriptContemOOperador(t *testing.T) {
	antes := FonteScript
	t.Cleanup(func() { FonteScript = antes })
	for _, tamEsq := range []int{0, 30, 69, 70, 71, 72, 90, 150} {
		for _, tamDir := range []int{1, 40, 100} {
			linha := "\tprint 'ção', " + strings.Repeat("é", tamEsq) + " % " + strings.Repeat("ã", tamDir) + ", 'fim'"
			DefFonteScript(linha)
			runas := []rune(linha)
			col, foco := 14, 14+tamEsq+1
			fonte, marca, ok := trechoScript(1, col, tamEsq+3+tamDir, foco)
			if !ok || runas[foco] != '%' || !strings.Contains(fonte, "%") {
				t.Errorf("esq=%d dir=%d: sem o operador: %q", tamEsq, tamDir, fonte)
				continue
			}
			mostrado := []rune(strings.TrimPrefix(fonte, "  1 | "))
			corpo := strings.TrimSuffix(strings.TrimPrefix(string(mostrado), "..."), "...")
			if n := len([]rune(corpo)); n > larguraTrecho {
				t.Errorf("esq=%d dir=%d: %d runas mostradas", tamEsq, tamDir, n)
			}
			// o primeiro "^" fica sob o começo do trecho ou do que é mostrado dele
			i := strings.IndexByte(marca, '^') - len("  1 | ")
			if i < 0 || i >= len(mostrado) || (mostrado[i] != 'é' && mostrado[i] != '%' && mostrado[i] != ' ') {
				t.Errorf("esq=%d dir=%d: marcador fora do lugar: %q\n%q", tamEsq, tamDir, fonte, marca)
			}
		}
	}
}

func TestTrechoDoScriptSemLinha(t *testing.T) {
	antes := FonteScript
	t.Cleanup(func() { FonteScript = antes })
	DefFonteScript("a\n")
	for _, linha := range []int{-1, 0, 2} {
		if _, _, ok := trechoScript(linha, 0, 1, 0); ok {
			t.Errorf("linha %d: deveria falhar", linha)
		}
	}
}
