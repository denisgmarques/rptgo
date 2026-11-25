# rptgo

**Dados em arquivos de texto, relatórios de verdade.** Descreva num script curto como os dados
devem ser ordenados, agrupados e totalizados, e o `rptgo` entrega o relatório: texto paginado
para impressora, **PDF com logo e layout**, CSV ou JSON, a partir da mesma definição.

![Fatura em PDF gerada pelo rptgo](docs/images/invoice.png)

## Para que serve

Você tem dados em arquivos de texto: um arquivo separado por delimitador, o CSV exportado do
Excel, um JSON, a saída de outro programa ou de uma consulta ao banco. E precisa de um
relatório com **layout**: colunas alinhadas, cabeçalho e rodapé por página, **várias quebras
em cascata** (grupo, subgrupo, item) e **totais, contagens e médias em cada nível**. Escrever
isso à mão em código é trabalhoso; o `rptgo` faz com um script declarativo de poucas linhas,
numa linguagem de relatórios no estilo dos geradores dos anos 80 e 90 (seções `input`, `sort`,
`before`, `after`, `detail`).

```
 fontes de dados            script .rp                          saídas
 ┌────────────────────┐   ┌────────────────────┐
 │ texto com |        │   │ input   (colunas)  │
 │ CSV (do Excel)     │──▶│ sort    (ordem)    │
 │ JSON               │   ├────────────────────┤ ──────────▶  texto paginado (matricial)
 └────────────────────┘   │ before / after     │ ─▶ template ▶  PDF / HTML com logo e caixas
                          │ detail  (layout)   │
                          └────────────────────┘
```

- **Várias fontes, um mesmo script:** texto com `|` (ou o `separator` do script), CSV com
  aspas e qualquer delimitador, inclusive o do Excel brasileiro (`;` e decimal com vírgula), e
  JSON (`--input`). O `input` do script diz o tipo de cada coluna, então datas, números e
  `bool` seguem as mesmas regras em qualquer fonte.
- **Layout para impressão:** o layout do script (`before`/`after`/`detail`) gera o relatório
  paginado e o PDF (`--format text|html`). Também dá para exportar os dados, sem layout, em CSV
  ou JSON (`--format csv|json`), na ordem do `sort`.
- **Quebras em cascata e acumuladores:** `before`/`after` em cada nível do `sort`, com `count`,
  `total`, `avg`, `min` e `max`, inclusive filtrados (`total(salary where active)`).
- **Layout em colunas:** `print ... in col N using 'máscara'`, datas, moldes numéricos,
  texto quebrado por largura e caixas desenhadas com `+---+` (`--box`).
- **PDF sem sair do terminal:** `--format html` junta o relatório a um template com logo e
  variáveis; o navegador imprime em PDF.
- **Um binário só**, em Go, sem dependências.

## Linha de comando

```sh
go build -o rptgo ./cmd/rptgo     # um binário estático, sem dependências
./rptgo [opções] script.rp [dados.txt | -]
```

Sem o arquivo de dados (ou com `-`), lê da entrada padrão. Todas as opções estão na ajuda:

