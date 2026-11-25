# Cadastro de clientes: datas, bool e nulos

[← Exemplos](../../README.md#exemplos) · [Referência da linguagem](../../REFERENCIA.md)

Mostra como cada tipo é impresso e exportado: datas (`date` com ano de 2 dígitos e `ldate` com 4),
a coluna `bool` (`YES`/`NO`, e `***` quando nula) e campos vazios (nulos). O mesmo script gera o
relatório, o JSON e o CSV.

## O script

<!-- file: examples/cadastro-clientes/relatorio.rp -->
```text
/* Cadastro de clientes: mostra datas, bool e valores nulos.
   Em texto e um relatorio simples; com --format json/csv as datas saem como AAAA-MM-DD,
   bool como true/false e campos vazios como null (JSON) ou vazio (CSV). */
input
   id          [numeric 4],
   nome        [string 16],
   nascimento  [date 10],
   cadastro    [ldate 10],
   ativo       [bool],
   limite      [amount 10]

width 70
length 20
sort id

before report
   print 'CADASTRO DE CLIENTES' centered
   print 'Id' in col 1, 'Nome' in col 6, 'Nasc.' in col 24, 'Cadastro' in col 34, 'Ativo' in col 46, 'Limite' in col 56
   print 68[-] in col 1

detail
   print id in col 1, nome in col 6, nascimento in col 24, cadastro in col 34, ativo in col 46, limite in col 54 using '###,##&.&&'

after report
   print 'Clientes ativos:' in col 1, count(* where ativo) in col 20 using '###&'
end
```

## Os dados

<!-- file: examples/cadastro-clientes/dados.txt -->
```text
1|Ana Souza|15/03/1985|2020-01-15|1|1500.00
2|Bruno Lima|02/11/1990|2019-07-04|0|0
3|Carla Dias||2021-12-31|yes|2500.50
4|Diego "Dd" Alves|30/06/1978|2018-02-28||
5|Elisa, Prado|09/09/2001|2022-05-05|no|800.25
```

## O relatório

<!-- run -->
```sh
./rptgo examples/cadastro-clientes/relatorio.rp examples/cadastro-clientes/dados.txt
```
<!-- expect -->
```text


                          CADASTRO DE CLIENTES
  Id   Nome              Nasc.     Cadastro    Ativo     Limite
  --------------------------------------------------------------------
  1    Ana Souza         15/03/85  15/01/2020  YES       1,500.00
  2    Bruno Lima        02/11/90  04/07/2019  NO            0.00
  3    Carla Dias                  31/12/2021  YES       2,500.50
  4    Diego "Dd" Alves  30/06/78  28/02/2018  ***
  5    Elisa, Prado      09/09/01  05/05/2022  NO          800.25
  Clientes ativos:      2
```

## Os mesmos dados em JSON e em CSV

`--format json` e `--format csv` ignoram o layout e entregam só os dados, na ordem do `sort`: datas em
`AAAA-MM-DD`, `bool` como `true`/`false`, nulos como `null` (JSON) ou vazio (CSV), e campos com vírgula
ou aspas entre aspas no CSV.

<!-- run -->
```sh
./rptgo --format json examples/cadastro-clientes/relatorio.rp examples/cadastro-clientes/dados.txt
```
<!-- expect -->
```text
[
{"id":1,"nome":"Ana Souza","nascimento":"1985-03-15","cadastro":"2020-01-15","ativo":true,"limite":1500},
{"id":2,"nome":"Bruno Lima","nascimento":"1990-11-02","cadastro":"2019-07-04","ativo":false,"limite":0},
{"id":3,"nome":"Carla Dias","nascimento":null,"cadastro":"2021-12-31","ativo":true,"limite":2500.5},
{"id":4,"nome":"Diego \"Dd\" Alves","nascimento":"1978-06-30","cadastro":"2018-02-28","ativo":null,"limite":null},
{"id":5,"nome":"Elisa, Prado","nascimento":"2001-09-09","cadastro":"2022-05-05","ativo":false,"limite":800.25}
]
```

<!-- run -->
```sh
./rptgo --format csv examples/cadastro-clientes/relatorio.rp examples/cadastro-clientes/dados.txt
```
<!-- expect -->
```text
id,nome,nascimento,cadastro,ativo,limite
1,Ana Souza,1985-03-15,2020-01-15,true,1500
2,Bruno Lima,1990-11-02,2019-07-04,false,0
3,Carla Dias,,2021-12-31,true,2500.5
4,"Diego ""Dd"" Alves",1978-06-30,2018-02-28,,
5,"Elisa, Prado",2001-09-09,2022-05-05,false,800.25
```

