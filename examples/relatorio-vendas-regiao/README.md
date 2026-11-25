# Vendas por região e mês: a mesma venda em quatro fontes

[← Exemplos](../../README.md#exemplos) · [Referência da linguagem](../../REFERENCIA.md)

Quebra em dois níveis: região e, dentro dela, ano/mês da venda. A chave do mês é **calculada** (`ano * 100
+ mês`) e ganha o apelido `<periodo>`, que dá nome às seções `before periodo` e `after periodo`. Mostra
subtotal por mês, total e média por região, total geral e a virada de ano (dezembro de 2025 para janeiro
de 2026). Depois, o mesmo relatório lido de **texto**, **CSV**, **JSON** e do **CSV do Excel**.

## O script

<!-- file: examples/relatorio-vendas-regiao/relatorio.rp -->
```text
/* Vendas por regiao com quebra em dois niveis: regiao e, dentro dela, ano/mes da venda.
   A chave do mes e calculada (ano * 100 + mes) e ganha o apelido <periodo>, que da nome
   as secoes before/after. Subtotal por mes, total e media por regiao, total geral. */
input
   regiao      [string 10],
   vendedor    [string 14],
   data_venda  [ldate 10],
   valor       [amount 10]

width 62
length 40
sort regiao, <periodo> (mdy(data_venda,2) * 100 + mdy(data_venda,0)), data_venda

before report
   print 'VENDAS POR REGIAO E MES' centered
   skip

before regiao
   need 16
   print 'REGIAO ' + regiao in col 1
   print 58[-] in col 1

before periodo
   print 'Mes ' in col 3, mdy(data_venda,0) in col 7 using '&&', '/' in col 9, mdy(data_venda,2) in col 10 using '&&&&'

detail
   print data_venda in col 5, vendedor in col 17, valor in col 40 using '#,###,##&.&&'

after periodo
   print 'Subtotal do mes' in col 5, count(*) in col 30 using '#&', ' venda(s)' in col 33, total(valor) in col 43 using '#,###,##&.&&'
   skip

after regiao
   print 'TOTAL ' + regiao in col 1, count(*) in col 30 using '#&', ' vendas' in col 33, total(valor) in col 43 using '#,###,##&.&&'
   print 'Media por venda' in col 3, avg(valor) in col 43 using '#,###,##&.&&'
   print 58[-] in col 1
   skip

after report
   print 'TOTAL GERAL' in col 1, count(*) in col 30 using '#&', ' vendas' in col 33, total(valor) in col 43 using '#,###,##&.&&'
end
```

## Os dados

Em qualquer ordem: o `sort` agrupa e ordena. Campos separados por `|`.

<!-- file: examples/relatorio-vendas-regiao/dados.txt -->
```text
Norte|Bruno Lima|2025-12-11|980.00
Sul|Gina Rocha|2026-02-19|920.00
Sul|Gina Rocha|2026-01-13|310.10
Sul|Ana Souza|2025-12-02|1250.50
Norte|Bruno Lima|2026-02-10|865.30
Norte|Elisa Prado|2026-02-24|495.50
Sul|Ana Souza|2026-02-27|755.45
Leste|Diego Alves|2026-02-26|388.60
Sul|Ana Souza|2026-01-22|1875.00
Sul|Carla Dias|2026-02-05|1640.20
Sul|Carla Dias|2025-12-16|2100.75
Norte|Elisa Prado|2025-12-29|1320.40
Leste|Diego Alves|2025-12-04|450.25
Norte|Bruno Lima|2026-01-06|1450.00
Leste|Helena Costa|2026-02-14|1275.90
Leste|Helena Costa|2026-01-27|640.00
Leste|Diego Alves|2026-01-09|1120.40
Leste|Fabio Neri|2025-12-18|780.00
Norte|Ivo Mendes|2026-02-17|2210.00
Leste|Fabio Neri|2026-02-03|930.15
Norte|Elisa Prado|2026-01-21|710.75
```

## Executar e o resultado

<!-- run -->
```sh
./rptgo examples/relatorio-vendas-regiao/relatorio.rp examples/relatorio-vendas-regiao/dados.txt
```
<!-- expect -->
```text


                     VENDAS POR REGIAO E MES

  REGIAO Leste
  ----------------------------------------------------------
    Mes 12/2025
      04/12/2025  Diego Alves                  450.25
      18/12/2025  Fabio Neri                   780.00
      Subtotal do mes           2  venda(s)     1,230.25

    Mes 01/2026
      09/01/2026  Diego Alves                1,120.40
      27/01/2026  Helena Costa                 640.00
      Subtotal do mes           2  venda(s)     1,760.40

    Mes 02/2026
      03/02/2026  Fabio Neri                   930.15
      14/02/2026  Helena Costa               1,275.90
      26/02/2026  Diego Alves                  388.60
      Subtotal do mes           3  venda(s)     2,594.65

  TOTAL Leste                   7  vendas       5,585.30
    Media por venda                               797.90
  ----------------------------------------------------------

















  REGIAO Norte
  ----------------------------------------------------------
    Mes 12/2025
      11/12/2025  Bruno Lima                   980.00
      29/12/2025  Elisa Prado                1,320.40
      Subtotal do mes           2  venda(s)     2,300.40

    Mes 01/2026
      06/01/2026  Bruno Lima                 1,450.00
      21/01/2026  Elisa Prado                  710.75
      Subtotal do mes           2  venda(s)     2,160.75

    Mes 02/2026
      10/02/2026  Bruno Lima                   865.30
      17/02/2026  Ivo Mendes                 2,210.00
      24/02/2026  Elisa Prado                  495.50
      Subtotal do mes           3  venda(s)     3,570.80

  TOTAL Norte                   7  vendas       8,031.95
    Media por venda                             1,147.42
  ----------------------------------------------------------



















  REGIAO Sul
  ----------------------------------------------------------
    Mes 12/2025
      02/12/2025  Ana Souza                  1,250.50
      16/12/2025  Carla Dias                 2,100.75
      Subtotal do mes           2  venda(s)     3,351.25

    Mes 01/2026
      13/01/2026  Gina Rocha                   310.10
      22/01/2026  Ana Souza                  1,875.00
      Subtotal do mes           2  venda(s)     2,185.10

    Mes 02/2026
      05/02/2026  Carla Dias                 1,640.20
      19/02/2026  Gina Rocha                   920.00
      27/02/2026  Ana Souza                    755.45
      Subtotal do mes           3  venda(s)     3,315.65

  TOTAL Sul                     7  vendas       8,852.00
    Media por venda                             1,264.57
  ----------------------------------------------------------

  TOTAL GERAL                  21  vendas      22,469.25
```

## Os mesmos dados em CSV, JSON e Excel

O script não muda: só a opção `--input`. No CSV, o delimitador (`,`, `;`, tab ou `|`) é detectado
sozinho (o `rptgo` escolhe o que divide as primeiras linhas no número de colunas do `input`), e as
aspas não precisam de opção, pois campos com e sem aspas convivem no mesmo arquivo. No JSON, cada
coluna do `input` busca a chave de mesmo nome.

<!-- file: examples/relatorio-vendas-regiao/dados.csv -->
```text
regiao,vendedor,data_venda,valor
Norte,Bruno Lima,2025-12-11,980.00
Sul,Gina Rocha,2026-02-19,920.00
Sul,Gina Rocha,2026-01-13,310.10
Sul,Ana Souza,2025-12-02,1250.50
Norte,Bruno Lima,2026-02-10,865.30
Norte,Elisa Prado,2026-02-24,495.50
Sul,Ana Souza,2026-02-27,755.45
Leste,Diego Alves,2026-02-26,388.60
Sul,Ana Souza,2026-01-22,1875.00
Sul,Carla Dias,2026-02-05,1640.20
Sul,Carla Dias,2025-12-16,2100.75
Norte,Elisa Prado,2025-12-29,1320.40
Leste,Diego Alves,2025-12-04,450.25
Norte,Bruno Lima,2026-01-06,1450.00
Leste,Helena Costa,2026-02-14,1275.90
Leste,Helena Costa,2026-01-27,640.00
Leste,Diego Alves,2026-01-09,1120.40
Leste,Fabio Neri,2025-12-18,780.00
Norte,Ivo Mendes,2026-02-17,2210.00
Leste,Fabio Neri,2026-02-03,930.15
Norte,Elisa Prado,2026-01-21,710.75
```

<!-- run-only -->
```sh
./rptgo --input csv examples/relatorio-vendas-regiao/relatorio.rp examples/relatorio-vendas-regiao/dados.csv
```

<!-- file: examples/relatorio-vendas-regiao/dados.json -->
```text
[
  {"regiao": "Norte", "vendedor": "Bruno Lima", "data_venda": "2025-12-11", "valor": 980.00},
  {"regiao": "Sul", "vendedor": "Gina Rocha", "data_venda": "2026-02-19", "valor": 920.00},
  {"regiao": "Sul", "vendedor": "Gina Rocha", "data_venda": "2026-01-13", "valor": 310.10},
  {"regiao": "Sul", "vendedor": "Ana Souza", "data_venda": "2025-12-02", "valor": 1250.50},
  {"regiao": "Norte", "vendedor": "Bruno Lima", "data_venda": "2026-02-10", "valor": 865.30},
  {"regiao": "Norte", "vendedor": "Elisa Prado", "data_venda": "2026-02-24", "valor": 495.50},
  {"regiao": "Sul", "vendedor": "Ana Souza", "data_venda": "2026-02-27", "valor": 755.45},
  {"regiao": "Leste", "vendedor": "Diego Alves", "data_venda": "2026-02-26", "valor": 388.60},
  {"regiao": "Sul", "vendedor": "Ana Souza", "data_venda": "2026-01-22", "valor": 1875.00},
  {"regiao": "Sul", "vendedor": "Carla Dias", "data_venda": "2026-02-05", "valor": 1640.20},
  {"regiao": "Sul", "vendedor": "Carla Dias", "data_venda": "2025-12-16", "valor": 2100.75},
  {"regiao": "Norte", "vendedor": "Elisa Prado", "data_venda": "2025-12-29", "valor": 1320.40},
  {"regiao": "Leste", "vendedor": "Diego Alves", "data_venda": "2025-12-04", "valor": 450.25},
  {"regiao": "Norte", "vendedor": "Bruno Lima", "data_venda": "2026-01-06", "valor": 1450.00},
  {"regiao": "Leste", "vendedor": "Helena Costa", "data_venda": "2026-02-14", "valor": 1275.90},
  {"regiao": "Leste", "vendedor": "Helena Costa", "data_venda": "2026-01-27", "valor": 640.00},
  {"regiao": "Leste", "vendedor": "Diego Alves", "data_venda": "2026-01-09", "valor": 1120.40},
  {"regiao": "Leste", "vendedor": "Fabio Neri", "data_venda": "2025-12-18", "valor": 780.00},
  {"regiao": "Norte", "vendedor": "Ivo Mendes", "data_venda": "2026-02-17", "valor": 2210.00},
  {"regiao": "Leste", "vendedor": "Fabio Neri", "data_venda": "2026-02-03", "valor": 930.15},
  {"regiao": "Norte", "vendedor": "Elisa Prado", "data_venda": "2026-01-21", "valor": 710.75}
]
```

<!-- run-only -->
```sh
./rptgo --input json examples/relatorio-vendas-regiao/relatorio.rp examples/relatorio-vendas-regiao/dados.json
```

Os dois produzem exatamente o mesmo relatório que o `dados.txt`.

### O CSV exportado pelo Excel brasileiro

Usa `;` como delimitador, vírgula decimal (`1.250,50`) e datas `DD/MM/AAAA`. O RPT original só lê o
ponto como separador decimal na entrada, então `--input-decimal` é uma extensão do `rptgo`, desligada
por padrão (`auto` a liga só se os valores tiverem a forma `1250,50`). O arquivo antigo "CSV (separado
por vírgulas)" grava em Windows-1252: nesse caso junte `--encoding latin-1`.
[`dados-excel-br.csv`](dados-excel-br.csv):

