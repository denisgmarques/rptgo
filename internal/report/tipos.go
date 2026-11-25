package report

import "sort"

// Tipagem estática lógico x aritmético. O tipo de uma expressão é o da última operação
// executada: and/or/not, comparações e 'is [not] null' dão lógico; + - * / e negação dão
// aritmético. Colunas bool são lógicas. Variáveis de 'set' não têm tipo conhecido em tempo
// de parse e são aceitas em qualquer lugar.
type tipoExpr int

const (
	tAritm   tipoExpr = iota // aritmética (ou qualquer valor não lógico: texto, data, número)
	tLogico                  // lógica
	tDesconh                 // indeterminado (variável de 'set', nome inexistente)
)

func tipoColuna(s *Script, nome string) string {
	for _, c := range s.Input {
		if c.Name == nome {
			return c.Type
		}
	}
	return ""
}

func tipoDe(s *Script, e Expr) tipoExpr {
	switch x := e.(type) {
	case AndExpr, OrExpr, NotExpr, CmpExpr, IsNullExpr:
		return tLogico
	case ColExpr:
		if s.Variaveis[x.Name] {
			return tDesconh
		}
		switch tipoColuna(s, x.Name) {
		case "":
			return tDesconh // o erro de nome desconhecido sai em tempo de execução
		case "bool":
			return tLogico
		}
		return tAritm
	}
	return tAritm
}

// verificaTipos garante que 'if', 'where' de count(* where ...) e os operandos de
// and/or/not sejam expressões lógicas. É executada uma vez, ao final do Parse.
func verificaTipos(s *Script) error {
	chaves := make([]EventKey, 0, len(s.Events))
	for k := range s.Events {
		chaves = append(chaves, k)
	}
	sort.Slice(chaves, func(i, j int) bool {
		if chaves[i].Kind != chaves[j].Kind {
			return chaves[i].Kind < chaves[j].Kind
		}
		return chaves[i].Target < chaves[j].Target
	})
	for _, k := range chaves {
		if err := verificaStmts(s, s.Events[k]); err != nil {
			return err
		}
	}
	for _, sk := range s.Sort {
		if err := verificaExpr(s, sk.Expr, 0); err != nil {
			return err
		}
	}
	return nil
}

func verificaStmts(s *Script, lst []Stmt) error {
	for _, st := range lst {
		var err error
		switch x := st.(type) {
		case PrintStmt:
			for _, it := range x.Items {
				for _, e := range []Expr{it.Expr, it.Col, it.Using} {
					if e != nil && err == nil {
						err = verificaExpr(s, e, 0)
					}
				}
			}
		case SetStmt:
			err = verificaExpr(s, x.Expr, 0)
		case IfStmt:
			if err = verificaExpr(s, x.Cond, x.Line); err == nil {
				err = exigeLogico(s, x.Cond, x.Line, "a condição do 'if'")
			}
			if err == nil {
				err = verificaStmts(s, x.Then)
			}
			if err == nil {
				err = verificaStmts(s, x.Else)
			}
		case BlockStmt:
			err = verificaStmts(s, x.Body)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func exigeLogico(s *Script, e Expr, linha int, onde string) error {
	if tipoDe(s, e) != tAritm {
		return nil
	}
	const dica = "precisa ser uma expressão lógica (comparação, 'is null', and/or/not ou coluna bool), não aritmética"
	if linha > 0 {
		return ErroLinhaScript(linha, "%s %s", onde, dica)
	}
	return Erro("%s %s", onde, dica)
}

// verificaExpr percorre a expressão checando operandos de and/or/not e 'where'.
func verificaExpr(s *Script, e Expr, linha int) error {
	switch x := e.(type) {
	case NotExpr:
		if err := exigeLogico(s, x.X, linha, "o operando de 'not'"); err != nil {
			return err
		}
		return verificaExpr(s, x.X, linha)
	case AndExpr:
		return verificaBin(s, x.L, x.R, linha, "'and'")
	case OrExpr:
		return verificaBin(s, x.L, x.R, linha, "'or'")
	case CmpExpr:
		return verificaPar(s, x.L, x.R, linha)
	case BinExpr:
		return verificaPar(s, x.L, x.R, linha)
	case NegExpr:
		return verificaExpr(s, x.X, linha)
	case IsNullExpr:
		return verificaExpr(s, x.X, linha)
	case CallExpr:
		if x.Where != nil {
			if err := exigeLogico(s, x.Where, linha, "o 'where' de count(* where ...)"); err != nil {
				return err
			}
			if err := verificaExpr(s, x.Where, linha); err != nil {
				return err
			}
		}
		for _, a := range x.Args {
			if err := verificaExpr(s, a, linha); err != nil {
				return err
			}
		}
	}
	return nil
}

func verificaPar(s *Script, l, r Expr, linha int) error {
	if err := verificaExpr(s, l, linha); err != nil {
		return err
	}
	return verificaExpr(s, r, linha)
}

func verificaBin(s *Script, l, r Expr, linha int, op string) error {
	for _, e := range []Expr{l, r} {
		if err := exigeLogico(s, e, linha, "o operando de "+op); err != nil {
			return err
		}
		if err := verificaExpr(s, e, linha); err != nil {
			return err
		}
	}
	return nil
}
