package report

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Row é um registro de entrada já convertido: os valores na mesma ordem das colunas do
// 'input' (nil representa nulo). O nome da coluna é resolvido por Interp.colIdx, um único
// mapa nome->posição compartilhado por todos os registros da execução.
type Row = []Value

var tiposInt = map[string]bool{"numeric": true, "integer": true, "long": true, "int": true}
var tiposFlt = map[string]bool{"float": true, "amount": true, "double": true, "real": true, "huge": true}

type extraItem struct {
	Col   int
	Texto string
}

// Interp executa um Script já interpretado sobre um conjunto de registros, produzindo o
// texto final do relatório, paginado.
type Interp struct {
	S        *Script
	FormFeed bool
	preenche bool

	semMargens   bool // sem margem superior/esquerda (usado por Paginas)
	linhasPagina int  // linhas de texto por página; 0 = length - margens do script

	vars    map[string]Value
	profNome int // profundidade de avaliação de expressões nomeadas (detecta recursão)
	colIdx  map[string]int
	row     Row
	pageno  int
	aggRows []Row
	aggAtivo bool

	paginas [][]string
	corpo   []string
	buf     []rune
	extras  []extraItem
	fimCol  int

	captura   *[]string
	quebrando bool
	altRodape int

	// Strict: com 'before report' no script, a primeira página não roda o 'header'.
	// Por padrão o 'header' roda sempre, logo depois do 'before report'.
	Strict bool

	agora time.Time
	rows  []Row
}

// NovoInterp converte os registros de entrada e devolve um Interp pronto pra Run().
func NovoInterp(s *Script, linhas [][]string, formFeed bool) (*Interp, error) {
	it := &Interp{
		S:        s,
		FormFeed: formFeed,
		preenche: !formFeed,
		vars:     map[string]Value{},
		colIdx:   make(map[string]int, len(s.Input)),
		fimCol:   1,
		agora:    agora(),
	}
	for j, c := range s.Input {
		it.colIdx[c.Name] = j
	}
	rows, err := it.converte(linhas)
	if err != nil {
		return nil, err
	}
	it.rows = rows
	if err := it.verificaEventos(); err != nil {
		return nil, err
	}
	return it, nil
}

// ---------------------------------------------------------------- entrada

func (it *Interp) converte(linhas [][]string) ([]Row, error) {
	cols := it.S.Input
	out := make([]Row, 0, len(linhas))
	for i, campos := range linhas {
		ln := i + 1
		if len(campos) != len(cols) {
			return nil, ErroLinhaDados(ln, "phase error no arquivo de entrada: registro %d tem %d campos, esperados %d",
				ln, len(campos), len(cols))
		}
		r := make(Row, len(cols))
		for j, col := range cols {
			v, err := it.tipa(campos[j], col, ln)
			if err != nil {
				return nil, err
			}
			r[j] = v
		}
		out = append(out, r)
	}
	return out, nil
}

func (it *Interp) tipa(txt string, col InputCol, ln int) (Value, error) {
	var t string
	if col.Type == "string" {
		t = strings.TrimRight(txt, " \t\r\n\v\f")
	} else {
		t = strings.TrimSpace(txt)
	}
	switch {
	case tiposInt[col.Type]:
		if t == "" {
			return nil, nil
		}
		f, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return nil, ErroLinhaDados(ln, "valor inválido em '%s' (registro %d): %q (%v)", col.Name, ln, txt, err)
		}
		return int64(math.Trunc(f)), nil
	case tiposFlt[col.Type]:
		if t == "" {
			return nil, nil
		}
		f, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return nil, ErroLinhaDados(ln, "valor inválido em '%s' (registro %d): %q (%v)", col.Name, ln, txt, err)
		}
		return f, nil
	case col.Type == "bool":
		if t == "" {
			return nil, nil
		}
		return normalizaBool(t), nil
	case col.Type == "time":
		h, ok, err := ParseHora(t)
		if err != nil {
			return nil, ErroLinhaDados(ln, "valor inválido em '%s' (registro %d): %q (%v)", col.Name, ln, txt, err)
		}
		if !ok {
			return nil, nil
		}
		return h, nil
	case col.Type == "date" || col.Type == "ldate":
		d, ok, err := ParseData(t)
		if err != nil {
			return nil, ErroLinhaDados(ln, "valor inválido em '%s' (registro %d): %q (%v)", col.Name, ln, txt, err)
		}
		if !ok {
			return nil, nil
		}
		return d, nil
	}
	if col.Type == "string" && col.HasLen {
		r := []rune(strings.TrimRight(txt, " \t\r\n\v\f"))
		if len(r) > col.Len {
			r = r[:col.Len]
		}
		return string(r), nil
	}
	return strings.TrimRight(txt, " \t\r\n\v\f"), nil
}