<!-- run -->
```sh
./rptgo --help
```
<!-- expect -->
```text
Uso: rptgo [OPÇÃO]... SCRIPT.rp [ENTRADA.txt | -]

Executa um script de relatório (.rp) sobre um arquivo de dados (texto com campos separados
por '|', CSV ou JSON) e escreve o relatório paginado, PDF/HTML, CSV ou JSON.
Sem ENTRADA.txt, ou com '-', lê os dados da entrada padrão.

Opções:
  -o, --saida, --output ARQUIVO  grava no ARQUIVO em vez da saída padrão
      --format FORMATO           formato da saída: text (padrão), csv, json ou html;
                                 csv, json e html não combinam com --form-feed
      --encoding CODIFICAÇÃO     leitura e escrita: utf-8 (padrão), latin-1 ou cp850
      --header MODO              1ª linha dos dados é cabeçalho: auto (padrão),
                                 sim ou nao
      --form-feed                separa páginas com \f em vez de completar com linhas
                                 em branco até o 'length' do script
      --newline MODO             fim de linha: auto (padrão), lf ou crlf
      --box                      troca caixas ASCII (+---+ e |) por caracteres de desenho
                                 de linha; com --format text (utf-8 ou cp850) ou html
      --strict                   exige que comentários /* */ abram e fechem na
                                 mesma linha

Fonte dos dados (padrão: texto com campos separados por '|', ou pelo 'separator' do script):
      --input FONTE              delimited (padrão), csv ou json. csv lê aspas do RFC 4180;
                                 json lê um array de objetos (ou um objeto por linha) e liga
                                 cada coluna do input à chave de mesmo nome
      --input-delimiter CARACTERE  delimitador do CSV de entrada: ; , | tab ou auto (padrão: detecta
                                 pelo número de colunas do input; aspas são sempre aceitas)
      --input-decimal CARACTERE  separador decimal dos números na entrada: . (padrão), , ou auto
                                 (com , lê 1.250,50 como 1250.50; auto escolhe pelos valores);
                                 CSV do Excel brasileiro: --input csv --input-decimal auto
                                 --encoding latin-1

Opções de --format csv (padrão: RFC 4180, vírgula, aspas só quando necessárias):
      --csv-delimiter CARACTERE  delimitador dos campos: ; | tab ... (padrão: vírgula)
      --csv-quote MODO           minimal (padrão), strings (todo texto e o cabeçalho entre
                                 aspas) ou all (todos os campos)
      --csv-decimal CARACTERE    separador decimal dos números: . (padrão) ou ,
      --csv-no-header            não escreve a linha de cabeçalho

Opções de --format html (a saída é sempre UTF-8, sem linhas em branco de preenchimento):
      --template ARQUIVO         modelo HTML (padrão: folhas A4 embutidas)
      --var NOME=VALOR           define {{NOME}} no modelo; pode repetir
      --logo ARQUIVO             imagem (png, jpg, gif, svg, webp) em {{Logo}}
      --title TEXTO              {{Title}} (padrão: nome do script)
      --lines N                  linhas de texto por página (padrão: length - margens
                                 superior e inferior do script)
  Modelo: {{#Pages}}...{{/Pages}} repete por página. Variáveis: Title, Script, Date, Time,
  Columns, PageCount, Logo, as de --var e, dentro do bloco, PageNumber e Text.

Outras opções:
  -r, --check                    só verifica a sintaxe do script, sem gerar relatório
  -h, --help                     mostra esta ajuda e sai

Variáveis de ambiente: DATEFMT, CENTURY_CUTOFF, TIMEFMT, WIDTH, PAGELN, SEPARATOR,
RADIXSEP, CURRSYM e RPT_NOW (fixa o relógio, ex.: 2026-11-24 17:47).

Código de saída: 0 sucesso; 1 erro no script, nos dados ou de E/S; 2 uso inválido.

Exemplos:
  rptgo relatorio.rp dados.txt > relatorio.txt
  rptgo --format json relatorio.rp dados.txt
  cat dados.txt | rptgo -o saida.txt relatorio.rp -
  rptgo --format csv --csv-delimiter ';' --csv-quote strings --csv-decimal ',' relatorio.rp dados.txt
  rptgo --format html --box --template modelo.html --logo logo.svg relatorio.rp dados.txt > saida.html
```

## Veja funcionando

Todos os exemplos abaixo estão em [`examples/`](examples/), cada um com um `run.sh`. Os
scripts, os dados e as saídas deste README são conferidos por testes, então o que você vê aqui
é o que o programa produz.

### 1. Folha de pagamento por departamento

Uma quebra (`dept`), só funcionários ativos (`if active`) e, no fim de cada departamento,
quantidade, total, média e maior salário com `where`.

O script, [`examples/payroll-by-department/report.rp`](examples/payroll-by-department/report.rp):

