# Contas a receber: prazos e faixas de atraso

[← Exemplos](../../README.md#exemplos) · [Referência da linguagem](../../REFERENCIA.md)

Faturas em aberto por cliente, com o prazo em dias (`due - issued`, aritmética de datas), uma faixa de
atraso (`current`, `overdue`, `OVERDUE 30+`) e o saldo do cliente. Mostra `not paid` dentro do `where`
dos agregados e `print ... no newline` para o status ficar na mesma linha. A data de referência é fixa
no script, e por isso ele só funciona com o `DATEFMT` padrão (`DD/MM/YY`).

## O script

<!-- file: examples/invoice-aging/report.rp -->
```text
/* Accounts receivable: open invoices per customer with payment terms in days
   (due - issued) and a flag for invoices that are 30 days or more past due at the
   as-of date. The as-of date is a constant so the output never changes. Date literals follow
   DATEFMT, so run it with the default DD/MM/YY (or adjust the literal). */
input
   customer  [string 16],
   invoice   [string 8],
   issued    [ldate 10],
   due       [ldate 10],
   amount    [amount 10],
   paid      [bool]

width 74
length 40
sort customer

before report
   print 'OPEN INVOICES (AS OF 2026-06-30)' centered
   print 'Invoice' in col 1, 'Issued' in col 12, 'Due' in col 24, 'Terms' in col 36, 'Amount' in col 49, 'Status' in col 62
   print 68[-] in col 1

before customer
   print customer in col 1

detail
   if not paid then
      begin
         print invoice in col 3, issued in col 12, due in col 24, due - issued in col 36 using '####&', amount in col 44 using '###,##&.&&' no newline
         if 30/06/2026 - due >= 30 then print 'OVERDUE 30+' in col 62 else if 30/06/2026 - due > 0 then print 'overdue' in col 62 else print 'current' in col 62
      end

after customer
   print 'Open balance:' in col 3, total(amount where not paid) in col 44 using '###,##&.&&'
   print 'Open invoices:' in col 3, count(* where not paid) in col 44 using '#####&'
   skip

after report
   print 'TOTAL OPEN' in col 1, total(amount where not paid) in col 44 using '###,##&.&&'
end
```

## Os dados

<!-- file: examples/invoice-aging/data.txt -->
```text
Acme Corp|INV-1001|2026-03-01|2026-03-31|12500.00|yes
Acme Corp|INV-1042|2026-04-15|2026-06-14|8300.50|no
Acme Corp|INV-1077|2026-06-10|2026-07-10|4100.00|no
Globex Ltd|INV-1010|2026-02-01|2026-03-03|22000.00|no
Globex Ltd|INV-1065|2026-05-20|2026-06-19|1500.75|no
Initech|INV-1029|2026-04-02|2026-05-02|9800.00|yes
Initech|INV-1081|2026-06-20|2026-08-04|2750.00|no
Umbrella Inc|INV-1055|2026-05-05|2026-06-04|640.25|no
```

## Executar e o resultado

<!-- run -->
```sh
./rptgo examples/invoice-aging/report.rp examples/invoice-aging/data.txt
```
<!-- expect -->
```text


                      OPEN INVOICES (AS OF 2026-06-30)
  Invoice    Issued      Due         Terms        Amount       Status
  --------------------------------------------------------------------
  Acme Corp
    INV-1042 15/04/2026  14/06/2026     60     8,300.50        overdue
    INV-1077 10/06/2026  10/07/2026     30     4,100.00        current
    Open balance:                             12,400.50
    Open invoices:                                2

  Globex Ltd
    INV-1010 01/02/2026  03/03/2026     30    22,000.00        OVERDUE 30+
    INV-1065 20/05/2026  19/06/2026     30     1,500.75        overdue
    Open balance:                             23,500.75
    Open invoices:                                2

  Initech
    INV-1081 20/06/2026  04/08/2026     45     2,750.00        current
    Open balance:                              2,750.00
    Open invoices:                                1

  Umbrella Inc
    INV-1055 05/05/2026  04/06/2026     30       640.25        overdue
    Open balance:                                640.25
    Open invoices:                                1

  TOTAL OPEN                                  39,291.50
```