func (it *Interp) verificaEventos() error {
	nomes := map[string]bool{}
	for _, k := range it.S.Sort {
		if k.Name != "" {
			nomes[k.Name] = true
		}
	}
	for k := range it.S.Events {
		if (k.Kind == "before" || k.Kind == "after") && k.Target != "report" && !nomes[k.Target] {
			return Erro("'%s' na seção %s não é uma expressão de sort existente", k.Target, k.Kind)
		}
	}
	for _, c := range it.S.Input {
		if it.S.Variaveis[c.Name] {
			return Erro("mesmo nome para variável e coluna: %s", c.Name)
		}
	}
	return nil
}

// ---------------------------------------------------------------- avaliação

func (it *Interp) ev(e Expr) (Value, error) {
	switch x := e.(type) {
	case ConstExpr:
		return x.Val, nil
	case ColExpr:
		if v, ok := it.vars[x.Name]; ok {
			return v, nil
		}
		if it.S.Variaveis[x.Name] {
			return nil, nil
		}
		if j, ok := it.colIdx[x.Name]; ok && j < len(it.row) {
			return it.row[j], nil
		}
		if e, ok := it.S.Nomeadas[x.Name]; ok {
			// expressão com apelido: usada como se fosse mais uma coluna
			if it.profNome >= 32 {
				return nil, Erro("a expressão nomeada '%s' refere-se a si mesma", x.Name)
			}
			it.profNome++
			defer func() { it.profNome-- }()
			return it.ev(e)
		}
		return nil, Erro("nome desconhecido: '%s'", x.Name)
	case TodayExpr:
		ano, mes, dia := it.agora.Date()
		return time.Date(ano, mes, dia, 0, 0, 0, 0, it.agora.Location()), nil
	case HourExpr:
		return horaDoDia(it.agora), nil
	case PageNoExpr:
		return int64(it.pageno), nil
	case NegExpr:
		v, err := it.ev(x.X)
		if err != nil {
			return nil, err
		}
		if !verdade(v) { // nil/0/0.0/""/false negam pra 0 (sem sinal)
			return int64(0), nil
		}
		if ehInteiro(v) {
			return -v.(int64), nil
		}
		return -paraFloat(v), nil
	case NotExpr:
		v, err := it.ev(x.X)
		if err != nil {
			return nil, err
		}
		if verdade(v) {
			return int64(0), nil
		}
		return int64(1), nil
	case AndExpr:
		a, err := it.ev(x.L)
		if err != nil {
			return nil, err
		}
		if !verdade(a) {
			return int64(0), nil
		}
		b, err := it.ev(x.R)
		if err != nil {
			return nil, err
		}
		if verdade(b) {
			return int64(1), nil
		}
		return int64(0), nil
	case OrExpr:
		a, err := it.ev(x.L)
		if err != nil {
			return nil, err
		}
		if verdade(a) {
			return int64(1), nil
		}
		b, err := it.ev(x.R)
		if err != nil {
			return nil, err
		}
		if verdade(b) {
			return int64(1), nil
		}
		return int64(0), nil
	case IsNullExpr:
		v, err := it.ev(x.X)
		if err != nil {
			return nil, err
		}
		nulo := v == nil || v == ""
		if nulo != x.Neg {
			return int64(1), nil
		}
		return int64(0), nil
	case CmpExpr:
		a, err := it.ev(x.L)
		if err != nil {
			return nil, err
		}
		b, err := it.ev(x.R)
		if err != nil {
			return nil, err
		}
		r, err := comparar(x.Op, a, b)
		if err != nil {
			return nil, err
		}
		if r {
			return int64(1), nil
		}
		return int64(0), nil
	case BinExpr:
		a, err := it.ev(x.L)
		if err != nil {
			return nil, err
		}
		b, err := it.ev(x.R)
		if err != nil {
			return nil, err
		}
		return aritm(x.Op, a, b)
	case SubstrExpr:
		v, err := it.ev(x.X)
		if err != nil {
			return nil, err
		}
		if v == nil {
			return nil, nil
		}
		s, ok := v.(string)
		if !ok {
			return nil, Erro("substr só pode ser usado com texto")
		}
		r := []rune(s)
		if x.Start > len(r) {
			return "", nil
		}
		fim := x.End
		if fim > len(r) {
			fim = len(r)
		}
		return string(r[x.Start-1 : fim]), nil
	case CallExpr:
		return it.chama(x)
	}
	return nil, Erro("expressão desconhecida")
}