<!-- file: examples/payroll-by-department/report.rp -->
```text
/* Payroll by department: active employees only, with headcount, total, average and
   highest salary per department. Inactive employees stay in the data but are skipped. */
input
   dept      [string 12],
   employee  [string 18],
   hired     [ldate 10],
   active    [bool],
   salary    [amount 10]

width 72
length 40
sort dept

before report
   print 'PAYROLL BY DEPARTMENT' centered
   print 'Department' in col 1, 'Employee' in col 14, 'Hired' in col 34, 'Salary' in col 52
   print 60[-] in col 1

detail
   if active then
      print dept in col 1, employee in col 14, hired in col 34, salary in col 46 using '###,##&.&&'

after dept
   print 'Headcount:' in col 14, count(* where active) in col 46 using '#####&'
   print 'Total:' in col 14, total(salary where active) in col 46 using '###,##&.&&'
   print 'Average:' in col 14, avg(salary where active) in col 46 using '###,##&.&&'
   print 'Highest:' in col 14, max(salary where active) in col 46 using '###,##&.&&'
   skip

after report
   print 'ALL DEPARTMENTS' in col 1, total(salary where active) in col 46 using '###,##&.&&'
end
```

Os dados, [`examples/payroll-by-department/data.txt`](examples/payroll-by-department/data.txt), campos separados por `|`:

<!-- file: examples/payroll-by-department/data.txt -->
```text
Engineering|Maria Santos|2018-03-12|yes|9200.00
Engineering|John Carter|2020-07-01|yes|8400.50
Engineering|Priya Nair|2022-01-17|yes|7600.00
Engineering|Tom Becker|2015-09-30|no|9800.00
Finance|Laura Chen|2017-05-22|yes|6800.75
Finance|Omar Haddad|2019-11-04|yes|5900.00
Finance|Sven Olsen|2012-02-13|no|7100.00
Sales|Ana Pereira|2021-06-14|yes|4300.00
Sales|Luke Turner|2023-02-27|yes|3900.25
Sales|Zoe Martin|2016-08-08|yes|5200.00
Sales|Ian Moore|2014-12-01|no|4800.00
```

Executar e o resultado:

<!-- run -->
```sh
./rptgo examples/payroll-by-department/report.rp examples/payroll-by-department/data.txt
```
<!-- expect -->
```text


                           PAYROLL BY DEPARTMENT
  Department   Employee            Hired             Salary
  ------------------------------------------------------------
  Engineering  Maria Santos        12/03/2018    9,200.00
  Engineering  John Carter         01/07/2020    8,400.50
  Engineering  Priya Nair          17/01/2022    7,600.00
               Headcount:                           3
               Total:                           25,200.50
               Average:                          8,400.17
               Highest:                          9,200.00

  Finance      Laura Chen          22/05/2017    6,800.75
  Finance      Omar Haddad         04/11/2019    5,900.00
               Headcount:                           2
               Total:                           12,700.75
               Average:                          6,350.38
               Highest:                          6,800.75

  Sales        Ana Pereira         14/06/2021    4,300.00
  Sales        Luke Turner         27/02/2023    3,900.25
  Sales        Zoe Martin          08/08/2016    5,200.00
               Headcount:                           3
               Total:                           13,400.25
               Average:                          4,466.75
               Highest:                          5,200.00

  ALL DEPARTMENTS                               51,301.50
```

### 2. Pedidos de compra em PDF: cliente > pedido > itens

Duas quebras em cascata com subtotal por pedido, total por cliente e total geral, marcas
condicionais (`BULK` para 100 unidades ou mais, `RUSH ORDER`) e frete para pedidos pequenos.

O script, [`examples/purchase-orders/report.rp`](examples/purchase-orders/report.rp):

