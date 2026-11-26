package report

// referencia é um nome (coluna, variável ou expressão nomeada) ou uma chamada de função usada
// numa expressão, com o token onde foi escrito.
type referencia struct {
	nome    string
	chamada bool
	tok     int
}

// faixa é o trecho [ini, fim) da lista de referências que pertence a uma expressão.
type faixa struct{ ini, fim int }

// verificaOrdenacao confere, antes de ler os dados, que as chaves do 'sort' só usam colunas,
// variáveis de 'set', expressões nomeadas e as funções que podem ser calculadas registro a
// registro. Sem isso o erro seria descartado na ordenação e os dados sairiam fora de ordem.
func (ps *parser) verificaOrdenacao(s *Script) error {
	vistos := map[string]bool{} // cada expressão nomeada é conferida uma vez (evita laços)
	var confere func(faixa) error
	confere = func(f faixa) error {
		for _, r := range ps.refs[f.ini:f.fim] {
			escrito := ps.t[r.tok].Src
			if r.chamada {
				switch r.nome {
				case "dow", "mdy", "index":
					continue
				case "count", "total", "avg", "min", "max":
					return ps.erroEm(r.tok, "funções de agregação não podem ser usadas no sort: '%s'", escrito)
				}
				return ps.erroEm(r.tok, "função desconhecida no sort: '%s'", escrito)
			}
			if tipoColuna(s, r.nome) != "" || s.Variaveis[r.nome] {
				continue
			}
			nomeada, ok := ps.faixaNomes[r.nome]
			if !ok {
				return ps.erroEm(r.tok, "nome desconhecido no sort: '%s'", escrito)
			}
			if !vistos[r.nome] {
				vistos[r.nome] = true
				if err := confere(nomeada); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, f := range ps.chaves {
		if err := confere(f); err != nil {
			return err
		}
	}
	return nil
}

// erroComparacaoSort completa o erro de uma chave com dois valores que não se comparam: o trecho
// da chave no script e os registros dos dois valores.
func erroComparacaoSort(err error, sk SortKey, a, b Row) error {
	if e, ok := err.(*Error); ok {
		e.msg = "no sort: " + e.msg
		e.marcado, e.pos, e.linha, e.outra = true, sk.pos, a, b
	}
	return naSecao(err, secaoSort)
}