func (it *Interp) chama(e CallExpr) (Value, error) {
	switch e.Name {
	case "count", "total", "avg", "min", "max":
		if !it.aggAtivo {
			return nil, Erro("a função %s() só pode ser usada em 'after sort_key' e 'after report'", e.Name)
		}
		salvo := it.row
		var vals []Value
		for _, r := range it.aggRows {
			it.row = r
			if e.Where != nil {
				ok, err := it.ev(e.Where)
				if err != nil {
					it.row = salvo
					return nil, err
				}
				if !verdade(ok) {
					continue
				}
			}
			if e.Star {
				vals = append(vals, int64(1))
				continue
			}
			v, err := it.ev(e.Args[0])
			if err != nil {
				it.row = salvo
				return nil, err
			}
			vals = append(vals, v)
		}
		it.row = salvo
		if e.Name == "count" {
			return int64(len(vals)), nil
		}
		naoNulos := vals[:0:0]
		for _, v := range vals {
			if v != nil {
				naoNulos = append(naoNulos, v)
			}
		}
		switch e.Name {
		case "total":
			if len(naoNulos) == 0 {
				return int64(0), nil
			}
			soma := 0.0
			todosInt := true
			for _, v := range naoNulos {
				soma += paraFloat(v)
				if !ehInteiro(v) {
					todosInt = false
				}
			}
			if todosInt {
				return int64(soma), nil
			}
			return soma, nil
		case "avg":
			if len(naoNulos) == 0 {
				return int64(0), nil
			}
			soma := 0.0
			for _, v := range naoNulos {
				soma += paraFloat(v)
			}
			return soma / float64(len(naoNulos)), nil
		case "min", "max":
			if len(naoNulos) == 0 {
				return nil, nil
			}
			melhor := naoNulos[0]
			for _, v := range naoNulos[1:] {
				menor, _ := comparar("<", v, melhor)
				maior, _ := comparar(">", v, melhor)
				if (e.Name == "min" && menor) || (e.Name == "max" && maior) {
					melhor = v
				}
			}
			return melhor, nil
		}
	}
	args := make([]Value, len(e.Args))
	for i, a := range e.Args {
		v, err := it.ev(a)
		if err != nil {
			return nil, err
		}
		args[i] = v
	}
	switch e.Name {
	case "dow":
		if args[0] == nil {
			return nil, nil
		}
		d := args[0].(time.Time)
		return int64(d.Weekday()), nil // time.Weekday: 0=domingo..6=sábado, igual a isoweekday()%7
	case "mdy":
		if args[0] == nil {
			return nil, nil
		}
		d := args[0].(time.Time)
		switch int(paraFloat(args[1])) {
		case 0:
			return int64(d.Month()), nil
		case 1:
			return int64(d.Day()), nil
		case 2:
			return int64(d.Year()), nil
		}
		return nil, nil
	case "index":
		i := int(paraFloat(args[0]))
		if i >= 0 && i < len(args)-1 {
			return args[1+i], nil
		}
		return "", nil
	}
	return nil, Erro("'%s' é um nome de função desconhecido", e.Name)
}