<!-- file: examples/purchase-orders/report.rp -->
```text
/* Purchase orders: customer > order > line items. Line total is quantity * unit price,
   lines with 100 or more units are tagged BULK, and orders with a subtotal under 500.00
   show a flat shipping charge (listed, not included in the totals). */
input
   customer   [string 14],
   order_no   [string 8],
   sku        [string 10],
   quantity   [numeric 6],
   unit_price [amount 10],
   rush       [bool]

width 76
length 50
sort customer, order_no

before report
   print 'PURCHASE ORDERS' centered
   skip

before customer
   print 'Customer: ' + customer in col 1

before order_no
   print 'Order ' + order_no in col 3
   print 'SKU' in col 5, 'Qty' in col 18, 'Price' in col 28, 'Total' in col 40

detail
   print sku in col 5, quantity in col 16 using '#####&', unit_price in col 26 using '###,##&.&&', quantity * unit_price in col 38 using '###,##&.&&' no newline
   if quantity >= 100 then print 'BULK' in col 52 else print ''

after order_no
   print 'Subtotal:' in col 5, total(quantity * unit_price) in col 38 using '###,##&.&&'
   if total(quantity * unit_price) < 500 then print 'Shipping:' in col 5, 25.00 in col 38 using '###,##&.&&'
   if count(* where rush) > 0 then print 'RUSH ORDER' in col 5
   skip

after customer
   print 'Customer total:' in col 3, total(quantity * unit_price) in col 38 using '###,##&.&&'
   print 'Lines:' in col 3, count(*) in col 38 using '#####&'
   skip

after report
   print 'ALL ORDERS' in col 1, total(quantity * unit_price) in col 38 using '###,##&.&&'
end
```

Os dados, [`examples/purchase-orders/data.txt`](examples/purchase-orders/data.txt):

<!-- file: examples/purchase-orders/data.txt -->
```text
Northwind|PO-2001|WIDGET-A|150|2.40|no
Northwind|PO-2001|WIDGET-B|40|7.95|no
Northwind|PO-2002|GADGET-X|3|49.99|yes
Contoso|PO-2003|BOLT-M8|500|0.18|no
Contoso|PO-2003|NUT-M8|500|0.07|no
Contoso|PO-2004|CABLE-2M|12|5.50|no
Fabrikam|PO-2005|PANEL-LG|2|310.00|yes
Fabrikam|PO-2005|SCREW-KIT|25|3.20|yes
```

Gerar o HTML (modelo A4 embutido) e imprimir em PDF pelo navegador:

<!-- run-only -->
```sh
./rptgo --format html --title "Purchase orders" examples/purchase-orders/report.rp examples/purchase-orders/data.txt > purchase-orders.html
```

```sh
google-chrome --headless --no-pdf-header-footer --print-to-pdf=purchase-orders.pdf purchase-orders.html
```

O resultado (página 1 do PDF; o relatório tem 2 páginas, com cabeçalho, rodapé e numeração):

![Pedidos de compra em PDF](docs/images/purchase-orders.png)

### 3. Fatura com logo, caixas e template próprio

O script desenha a tabela com `+---+` e `|`; `--box` troca por caracteres de linha, e um
template HTML (`{{Title}}`, `{{Logo}}`, `{{Text}}`...) coloca o logo, o cabeçalho e o rodapé.

O script, [`examples/invoice-html/report.rp`](examples/invoice-html/report.rp):

<!-- file: examples/invoice-html/report.rp -->
```text
/* Invoice lines drawn with ASCII boxes (+---+ and |). Run with --box to get line-drawing
   characters, and with --format html --template template.html for the printable page. */
input
   item   [string 26],
   qty    [numeric 6],
   price  [amount 10]

width 70
length 40
sort item

before report
   print '+----------------------------+-------+------------+------------+'
   print '| Item                       |   Qty |      Price |      Total |'
   print '+----------------------------+-------+------------+------------+'

detail
   print '|' in col 1, item in col 3, '|' in col 30, qty in col 32 using '####&', '|' in col 38, price in col 40 using '###,##&.&&', '|' in col 51, qty * price in col 53 using '###,##&.&&', '|' in col 64

after report
   print '+----------------------------+-------+------------+------------+'
   print '|' in col 1, 'Items' in col 3, '|' in col 30, total(qty) in col 32 using '####&', '|' in col 38, '|' in col 51, total(qty * price) in col 53 using '###,##&.&&', '|' in col 64
   print '+----------------------------+-------+------------+------------+'
end
```