```text
regiao;vendedor;data_venda;valor
Norte;Bruno Lima;11/12/2025;980,00
Sul;Gina Rocha;19/02/2026;920,00
Sul;Gina Rocha;13/01/2026;310,10
Sul;Ana Souza;02/12/2025;1.250,50
Norte;Bruno Lima;10/02/2026;865,30
Norte;Elisa Prado;24/02/2026;495,50
Sul;Ana Souza;27/02/2026;755,45
Leste;Diego Alves;26/02/2026;388,60
Sul;Ana Souza;22/01/2026;1.875,00
Sul;Carla Dias;05/02/2026;1.640,20
Sul;Carla Dias;16/12/2025;2.100,75
Norte;Elisa Prado;29/12/2025;1.320,40
Leste;Diego Alves;04/12/2025;450,25
Norte;Bruno Lima;06/01/2026;1.450,00
Leste;Helena Costa;14/02/2026;1.275,90
Leste;Helena Costa;27/01/2026;640,00
Leste;Diego Alves;09/01/2026;1.120,40
Leste;Fabio Neri;18/12/2025;780,00
Norte;Ivo Mendes;17/02/2026;2.210,00
Leste;Fabio Neri;03/02/2026;930,15
Norte;Elisa Prado;21/01/2026;710,75
```

