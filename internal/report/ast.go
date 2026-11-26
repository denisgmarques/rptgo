package report

// Expr é um nó de expressão. Cada tipo concreto implementa a interface só pra marcação;
// a avaliação de fato (em interp.go) usa um type switch sobre o tipo concreto.
type Expr interface{ exprNode() }

// posicao é o trecho do script de um nó, em índices de Script.toks: ini é o primeiro token, foco
// o que identifica a construção (o operador, o nome ou a palavra 'substr') e fim o último. O
// valor zero quer dizer posição desconhecida (o token 0 nunca faz parte de uma expressão).
type posicao struct{ ini, foco, fim int32 }

type ConstExpr struct{ Val Value }
type ColExpr struct {
	Name string
	pos  posicao
}
type TodayExpr struct{}
type HourExpr struct{}
type PageNoExpr struct{}
type NegExpr struct{ X Expr }
type NotExpr struct{ X Expr }
type AndExpr struct{ L, R Expr }
type OrExpr struct{ L, R Expr }
type IsNullExpr struct {
	X   Expr
	Neg bool // "is not null"
}
type CmpExpr struct {
	Op   string // = <> < > <= >=
	L, R Expr
	pos  posicao
}
type BinExpr struct {
	Op   string // + - * /
	L, R Expr
	pos  posicao
}

// SubstrExpr é "texto substr [início-fim]": posições de 1 em diante, fim inclusivo.
type SubstrExpr struct {
	X          Expr
	Start, End int
	pos        posicao
}
type CallExpr struct {
	Name  string
	Args  []Expr
	Star  bool // count(*...)
	Where Expr // count(* where expr)
	pos   posicao
}

func (ConstExpr) exprNode()  {}
func (ColExpr) exprNode()    {}
func (TodayExpr) exprNode()  {}
func (HourExpr) exprNode()   {}
func (PageNoExpr) exprNode() {}
func (NegExpr) exprNode()    {}
func (NotExpr) exprNode()    {}
func (AndExpr) exprNode()    {}
func (OrExpr) exprNode()     {}
func (IsNullExpr) exprNode() {}
func (CmpExpr) exprNode()    {}
func (BinExpr) exprNode()    {}
func (CallExpr) exprNode()   {}
func (SubstrExpr) exprNode() {}

// Stmt é uma instrução da seção de formato (print, skip, page, need, set, if, begin/end).
type Stmt interface{ stmtNode() }

type PrintItem struct {
	Expr     Expr
	Col      Expr // nil = posição automática
	Using    Expr // nil = sem máscara
	Centered bool

	posCol, posMascara posicao // trechos de Col e de Using
}
type PrintStmt struct {
	Items     []PrintItem
	NoNewline bool
}
type SkipStmt struct{ N int }
type PageStmt struct{ NoFooter, NoHeader bool }
type NeedStmt struct{ N int }
type SetStmt struct {
	Name string
	Expr Expr
}
type IfStmt struct {
	Line       int // linha do script onde está o 'if' (0 se desconhecida)
	Cond       Expr
	Then, Else []Stmt
}
type BlockStmt struct{ Body []Stmt }

func (PrintStmt) stmtNode() {}
func (SkipStmt) stmtNode()  {}
func (PageStmt) stmtNode()  {}
func (NeedStmt) stmtNode()  {}
func (SetStmt) stmtNode()   {}
func (IfStmt) stmtNode()    {}
func (BlockStmt) stmtNode() {}

// InputCol descreve uma coluna declarada na seção 'input'.
type InputCol struct {
	Name   string
	Type   string // string, numeric, float, amount, date, ldate, time, text, bool...
	Len    int
	HasLen bool
}

// SortKey é um elemento da instrução 'sort' - pode ter nome explícito (<nome> expr) usado
// depois em 'before <nome>'/'after <nome>', ou herdar o nome quando a expressão é uma coluna.
type SortKey struct {
	Name string // "" se a expressão não tiver nome (não pode ser usada em before/after)
	Expr Expr
	Desc bool
	pos  posicao // trecho da expressão, sem o "<nome>"
}

// EventKey identifica uma seção de evento: (Kind, Target). Target só é usado com
// Kind "before"/"after" (ex.: ("before","report"), ("after","nome_loja")); as demais
// seções (header, footer, detail) usam Target="".
type EventKey struct {
	Kind   string
	Target string
}

// Script é a árvore sintática completa de um relatório, já com os padrões aplicados.
type Script struct {
	Input      []InputCol
	Width      int
	Length     int
	Top        int
	Bottom     int
	Left       int
	Separator  string
	Sort       []SortKey
	Events     map[EventKey][]Stmt
	Variaveis  map[string]bool // nomes usados em 'set' (precisam de set antes do primeiro uso)
	Nomeadas   map[string]Expr // expressões com apelido: "<nome> expr" no sort ou num print
	ordemEvent []EventKey      // ordem de inserção, só pra mensagens/depuração estáveis
	toks       []Token         // tokens do script, para situar os nós nas mensagens de erro
}