Os dados, [`examples/invoice-html/data.txt`](examples/invoice-html/data.txt):

<!-- file: examples/invoice-html/data.txt -->
```text
Consulting - design review|8|120.00
Cloud hosting (monthly)|1|349.90
Support plan - premium|12|45.50
Domain registration|2|14.99
```

Só o texto, com as caixas convertidas:

<!-- run -->
```sh
./rptgo --box examples/invoice-html/report.rp examples/invoice-html/data.txt
```
<!-- expect -->
```text


  ┌────────────────────────────┬───────┬────────────┬────────────┐
  │ Item                       │   Qty │      Price │      Total │
  ├────────────────────────────┼───────┼────────────┼────────────┤
  │ Cloud hosting (monthly)    │     1 │     349.90 │     349.90 │
  │ Consulting - design review │     8 │     120.00 │     960.00 │
  │ Domain registration        │     2 │      14.99 │      29.98 │
  │ Support plan - premium     │    12 │      45.50 │     546.00 │
  ├────────────────────────────┼───────┼────────────┼────────────┤
  │ Items                      │    23 │            │   1,885.88 │
  └────────────────────────────┴───────┴────────────┴────────────┘
```

E a fatura completa em PDF, com o template [`template.html`](examples/invoice-html/template.html), o logo
[`logo.svg`](examples/invoice-html/logo.svg) e as variáveis passadas na linha de comando:

<!-- run-only -->
```sh
./rptgo --format html --box \
  --template examples/invoice-html/template.html --logo examples/invoice-html/logo.svg \
  --var Company="Northwind Ltd" --var Invoice=2026-0142 \
  --var BillTo="Acme Corp, 100 Main St, Springfield" --var Footer="payment due in 30 days" \
  examples/invoice-html/report.rp examples/invoice-html/data.txt > invoice.html
```

```sh
google-chrome --headless --no-pdf-header-footer --print-to-pdf=invoice.pdf invoice.html
```

O resultado é a imagem do topo deste README. Para gerar o PDF direto, rode
[`examples/invoice-html/run.sh`](examples/invoice-html/run.sh) (ele já escolhe o Chrome instalado).

### 4. Os mesmos dados em CSV e JSON

Sem mudar uma linha do script: `--format csv` e `--format json` ignoram o layout e entregam um
registro por linha de entrada, na ordem do `sort`, com datas em `AAAA-MM-DD`, números como
números e nulos como vazio ou `null`.

<!-- run -->
```sh
./rptgo --format csv examples/relatorio-vendas-regiao/relatorio.rp examples/relatorio-vendas-regiao/dados.txt
```
<!-- expect -->
```text
regiao,vendedor,valor
Leste,Diego Alves,450.25
Leste,Fabio Neri,780
Norte,Bruno Lima,980
Norte,Elisa Prado,1320.4
Sul,Ana Souza,1250.5
Sul,Carla Dias,2100.75
Sul,Gina Rocha,310.1
```

<!-- run -->
```sh
./rptgo --format json examples/relatorio-vendas-regiao/relatorio.rp examples/relatorio-vendas-regiao/dados.txt
```
<!-- expect -->
```text
[
{"regiao":"Leste","vendedor":"Diego Alves","valor":450.25},
{"regiao":"Leste","vendedor":"Fabio Neri","valor":780},
{"regiao":"Norte","vendedor":"Bruno Lima","valor":980},
{"regiao":"Norte","vendedor":"Elisa Prado","valor":1320.4},
{"regiao":"Sul","vendedor":"Ana Souza","valor":1250.5},
{"regiao":"Sul","vendedor":"Carla Dias","valor":2100.75},
{"regiao":"Sul","vendedor":"Gina Rocha","valor":310.1}
]
```

### 5. Fontes de dados: texto, CSV do Excel e JSON

