package report

import (
	"os"
	"strconv"
	"strings"
)

var eventos = map[string]bool{"before": true, "after": true, "header": true, "footer": true, "detail": true}

type parser struct {
	t []Token
	p int
}

func (ps *parser) pk(k int) Token {
	i := ps.p + k
	if i >= len(ps.t) {
		i = len(ps.t) - 1
	}
	return ps.t[i]
}

func (ps *parser) nx() Token {
	tk := ps.t[ps.p]
	if ps.p < len(ps.t)-1 {
		ps.p++
	}
	return tk
}

func (ps *parser) errf(formato string, args ...any) error {
	return ErroLinhaScript(ps.pk(0).Line, formato, args...)
}

func (ps *parser) isKw(nomes ...string) bool {
	tk := ps.pk(0)
	if tk.Kind != TokID {
		return false
	}
	for _, n := range nomes {
		if tk.Text == n {
			return true
		}
	}
	return false
}

func (ps *parser) isKwAt(k int, nomes ...string) bool {
	tk := ps.pk(k)
	if tk.Kind != TokID {
		return false
	}
	for _, n := range nomes {
		if tk.Text == n {
			return true
		}
	}
	return false
}

func (ps *parser) isOp(ops ...string) bool {
	tk := ps.pk(0)
	if tk.Kind != TokOp {
		return false
	}
	for _, o := range ops {
		if tk.Text == o {
			return true
		}
	}
	return false
}

func (ps *parser) kw(nome string) (Token, error) {
	if !ps.isKw(nome) {
		return Token{}, ps.errf("esperado '%s'", nome)
	}
	return ps.nx(), nil
}

func (ps *parser) op(o string) (Token, error) {
	if !ps.isOp(o) {
		return Token{}, ps.errf("esperado '%s'", o)
	}
	return ps.nx(), nil
}

func (ps *parser) numInt() (int, error) {
	tk := ps.nx()
	if tk.Kind != TokNum || !tk.IsInt {
		return 0, ErroLinhaScript(tk.Line, "esperada uma constante inteira")
	}
	return int(tk.Num), nil
}

// Parse interpreta a lista de tokens e devolve o Script completo.
func Parse(toks []Token) (*Script, error) {
	ps := &parser{t: toks}
	s := &Script{
		Width:     80,
		Length:    66,
		Top:       2,
		Bottom:    2,
		Left:      2,
		Separator: "|",
		Events:    map[EventKey][]Stmt{},
		Variaveis: map[string]bool{},
	}
	s.Width = cfgInt("WIDTH", 80)
	s.Length = cfgInt("PAGELN", 66)
	if v, ok := os.LookupEnv("SEPARATOR"); ok {
		s.Separator = v
	}

	terminou := false
	for ps.pk(0).Kind != TokEOF {
		tk := ps.pk(0)
		if tk.Kind != TokID {
			return nil, ps.errf("instrução inválida fora de seção")
		}
		n := tk.Text
		var err error
		switch {
		case n == "input":
			ps.nx()
			err = ps.pInput(s)
		case n == "width":
			ps.nx()
			s.Width, err = ps.numInt()
		case n == "length":
			ps.nx()
			s.Length, err = ps.numInt()
		case n == "top" || n == "bottom" || n == "left":
			ps.nx()
			if _, err = ps.kw("margin"); err == nil {
				var v int
				v, err = ps.numInt()
				if err == nil {
					switch n {
					case "top":
						s.Top = v
					case "bottom":
						s.Bottom = v
					default:
						s.Left = v
					}
				}
			}
		case n == "separator":
			ps.nx()
			s.Separator = ps.nx().Text
		case n == "sort":
			ps.nx()
			err = ps.pSort(s)
		case eventos[n]:
			err = ps.pEvento(s)
		case n == "end":
			ps.nx()
			terminou = true
		default:
			err = ps.errf("instrução inválida fora de seção: '%s'", n)
		}
		if err != nil {
			return nil, err
		}
		if terminou {
			break
		}
	}
	if !terminou {
		return nil, ps.errf("falta o 'end' que termina o script")
	}
	if len(s.Input) == 0 {
		return nil, Erro("o script precisa de uma instrução 'input'")
	}
	coletarVariaveis(s)
	if err := verificaTipos(s); err != nil {
		return nil, err
	}
	return s, nil
}

