package report

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Contexto dos erros de execução. Ele é anexado em dois passos: onde o erro nasce são anotados o
// trecho do script, o registro que estava sendo lido e, na divisão por zero, os valores dos
// operandos; o limite da seção acrescenta o nome dela. O texto é montado uma vez só, e apenas
// para o erro que vai ser mostrado: um erro descartado numa chave do sort nunca chega aqui.

// secaoSort é o nome da seção dos erros nas chaves do 'sort'.
const secaoSort = "sort"

// marcaOrigem anota no erro o trecho do script que falhou e o registro que estava sendo lido. Só a
// primeira anotação vale.
func (it *Interp) marcaOrigem(err error, p posicao) error {
	if e, ok := err.(*Error); ok && !e.marcado {
		e.marcado, e.pos, e.linha = true, p, it.row
	}
	return err
}

// marcaConta é marcaOrigem para uma operação, guardando também os valores dos dois operandos.
func (it *Interp) marcaConta(err error, p posicao, a, b Value) error {
	if e, ok := err.(*Error); ok && !e.marcado {
		e.operandos = [2]Value{a, b}
	}
	return it.marcaOrigem(err, p)
}

// naSecao dá ao erro o nome da seção, se ele ainda não tiver um: vale a seção mais interna.
func naSecao(err error, secao string) error {
	if e, ok := err.(*Error); ok && e.secao == "" {
		e.secao = secao
	}
	return err
}

// rodaSecao executa os comandos da seção k. É o limite de seção dos erros de execução; o nome da
// seção só é montado quando há um erro.
func (it *Interp) rodaSecao(k EventKey, corpo []Stmt) error {
	err := it.exec(corpo)
	if err == nil {
		return nil
	}
	nome := k.Kind
	if k.Target != "" {
		nome += " " + k.Target
	}
	return naSecao(err, nome)
}

// relata monta o texto de um erro de execução que vai ser mostrado: a mensagem com a seção, o
// trecho do script, a conta (na divisão por zero) e o registro lido, ou os dois registros de uma
// comparação do sort, na ordem da entrada. Um erro sem seção fica como está.
func (it *Interp) relata(err error) error {
	e, ok := err.(*Error)
	if !ok || e.secao == "" {
		return err
	}
	partes := []string{e.msg + " na seção " + e.secao}
	if p := e.pos; p != (posicao{}) && int(p.fim) < len(it.S.toks) {
		partes = append(partes, it.parteScript(p)...)
		if e.divisao {
			partes = append(partes, it.parteConta(p, e.operandos))
		}
	}
	var registros []int
	for _, r := range []Row{e.linha, e.outra} {
		if n, ok := it.numeroDoRegistro(r); ok {
			registros = append(registros, n)
		}
	}
	slices.Sort(registros)
	for _, n := range registros {
		partes = append(partes, parteDados(n)...)
	}
	return &Error{msg: strings.Join(partes, "\n")}
}

// parteScript mostra a linha do token de foco, com o marcador sob o que o trecho ocupa nela.
func (it *Interp) parteScript(p posicao) []string {
	toks := it.S.toks
	foco := toks[p.foco]
	ini, fim := p.foco, p.foco
	for ini > p.ini && toks[ini-1].Line == foco.Line {
		ini--
	}
	for fim < p.fim && toks[fim+1].Line == foco.Line {
		fim++
	}
	col := toks[ini].Col
	tam := toks[fim].Col + utf8.RuneCountInString(toks[fim].Src) - col
	partes := []string{fmt.Sprintf("script, linha %d:", foco.Line)}
	if fonte, marcador, ok := trechoScript(foco.Line, col, tam, foco.Col); ok {
		partes = append(partes, fonte, marcador)
	}
	return partes
}

// parteConta mostra a operação como foi escrita, seguida do valor de cada operando que não é um
// número escrito no script. Um nulo ou um texto escrito no script também aparece, para dizer que
// conta como 0.
func (it *Interp) parteConta(p posicao, operandos [2]Value) string {
	conta := "conta: " + it.escrito(p.ini, p.fim)
	var valores []string
	for i, f := range [2][2]int32{{p.ini, p.foco - 1}, {p.foco + 1, p.fim}} {
		if !it.ehNumeroEscrito(f[0], f[1]) {
			valores = append(valores, it.escrito(f[0], f[1])+" = "+descreveOperando(operandos[i]))
		}
	}
	if len(valores) > 0 {
		conta += ", com " + strings.Join(valores, " e ")
	}
	return conta
}

// escrito junta os tokens de ini a fim como estão no script, com um só espaço no lugar de cada
// intervalo ou quebra de linha entre eles.
func (it *Interp) escrito(ini, fim int32) string {
	var sb strings.Builder
	for i := ini; i <= fim; i++ {
		tk := it.S.toks[i]
		if i > ini {
			ant := it.S.toks[i-1]
			if tk.Line != ant.Line || tk.Col != ant.Col+utf8.RuneCountInString(ant.Src) {
				sb.WriteByte(' ')
			}
		}
		sb.WriteString(tk.Src)
	}
	return sb.String()
}

// ehNumeroEscrito diz se os tokens de ini a fim são um número escrito no script, com ou sem sinal.
// Entre parênteses ele conta como uma expressão.
func (it *Interp) ehNumeroEscrito(ini, fim int32) bool {
	toks := it.S.toks
	for ini < fim && toks[ini].Kind == TokOp && (toks[ini].Text == "-" || toks[ini].Text == "+") {
		ini++
	}
	return ini == fim && toks[ini].Kind == TokNum
}

// descreveOperando descreve o valor de um operando, dizendo quanto vale na conta o que não é
// número.
func descreveOperando(v Value) string {
	switch x := v.(type) {
	case nil:
		return "nulo (conta como 0)"
	case string:
		if x == "" {
			return "texto vazio (conta como 0)"
		}
		return "texto " + strconv.Quote(x) + " (conta como 0)"
	case bool:
		if x {
			return "verdadeiro (conta como 1)"
		}
		return "falso (conta como 0)"
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}

// numeroDoRegistro acha o número (a posição na entrada, a partir de 1) do registro r, que é
// reconhecido por ser o mesmo de it.rows, e não só igual a ele.
func (it *Interp) numeroDoRegistro(r Row) (int, bool) {
	if len(r) == 0 {
		return 0, false
	}
	for i, outro := range it.rows {
		if len(outro) > 0 && &outro[0] == &r[0] {
			return it.numeroNaPosicao(i), true
		}
	}
	return 0, false
}

// numeroNaPosicao é o número do registro que está na posição i de it.rows.
func (it *Interp) numeroNaPosicao(i int) int {
	if it.numeros != nil {
		return it.numeros[i]
	}
	return i + 1
}

// parteDados mostra onde começa o registro no texto de dados.
func parteDados(registro int) []string {
	pos := posicaoDados(registro)
	titulo := fmt.Sprintf("dados (registro %d):", registro)
	if pos.Linha > 0 {
		titulo = fmt.Sprintf("dados, linha %d (registro %d):", pos.Linha, registro)
	}
	partes := []string{titulo}
	if linha, _, _, _, ok := trechoDados(pos); ok {
		partes = append(partes, strings.TrimSuffix(linha, "\r"))
	}
	return partes
}