<!-- run -->
```sh
./rptgo --input csv --input-decimal auto examples/relatorio-vendas-regiao/relatorio.rp examples/relatorio-vendas-regiao/dados-excel-br.csv
```
<!-- expect -->
```text


                     VENDAS POR REGIAO E MES

  REGIAO Leste
  ----------------------------------------------------------
    Mes 12/2025
      04/12/2025  Diego Alves                  450.25
      18/12/2025  Fabio Neri                   780.00
      Subtotal do mes           2  venda(s)     1,230.25

    Mes 01/2026
      09/01/2026  Diego Alves                1,120.40
      27/01/2026  Helena Costa                 640.00
      Subtotal do mes           2  venda(s)     1,760.40

    Mes 02/2026
      03/02/2026  Fabio Neri                   930.15
      14/02/2026  Helena Costa               1,275.90
      26/02/2026  Diego Alves                  388.60
      Subtotal do mes           3  venda(s)     2,594.65

  TOTAL Leste                   7  vendas       5,585.30
    Media por venda                               797.90
  ----------------------------------------------------------

















  REGIAO Norte
  ----------------------------------------------------------
    Mes 12/2025
      11/12/2025  Bruno Lima                   980.00
      29/12/2025  Elisa Prado                1,320.40
      Subtotal do mes           2  venda(s)     2,300.40

    Mes 01/2026
      06/01/2026  Bruno Lima                 1,450.00
      21/01/2026  Elisa Prado                  710.75
      Subtotal do mes           2  venda(s)     2,160.75

    Mes 02/2026
      10/02/2026  Bruno Lima                   865.30
      17/02/2026  Ivo Mendes                 2,210.00
      24/02/2026  Elisa Prado                  495.50
      Subtotal do mes           3  venda(s)     3,570.80

  TOTAL Norte                   7  vendas       8,031.95
    Media por venda                             1,147.42
  ----------------------------------------------------------



















  REGIAO Sul
  ----------------------------------------------------------
    Mes 12/2025
      02/12/2025  Ana Souza                  1,250.50
      16/12/2025  Carla Dias                 2,100.75
      Subtotal do mes           2  venda(s)     3,351.25

    Mes 01/2026
      13/01/2026  Gina Rocha                   310.10
      22/01/2026  Ana Souza                  1,875.00
      Subtotal do mes           2  venda(s)     2,185.10

    Mes 02/2026
      05/02/2026  Carla Dias                 1,640.20
      19/02/2026  Gina Rocha                   920.00
      27/02/2026  Ana Souza                    755.45
      Subtotal do mes           3  venda(s)     3,315.65

  TOTAL Sul                     7  vendas       8,852.00
    Media por venda                             1,264.57
  ----------------------------------------------------------

  TOTAL GERAL                  21  vendas      22,469.25
```

