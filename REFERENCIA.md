# Referência da linguagem de relatórios (`.rp`)

Esta página descreve a linguagem dos scripts do `rptgo`: as seções, os comandos, as expressões, as
funções e as máscaras de impressão. Cada exemplo mostra o **script**, os **dados** de entrada e a
**saída** que o programa gera, e todos são conferidos por testes automáticos, então o que está
aqui é o que o programa faz. Para as opções de linha de comando e as fontes de dados (texto, CSV,
JSON), veja o [README](README.md).

**Como ler os exemplos.** Os dados têm um registro por linha, com os campos separados por `|`
(o separador padrão). Para deixar a saída curta, os exemplos mostram o relatório sem a margem
esquerda e sem as linhas em branco que completam a página.

## Sumário

1. [Estrutura de um script](#1-estrutura-de-um-script)
2. [Configurações da página](#2-configurações-da-página)
3. [Colunas de entrada e tipos (`input`)](#3-colunas-de-entrada-e-tipos-input)
4. [Ordenação e quebras (`sort`)](#4-ordenação-e-quebras-sort)
5. [Seções do relatório](#5-seções-do-relatório)
6. [Comandos](#6-comandos)
7. [Expressões](#7-expressões)
8. [Funções](#8-funções)
9. [Máscaras de impressão (`using`)](#9-máscaras-de-impressão-using)
10. [Configuração por variáveis de ambiente](#10-configuração-por-variáveis-de-ambiente)
11. [Modo `--strict`](#11-modo---strict)
12. [Mensagens de erro comuns](#12-mensagens-de-erro-comuns)
13. [Resumo rápido](#13-resumo-rápido)

---

## 1. Estrutura de um script

Um script é uma lista de **instruções de configuração** (`input`, `sort`, `width`...) seguida das
**seções** (`before report`, `detail`...) e termina com `end`. O `end` é obrigatório, e o `input`
também. Maiúsculas e minúsculas não importam nas palavras e nos nomes.

O menor script possível lê duas colunas e imprime uma linha por registro:

<!-- script -->
```text
input
   nome  [string 8],
   valor [numeric]

detail
   print nome, valor
end
```
<!-- data -->
```text
Ana|10
Beto|20
```
<!-- output -->
```text
Ana      10
Beto     20
```

**Comentários** vão entre `/*` e `*/`. Por padrão podem ocupar várias linhas; com a opção
`--strict` precisam abrir e fechar na mesma linha (essa opção também muda o `header`: veja a
seção 11).

<!-- script -->
```text
/* Lista simples.
   Comentários podem ter varias linhas. */
input
   nome  [string 8],     /* coluna de texto */
   valor [numeric]

detail
   print nome, valor
end
```
<!-- data -->
```text
Ana|10
```
<!-- output -->
```text
Ana      10
```

**Ordem das partes.** As configurações (`input`, `width`, `length`, margens, `separator`, `sort`)
podem vir em qualquer ordem antes das seções. Cada seção (`before report`, `after report`,
`header`, `footer`, `detail`, `before <chave>`, `after <chave>`) só pode aparecer **uma vez**.

---

## 2. Configurações da página

| Instrução | Efeito | Padrão | Variável de ambiente |
|---|---|---|---|
| `width N` | largura da linha, em caracteres | 80 | `WIDTH` |
| `length N` | linhas por página (quando preenche a página) | 66 | `PAGELN` |
| `top margin N` | linhas em branco no topo de cada página | 2 | |
| `bottom margin N` | linhas reservadas no fim de cada página | 2 | |
| `left margin N` | espaços à esquerda de cada linha | 2 | |
| `separator 'c'` | caractere que separa os campos nos dados | `'\|'` | `SEPARATOR` |

O texto que passa da largura é cortado. As linhas por página que sobram para o conteúdo são
`length - top margin - bottom margin`; quando elas acabam, começa uma nova página.

Exemplo 1: dados separados por `;` e texto cortado na largura (`width`):

<!-- script -->
```text
input
   fruta [string 10],
   obs   [string 30]
width 20
left margin 0
separator ';'
detail
   print fruta, obs
end
```
<!-- data -->
```text
Maca;uma fruta muito vermelha e doce
Uva;roxa
```
<!-- output -->
```text
Maca       uma fruta
Uva        roxa
```

Exemplo 2: quebra de página com `length`. A seção `header` roda no topo de cada página, e `pageno`
é o número da página (veja a seção 5):

<!-- script -->
```text
input
   n [numeric]
length 4
top margin 0
bottom margin 0
left margin 0
header
   print 'Pagina', pageno
detail
   print n
end
```
<!-- data -->
```text
1
2
3
4
5
```
<!-- output -->
```text
Pagina 1
1
2
3
Pagina 2
4
5
```

Cada página tem 4 linhas: o cabeçalho e três registros. Numa impressora de formulário contínuo,
cada página é completada com linhas em branco até o `length`; nos exemplos elas são omitidas. A
opção `--form-feed` troca esse preenchimento por um caractere de avanço de formulário entre as
páginas, e o formato HTML não preenche páginas.

---

## 3. Colunas de entrada e tipos (`input`)

A instrução `input` declara, **na ordem em que aparecem em cada registro**, as colunas dos dados.
Cada coluna é `nome [tipo tamanho]`, separadas por vírgula. Sem `[...]` o tipo é `string`, e um
tipo fora da tabela abaixo é erro de script (seção 12). O tamanho só importa para `string` (e é opcional nas datas e horas). A palavra `sorted` depois de
uma coluna é aceita e ignorada.

| Tipo | O que guarda | Observações |
|---|---|---|
| `string`, `text` | texto | `[string N]` corta o valor em N caracteres e o imprime com largura N; os brancos do fim são removidos |
| `numeric`, `integer`, `long`, `int` | número inteiro | um valor com fração é truncado (`3.7` vira `3`) |
| `float`, `amount`, `double`, `real`, `huge` | número com fração | sem máscara, imprime com 2 casas decimais |
| `date` | data | imprime com ano de 2 dígitos (formato `DATEFMT`) |
| `ldate` | data | imprime com ano de 4 dígitos |
| `time` | hora do dia | `H:MM`, `HH:MM` ou `HH:MM:SS`; imprime como `TIMEFMT` |
| `bool` | verdadeiro ou falso | veja a tabela abaixo |

Um campo vazio é **nulo** em qualquer tipo (o texto vazio também conta como nulo no `is null`).
Nulos imprimem em branco, exceto o `bool`, que imprime `***`. Um valor inválido para o tipo, como
`abc` numa coluna numérica, interrompe o relatório com um erro que mostra o registro (seção 12).

Exemplo com todos os tipos, num registro completo e num com campos vazios:

<!-- script -->
```text
input
   id    [numeric],
   preco [amount 10],
   nasc  [date],
   cad   [ldate],
   hora  [time],
   ativo [bool],
   nome  [string 6]

detail
   print id, preco, nasc, cad, hora, ativo, nome
end
```
<!-- data -->
```text
1|1250.5|15/03/1985|2026-01-15|8:05|yes|Ana
2||||||Beto
3|3.99|||||Joana Maria Santos
```
<!-- output -->
```text
1 1250.50 15/03/85 15/01/2026 08:05 YES Ana
2     *** Beto
3 3.99    *** Joana
```

Repare que `Joana Maria Santos` foi cortado em 6 caracteres (`[string 6]`), que `3.99` num campo
`amount` imprime com 2 casas, e que o campo `ativo` vazio do registro 2 imprime `***`.

**Valores aceitos em `bool`** (a comparação não diferencia maiúsculas e minúsculas):

| Entrada | Valor |
|---|---|
| `0`, `0.0`, `FALSE`, `NO`, qualquer texto começando com `N` ou `F` | falso (`NO`) |
| qualquer número diferente de zero, `TRUE`, `YES`, qualquer texto começando com `Y` ou `T` | verdadeiro (`YES`) |
| qualquer outra coisa (`5xx`, `xxx`, `*`) ou campo vazio | nulo (`***`) |

<!-- script -->
```text
input
   entrada [string 8],
   ativo   [bool]
detail
   print entrada, ativo
end
```
<!-- data -->
```text
0|0
1|1
999|999
false|false
Yes|Yes
nao|nao
t|t
xxx|xxx
vazio|
```
<!-- output -->
```text
0        NO
1        YES
999      YES
false    NO
Yes      YES
nao      NO
t        YES
xxx      ***
vazio    ***
```

(Em cada linha dos dados acima, o primeiro campo é só um rótulo para você ver o que foi lido.)

**Datas.** Uma data de entrada pode vir como `DD/MM/AA`, `DD/MM/AAAA` ou `AAAA-MM-DD`
(o formato ISO é aceito sempre). A ordem dos campos nas formas com barra depende de `DATEFMT`
(padrão `DD/MM/YY`, veja a seção 10). Um ano de 2 dígitos usa `CENTURY_CUTOFF` (padrão 30): de 30
em diante é 19xx, abaixo disso é 20xx.

<!-- script -->
```text
input
   texto [string 14],
   d     [date],
   l     [ldate]
detail
   print texto, d, l
end
```
<!-- data -->
```text
ano curto|05/03/85|05/03/85
ano 2000|05/03/29|05/03/29
ano 4 digitos|05/03/2026|05/03/2026
ISO|2026-12-31|2026-12-31
```
<!-- output -->
```text
ano curto      05/03/85 05/03/1985
ano 2000       05/03/29 05/03/2029
ano 4 digitos  05/03/26 05/03/2026
ISO            31/12/26 31/12/2026
```

A coluna `date` imprime o ano com 2 dígitos e a `ldate` com 4: é a **única** diferença entre elas.

**Horas.** O tipo `time` é uma hora do dia (de `0:00` a `23:59`), não uma duração. Veja a aritmética
de horas na seção 7.

**Nomes de colunas e palavras reservadas.** O nome de uma coluna começa com uma letra (ou `_`),
tem letras, dígitos e `_`, não diferencia maiúsculas de minúsculas e deve ser único. Palavras-chave e palavras
reservadas não deveriam ser usadas como nome de coluna, mas o `rptgo` é tolerante: as palavras das
instruções e os nomes de tipos servem de nome de coluna, e é por isso que o exemplo dos feriados usa
`date`, `name` e `type` (os nomes das chaves do JSON):

<!-- script -->
```text
input
   date  [ldate],
   name  [string 8],
   type  [string 6]
left margin 0
sort name
detail
   print date, name, type
end
```
<!-- data -->
```text
2026-03-05|Ana|a
2026-01-20|Beto|b
```
<!-- output -->
```text
05/03/2026 Ana      a
20/01/2026 Beto     b
```

Mas alguns nomes **já têm significado dentro das expressões** e vencem a coluna: `today`, `hour` e
`pageno` (variáveis embutidas), `null` (a constante) e `not` (o operador). Uma coluna chamada
`hour` imprimiria o relógio, não o dado:

<!-- script -->
```text
input
   hour [numeric]
left margin 0
detail
   print 'o dado era 5, mas hour vale', hour
end
```
<!-- data -->
```text
5
```
<!-- output -->
```text
o dado era 5, mas hour vale 17:47
```

Palavras reservadas:

| Grupo | Palavras |
|---|---|
| configuração e seções | `input` `sort` `width` `length` `top` `bottom` `left` `margin` `separator` `end` `before` `after` `header` `footer` `detail` `report` |
| comandos | `print` `skip` `page` `need` `set` `if` `then` `else` `begin` |
| opções | `col` `column` `using` `centered` `in` `no` `newline` `desc` `sorted` `where` `to` |
| operadores e constantes | `and` `or` `not` `is` `null` `substr` |
| tipos | `string` `text` `numeric` `integer` `long` `int` `float` `amount` `double` `real` `huge` `date` `ldate` `time` `bool` |
| funções e variáveis embutidas | `count` `total` `avg` `min` `max` `dow` `mdy` `index` `today` `hour` `pageno` |


---

## 4. Ordenação e quebras (`sort`)

`sort` lista as chaves de ordenação, separadas por vírgula. O relatório sai ordenado pelas chaves,
e **cada chave é um nível de quebra**: quando o valor dela muda, rodam as seções `after <chave>` e
`before <chave>` (seção 5). O nome do nível é o nome da coluna. Depois de uma chave pode vir `desc`
para a ordem decrescente (o padrão é crescente).

Um nível, com subtotal no fim de cada grupo:

<!-- script -->
```text
input
   regiao [string 6],
   valor  [numeric]
left margin 0
sort regiao

before regiao
   print 'Regiao', regiao
detail
   print '  ', valor
after regiao
   print '  total', total(valor)
end
```
<!-- data -->
```text
Sul|10
Norte|5
Sul|30
Norte|7
```
<!-- output -->
```text
Regiao Norte
   5
   7
  total 12
Regiao Sul
   10
   30
  total 40
```

Os dados chegaram fora de ordem, e o `sort` os agrupou. Com **dois níveis** (ou mais), as quebras
acontecem em cascata: quando o nível de fora muda, o de dentro também fecha e reabre.

<!-- script -->
```text
input
   regiao  [string 6],
   produto [string 8],
   valor   [numeric]
left margin 0
sort regiao, produto

before regiao
   print 'REGIAO', regiao
before produto
   print '  produto', produto
detail
   print '    ', valor
after produto
   print '  soma', total(valor)
after regiao
   print 'TOTAL', total(valor)
end
```
<!-- data -->
```text
Sul|lapis|10
Norte|caneta|5
Sul|lapis|30
Sul|caderno|8
Norte|caneta|7
```
<!-- output -->
```text
REGIAO Norte
  produto caneta
     5
     7
  soma 12
TOTAL 12
REGIAO Sul
  produto caderno
     8
  soma 8
  produto lapis
     10
     30
  soma 40
TOTAL 48
```

**Ordem decrescente** com `desc`, e ordenação por mais de uma coluna sem seções de quebra (as chaves
só ordenam):

<!-- script -->
```text
input
   nome  [string 6],
   idade [numeric]
left margin 0
sort idade desc, nome
detail
   print idade, nome
end
```
<!-- data -->
```text
Ana|30
Caio|25
Beto|30
Dora|25
```
<!-- output -->
```text
30 Ana
30 Beto
25 Caio
25 Dora
```

**Chave calculada.** Uma chave pode ser uma expressão. Como ela não é uma coluna, precisa de um
**nome** entre `<` e `>` antes da expressão, que é o nome usado nas seções `before` e `after`. No
exemplo, as vendas são agrupadas pelo mês da data (`mdy(data, 0)` é o mês, veja a seção 8):

<!-- script -->
```text
input
   data  [ldate],
   valor [numeric]
left margin 0
sort <mes> mdy(data, 0), data

before mes
   print 'Mes', mdy(data, 0)
detail
   print '  ', data, valor
after mes
   print '  subtotal', total(valor)
end
```
<!-- data -->
```text
2026-02-10|5
2026-01-20|7
2026-02-03|11
2026-01-05|2
```
<!-- output -->
```text
Mes 1
   05/01/2026 2
   20/01/2026 7
  subtotal 9
Mes 2
   03/02/2026 11
   10/02/2026 5
  subtotal 16
```

Chaves que são só uma coluna não precisam de nome: `sort regiao, produto` já cria os níveis
`regiao` e `produto`. A forma `<regiao> regiao` também é aceita, mas é redundante.

**O que uma chave pode usar:** colunas do `input`, variáveis de `set`, expressões nomeadas, `today`,
`hour`, `pageno`, `null` e as funções `dow`, `mdy` e `index`. Um nome desconhecido, outra função ou
uma função de agregação (`total`, `count`...) é erro de script, apontado antes da leitura dos dados
(seção 12).

---

## 5. Seções do relatório

As seções dizem **o que imprimir em cada momento**. Cada uma é uma lista de comandos (seção 6) e só
pode aparecer uma vez no script.

| Seção | Quando roda | Que registro enxerga |
|---|---|---|
| `before report` | uma vez, antes do primeiro registro | o primeiro registro |
| `after report` | uma vez, depois do último registro | o último registro |
| `header` | no topo de cada página | o primeiro registro |
| `footer` | no fim de cada página | o primeiro registro |
| `detail` | para cada registro | o próprio registro |
| `before <chave>` | quando começa um grupo novo da chave (inclusive o primeiro) | o primeiro registro do grupo |
| `after <chave>` | quando o grupo da chave termina | o último registro do grupo |

`<chave>` é o nome de um nível do `sort` (seção 4). As funções de agregação (`count`, `total`,
`avg`, `min` e `max`) só calculam nas seções `after`, onde enxergam todos os registros do grupo (seção 8).

**A ordem em cascata.** Quando uma chave de nível k muda, rodam primeiro os `after` dos níveis de
dentro para fora, até k, e depois os `before` de k para dentro. O exemplo imprime o nome de cada
seção para você ver a sequência com dois níveis:

<!-- script -->
```text
input
   a [string 2],
   b [string 2],
   v [numeric]
left margin 0
sort a, b

before report
   print '[before report]'
before a
   print '[before a]', a
before b
   print '  [before b]', b
detail
   print '    [detail]', v
after b
   print '  [after b]', b
after a
   print '[after a]', a
after report
   print '[after report]'
end
```
<!-- data -->
```text
x|p|1
x|p|2
x|q|3
y|p|4
```
<!-- output -->
```text
[before report]
[before a] x
  [before b] p
    [detail] 1
    [detail] 2
  [after b] p
  [before b] q
    [detail] 3
  [after b] q
[after a] x
[before a] y
  [before b] p
    [detail] 4
  [after b] p
[after a] y
[after report]
```

Há **um único `detail`** no script, e ele roda **uma vez para cada registro**, não uma vez por quebra:
o grupo `x`/`p` tem dois registros, então `[detail]` aparece duas vezes seguidas sob o mesmo
`[before b]`. Já o `before` e o `after` de um nível rodam uma vez por grupo (e, como ao mudar `a` o
nível `b` também fecha, o `[after b]` vem antes do `[after a]`).

**Cabeçalho e rodapé.** O `header` roda no topo de cada página nova e o `footer` no fim, e é
empurrado até a última linha da página. O rodapé reserva espaço no fim, e `pageno` dá o número da
página (seção 8).

<!-- script -->
```text
input
   n [numeric]
length 7
top margin 0
bottom margin 0
left margin 0
header
   print 'cabecalho, pagina', pageno
footer
   print 'rodape, pagina', pageno
detail
   print n
end
```
<!-- data -->
```text
1
2
3
4
5
6
7
```
<!-- output -->
```text
cabecalho, pagina 1
1
2
3
4
5
rodape, pagina 1
cabecalho, pagina 2
6
7



rodape, pagina 2
```

**`before report` e `header`.** O `before report` roda **antes** do `header` na primeira página, e
o `header` roda em todas as páginas, inclusive na primeira. Assim o `before report` serve para
inicializar variáveis e imprimir um título, e o cabeçalho é escrito uma única vez, no `header`.
Com a opção `--strict`, quando o script tem `before report` o `header` **não** roda na primeira
página (o `before report` ocupa o lugar dele); veja a [seção 11](#11-modo---strict).

<!-- script -->
```text
input
   n [numeric]
length 7
top margin 0
bottom margin 0
left margin 0
before report
   print 'TITULO DO RELATORIO'
header
   print 'cabecalho, pagina', pageno
footer
   print 'rodape, pagina', pageno
detail
   print n
end
```
<!-- data -->
```text
1
2
3
4
5
6
7
```
<!-- output -->
```text
TITULO DO RELATORIO
cabecalho, pagina 1
1
2
3
4
rodape, pagina 1
cabecalho, pagina 2
5
6
7


rodape, pagina 2
```

Com `--strict`, o `header` não roda na primeira página, porque o script tem `before report`:

<!-- output: --strict -->
```text
TITULO DO RELATORIO
1
2
3
4
5
rodape, pagina 1
cabecalho, pagina 2
6
7



rodape, pagina 2
```

---

## 6. Comandos

Dentro de uma seção você escreve comandos, um por instrução. Os comandos são `print`, `skip`, `page`,
`need`, `set`, `if` e `begin ... end`.

### `print`

`print` imprime uma lista de itens separados por vírgula. Cada item é uma expressão (seção 7) com
opções opcionais, na ordem `in col N`, `using 'máscara'`, `centered`:

| Opção | Efeito |
|---|---|
| (nenhuma) | o item começa uma posição depois do anterior, ou na coluna 1 se é o primeiro |
| `in col N` ou `col N` | o item começa na coluna N (a expressão N é calculada) |
| `using 'máscara'` | formata o valor com uma máscara (seção 9) |
| `centered` | centraliza o item na largura da linha |
| `no newline` (no fim do comando) | não quebra a linha: o próximo `print` continua nela |

Uma coluna `string` com tamanho declarado é impressa **com a largura declarada** (completada com
brancos), o que alinha o item seguinte.

<!-- script -->
```text
input
   nome  [string 8],
   valor [numeric]
width 30
left margin 0
detail
   print nome, valor
   print 'Nome:' in col 1, nome in col 8, 'Valor:' in col 20, valor in col 27
   print 'centro' centered
   print 'a' no newline
   print 'b'
   skip
end
```
<!-- data -->
```text
Ana|10
Beto|20
```
<!-- output -->
```text
Ana      10
Nome:  Ana         Valor: 10
            centro
a b

Beto     20
Nome:  Beto        Valor: 20
            centro
a b
```

Dois itens na mesma coluna se sobrepõem (o último vence). Texto além da largura (`width`) é cortado.

### `skip`, `page` e `need`

- `skip` imprime uma linha em branco; `skip N`, N linhas.
- `page` termina a página agora e começa outra (rodando o `footer` e o `header`). `page, no footer`
  e `page, no header` pulam um deles, e dá para combinar: `page, no footer, no header`.
- `need N` garante que ainda cabem N linhas na página; se não couberem, começa uma página nova.
  Serve para não separar um título do seu grupo.

O exemplo força uma quebra depois do registro 3 com `page`:

<!-- script -->
```text
input
   n [numeric]
length 8
top margin 0
bottom margin 0
left margin 0
header
   print '--- pagina', pageno, '---'
detail
   print n
   if n = 3 then page
end
```
<!-- data -->
```text
1
2
3
4
5
```
<!-- output -->
```text
--- pagina 1 ---
1
2
3




--- pagina 2 ---
4
5
```

E aqui o `need 3` impede que o título de um grupo fique sozinho no fim de uma página. Cada grupo
tem 3 linhas (o título e dois registros), e só sobra 1 linha na página depois do primeiro grupo,
então o segundo grupo começa na página seguinte:

<!-- script -->
```text
input
   g [string 2],
   v [numeric]
length 5
top margin 0
bottom margin 0
left margin 0
sort g
header
   print '== pagina', pageno
before g
   need 3
   print 'grupo', g
detail
   print '  ', v
end
```
<!-- data -->
```text
g1|1
g1|2
g2|3
g2|4
g3|5
g3|6
```
<!-- output -->
```text
== pagina 1
grupo g1
   1
   2

== pagina 2
grupo g2
   3
   4

== pagina 3
grupo g3
   5
   6
```

### `set`: variáveis

`set nome to expressão` guarda um valor numa variável, que vale até o fim do relatório e pode ser
usada em qualquer seção. Uma variável ainda não atribuída vale **nulo** (que na aritmética conta
como 0), então dá para acumular sem inicializar. O nome de uma variável não pode ser igual ao de uma
coluna do `input`.

Totais correntes, zerados a cada grupo:

<!-- script -->
```text
input
   g [string 2],
   valor [numeric]
left margin 0
sort g
before g
   set soma to 0
   print 'grupo', g
detail
   set soma to soma + valor
   print '  ', valor, 'acumulado', soma
end
```
<!-- data -->
```text
a|10
a|5
b|7
b|1
b|2
```
<!-- output -->
```text
grupo a
   10 acumulado 10
   5 acumulado 15
grupo b
   7 acumulado 7
   1 acumulado 8
   2 acumulado 10
```

### `if`, `else` e `begin ... end`

`if condição then comando [else comando]`. A condição precisa ser uma expressão **lógica**
(seção 7). Para executar mais de um comando, agrupe com `begin ... end`. O `else if` encadeia
condições.

<!-- script -->
```text
input
   nome  [string 6],
   valor [numeric]
left margin 0
detail
   if valor >= 100 then print nome, 'alto'
   else if valor >= 50 then print nome, 'medio'
   else print nome, 'baixo'
   if valor = 0 then
      begin
         print '  (valor zero!)'
         skip
      end
end
```
<!-- data -->
```text
Ana|150
Beto|70
Caio|0
Dora|10
```
<!-- output -->
```text
Ana    alto
Beto   medio
Caio   baixo
  (valor zero!)

Dora   baixo
```

---

## 7. Expressões

Uma expressão calcula um valor a partir de constantes, colunas, variáveis, operadores e funções.
Aparece em `print`, `set`, `if`, `sort`, no `where` das funções de agregação e nos argumentos das funções.

### 7.1 Constantes

| Constante | Exemplos | Observações |
|---|---|---|
| número | `42`, `3.14`, `-5` | sem ponto é inteiro; com ponto, decimal |
| texto | `'texto'`, `"texto"` | aspas simples ou duplas; precisa estar numa única linha; `\` escapa o caractere seguinte |
| data | `31/12/2026` | sem aspas, na ordem de `DATEFMT` |
| hora | `14:30` | sem aspas, de `0:00` a `23:59` |
| repetição de caractere | `12[-]` | repete o caractere entre colchetes: `------------` |
| nulo | `null` | valor ausente |

<!-- script -->
```text
input x [numeric]
left margin 0
detail
   print '42' in col 1, 42 in col 20
   print '3.14' in col 1, 3.14 in col 20
   print 'texto' in col 1, 'texto' in col 20
   print 'aspas duplas' in col 1, "com 'aspas' dentro" in col 20
   print 'data' in col 1, 31/12/2026 in col 20
   print 'hora' in col 1, 14:30 in col 20
   print 'repeticao' in col 1, 12[-] in col 20
   print 'null' in col 1, null in col 20
end
```
<!-- data -->
```text
1
```
<!-- output -->
```text
42                 42
3.14               3.14
texto              texto
aspas duplas       com 'aspas' dentro
data               31/12/26
hora               14:30
repeticao          ------------
null
```

### 7.2 Operadores aritméticos

`+  -  *  /  %` e o `-` unário. **A divisão entre dois inteiros é inteira** (`7 / 2` dá `3`, e
`-7 / 2` dá `-3`); com um lado decimal o resultado é decimal. `%` é o resto da divisão. Os
parênteses mudam a ordem. A tabela de precedência, da mais forte para a mais fraca:

| Nível | Operadores |
|---|---|
| 1 | `-` unário e `substr` |
| 2 | `*  /  %` |
| 3 | `+  -  /+` |
| 4 | `=  <>  ^=  !=  <  >  <=  >=  is null  is not null` |
| 5 | `not` |
| 6 | `and` |
| 7 | `or` |

<!-- script -->
```text
input x [numeric]
left margin 0
detail
   print '7 + 2' in col 1, 7 + 2 in col 18
   print '7 / 2' in col 1, 7 / 2 in col 18
   print '7.0 / 2' in col 1, 7.0 / 2 in col 18
   print '-7 / 2' in col 1, -7 / 2 in col 18
   print '7 % 2' in col 1, 7 % 2 in col 18
   print '2 + 3 * 4' in col 1, 2 + 3 * 4 in col 18
   print '(2 + 3) * 4' in col 1, (2 + 3) * 4 in col 18
   print '10 - 4 - 3' in col 1, 10 - 4 - 3 in col 18
   print '-x' in col 1, -x in col 18
   print '1 / 3' in col 1, 1 / 3 in col 18
   print '1.0 / 3' in col 1, 1.0 / 3 in col 18
end
```
<!-- data -->
```text
5
```
<!-- output -->
```text
7 + 2            9
7 / 2            3
7.0 / 2          3.50
-7 / 2           -3
7 % 2            1
2 + 3 * 4        14
(2 + 3) * 4      20
10 - 4 - 3       3
-x               -5
1 / 3            0
1.0 / 3          0.33
```

### 7.3 Texto

- `+` junta dois textos **mantendo** os brancos do fim do primeiro.
- `/+` junta tirando antes os brancos do fim do primeiro.
- `texto substr [início-fim]` pega um pedaço: as posições começam em 1, o fim é inclusivo, um fim
  além do tamanho é cortado, e um início além do tamanho dá texto vazio. Início e fim são
  números inteiros, com `início >= 1` e `fim >= início`.

Os operadores de texto não funcionam com números (`'a' + 1` é um erro).

<!-- script -->
```text
input x [numeric]
left margin 0
detail
   print 'ab + cd' in col 1, '[' + 'ab' + 'cd' + ']' in col 24
   print "'ab  ' + 'cd'" in col 1, '[' + ('ab  ' + 'cd') + ']' in col 24
   print "'ab  ' /+ 'cd'" in col 1, '[' + ('ab  ' /+ 'cd') + ']' in col 24
   print 'ABCDEF substr [2-4]' in col 1, 'ABCDEF' substr [2-4] in col 24
   print 'ABCDEF substr [4-99]' in col 1, 'ABCDEF' substr [4-99] in col 24
   print 'ABCDEF substr [9-9]' in col 1, '[' + ('ABCDEF' substr [9-9]) + ']' in col 24
end
```
<!-- data -->
```text
1
```
<!-- output -->
```text
ab + cd                [abcd]
'ab  ' + 'cd'          [ab  cd]
'ab  ' /+ 'cd'         [abcd]
ABCDEF substr [2-4]    BCD
ABCDEF substr [4-99]   DEF
ABCDEF substr [9-9]    []
```

### 7.4 Comparações e lógica

As comparações são `=`, `<>` (ou `^=`, `!=`), `<`, `>`, `<=`, `>=`, e o teste de nulo `is null` /
`is not null`. Os operadores lógicos são `not`, `and` e `or`. **Verdadeiro vale 1 e falso vale 0.**
Textos são comparados pela ordem dos caracteres, e só se compara data com data, hora com hora, e
texto com texto (número com número).

<!-- script -->
```text
input x [numeric]
left margin 0
detail
   print '3 > 2' in col 1, 3 > 2 in col 28
   print "'a' < 'b'" in col 1, 'a' < 'b' in col 28
   print '3 <> 3' in col 1, 3 <> 3 in col 28
   print '3 ^= 4' in col 1, 3 ^= 4 in col 28
   print 'not 1 = 2' in col 1, not 1 = 2 in col 28
   print '1 = 1 and 2 = 3' in col 1, 1 = 1 and 2 = 3 in col 28
   print '1 = 1 or 2 = 3' in col 1, 1 = 1 or 2 = 3 in col 28
   print '1 = 1 or 1 = 2 and 1 = 3' in col 1, 1 = 1 or 1 = 2 and 1 = 3 in col 28
   print '(1 = 1 or 1 = 2) and 1 = 3' in col 1, (1 = 1 or 1 = 2) and 1 = 3 in col 28
end
```
<!-- data -->
```text
1
```
<!-- output -->
```text
3 > 2                      1
'a' < 'b'                  1
3 <> 3                     0
3 ^= 4                     1
not 1 = 2                  1
1 = 1 and 2 = 3            0
1 = 1 or 2 = 3             1
1 = 1 or 1 = 2 and 1 = 3   1
(1 = 1 or 1 = 2) and 1 = 3 0
```

**Condições são expressões lógicas.** O tipo de uma expressão é o da **última operação** que ela
executa: `and`, `or`, `not`, as comparações e `is null` dão um resultado **lógico**; `+ - * / %`
dão um resultado **aritmético**. Uma expressão lógica pode entrar numa conta (valendo 1 ou 0), mas
uma expressão aritmética **não** pode ser usada onde se espera uma condição:

- a condição de um `if`;
- os operandos de `and`, `or` e `not`;
- o `where` de uma função de agregação (seção 8).

Uma coluna `bool` é lógica e pode ser usada direto. Uma variável criada com `set` não tem tipo
fixo e é aceita em qualquer lugar. O erro aparece **antes** de o relatório rodar:

<!-- script -->
```text
input
   n    [numeric],
   ok   [bool]
detail
   if n > 5 then print 'n maior que 5'
   if ok then print 'ok verdadeiro'
   if n then print 'isto e um erro: n e aritmetico'
end
```
<!-- data -->
```text
10|yes
```
<!-- error -->
```text
rptgo: linha 7: a condição do 'if' precisa ser uma expressão lógica (comparação, 'is null', and/or/not ou coluna bool), não aritmética
  7 |    if n then print 'isto e um erro: n e aritmetico'
      ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
```

Uma expressão lógica dentro de uma conta vale 1 ou 0, então dá para contar condições:

<!-- script -->
```text
input n [numeric]
left margin 0
detail
   print n, 'passa de 3:', n > 3, ' passa de 5:', n > 5, ' quantas:', (n > 3) + (n > 5)
end
```
<!-- data -->
```text
2
4
10
```
<!-- output -->
```text
2 passa de 3: 0  passa de 5: 0  quantas: 0
4 passa de 3: 1  passa de 5: 0  quantas: 1
10 passa de 3: 1  passa de 5: 1  quantas: 2
```

### 7.5 Datas

Uma data pode ser comparada com outra data, e dá para somar ou subtrair **dias**. A diferença
entre duas datas é o **número de dias**. Uma conta com datas devolve uma `date`, que imprime com o
ano de 2 dígitos, mesmo que as colunas sejam `ldate`.

<!-- script -->
```text
input
   d [ldate],
   e [ldate]
left margin 0
detail
   print 'd' in col 1, d in col 28
   print 'd + 30' in col 1, d + 30 in col 28
   print 'd - 7' in col 1, d - 7 in col 28
   print 'e - d (dias)' in col 1, e - d in col 28
   print 'd < e' in col 1, d < e in col 28
   print 'd = 05/03/2026' in col 1, d = 05/03/2026 in col 28
   print '31/12/2026 - 01/01/2026' in col 1, 31/12/2026 - 01/01/2026 in col 28
end
```
<!-- data -->
```text
2026-03-05|2026-04-14
```
<!-- output -->
```text
d                          05/03/2026
d + 30                     04/04/26
d - 7                      26/02/26
e - d (dias)               40
d < e                      1
d = 05/03/2026             1
31/12/2026 - 01/01/2026    364
```

### 7.6 Horas

O tipo `time` é uma hora do dia. Compara-se hora com hora; somar ou subtrair um número soma ou
subtrai **minutos** (dando a volta em 24 horas, então `23:30 + 60` é `00:30`); a diferença entre
duas horas é o número de **minutos**. Somar duas horas não existe.

<!-- script -->
```text
input
   h [time],
   g [time]
left margin 0
detail
   print 'h' in col 1, h in col 24
   print 'h + 90 (minutos)' in col 1, h + 90 in col 24
   print 'h - 30' in col 1, h - 30 in col 24
   print '23:30 + 60' in col 1, 23:30 + 60 in col 24
   print 'g - h (minutos)' in col 1, g - h in col 24
   print 'h < g' in col 1, h < g in col 24
   print 'h >= 08:00' in col 1, h >= 08:00 in col 24
end
```
<!-- data -->
```text
8:30|17:45
```
<!-- output -->
```text
h                      08:30
h + 90 (minutos)       10:00
h - 30                 08:00
23:30 + 60             00:30
g - h (minutos)        555
h < g                  1
h >= 08:00             1
```

### 7.7 Nulos

Um valor nulo imprime em branco. Em uma conta, o nulo conta como 0; numa comparação, como o
valor "vazio" do tipo do outro lado (0, texto vazio, data mínima). Para testar nulidade use
`is null` / `is not null`, que também consideram nulo o texto vazio. As funções de agregação
ignoram nulos (seção 8).

<!-- script -->
```text
input
   a [numeric],
   b [numeric],
   t [string 4]
left margin 0
detail
   print 'a + 5' in col 1, a + 5 in col 20
   print 'a = 0' in col 1, a = 0 in col 20
   print 'a is null' in col 1, a is null in col 20
   print 'b is null' in col 1, b is null in col 20
   print 't is null' in col 1, t is null in col 20
end
```
<!-- data -->
```text
|7|
```
<!-- output -->
```text
a + 5              5
a = 0              1
a is null          1
b is null          0
t is null          1
```

(Neste registro `a` e `t` estão vazios e `b` vale 7.)

### 7.8 Expressões nomeadas

`<nome> expressão` dá um **nome** a uma expressão, no `sort` (seção 4) ou num `print`. Depois de
nomeada, o nome pode ser usado nas expressões seguintes como se fosse uma coluna: em outros
`print`, em condições, em funções de agregação. O nome tem o tipo da expressão que ele nomeia. Repetir o
mesmo nome só é aceito com a mesma expressão, e o nome não pode ser o de uma coluna nem de uma
variável.

<!-- script -->
```text
input
   produto  [string 8],
   preco    [amount 10],
   desconto [amount 10]
left margin 0
sort produto
detail
   print produto, <liquido> preco - desconto
   print '  com imposto', <final> liquido * 1.10, ' diferenca', preco - final
end
```
<!-- data -->
```text
lapis|10.00|1.50
caderno|20.00|5.00
```
<!-- output -->
```text
caderno  15.00
  com imposto 16.50  diferenca 3.50
lapis    8.50
  com imposto 9.35  diferenca 0.65
```

---

## 8. Funções

### 8.1 Funções de agregação: `count`, `total`, `avg`, `min`, `max`

As funções de agregação calculam sobre o **grupo** de registros nas seções `after <chave>` (o grupo que acabou
de fechar) e `after report` (todos os registros). Cada grupo começa do zero. Em `detail` e nas seções
`before` não existe um grupo, e o resultado é vazio (`0`); em `header` e `footer` o programa avisa
com um erro.

| Função | Resultado |
|---|---|
| `count(*)` | quantos registros |
| `count(expr)` | quantos registros (inclusive os de `expr` nula) |
| `total(expr)` | a soma de `expr` (ignorando nulos; 0 se não houver valores) |
| `avg(expr)` | a média de `expr` (ignorando nulos; 0 se não houver valores) |
| `min(expr)`, `max(expr)` | o menor e o maior valor (números, textos ou datas; nulo se não houver) |

Todos aceitam `where condição` dentro dos parênteses, para considerar só os registros em que a
condição é verdadeira: `total(valor where regiao = 'Sul')`, `count(* where valor > 100)`. A condição
precisa ser lógica (seção 7.4).

<!-- script -->
```text
input
   regiao [string 6],
   valor  [numeric]
left margin 0
sort regiao
after regiao
   print regiao, 'n', count(*), 'soma', total(valor), 'media', avg(valor), 'min', min(valor), 'max', max(valor)
after report
   print 'GERAL ', 'n', count(*), 'soma', total(valor), 'media', avg(valor)
end
```
<!-- data -->
```text
Norte|10
Norte|
Norte|30
Sul|5
Sul|15
```
<!-- output -->
```text
Norte  n 3 soma 40 media 20.00 min 10 max 30
Sul    n 2 soma 20 media 10.00 min 5 max 15
GERAL  n 5 soma 60 media 15.00
```

Repare no grupo `Norte`: há 3 registros (`n 3`), mas o valor nulo é ignorado na soma e na média (a
média é 20, de 10 e 30). Agora com `where`:

<!-- script -->
```text
input
   regiao [string 6],
   valor  [numeric]
left margin 0
after report
   print 'registros:', count(*)
   print 'acima de 8:', count(* where valor > 8)
   print 'soma no Sul:', total(valor where regiao = 'Sul')
   print 'media dos pares:', avg(valor where valor % 2 = 0)
   print 'maior no Norte:', max(valor where regiao = 'Norte')
end
```
<!-- data -->
```text
Norte|10
Norte|20
Sul|5
Sul|15
Sul|9
```
<!-- output -->
```text
registros: 5
acima de 8: 4
soma no Sul: 29
media dos pares: 15.00
maior no Norte: 20
```

### 8.2 Datas: `dow`, `mdy`

| Função | Resultado |
|---|---|
| `dow(data)` | o dia da semana: 0 = domingo, 1 = segunda, ..., 6 = sábado |
| `mdy(data, 0)` | o mês (1 a 12) |
| `mdy(data, 1)` | o dia do mês |
| `mdy(data, 2)` | o ano (com 4 dígitos) |

Com data nula, devolvem nulo.

### 8.3 Escolha: `index`

`index(n, a, b, c, ...)` devolve o argumento de posição `n`, **começando em 0** (`index(0, 'a', 'b')`
é `'a'`); se `n` está fora da lista, devolve texto vazio. É útil para trocar um número por um nome.

<!-- script -->
```text
input d [ldate]
left margin 0
detail
   print d, 'dia da semana', dow(d), index(dow(d), 'domingo', 'segunda', 'terca', 'quarta', 'quinta', 'sexta', 'sabado')
   print '  mes', mdy(d, 0), index(mdy(d, 0) - 1, 'jan', 'fev', 'mar', 'abr', 'mai', 'jun', 'jul', 'ago', 'set', 'out', 'nov', 'dez'), ' dia', mdy(d, 1), ' ano', mdy(d, 2)
end
```
<!-- data -->
```text
2026-03-05
2026-12-25
```
<!-- output -->
```text
05/03/2026 dia da semana 4 quinta
  mes 3 mar  dia 5  ano 2026
25/12/2026 dia da semana 5 sexta
  mes 12 dez  dia 25  ano 2026
```

### 8.4 Variáveis embutidas: `today`, `hour`, `pageno`

| Nome | Valor |
|---|---|
| `today` | a data de hoje (`ldate`, com ano de 4 dígitos; com `--strict` é `date`, com 2) |
| `hour` | a hora de agora (`time`) |
| `pageno` | o número da página atual, começando em 1 |

Para que o relatório seja reproduzível, a variável de ambiente `RPT_NOW` fixa o relógio (por
exemplo `2026-11-24 17:47`); nos exemplos desta página ela está fixada nesse valor.

<!-- script -->
```text
input n [numeric]
left margin 0
detail
   print 'hoje', today, 'hora', hour, 'pagina', pageno
end
```
<!-- data -->
```text
1
```
<!-- output -->
```text
hoje 24/11/2026 hora 17:47 pagina 1
```

---

## 9. Máscaras de impressão (`using`)

`print valor using 'máscara'` formata o valor. Sem máscara, um inteiro imprime como está, um decimal
com 2 casas (arredondando para cima na metade), e as datas e horas pelos formatos da seção 10.

### Máscaras numéricas

A máscara descreve **uma posição por caractere**, e os dígitos entram da direita para a esquerda:

| Caractere | O que faz |
|---|---|
| `#` | posição de dígito; se não há dígito, fica em branco |
| `&` | posição de dígito; se não há dígito, imprime `0` (e vale para as posições `#` à esquerda dela) |
| `*` | posição de dígito; se não há dígito, imprime `*` (proteção de cheque) |
| `+` | posição do sinal: imprime `+` ou `-` |
| `-` | posição do sinal: imprime `-` só para negativos |
| `$` | posição do símbolo de moeda (`CURRSYM`, padrão `$`) |
| `(` e `)` | parênteses para negativos |
| `,` | separador de milhares (`TRIADSEP`, padrão `,`), impresso só se há dígito à esquerda |
| `.` | separador decimal (`RADIXSEP`, padrão `.`); os caracteres depois dele dão as casas decimais |

Os dígitos depois do ponto decimal arredondam o valor (a metade arredonda para cima). Se o número tem
**mais dígitos do que a máscara comporta**, os dígitos de ordem mais alta que sobram são impressos
inteiros, à esquerda (nada é cortado).

<!-- script -->
```text
input v [float]
left margin 0
detail
   print v in col 1, ' ###,##&.&&' in col 12, v in col 26 using '###,##&.&&'
   print v in col 1, ' &&&&&.&&' in col 12, v in col 26 using '&&&&&.&&'
   print v in col 1, ' -###,##&.&&' in col 12, v in col 26 using '-###,##&.&&'
   print v in col 1, ' +###,##&.&&' in col 12, v in col 26 using '+###,##&.&&'
   print v in col 1, ' $###,##&.&&' in col 12, v in col 26 using '$###,##&.&&'
   print v in col 1, ' (###,##&.&&)' in col 12, v in col 26 using '(###,##&.&&)'
   print v in col 1, ' ***,**&.&&' in col 12, v in col 26 using '***,**&.&&'
   skip
end
```
<!-- data -->
```text
1234.5
-1234.5
```
<!-- output -->
```text
1234.50     ###,##&.&&     1,234.50
1234.50     &&&&&.&&     01234.50
1234.50     -###,##&.&&     1,234.50
1234.50     +###,##&.&&  +  1,234.50
1234.50     $###,##&.&&  $  1,234.50
1234.50     (###,##&.&&)    1,234.50
1234.50     ***,**&.&&   **1,234.50

-1234.50    ###,##&.&&     1,234.50
-1234.50    &&&&&.&&     01234.50
-1234.50    -###,##&.&&  -  1,234.50
-1234.50    +###,##&.&&  -  1,234.50
-1234.50    $###,##&.&&  $  1,234.50
-1234.50    (###,##&.&&) (  1,234.50)
-1234.50    ***,**&.&&   **1,234.50
```

Repare que `&` completa com zeros e `#` com brancos, que um sinal ou símbolo de moeda **único** fica
na primeira posição da máscara, e que os parênteses só aparecem nos negativos.

### Sinal e moeda colados no primeiro dígito

Repita o caractere (`--,---,--&.&&`) e ele passa a **flutuar**: as posições repetidas recebem os
dígitos, e o sinal é impresso uma única vez, logo antes do primeiro deles. Os valores continuam
alinhados pela direita. O mesmo vale para `+`, `$` e `(`. Reserve uma posição a mais para o sinal: a
máscara abaixo comporta 8 dígitos inteiros mais o sinal.

<!-- script -->
```text
input v [float]
left margin 0
detail
   print v using '-##,###,##&.&&' in col 1,
         v using '--,---,--&.&&' in col 18,
         v using '$$,$$$,$$&.&&' in col 34,
         v using '((,(((,((&.&&)' in col 50
end
```
<!-- data -->
```text
-5
-123
-1234.5
-123456
-1234567.89
123
```
<!-- output -->
```text
-         5.00           -5.00           $5.00           (5.00)
-       123.00         -123.00         $123.00         (123.00)
-     1,234.50       -1,234.50       $1,234.50       (1,234.50)
-   123,456.00     -123,456.00     $123,456.00     (123,456.00)
- 1,234,567.89   -1,234,567.89   $1,234,567.89   (1,234,567.89)
        123.00          123.00         $123.00          123.00
```

Na primeira coluna o sinal único fica parado à esquerda; nas outras ele (ou o `$`, ou o parêntese)
acompanha o número.

### Máscaras no estilo `printf`

Uma máscara que começa com `%` usa a notação do `printf`: `%d` e `%i` para inteiros (truncam),
`%f` para decimais, com largura e casas opcionais (`%8.2f`) e `-` para alinhar à esquerda (`%-8.1f`).

<!-- script -->
```text
input v [float]
left margin 0
detail
   print '[' + 'x' + ']' in col 1, v using '%8.2f', v using '%d', v using '%5d', v using '%-8.1f'
end
```
<!-- data -->
```text
1234.567
```
<!-- output -->
```text
[x]  1234.57 1234  1234 1234.6
```

### Máscaras de texto

Uma máscara que é só um número, como `'14'` (ou `'14r'`), formata um **texto**: quebra o texto em
linhas de até esse número de caracteres, sem cortar palavras. A primeira linha fica onde o item foi
impresso, e as outras ficam logo abaixo, na mesma coluna. (Uma máscara numérica aplicada a um
texto imprime em branco, e uma máscara de texto aplicada a um número não faz efeito.)

<!-- script -->
```text
input t [string 60]
left margin 0
detail
   print 'Obs:' in col 1, t in col 6 using '14'
   print 'fim'
end
```
<!-- data -->
```text
uma frase um pouco longa para quebrar em varias linhas
```
<!-- output -->
```text
Obs: uma frase um
     pouco longa
     para quebrar
     em varias
     linhas
fim
```

### Separadores e moeda por variável de ambiente

`RADIXSEP` troca o separador decimal, `TRIADSEP` o de milhares e `CURRSYM` o símbolo de moeda
(só o primeiro caractere é usado, na posição do `$`). Para o formato brasileiro:

<!-- script -->
```text
input v [float]
left margin 0
detail
   print 'R$', v using '###,##&.&&'
end
```
<!-- data -->
```text
54390.5
```
<!-- output: RADIXSEP=, TRIADSEP=. -->
```text
R$  54.390,50
```

---

## 10. Configuração por variáveis de ambiente

Estas variáveis ajustam o comportamento do programa **sem mudar o script**:

| Variável | O que controla | Padrão |
|---|---|---|
| `DATEFMT` | formato das datas na entrada e na saída (use `DD`, `MM`, `YY` ou `YYYY`, com qualquer separador) | `DD/MM/YY` |
| `CENTURY_CUTOFF` | o ano de 2 dígitos a partir do qual é 19xx (abaixo dele, 20xx) | `30` |
| `TIMEFMT` | formato das horas (`HH`, `MM`, `SS`) | `HH:MM` |
| `WIDTH` | largura padrão da linha (o `width` do script vence) | `80` |
| `PAGELN` | linhas por página padrão (o `length` do script vence) | `66` |
| `SEPARATOR` | separador de campos padrão (o `separator` do script vence) | `\|` |
| `RADIXSEP` | separador decimal das máscaras | `.` |
| `TRIADSEP` | separador de milhares das máscaras | `,` |
| `CURRSYM` | símbolo de moeda da máscara `$` | `$` |
| `RPT_NOW` | fixa o relógio (`today`, `hour`): `2026-11-24 17:47` | relógio do sistema |

O `DATEFMT` define a **ordem** dos campos de uma data com barras na entrada e o formato de saída.
Uma data no formato ISO (`2026-03-05`) é aceita sempre. Exemplo com o formato americano, ano de 4
dígitos, o corte do século em 50 e as horas com ponto:

<!-- script -->
```text
input
   d [date],
   l [ldate],
   h [time]
left margin 0
detail
   print d, l, h
end
```
<!-- data -->
```text
03/05/85|03/05/85|8:05
12/31/49|12/31/49|23:59
```
<!-- output: DATEFMT=MM/DD/YYYY CENTURY_CUTOFF=50 TIMEFMT=HH.MM -->
```text
03/05/85 03/05/1985 08.05
12/31/49 12/31/2049 23.59
```

(Com o corte em 50, o ano `85` é 1985 e o `49` é 2049.)

---

## 11. Modo `--strict`

A opção `--strict` liga três regras mais rígidas. Sem ela, o `rptgo` é mais permissivo.

| | Sem `--strict` (padrão) | Com `--strict` |
|---|---|---|
| Comentários `/* */` | podem ocupar várias linhas | precisam abrir e fechar na mesma linha |
| `header` na 1ª página, com `before report` no script | roda, logo depois do `before report` | **não** roda: o `before report` ocupa o lugar dele |
| `today` impresso sozinho | `ldate`: ano de 4 dígitos (`24/11/2026`) | `date`: ano de 2 dígitos (`24/11/26`) |

Scripts sem comentários de várias linhas, sem `before report` junto com `header` e sem `print today`
saem iguais nos dois modos.

### Comentários de uma linha só

No padrão, um comentário pode se estender por várias linhas. Com `--strict`, o mesmo script dá erro:

<!-- script -->
```text
/* comentario
   em duas linhas */
input
   n [numeric]
detail
   print n
end
```
<!-- data -->
```text
1
```
<!-- error: --strict -->
```text
rptgo: linha 1: comentário precisa abrir e fechar na mesma linha (--strict)
  1 | /* comentario
      ^^^^^^^^^^^^^
```

Para passar no modo estrito, feche o comentário na mesma linha em que ele abre, ou use um comentário
por linha.

### O `header` na primeira página

No padrão, o `before report` roda primeiro e o `header` vem logo depois, então o cabeçalho é escrito
uma vez só, no `header`, e o `before report` fica para o título e para inicializar variáveis:

<!-- script -->
```text
input
   n [numeric]
length 7
top margin 0
bottom margin 0
left margin 0
before report
   print 'TITULO'
header
   print 'CABECALHO'
detail
   print n
end
```
<!-- data -->
```text
1
2
3
```
<!-- output -->
```text
TITULO
CABECALHO
1
2
3
```

Com `--strict`, o `header` não roda na primeira página, porque o script tem `before report`. As
páginas seguintes continuam com o `header`:

<!-- output: --strict -->
```text
TITULO
1
2
3
```

Use `--strict` para rodar scripts que dependem da regra estrita, em que o `before report` imprime o
cabeçalho da primeira página e o `header` só cuida das demais. Se o `before report` já repete o
cabeçalho que está no `header`, rode sem a opção e tire a cópia.

---

## 12. Mensagens de erro comuns

Os erros de sintaxe e de tipo aparecem antes de o relatório rodar; os de dados e os de execução,
quando o registro problemático é lido ou calculado. Quase todos mostram a linha envolvida.

Num erro de script, o marcador `^` fica sob a palavra em que o problema foi encontrado. De uma linha
com mais de 72 caracteres sai só o trecho em volta dela, com `...` no lado cortado.

**Tipo desconhecido no `input`:**

<!-- script -->
```text
input nome, valor [numerc]
detail
   print nome, valor
end
```
<!-- data -->
```text
Ana|10
```
<!-- error -->
```text
rptgo: linha 1: tipo desconhecido 'numerc' na coluna 'valor'; tipos válidos: string, text, numeric, integer, long, int, float, amount, double, real, huge, date, ldate, time, bool
  1 | input nome, valor [numerc]
                         ^^^^^^
```

**Nome desconhecido no `sort`:**

<!-- script -->
```text
input idade [numeric]
sort idades
detail
   print idade
end
```
<!-- data -->
```text
30
```
<!-- error -->
```text
rptgo: linha 2: nome desconhecido no sort: 'idades'
  2 | sort idades
           ^^^^^^
```

**Vírgula sobrando no `input`.** Como uma coluna pode ter o nome de uma instrução, a palavra depois da
vírgula é lida como coluna, e a mensagem avisa:

<!-- script -->
```text
input nome, idade [numeric],
detail
   print nome
end
```
<!-- data -->
```text
Ana|30
```
<!-- error -->
```text
rptgo: linha 3: instrução inválida fora de seção: 'print'; 'detail' foi lido como nome de coluna do input; pode haver uma vírgula sobrando na linha 1
  3 |    print nome
         ^^^^^
```

**Nome desconhecido** (uma coluna ou variável que não existe) num comando:

<!-- script -->
```text
input n [numeric]
detail
   print n, total
end
```
<!-- data -->
```text
5
```
<!-- error -->
```text
rptgo: nome desconhecido: 'total' na seção detail
script, linha 3:
  3 |    print n, total
                  ^^^^^
dados, linha 1 (registro 1):
  1 | 5
```

**Valor inválido nos dados** (o texto não é do tipo da coluna), com o número do registro:

<!-- script -->
```text
input
   nome  [string 6],
   valor [numeric]
detail
   print nome, valor
end
```
<!-- data -->
```text
Ana|10
Beto|doze
```
<!-- error -->
```text
rptgo: valor inválido em 'valor' (registro 2): "doze" (strconv.ParseFloat: parsing "doze": invalid syntax)
  2 | Beto|doze
      ^^^^^^^^^
```

**Número de campos diferente do `input`** (aqui o segundo registro tem um campo a mais):

<!-- script -->
```text
input
   nome  [string 6],
   valor [numeric]
detail
   print nome, valor
end
```
<!-- data -->
```text
Ana|10
Beto|20|30
```
<!-- error -->
```text
rptgo: phase error no arquivo de entrada: registro 2 tem 3 campos, esperados 2
  2 | Beto|20|30
      ^^^^^^^^^^
```

**Divisão por zero:**

<!-- script -->
```text
input n [numeric]
detail
   print 10 / n
end
```
<!-- data -->
```text
0
```
<!-- error -->
```text
rptgo: divisão por zero na seção detail
script, linha 3:
  3 |    print 10 / n
               ^^^^^^
conta: 10 / n, com n = 0
dados, linha 1 (registro 1):
  1 | 0
```

**Função de agregação numa seção onde não há grupo** (`header` e `footer`; em `detail` ele devolve 0):

<!-- script -->
```text
input n [numeric]
header
   print 'total ate aqui:', total(n)
detail
   print n
end
```
<!-- data -->
```text
5
```
<!-- error -->
```text
rptgo: a função total() só pode ser usada em 'after sort_key' e 'after report' na seção header
script, linha 3:
  3 |    print 'total ate aqui:', total(n)
                                  ^^^^^
dados, linha 1 (registro 1):
  1 | 5
```

**Seção repetida** e **`before`/`after` de um nível que não existe no `sort`:**

<!-- script -->
```text
input n [numeric]
detail
   print n
detail
   print n
end
```
<!-- data -->
```text
5
```
<!-- error -->
```text
rptgo: linha 4: a seção 'detail' só pode aparecer uma vez
  4 | detail
      ^^^^^^
```
<!-- script -->
```text
input
   g [string 2],
   n [numeric]
sort g
before grupo
   print n
detail
   print n
end
```
<!-- data -->
```text
a|5
```
<!-- error -->
```text
rptgo: 'grupo' na seção before não é uma expressão de sort existente
```

Outros erros frequentes: falta do `end`, falta do `input`, condição aritmética num `if` (seção 7.4),
operação inválida com horas (`h + g`), comparar hora ou data com número, `dow` ou `mdy` sobre o que
não é data, número errado de argumentos numa função, e `substr` com início menor que 1 ou fim menor
que o início.

---

## 13. Resumo rápido

**Estrutura**

```text
input col [tipo tam], ...        colunas dos dados (obrigatório)
width N | length N | top|bottom|left margin N | separator 'c'
sort [<nome>] expr [desc], ...   ordenação e níveis de quebra
before report | after report | header | footer | detail
before <chave> | after <chave>
end
```

**Comandos:** `print item [in col N] [using 'm'] [centered], ... [no newline]` · `skip [N]` ·
`page [, no footer] [, no header]` · `need N` · `set v to expr` · `if c then cmd [else cmd]` ·
`begin ... end`

**Operadores:** `+ - * / %` · `+ /+ substr[a-b]` · `= <> ^= != < > <= >=` · `is [not] null` ·
`not and or` · parênteses. Condição = expressão lógica.

**Funções:** `count(*)` `count(e)` `total(e)` `avg(e)` `min(e)` `max(e)` (só em `after`, aceitam
`where`) · `dow(d)` `mdy(d, 0|1|2)` `index(n, a, b, ...)` · embutidas `today` `hour` `pageno`

**Tipos:** `string text numeric integer long int float amount double real huge date ldate time bool`

**Máscaras:** `# & * + - $ ( ) , .` · `%8.2f %d` · `'14'` (texto)

Veja também o [README](README.md) para as opções de linha de comando, as fontes de dados (texto,
CSV, JSON) e os formatos de saída (texto, HTML/PDF, CSV, JSON).