// ---------------------------------------------------------------- saída / páginas

func (it *Interp) cap() int {
	if it.linhasPagina > 0 {
		return it.linhasPagina - it.altRodape
	}
	return it.S.Length - it.S.Top - it.S.Bottom - it.altRodape
}

func (it *Interp) emite(texto string) {
	if it.captura != nil {
		*it.captura = append(*it.captura, texto)
		return
	}
	if len(it.corpo) >= it.cap() && !it.quebrando {
		it.novaPagina(false, false)
	}
	it.corpo = append(it.corpo, texto)
}

func (it *Interp) flush() {
	if len(it.buf) == 0 && len(it.extras) == 0 {
		return
	}
	buf, extras := it.buf, it.extras
	it.buf, it.extras, it.fimCol = nil, nil, 1
	esq := strings.Repeat(" ", it.S.Left)
	if it.semMargens {
		esq = ""
	}
	limite := it.S.Width - it.S.Left
	if len(buf) > 0 {
		txt := strings.TrimRight(string(buf), " ")
		it.emite(esq + recortaRunas(txt, limite))
	}
	for _, ex := range extras {
		pad := strings.Repeat(" ", ex.Col-1) + ex.Texto
		it.emite(esq + recortaRunas(pad, limite))
	}
}

func recortaRunas(s string, limite int) string {
	if limite < 0 {
		return ""
	}
	r := []rune(s)
	if len(r) > limite {
		r = r[:limite]
	}
	return string(r)
}

func (it *Interp) fechaPagina(semRodape bool) {
	it.flush()
	if _, temRodape := it.S.Events[EventKey{Kind: "footer"}]; temRodape && !semRodape {
		var cap []string
		it.captura = &cap
		it.exec(it.S.Events[EventKey{Kind: "footer"}])
		it.flush()
		it.captura = nil
		falta := it.cap() - len(it.corpo)
		for i := 0; i < falta; i++ {
			it.corpo = append(it.corpo, "")
		}
		it.corpo = append(it.corpo, cap...)
	}
	var pag []string
	if !it.semMargens {
		pag = make([]string, it.S.Top)
	}
	pag = append(pag, it.corpo...)
	if it.preenche {
		for len(pag) < it.S.Length {
			pag = append(pag, "")
		}
	}
	it.paginas = append(it.paginas, pag)
	it.corpo = nil
}

func (it *Interp) novaPagina(semRodape, semCabecalho bool) {
	it.flush()
	if len(it.corpo) == 0 {
		return
	}
	it.quebrando = true
	it.fechaPagina(semRodape)
	it.pageno++
	if corpo, ok := it.S.Events[EventKey{Kind: "header"}]; ok && !semCabecalho {
		it.exec(corpo)
		it.flush()
	}
	it.quebrando = false
}

// ---------------------------------------------------------------- comandos