## Exportar os dados (CSV e JSON)

`--format csv` e `--format json` ignoram o layout e entregam só os dados, na ordem do `sort`, com datas
em `AAAA-MM-DD`:

<!-- run -->
```sh
./rptgo --format csv examples/relatorio-vendas-regiao/relatorio.rp examples/relatorio-vendas-regiao/dados.txt
```
<!-- expect -->
```text
regiao,vendedor,data_venda,valor
Leste,Diego Alves,2025-12-04,450.25
Leste,Fabio Neri,2025-12-18,780
Leste,Diego Alves,2026-01-09,1120.4
Leste,Helena Costa,2026-01-27,640
Leste,Fabio Neri,2026-02-03,930.15
Leste,Helena Costa,2026-02-14,1275.9
Leste,Diego Alves,2026-02-26,388.6
Norte,Bruno Lima,2025-12-11,980
Norte,Elisa Prado,2025-12-29,1320.4
Norte,Bruno Lima,2026-01-06,1450
Norte,Elisa Prado,2026-01-21,710.75
Norte,Bruno Lima,2026-02-10,865.3
Norte,Ivo Mendes,2026-02-17,2210
Norte,Elisa Prado,2026-02-24,495.5
Sul,Ana Souza,2025-12-02,1250.5
Sul,Carla Dias,2025-12-16,2100.75
Sul,Gina Rocha,2026-01-13,310.1
Sul,Ana Souza,2026-01-22,1875
Sul,Carla Dias,2026-02-05,1640.2
Sul,Gina Rocha,2026-02-19,920
Sul,Ana Souza,2026-02-27,755.45
```

