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
- `--encoding utf-8|latin-1` — codificação de leitura/escrita.

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
- **Expressões:** aritmética (`+ - * /`), comparações (`= <> < > <= >=`), `and`/`or`/`not`,
  `is [not] null`, constantes (números, `'texto'`, datas sem aspas como `31/12/2026`,
  repetição de caractere como `40[-]`), `today`, `hour`, `pageno`.
- **Funções:** `count(* [where expr])`, `total`, `avg`, `min`, `max` (só em `after <chave>`/
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