func (it *Interp) exec(stmts []Stmt) error {
	for _, st := range stmts {
		switch x := st.(type) {
		case PrintStmt:
			if err := it.printStmt(x); err != nil {
				return err
			}
		case SkipStmt:
			it.flush()
			for i := 0; i < x.N; i++ {
				it.emite("")
			}
		case PageStmt:
			it.novaPagina(x.NoFooter, x.NoHeader)
		case NeedStmt:
			it.flush()
			if it.cap()-len(it.corpo) < x.N {
				it.novaPagina(false, false)
			}
		case SetStmt:
			v, err := it.ev(x.Expr)
			if err != nil {
				return err
			}
			it.vars[x.Name] = v
		case IfStmt:
			v, err := it.ev(x.Cond)
			if err != nil {
				return err
			}
			if verdade(v) {
				if err := it.exec(x.Then); err != nil {
					return err
				}
			} else if err := it.exec(x.Else); err != nil {
				return err
			}
		case BlockStmt:
			if err := it.exec(x.Body); err != nil {
				return err
			}
		}
	}
	return nil
}

// ehColBool diz se a expressão é uma referência direta a uma coluna do input declarada
// como bool (e não a uma variável de 'set' com o mesmo nome).
func (it *Interp) ehColBool(e Expr) bool {
	c, ok := e.(ColExpr)
	if !ok || it.S.Variaveis[c.Name] {
		return false
	}
	return tipoColuna(it.S, c.Name) == "bool"
}

func (it *Interp) ehLdate(e Expr) bool {
	col, ok := e.(ColExpr)
	if !ok {
		return false
	}
	for _, c := range it.S.Input {
		if c.Name == col.Name && c.Type == "ldate" {
			return true
		}
	}
	return false
}

// larguraCol devolve a largura declarada no 'input' pra uma referência direta a coluna
// STRING (usada pro preenchimento padrão com espaços quando não há 'using': uma string sem
// 'using' é alinhada até a largura do campo, não só até o tamanho do valor).
func (it *Interp) larguraCol(e Expr) (int, bool) {
	col, ok := e.(ColExpr)
	if !ok {
		return 0, false
	}
	for _, c := range it.S.Input {
		if c.Name == col.Name && c.Type == "string" && c.HasLen {
			return c.Len, true
		}
	}
	return 0, false
}

func (it *Interp) printStmt(st PrintStmt) error {
	for _, item := range st.Items {
		v, err := it.ev(item.Expr)
		if err != nil {
			return err
		}
		var mascara Value
		if item.Using != nil {
			mascara, err = it.ev(item.Using)
			if err != nil {
				return err
			}
		}
		linhas, err := it.formata(v, mascara, it.ehLdate(item.Expr))
		if err != nil {
			return err
		}
		if v == nil && it.ehColBool(item.Expr) {
			linhas = []string{"***"}
		}
		if item.Using == nil {
			if s, ok := v.(string); ok {
				if larg, ok := it.larguraCol(item.Expr); ok && len([]rune(linhas[0])) < larg {
					linhas[0] = s + strings.Repeat(" ", larg-len([]rune(s)))
				}
			}
		}
		texto := linhas[0]
		col := 1
		if len(it.buf) > 0 {
			col = it.fimCol + 1 // sem 'col' explícita: pelo menos 1 espaço do item anterior
		}
		if item.Col != nil {
			cv, err := it.ev(item.Col)
			if err != nil {
				return err
			}
			col = int(paraFloat(cv))
		}
		if item.Centered {
			espaco := it.S.Width - it.S.Left - len([]rune(texto))
			col = int(math.Ceil(float64(espaco)/2.0)) + 1
			if col < 1 {
				col = 1
			}
		}
		if col < 1 {
			return Erro("número de coluna inválido: %d", col)
		}
		ini := col - 1
		tr := []rune(texto)
		if len(it.buf) < ini+len(tr) {
			it.buf = append(it.buf, make([]rune, ini+len(tr)-len(it.buf))...)
			for i := range it.buf {
				if it.buf[i] == 0 {
					it.buf[i] = ' '
				}
			}
		}
		copy(it.buf[ini:ini+len(tr)], tr)
		it.fimCol = col + len(tr)
		for _, extra := range linhas[1:] {
			it.extras = append(it.extras, extraItem{Col: col, Texto: extra})
		}
	}
	if !st.NoNewline {
		it.flush()
	}
	return nil
}