func (ps *parser) pInput(s *Script) error {
	for {
		nome := ps.nx()
		if nome.Kind != TokID {
			return ps.errf("nome de coluna esperado no input")
		}
		col := InputCol{Name: nome.Text, Type: "string"}
		if ps.isOp("[") {
			ps.nx()
			col.Type = ps.nx().Text
			if ps.pk(0).Kind == TokNum {
				tk := ps.nx()
				col.Len, col.HasLen = int(tk.Num), true
			}
			if _, err := ps.op("]"); err != nil {
				return err
			}
		}
		if ps.isKw("sorted") {
			ps.nx()
		}
		s.Input = append(s.Input, col)
		if ps.isOp(",") {
			ps.nx()
			continue
		}
		break
	}
	return nil
}

func (ps *parser) pSort(s *Script) error {
	for {
		nome := ""
		if ps.isOp("<") {
			ps.nx()
			nome = ps.nx().Text
			if _, err := ps.op(">"); err != nil {
				return err
			}
		}
		e, err := ps.expr()
		if err != nil {
			return err
		}
		if nome == "" {
			if col, ok := e.(ColExpr); ok {
				nome = col.Name
			}
		}
		desc := false
		if ps.isKw("desc") {
			ps.nx()
			desc = true
		}
		s.Sort = append(s.Sort, SortKey{Name: nome, Expr: e, Desc: desc})
		if ps.isOp(",") {
			ps.nx()
			continue
		}
		break
	}
	return nil
}

func (ps *parser) pEvento(s *Script) error {
	ev := ps.nx().Text
	chave := EventKey{Kind: ev}
	if ev == "before" || ev == "after" {
		alvo := ps.nx()
		if alvo.Kind != TokID {
			return ps.errf("nome da seção esperado")
		}
		chave.Target = alvo.Text
	}
	if _, existe := s.Events[chave]; existe {
		return ps.errf("a seção '%s' só pode aparecer uma vez", ev)
	}
	corpo, err := ps.listaStmts()
	if err != nil {
		return err
	}
	s.Events[chave] = corpo
	s.ordemEvent = append(s.ordemEvent, chave)
	return nil
}