O mesmo script lê qualquer fonte; só muda a opção `--input`. No CSV, o delimitador (`,`, `;`,
tab ou `|`) é detectado sozinho: o `rptgo` escolhe o que divide as primeiras linhas exatamente
no número de colunas do `input`, e aspas não precisam de opção, pois campos com e sem aspas
convivem no mesmo arquivo. O JSON pode ser um array de objetos
(inclusive o que o próprio `--format json` gera) ou um objeto por linha; cada coluna do `input`
busca a chave de mesmo nome, e chave ausente ou `null` vira campo vazio. O CSV aceita campos
entre aspas, com delimitador e quebra de linha dentro, BOM e CRLF.

<!-- file: examples/relatorio-vendas-regiao/dados.csv -->
```text
regiao,vendedor,valor
Sul,Ana Souza,1250.50
Norte,Bruno Lima,980.00
Sul,Carla Dias,2100.75
Leste,Diego Alves,450.25
Norte,Elisa Prado,1320.40
Leste,Fabio Neri,780.00
Sul,Gina Rocha,310.10
```

<!-- file: examples/relatorio-vendas-regiao/dados.json -->
```text
[
  {"regiao": "Sul", "vendedor": "Ana Souza", "valor": 1250.50},
  {"regiao": "Norte", "vendedor": "Bruno Lima", "valor": 980.00},
  {"regiao": "Sul", "vendedor": "Carla Dias", "valor": 2100.75},
  {"regiao": "Leste", "vendedor": "Diego Alves", "valor": 450.25},
  {"regiao": "Norte", "vendedor": "Elisa Prado", "valor": 1320.40},
  {"regiao": "Leste", "vendedor": "Fabio Neri", "valor": 780.00},
  {"regiao": "Sul", "vendedor": "Gina Rocha", "valor": 310.10}
]
```

<!-- run-only -->
```sh
./rptgo --input csv examples/relatorio-vendas-regiao/relatorio.rp examples/relatorio-vendas-regiao/dados.csv
```

<!-- run-only -->
```sh
./rptgo --input json examples/relatorio-vendas-regiao/relatorio.rp examples/relatorio-vendas-regiao/dados.json
```

Os dois produzem exatamente o mesmo relatório do exemplo 1 de dados em texto. O CSV exportado
pelo Excel com configuração regional brasileira usa `;` como delimitador e vírgula decimal
(`1.250,50`). O RPT original só lê o ponto como separador decimal na entrada (o `RADIXSEP` só
muda a exibição dos valores `amount`), então `--input-decimal ','` é uma extensão do `rptgo`,
desligada por padrão (`auto` a liga só se os valores tiverem a forma `1250,50`). A opção
"CSV (separado por vírgulas)" antiga grava em Windows-1252: nesse caso junte `--encoding latin-1`. O arquivo de exemplo
[`dados-excel-br.csv`](examples/relatorio-vendas-regiao/dados-excel-br.csv) está nesse formato:

```text
regiao;vendedor;valor
Sul;Ana Souza;1.250,50
Norte;Bruno Lima;980,00
Sul;Carla Dias;2.100,75
Leste;Diego Alves;450,25
Norte;Elisa Prado;1.320,40
Leste;Fabio Neri;780,00
Sul;Gina Rocha;310,10
```