func (it *Interp) formata(v, m Value, longa bool) ([]string, error) {
	if v == nil {
		return []string{""}, nil
	}
	if h, ok := v.(Hora); ok {
		return []string{FmtHora(h)}, nil
	}
	if d, ok := v.(time.Time); ok {
		return []string{FmtData(d, true, longa)}, nil
	}
	if s, ok := v.(string); ok {
		if m != nil {
			ms, _ := m.(string)
			if larg, ok := ehMascaraString(ms); ok {
				return QuebraTexto(s, larg), nil
			}
			// uma máscara numérica aplicada a um campo de texto não dá erro: imprime em branco.
			return []string{""}, nil
		}
		return []string{s}, nil
	}
	if b, ok := v.(bool); ok {
		if m == nil {
			if b {
				return []string{"YES"}, nil
			}
			return []string{"NO"}, nil
		}
		v = boolParaInt(b)
	}
	if m != nil {
		ms, _ := m.(string)
		s, err := FmtMascara(v, ms)
		if err != nil {
			return nil, err
		}
		return []string{s}, nil
	}
	if ehInteiro(v) {
		return []string{strconv.FormatInt(v.(int64), 10)}, nil
	}
	return []string{strconv.FormatFloat(paraFloat(v), 'f', 2, 64)}, nil
}

// ---------------------------------------------------------------- ordenação e execução

func (it *Interp) chave(r Row) []Value {
	salvo := it.row
	it.row = r
	ks := make([]Value, len(it.S.Sort))
	for i, sk := range it.S.Sort {
		v, _ := it.ev(sk.Expr) // erro de expressão de sort já teria sido pego antes (coluna existe)
		ks[i] = v
	}
	it.row = salvo
	return ks
}

// LinhasOrdenadas devolve os registros na ordem da cláusula 'sort', sem executar nenhuma
// seção de evento (before/after/header/footer/detail), sem tocar em variáveis nem em
// paginação. É a base das saídas CSV/JSON, que só consultam 'input' e 'sort'.
func (it *Interp) LinhasOrdenadas() []Row {
	it.ordena()
	return it.rows
}

func (it *Interp) ordena() {
	if len(it.S.Sort) == 0 {
		return
	}
	type par struct {
		k []Value
		r Row
	}
	pares := make([]par, len(it.rows))
	for i, r := range it.rows {
		pares[i] = par{k: it.chave(r), r: r}
	}
	sort.SliceStable(pares, func(i, j int) bool {
		for n, sk := range it.S.Sort {
			a, b := pares[i].k[n], pares[j].k[n]
			na, nb := a == nil, b == nil
			if na && nb {
				continue
			}
			var c int
			switch {
			case na:
				c = -1
			case nb:
				c = 1
			default:
				menor, _ := comparar("<", a, b)
				maior, _ := comparar(">", a, b)
				switch {
				case menor:
					c = -1
				case maior:
					c = 1
				default:
					c = 0
				}
			}
			if c != 0 {
				if sk.Desc {
					return c > 0
				}
				return c < 0
			}
		}
		return false
	})
	for i, p := range pares {
		it.rows[i] = p.r
	}
}