func (ps *parser) listaStmts() ([]Stmt, error) {
	var out []Stmt
	for {
		tk := ps.pk(0)
		if tk.Kind == TokEOF {
			break
		}
		if tk.Kind == TokID && (eventos[tk.Text] || tk.Text == "end") {
			break
		}
		st, err := ps.stmt()
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, nil
}

func (ps *parser) stmtOuBloco() ([]Stmt, error) {
	if ps.isKw("begin") {
		ps.nx()
		var out []Stmt
		for !ps.isKw("end") {
			if ps.pk(0).Kind == TokEOF {
				return nil, ps.errf("falta 'end' do bloco 'begin'")
			}
			st, err := ps.stmt()
			if err != nil {
				return nil, err
			}
			out = append(out, st)
		}
		ps.nx()
		return out, nil
	}
	st, err := ps.stmt()
	if err != nil {
		return nil, err
	}
	return []Stmt{st}, nil
}

func (ps *parser) stmt() (Stmt, error) {
	tk := ps.pk(0)
	if tk.Kind != TokID {
		return nil, ps.errf("instrução inválida")
	}
	n := tk.Text
	switch n {
	case "print":
		ps.nx()
		return ps.pPrint()
	case "skip":
		ps.nx()
		q := 1
		if ps.pk(0).Kind == TokNum && ps.pk(0).IsInt {
			q = int(ps.nx().Num)
		}
		return SkipStmt{N: q}, nil
	case "page":
		ps.nx()
		nf, nh := false, false
		for ps.isOp(",") {
			ps.nx()
			if _, err := ps.kw("no"); err != nil {
				return nil, err
			}
			w := ps.nx().Text
			switch w {
			case "footer":
				nf = true
			case "header":
				nh = true
			default:
				return nil, ps.errf("'footer' ou 'header' esperado")
			}
		}
		return PageStmt{NoFooter: nf, NoHeader: nh}, nil
	case "need":
		ps.nx()
		v, err := ps.numInt()
		if err != nil {
			return nil, err
		}
		return NeedStmt{N: v}, nil
	case "set":
		ps.nx()
		nome := ps.nx()
		if nome.Kind != TokID {
			return nil, ps.errf("nome de variável esperado")
		}
		if _, err := ps.kw("to"); err != nil {
			return nil, err
		}
		if ps.isOp("<") { // set x to <nome> expr
			ps.nx()
			ps.nx()
			if _, err := ps.op(">"); err != nil {
				return nil, err
			}
		}
		e, err := ps.expr()
		if err != nil {
			return nil, err
		}
		return SetStmt{Name: nome.Text, Expr: e}, nil
	case "if":
		linhaIf := ps.nx().Line
		c, err := ps.expr()
		if err != nil {
			return nil, err
		}
		if _, err := ps.kw("then"); err != nil {
			return nil, err
		}
		a, err := ps.stmtOuBloco()
		if err != nil {
			return nil, err
		}
		var b []Stmt
		if ps.isKw("else") {
			ps.nx()
			b, err = ps.stmtOuBloco()
			if err != nil {
				return nil, err
			}
		}
		return IfStmt{Line: linhaIf, Cond: c, Then: a, Else: b}, nil
	case "begin":
		b, err := ps.stmtOuBloco()
		if err != nil {
			return nil, err
		}
		return BlockStmt{Body: b}, nil
	}
	return nil, ps.errf("instrução inválida: '%s'", n)
}

func (ps *parser) pPrint() (Stmt, error) {
	var itens []PrintItem
	for {
		e, err := ps.expr()
		if err != nil {
			return nil, err
		}
		item := PrintItem{Expr: e}
		for {
			switch {
			case ps.isKw("in") && ps.isKwAt(1, "col", "column"):
				ps.nx()
				ps.nx()
				item.Col, err = ps.expr()
			case ps.isKw("col", "column"):
				ps.nx()
				item.Col, err = ps.expr()
			case ps.isKw("using"):
				ps.nx()
				item.Using, err = ps.expr()
			case ps.isKw("centered"):
				ps.nx()
				item.Centered = true
			default:
				goto fimOpcoes
			}
			if err != nil {
				return nil, err
			}
		}
	fimOpcoes:
		itens = append(itens, item)
		if ps.isOp(",") {
			ps.nx()
			continue
		}
		break
	}
	nonl := false
	if ps.isKw("no") && ps.isKwAt(1, "newline") {
		ps.nx()
		ps.nx()
		nonl = true
	}
	return PrintStmt{Items: itens, NoNewline: nonl}, nil
}

// --- expressões (precedência: or > and > not > comparação > + - > * / > unário > primário)

func (ps *parser) expr() (Expr, error) {
	a, err := ps.eAnd()
	if err != nil {
		return nil, err
	}
	for ps.isKw("or") {
		ps.nx()
		b, err := ps.eAnd()
		if err != nil {
			return nil, err
		}
		a = OrExpr{L: a, R: b}
	}
	return a, nil
}

func (ps *parser) eAnd() (Expr, error) {
	a, err := ps.eNot()
	if err != nil {
		return nil, err
	}
	for ps.isKw("and") {
		ps.nx()
		b, err := ps.eNot()
		if err != nil {
			return nil, err
		}
		a = AndExpr{L: a, R: b}
	}
	return a, nil
}

func (ps *parser) eNot() (Expr, error) {
	if ps.isKw("not") {
		ps.nx()
		x, err := ps.eNot()
		if err != nil {
			return nil, err
		}
		return NotExpr{X: x}, nil
	}
	return ps.eCmp()
}

var opsCmp = map[string]bool{"=": true, "<>": true, "^=": true, "!=": true, "<": true, ">": true, "<=": true, ">=": true}

func (ps *parser) eCmp() (Expr, error) {
	a, err := ps.eAdd()
	if err != nil {
		return nil, err
	}
	for {
		tk := ps.pk(0)
		if tk.Kind == TokOp && opsCmp[tk.Text] {
			ps.nx()
			b, err := ps.eAdd()
			if err != nil {
				return nil, err
			}
			op := tk.Text
			if op == "^=" || op == "!=" {
				op = "<>"
			}
			a = CmpExpr{Op: op, L: a, R: b}
		} else if ps.isKw("is") {
			ps.nx()
			neg := false
			if ps.isKw("not") {
				ps.nx()
				neg = true
			}
			if _, err := ps.kw("null"); err != nil {
				return nil, err
			}
			a = IsNullExpr{X: a, Neg: neg}
		} else {
			return a, nil
		}
	}
}

func (ps *parser) eAdd() (Expr, error) {
	a, err := ps.eMul()
	if err != nil {
		return nil, err
	}
	for ps.isOp("+", "-", "/+") {
		op := ps.nx().Text
		b, err := ps.eMul()
		if err != nil {
			return nil, err
		}
		a = BinExpr{Op: op, L: a, R: b}
	}
	return a, nil
}

func (ps *parser) eMul() (Expr, error) {
	a, err := ps.eUn()
	if err != nil {
		return nil, err
	}
	for ps.isOp("*", "/", "%") {
		op := ps.nx().Text
		b, err := ps.eUn()
		if err != nil {
			return nil, err
		}
		a = BinExpr{Op: op, L: a, R: b}
	}
	return a, nil
}

func (ps *parser) eUn() (Expr, error) {
	if ps.isOp("-") {
		ps.nx()
		x, err := ps.eUn()
		if err != nil {
			return nil, err
		}
		return NegExpr{X: x}, nil
	}
	if ps.isOp("+") {
		ps.nx()
	}
	e, err := ps.ePrim()
	if err != nil {
		return nil, err
	}
	// operador de substring (posfixo): texto substr [início-fim]
	for ps.isKw("substr") {
		linha := ps.nx().Line
		sub, err := ps.substrEspec(linha)
		if err != nil {
			return nil, err
		}
		sub.X = e
		e = sub
	}
	return e, nil
}

// substrEspec lê "[início-fim]" (inteiros, início >= 1 e fim >= início, como no manual).
func (ps *parser) substrEspec(linha int) (SubstrExpr, error) {
	var sub SubstrExpr
	if _, err := ps.op("["); err != nil {
		return sub, err
	}
	ini := ps.nx()
	if ini.Kind != TokNum || !ini.IsInt {
		return sub, ErroLinhaScript(linha, "substr espera [início-fim] com números inteiros")
	}
	if _, err := ps.op("-"); err != nil {
		return sub, err
	}
	fim := ps.nx()
	if fim.Kind != TokNum || !fim.IsInt {
		return sub, ErroLinhaScript(linha, "substr espera [início-fim] com números inteiros")
	}
	if _, err := ps.op("]"); err != nil {
		return sub, err
	}
	sub.Start, sub.End = int(ini.Num), int(fim.Num)
	if sub.Start < 1 || sub.End < sub.Start {
		return sub, ErroLinhaScript(linha, "substr [%d-%d] inválido: o início começa em 1 e o fim não pode ser menor que o início", sub.Start, sub.End)
	}
	return sub, nil
}

func (ps *parser) ePrim() (Expr, error) {
	tk := ps.nx()
	switch tk.Kind {
	case TokNum:
		if tk.IsInt {
			return ConstExpr{Val: int64(tk.Num)}, nil
		}
		return ConstExpr{Val: tk.Num}, nil
	case TokStr:
		return ConstExpr{Val: tk.Text}, nil
	case TokRep:
		return ConstExpr{Val: strings.Repeat(tk.RepChar, tk.RepN)}, nil
	case TokData:
		d, ok, err := ParseData(tk.Text)
		if err != nil {
			return nil, err
		}
		if !ok {
			return ConstExpr{Val: nil}, nil
		}
		return ConstExpr{Val: d}, nil
	case TokOp:
		if tk.Text == "(" {
			e, err := ps.expr()
			if err != nil {
				return nil, err
			}
			if _, err := ps.op(")"); err != nil {
				return nil, err
			}
			return e, nil
		}
	case TokID:
		v := tk.Text
		if v == "null" {
			return ConstExpr{Val: nil}, nil
		}
		if ps.isOp("(") {
			ps.nx()
			var args []Expr
			star := false
			var where Expr
			if ps.isOp("*") {
				ps.nx()
				star = true
				if ps.isKw("where") {
					ps.nx()
					w, err := ps.expr()
					if err != nil {
						return nil, err
					}
					where = w
				}
			} else if !ps.isOp(")") {
				e, err := ps.expr()
				if err != nil {
					return nil, err
				}
				args = append(args, e)
				if ps.isKw("where") {
					ps.nx()
					w, err := ps.expr()
					if err != nil {
						return nil, err
					}
					where = w
				}
				for ps.isOp(",") {
					ps.nx()
					e, err := ps.expr()
					if err != nil {
						return nil, err
					}
					args = append(args, e)
				}
			}
			if _, err := ps.op(")"); err != nil {
				return nil, err
			}
			return CallExpr{Name: v, Args: args, Star: star, Where: where}, nil
		}
		switch v {
		case "today":
			return TodayExpr{}, nil
		case "hour":
			return HourExpr{}, nil
		case "pageno":
			return PageNoExpr{}, nil
		}
		return ColExpr{Name: v}, nil
	}
	return nil, ErroLinhaScript(tk.Line, "expressão inválida")
}

// coletarVariaveis percorre todas as seções marcando quais nomes recebem 'set' em algum
// ponto (precisam ter sido inicializados antes do primeiro uso, e não podem coincidir com
// nome de coluna do input).
func coletarVariaveis(s *Script) {
	var anda func([]Stmt)
	anda = func(lst []Stmt) {
		for _, st := range lst {
			switch x := st.(type) {
			case SetStmt:
				s.Variaveis[x.Name] = true
			case IfStmt:
				anda(x.Then)
				anda(x.Else)
			case BlockStmt:
				anda(x.Body)
			}
		}
	}
	for _, corpo := range s.Events {
		anda(corpo)
	}
}

func cfgInt(nome string, padrao int) int {
	if v, ok := os.LookupEnv(nome); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return padrao
}
