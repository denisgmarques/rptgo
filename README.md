# rptgo

Um interpretador, em Go, de uma linguagem de definição de relatórios em colunas fixas:
você descreve os campos de entrada, a ordenação, os níveis de quebra com seus totais, e o
layout exato (coluna a coluna) de cada linha impressa — e o `rptgo` lê os dados e produz o
texto final, paginado, pronto pra uma impressora de formulário contínuo ou pra um arquivo.

Esse tipo de linguagem de relatório — com seções `input`/`sort`/`before`/`after`/`detail`,
comandos `print ... in col N using 'máscara'`, quebras em cascata com acumuladores e um
cabeçalho/rodapé por página — foi comum em sistemas de gestão dos anos 80 e 90. O `rptgo` é
uma implementação independente desse dialeto, escrita a partir de documentação pública e de
testes de comportamento (entrada/saída), sem copiar código de nenhum produto específico.

## Uso

```
rptgo [opções] script.rp [entrada.txt | -]  > relatorio.txt
```

- `-o, --saida <arquivo>` — grava no arquivo em vez da saída padrão.
- `--check` — só verifica a sintaxe do script, não gera relatório.
- `--header auto|sim|nao` — se a 1ª linha dos dados é um cabeçalho com nomes de coluna
  (padrão: detecta automaticamente).
- `--form-feed` — separa páginas com `\f` em vez de completar com linhas em branco até o
  `length` (o padrão imita o comportamento de uma impressora matricial real).
- `--newline auto|lf|crlf` — fim de linha da saída.
- `--strict` — valida um detalhe em que este programa é deliberadamente mais permissivo que
  o comportamento de referência (hoje: aceita comentário `/* */` de várias linhas; o
  `--strict` exige abrir e fechar na mesma linha).
- `--encoding utf-8|latin-1|cp850` — codificação de leitura/escrita. O `cp850` (página de
  código DOS, comum em impressoras matriciais) vale para o script, para os dados e para a
  saída.
- `--format text|csv|json` — formato da saída (padrão: `text`, o relatório paginado de
  colunas fixas). `csv` e `json` geram um registro por linha de entrada, na ordem do `sort`,
  com as colunas do `input`; as seções `before`/`after`/`header`/`footer`/`detail` não são
  executadas e `width`/`length`/margens são ignorados. Datas (`date` e `ldate`) saem como
  `AAAA-MM-DD`, nulos como vazio (CSV) ou `null` (JSON), e `bool` como `true`/`false`.
  `--format csv|json` não pode ser combinado com `--form-feed` (erro, código de saída 2).

## Primeiros passos

Dois exemplos prontos ficam em `examples/`: `relatorio-vendas-regiao/` (uma quebra, com
total e contagem por grupo) e `relatorio-estoque-categoria/` (duas quebras em cascata, com
`count`, `total` e `avg`). Ambos usam dados fictícios e não chamam `today`/`hour`, então a
saída é sempre a mesma. A partir da raiz do repositório:

```
go build -o rptgo ./cmd/rptgo
./rptgo examples/relatorio-vendas-regiao/relatorio.rp examples/relatorio-vendas-regiao/dados.txt
```

Saída (sem as linhas em branco que completam a página no final):

```


                       VENDAS POR REGIAO
  Regiao       Vendedor                Valor
  ----------------------------------------
  Leste        Diego Alves           450.25
  Leste        Fabio Neri            780.00
  Total Leste                      1,230.25
  Vendas:                              2

  Norte        Bruno Lima            980.00
  Norte        Elisa Prado         1,320.40
  Total Norte                      2,300.40
  Vendas:                              2

  Sul          Ana Souza           1,250.50
  Sul          Carla Dias          2,100.75
  Sul          Gina Rocha            310.10
  Total Sul                        3,661.35
  Vendas:                              3

  TOTAL GERAL                      7,192.00
```

O mesmo script e os mesmos dados em CSV (o `sort regiao` ordena as linhas; só as colunas do
`input` saem):

```
./rptgo --format csv examples/relatorio-vendas-regiao/relatorio.rp examples/relatorio-vendas-regiao/dados.txt
```