<!-- run -->
```sh
./rptgo --format json examples/relatorio-vendas-regiao/relatorio.rp examples/relatorio-vendas-regiao/dados.txt
```
<!-- expect -->
```text
[
{"regiao":"Leste","vendedor":"Diego Alves","data_venda":"2025-12-04","valor":450.25},
{"regiao":"Leste","vendedor":"Fabio Neri","data_venda":"2025-12-18","valor":780},
{"regiao":"Leste","vendedor":"Diego Alves","data_venda":"2026-01-09","valor":1120.4},
{"regiao":"Leste","vendedor":"Helena Costa","data_venda":"2026-01-27","valor":640},
{"regiao":"Leste","vendedor":"Fabio Neri","data_venda":"2026-02-03","valor":930.15},
{"regiao":"Leste","vendedor":"Helena Costa","data_venda":"2026-02-14","valor":1275.9},
{"regiao":"Leste","vendedor":"Diego Alves","data_venda":"2026-02-26","valor":388.6},
{"regiao":"Norte","vendedor":"Bruno Lima","data_venda":"2025-12-11","valor":980},
{"regiao":"Norte","vendedor":"Elisa Prado","data_venda":"2025-12-29","valor":1320.4},
{"regiao":"Norte","vendedor":"Bruno Lima","data_venda":"2026-01-06","valor":1450},
{"regiao":"Norte","vendedor":"Elisa Prado","data_venda":"2026-01-21","valor":710.75},
{"regiao":"Norte","vendedor":"Bruno Lima","data_venda":"2026-02-10","valor":865.3},
{"regiao":"Norte","vendedor":"Ivo Mendes","data_venda":"2026-02-17","valor":2210},
{"regiao":"Norte","vendedor":"Elisa Prado","data_venda":"2026-02-24","valor":495.5},
{"regiao":"Sul","vendedor":"Ana Souza","data_venda":"2025-12-02","valor":1250.5},
{"regiao":"Sul","vendedor":"Carla Dias","data_venda":"2025-12-16","valor":2100.75},
{"regiao":"Sul","vendedor":"Gina Rocha","data_venda":"2026-01-13","valor":310.1},
{"regiao":"Sul","vendedor":"Ana Souza","data_venda":"2026-01-22","valor":1875},
{"regiao":"Sul","vendedor":"Carla Dias","data_venda":"2026-02-05","valor":1640.2},
{"regiao":"Sul","vendedor":"Gina Rocha","data_venda":"2026-02-19","valor":920},
{"regiao":"Sul","vendedor":"Ana Souza","data_venda":"2026-02-27","valor":755.45}
]
```

## Direto de um banco, sem driver

Os clientes de linha de comando já exportam em texto, e o `rptgo` lê da entrada padrão (`-`). O padrão do
`psql -A` já usa `|`:

```sh
psql -At -F'|' -c "select regiao, vendedor, data_venda, valor from vendas" | ./rptgo relatorio.rp -
sqlite3 -separator '|' vendas.db "select regiao, vendedor, data_venda, valor from vendas" | ./rptgo relatorio.rp -
mysql -B -e "select regiao, vendedor, data_venda, valor from vendas" | ./rptgo --input csv relatorio.rp -
```

## Desempenho das fontes

Com 200.000 registros, só a leitura leva cerca de 40 ms no texto delimitado, 100 ms no CSV e 700 ms no
JSON. Como a ordenação, as quebras e a paginação dominam, o relatório completo fica em torno de 2,1 s
com texto, 2,3 s com CSV e 3,0 s com JSON: o JSON custa uns 40% a mais no total, não uma ordem de
grandeza. Para repetir a medição: `go test -run '^$' -bench 'Entrada|Relatorio' -benchmem ./cmd/rptgo`.