<!-- run -->
```sh
./rptgo --input csv --input-decimal auto examples/relatorio-vendas-regiao/relatorio.rp examples/relatorio-vendas-regiao/dados-excel-br.csv
```
<!-- expect -->
```text


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

**Direto de um banco, sem driver:** os clientes de linha de comando já exportam em texto, e o
`rptgo` lê da entrada padrão (`-`). O padrão do `psql -A` já usa `|`:

```sh
psql -At -F'|' -c "select regiao, vendedor, valor from vendas" | ./rptgo relatorio.rp -
sqlite3 -separator '|' vendas.db "select regiao, vendedor, valor from vendas" | ./rptgo relatorio.rp -
mysql -B -e "select regiao, vendedor, valor from vendas" | ./rptgo --input csv --input-delimiter tab relatorio.rp -
```

Há mais exemplos em `examples/`: estoque por categoria (duas quebras com `count`/`total`/`avg`),
cadastro de clientes (datas, `bool` e nulos), contas a receber (prazos e faixas de atraso) e
vendas por região.

## Opções

```
rptgo [opções] script.rp [entrada.txt | -]  > relatorio.txt
```

- `-h, --help` — mostra a ajuda (uso, opções, variáveis de ambiente, códigos de saída e
  exemplos) e sai. Sem o script, o programa mostra a mesma ajuda no erro padrão (código 2).
  O nome do comando na ajuda e nas mensagens é o nome pelo qual o binário foi chamado.
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
- `--format text|csv|json|html` — formato da saída (padrão: `text`, o relatório paginado de
  colunas fixas). `csv` e `json` geram um registro por linha de entrada, na ordem do `sort`,
  com as colunas do `input`; as seções `before`/`after`/`header`/`footer`/`detail` não são
  executadas e `width`/`length`/margens são ignorados. Datas (`date` e `ldate`) saem como
  `AAAA-MM-DD`, nulos como vazio (CSV) ou `null` (JSON), e `bool` como `true`/`false`. O
  `html` junta as páginas do relatório a um modelo HTML (veja "Saída HTML e caixas").
- `--box` — troca caixas ASCII (`+---+` e `|`) por caracteres de desenho de linha (`┌─┬┐│├┼┤└┴┘`).
  Vale com `--format text` (`--encoding utf-8` ou `cp850`, que usa os bytes de caixa do DOS) e
  com `html`; não vale com `latin-1`, `csv` nem `json`.
- `--input delimited|csv|json` — fonte dos dados (padrão: `delimited`, texto com o separador do
  script). `csv` lê aspas do RFC 4180 e `json` lê um array de objetos (ou um objeto por linha),
  ligando cada coluna do `input` à chave de mesmo nome (sem diferenciar maiúsculas).
  `--input-delimiter CARACTERE` (`;`, `|`, `tab`...) fixa o delimitador do CSV; sem ele (ou com
  `auto`) o `rptgo` detecta pelo número de colunas do `input`. `--input-decimal ,` lê números
  como `1.250,50` e `--input-decimal auto` escolhe pelos valores. Veja "Fontes de dados".
- `--csv-delimiter`, `--csv-quote minimal|strings|all`, `--csv-decimal`, `--csv-no-header` —
  apresentação do `--format csv`: delimitador (`;`, `tab`...), aspas só quando necessárias
  (padrão), em todo texto e no cabeçalho (`strings`) ou em todos os campos (`all`), vírgula
  decimal e cabeçalho opcional.
- `--template`, `--var NOME=VALOR`, `--logo`, `--title`, `--lines` — opções do `--format html`.
  `--format csv|json|html` não pode ser combinado com `--form-feed` (erro, código de saída 2).
  Valores inválidos em `--format`, `--header`, `--newline` e `--encoding` também dão erro
  com código 2.

## Saída HTML e caixas

`--format html` junta as páginas do relatório a um modelo HTML, para imprimir numa jato de tinta
ou salvar em PDF pelo navegador, sem mudar o script `.rp`. O RPT continua cuidando da
ordenação, das quebras, dos totais e do alinhamento; o modelo só cuida da moldura (logo,
cabeçalho, rodapé, cores).

```
rptgo --format html --box --template examples/invoice-html/template.html \
      --logo examples/invoice-html/logo.svg \
      --var Company="Northwind Ltd" --var Invoice=2026-0142 \
      --var BillTo="Acme Corp, 100 Main St, Springfield" --var Footer="payment due in 30 days" \
      examples/invoice-html/report.rp examples/invoice-html/data.txt > invoice.html