```
regiao,vendedor,valor
Leste,Diego Alves,450.25
Leste,Fabio Neri,780
Norte,Bruno Lima,980
Norte,Elisa Prado,1320.4
Sul,Ana Souza,1250.5
Sul,Carla Dias,2100.75
Sul,Gina Rocha,310.1
```

## A linguagem do script (`.rp`)

```
input
   nome    [string 30],
   data    [date 8],
   valor   [amount 10]

width 80
length 66
sort nome

before report
   print 'RELATORIO' centered

header
   print 'pag' in col 70, pageno using '####&'

detail
   print nome in col 1, data in col 35, valor in col 50 using '-----&.&&'

after report
   print 'total:' in col 1, total(valor) using '-----&.&&'
end
```

- **Seções de formato:** `input`, `sort`, `width`, `length`, `top margin`/`bottom margin`/
  `left margin`, `separator`, e um `end` final (obrigatório).
- **Eventos:** `before report`, `after report`, `header`, `footer`, `before <chave>`/
  `after <chave>` (uma por nível de quebra declarado em `sort`), `detail`. As quebras
  disparam em cascata: ao mudar uma chave, os níveis mais internos fecham primeiro
  (`after` em ordem do mais específico pro mais geral), depois os novos níveis abrem
  (`before` em ordem do mais geral pro mais específico).
- **Comandos:** `print` (com `in col N`, `using 'máscara'`, `centered`, `no newline`),
  `skip [n]`, `page [, no footer][, no header]`, `need n`, `set var to expr`,
  `if expr then ... else ...`, `begin ... end`.
- **Expressões:** aritmética (`+ - * / %`; `/` e `%` entre inteiros dão inteiro), comparações (`= <> < > <= >=`), `and`/`or`/`not`,
  `is [not] null` (expressões lógicas valem 1 ou 0; `if`, `and`/`or`/`not` e `where` exigem
  expressão lógica, não aritmética), constantes (números, `'texto'`, datas sem aspas como `31/12/2026`,
  repetição de caractere como `40[-]`), `today`, `hour`, `pageno`.
- **Funções:** `count(* [where expr])`, `total(expr [where expr])`, `avg`, `min`, `max` (só em `after <chave>`/
  `after report`), `dow`, `mdy`, `index`.
- **Tipos de coluna:** `numeric`, `float`/`amount`, `date` (ano com 2 dígitos na impressão),
  `ldate` (ano com 4 dígitos), `string`, `text`, `bool`.
- **Máscaras `using`:** notação própria com `# & * + - $ ( ) , .` pra números, `%ld`/`%7.2f`
  estilo printf, e `'N'`/`'Nr'` (largura de coluna com quebra por palavra) pra texto.

## Variáveis de configuração (lidas do ambiente)

`DATEFMT` (padrão `DD/MM/YY`), `CENTURY_CUTOFF` (padrão `30`: ano de 2 dígitos `>= corte`
soma 1900, abaixo soma 2000), `TIMEFMT` (padrão `HH:MM`), `WIDTH`, `PAGELN`, `SEPARATOR`,
`RADIXSEP`, `CURRSYM`. `RPT_NOW` (ex. `2026-11-24 17:47`) fixa o relógio pra testes
reprodutíveis.

## Estrutura

```
cmd/rptgo/          interface de linha de comando
  cp850.go          tabela da página de código CP850 (--encoding cp850)
  csv.go            saída --format csv
  json.go           saída --format json
examples/           relatórios de exemplo (vendas por região, estoque por categoria)
internal/report/
  lexer.go          análise léxica (tokens, comentários, literais de data/repetição)
  parser.go         análise sintática recursiva descendente -> AST
  ast.go             tipos da árvore sintática
  interp.go         execução: conversão de entrada, ordenação, cascata de quebras,
                    paginação, avaliação de expressões
  format.go         máscaras `using` (números e texto)
  dates.go          formatação/leitura de datas
  value.go          valores em tempo de execução, comparação, aritmética
  config.go         leitura de variáveis de ambiente
  errors.go         erros com a linha de origem anexada (script ou dados)
```