// Run executa o relatório completo e devolve o texto final, já paginado.
func (it *Interp) executa() error {
	it.ordena()

	// altura do rodapé: medida com uma execução "seca" antes de começar de verdade
	if corpo, ok := it.S.Events[EventKey{Kind: "footer"}]; ok && len(it.rows) > 0 {
		varsSalvas := map[string]Value{}
		for k, v := range it.vars {
			varsSalvas[k] = v
		}
		it.row, it.pageno = it.rows[0], 1
		var cap []string
		it.captura = &cap
		if err := it.exec(corpo); err != nil {
			return err
		}
		it.flush()
		it.altRodape = len(cap)
		it.captura, it.vars, it.pageno = nil, varsSalvas, 0
	}
	it.pageno = 1
	if len(it.rows) == 0 {
		return Erro("não há registros de entrada")
	}

	niveis := make([]string, len(it.S.Sort))
	for i, sk := range it.S.Sort {
		niveis[i] = sk.Name
	}
	nn := len(niveis)

	sec := func(tipo, nome string) ([]Stmt, bool) {
		corpo, ok := it.S.Events[EventKey{Kind: tipo, Target: nome}]
		return corpo, ok
	}
	roda := func(corpo []Stmt, ok bool, linha Row, grupo []Row) error {
		if !ok {
			return nil
		}
		it.row = linha
		it.aggRows, it.aggAtivo = grupo, true
		err := it.exec(corpo)
		it.aggAtivo = false
		return err
	}

	if _, temBeforeReport := sec("before", "report"); !temBeforeReport {
		if corpo, ok := sec("header", ""); ok {
			it.row = it.rows[0]
			if err := it.exec(corpo); err != nil {
				return err
			}
			it.flush()
		}
	}

	var prev Row
	var prevK []Value
	ini := make([]int, nn)
	if nn == 0 {
		ini = []int{0}
	}
	for i, r := range it.rows {
		k := it.chave(r)
		quebra := 0
		if prev == nil {
			if corpo, ok := sec("before", "report"); ok {
				if err := roda(corpo, ok, r, nil); err != nil {
					return err
				}
				if cab, temCab := sec("header", ""); temCab && !it.Strict {
					it.row = r
					if err := it.exec(cab); err != nil {
						return err
					}
					it.flush()
				}
			}
		} else {
			quebra = nn
			for j := 0; j < nn; j++ {
				if k[j] != prevK[j] {
					quebra = j
					break
				}
			}
			for j := nn - 1; j >= quebra; j-- {
				if niveis[j] == "" {
					continue
				}
				corpo, ok := sec("after", niveis[j])
				if err := roda(corpo, ok, prev, it.rows[ini[j]:i]); err != nil {
					return err
				}
			}
		}
		for j := quebra; j < nn; j++ {
			ini[j] = i
			if niveis[j] != "" {
				corpo, ok := sec("before", niveis[j])
				if err := roda(corpo, ok, r, nil); err != nil {
					return err
				}
			}
		}
		if corpo, ok := sec("detail", ""); ok {
			if err := roda(corpo, ok, r, nil); err != nil {
				return err
			}
		}
		prev, prevK = r, k
	}
	for j := nn - 1; j >= 0; j-- {
		if niveis[j] == "" {
			continue
		}
		corpo, ok := sec("after", niveis[j])
		if err := roda(corpo, ok, prev, it.rows[ini[j]:]); err != nil {
			return err
		}
	}
	if corpo, ok := sec("after", "report"); ok {
		if err := roda(corpo, ok, prev, it.rows); err != nil {
			return err
		}
	}
	it.flush()
	it.fechaPagina(false)

	return nil
}

// Run executa o relatório e devolve o texto final, paginado.
func (it *Interp) Run() (string, error) {
	if err := it.executa(); err != nil {
		return "", err
	}
	sep := ""
	if it.FormFeed {
		sep = "\f"
	}
	var sb strings.Builder
	for i, p := range it.paginas {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(strings.Join(p, "\n"))
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

// Paginas executa o relatório e devolve cada página como lista de linhas, sem completar com
// linhas em branco, sem as linhas da margem superior e sem a margem esquerda (quem apresenta
// as páginas, como um modelo HTML, cuida das margens). linhasPorPagina é o número de linhas de
// texto por página, incluindo cabeçalho e rodapé; 0 usa length - top margin - bottom margin
// do script.
func (it *Interp) Paginas(linhasPorPagina int) ([][]string, error) {
	it.preenche = false
	it.semMargens = true
	it.linhasPagina = linhasPorPagina
	if err := it.executa(); err != nil {
		return nil, err
	}
	return it.paginas, nil
}