```

- **Páginas sem preenchimento:** diferente do `text` (que completa cada página com linhas em
  branco até o `length`, como numa impressora matricial), o HTML traz só as linhas de cada
  página, sem a margem superior nem a esquerda do script. Quem quebra a página fisicamente é
  o CSS (`@page`, `break-after`). O número de linhas por página vem de `length` menos
  `top margin` e `bottom margin`, ou de `--lines N`. A saída é sempre UTF-8.
- **Modelo:** texto HTML com marcadores `{{Nome}}`. `{{#Pages}}...{{/Pages}}` repete o bloco
  uma vez por página. Todo valor é escapado para HTML. Variável desconhecida é erro, com a
  linha do modelo. Sem `--template`, um modelo embutido (folhas A4) é usado.

  | Marcador | Valor |
  |---|---|
  | `{{Title}}` | `--title` (padrão: nome do script, sem extensão) |
  | `{{Script}}` | nome do arquivo do script |
  | `{{Date}}`, `{{Time}}` | data (`AAAA-MM-DD`) e hora (`HH:MM`); `RPT_NOW` fixa o relógio |
  | `{{Columns}}` | largura do texto em caracteres (para calcular o tamanho da fonte) |
  | `{{PageCount}}` | número de páginas |
  | `{{Logo}}` | `--logo` como imagem embutida (data URI: png, jpg, gif, svg ou webp) |
  | `{{NOME}}` | cada `--var NOME=VALOR` (os nomes acima são reservados) |
  | `{{PageNumber}}`, `{{Text}}` | só dentro do bloco: número e texto da página |

- **Caixas:** com `--box`, as caixas ASCII viram `┌─┬┐│├┼┤└┴┘`. Cada `+` vira canto ou
  junção conforme os vizinhos; `-` e `|` só mudam quando fazem parte de uma linha, então
  `2026-06` ou um `|` solto no texto ficam como estão. Com `--format text --encoding cp850` os
  caracteres saem como os bytes de caixa do DOS.

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
- **Níveis de quebra:** cada chave do `sort` é um nível, e o nome é o da coluna:
  `sort depto, nome` já permite `before depto`/`after depto` e `before nome`/`after nome`. Só
  uma chave calculada precisa de nome explícito, entre `<>`, antes da expressão:
  `sort <periodo> (mdy(data,2) * 100 + mdy(data,0)), loja` e depois `before periodo`. Depois da
  chave pode vir `desc`.
- **Expressões nomeadas:** `<nome> expressão` dá um apelido a uma expressão, no `sort` ou num
  `print` (`print item, <total> preco * qtd in col 30`). Depois de nomeado, o nome vale nas
  expressões seguintes como se fosse mais uma coluna (`print total * 2`, `if total > 100 then`,
  `total(total)`), e o apelido tem o tipo da expressão (um apelido aritmético não serve de
  condição). Repetir o mesmo nome com a mesma expressão é aceito; com outra expressão, ou com o
  nome de uma coluna ou variável, é erro.
- **Comandos:** `print` (com `in col N`, `using 'máscara'`, `centered`, `no newline`),
  `skip [n]`, `page [, no footer][, no header]`, `need n`, `set var to expr`,
  `if expr then ... else ...`, `begin ... end`.
- **Expressões:** aritmética (`+ - * / %`; `/` e `%` entre inteiros dão inteiro), comparações (`= <> < > <= >=`), `and`/`or`/`not`,
  `is [not] null` (expressões lógicas valem 1 ou 0; `if`, `and`/`or`/`not` e `where` exigem
  expressão lógica, não aritmética), constantes (números, `'texto'`, datas sem aspas como `31/12/2026`,
  repetição de caractere como `40[-]`), `today`, `hour`, `pageno`. Texto: `+` (concatena e mantém
  os brancos do fim), `/+` (concatena tirando os brancos do fim do lado esquerdo) e
  `texto substr [início-fim]` (posições de 1 em diante, fim inclusivo; o fim além do tamanho é
  cortado e um início além do tamanho dá texto vazio).
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
  box.go            --box: caixas ASCII -> caracteres de desenho de linha
  csv.go            saída --format csv
  json.go           saída --format json
  html.go           saída --format html: modelo com {{marcadores}}, logo embutido
  ajuda.go          --help e valores aceitos pelas opções
scripts/            render-readme-images.sh gera as imagens do README (docs/images)
examples/           relatórios de exemplo (vendas, estoque, cadastro, folha, contas a receber, pedidos, fatura HTML)
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

