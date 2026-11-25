# Feriados do ano, com um único curl

[← Exemplos](../../README.md#exemplos) · [Referência da linguagem](../../REFERENCIA.md)

O JSON da resposta de uma API é o dado: uma única chamada à BrasilAPI traz os feriados nacionais do
ano, e o `rptgo` lê da entrada padrão (`-`), sem arquivo intermediário e sem `jq`:

```sh
curl -s https://brasilapi.com.br/api/feriados/v1/2026 | ./rptgo --input json examples/feriados-brasil/relatorio.rp -
```

A resposta é um array de objetos (`date`, `name`, `type`, `weekday`), e o `input` do script só escolhe
as chaves de que precisa. O script agrupa por mês com uma chave calculada, escreve o nome do mês com
`index(...)` e marca, com `dow` e `or`, os feriados que caem no fim de semana, os que prolongam o fim
de semana e os que permitem emenda.

## O script

<!-- file: examples/feriados-brasil/relatorio.rp -->
```text
/* Feriados nacionais do ano, por mes, a partir de UMA chamada a uma API:
     curl -s https://brasilapi.com.br/api/feriados/v1/2026 | rpt --input json relatorio.rp -
   O JSON da resposta e o dado: o input so escolhe as chaves que usa (date, name, type e
   weekday). A chave do mes e calculada (<mes>) e da nome as secoes before/after. */
input
   date     [ldate 10],
   name     [string 40],
   type     [string 12],
   weekday  [string 14]

width 78
length 60
sort <mes> mdy(date,0), date

before report
   print 'FERIADOS NACIONAIS' in col 1, mdy(date,2) in col 20 using '&&&&'
   print 'Quando cai no fim de semana, o feriado nao gera folga.' in col 1
   skip

before mes
   print index(mdy(date,0) - 1, 'Janeiro', 'Fevereiro', 'Marco', 'Abril', 'Maio', 'Junho', 'Julho', 'Agosto', 'Setembro', 'Outubro', 'Novembro', 'Dezembro') in col 1
   print 70[-] in col 1

detail
   print date in col 3, weekday in col 15, name in col 31
   if dow(date) = 0 or dow(date) = 6 then
      print '(cai no fim de semana)' in col 31
   else if dow(date) = 1 or dow(date) = 5 then
      print '(fim de semana prolongado)' in col 31
   else if dow(date) = 2 or dow(date) = 4 then
      print '(emenda possivel)' in col 31

after mes
   print 'Feriados no mes:' in col 3, count(*) in col 31 using '#&'
   skip

after report
   print 'Total no ano:' in col 3, count(*) in col 31 using '#&'
   print 'No fim de semana:' in col 3, count(* where dow(date) = 0 or dow(date) = 6) in col 31 using '#&'
end
```

## A resposta da API (amostra)

O arquivo [`dados-2026.json`](dados-2026.json) guarda a resposta (um objeto por linha) e dá exatamente
o mesmo relatório, sem rede:

<!-- file: examples/feriados-brasil/dados-2026.json -->
```text
[
  {"date":"2026-01-01","name":"Confraternização mundial","type":"national","weekday":"quinta-feira"},
  {"date":"2026-02-16","name":"Carnaval","type":"national","weekday":"segunda-feira"},
  {"date":"2026-02-17","name":"Carnaval","type":"national","weekday":"terça-feira"},
  {"date":"2026-04-03","name":"Sexta-feira Santa","type":"national","weekday":"sexta-feira"},
  {"date":"2026-04-05","name":"Páscoa","type":"national","weekday":"domingo"},
  {"date":"2026-04-21","name":"Tiradentes","type":"national","weekday":"terça-feira"},
  {"date":"2026-05-01","name":"Dia do trabalho","type":"national","weekday":"sexta-feira"},
  {"date":"2026-06-04","name":"Corpus Christi","type":"national","weekday":"quinta-feira"},
  {"date":"2026-09-07","name":"Independência do Brasil","type":"national","weekday":"segunda-feira"},
  {"date":"2026-10-12","name":"Nossa Senhora Aparecida","type":"national","weekday":"segunda-feira"},
  {"date":"2026-11-02","name":"Finados","type":"national","weekday":"segunda-feira"},
  {"date":"2026-11-15","name":"Proclamação da República","type":"national","weekday":"domingo"},
  {"date":"2026-11-20","name":"Dia da consciência negra","type":"national","weekday":"sexta-feira"},
  {"date":"2026-12-25","name":"Natal","type":"national","weekday":"sexta-feira"}
]
```

## Executar e o resultado

<!-- run -->
```sh
./rptgo --input json examples/feriados-brasil/relatorio.rp examples/feriados-brasil/dados-2026.json
```
<!-- expect -->
```text


  FERIADOS NACIONAIS 2026
  Quando cai no fim de semana, o feriado nao gera folga.

  Janeiro
  ----------------------------------------------------------------------
    01/01/2026  quinta-feira    Confraternização mundial
                                (emenda possivel)
    Feriados no mes:             1

  Fevereiro
  ----------------------------------------------------------------------
    16/02/2026  segunda-feira   Carnaval
                                (fim de semana prolongado)
    17/02/2026  terça-feira     Carnaval
                                (emenda possivel)
    Feriados no mes:             2

  Abril
  ----------------------------------------------------------------------
    03/04/2026  sexta-feira     Sexta-feira Santa
                                (fim de semana prolongado)
    05/04/2026  domingo         Páscoa
                                (cai no fim de semana)
    21/04/2026  terça-feira     Tiradentes
                                (emenda possivel)
    Feriados no mes:             3

  Maio
  ----------------------------------------------------------------------
    01/05/2026  sexta-feira     Dia do trabalho
                                (fim de semana prolongado)
    Feriados no mes:             1

  Junho
  ----------------------------------------------------------------------
    04/06/2026  quinta-feira    Corpus Christi
                                (emenda possivel)
    Feriados no mes:             1

  Setembro
  ----------------------------------------------------------------------
    07/09/2026  segunda-feira   Independência do Brasil
                                (fim de semana prolongado)
    Feriados no mes:             1

  Outubro
  ----------------------------------------------------------------------
    12/10/2026  segunda-feira   Nossa Senhora Aparecida
                                (fim de semana prolongado)
    Feriados no mes:             1

  Novembro
  ----------------------------------------------------------------------
    02/11/2026  segunda-feira   Finados
                                (fim de semana prolongado)
    15/11/2026  domingo         Proclamação da República




                                (cai no fim de semana)
    20/11/2026  sexta-feira     Dia da consciência negra
                                (fim de semana prolongado)
    Feriados no mes:             3

  Dezembro
  ----------------------------------------------------------------------
    25/12/2026  sexta-feira     Natal
                                (fim de semana prolongado)
    Feriados no mes:             1

    Total no ano:               14
    No fim de semana:            2
```

O [`run.sh`](run.sh) faz a chamada de verdade (`./run.sh 2027` para outro ano).
