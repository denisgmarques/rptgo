# Estoque por categoria e subcategoria

[← Exemplos](../../README.md#exemplos) · [Referência da linguagem](../../REFERENCIA.md)

Duas quebras em cascata (`categoria`, depois `subcategoria`), cada uma com a sua contagem, o seu
total e, na subcategoria, a média. Mostra `count`, `total` e `avg` em níveis diferentes, expressões
dentro das funções de agregação (`total(quantidade * preco)`) e o total geral.

## O script

<!-- file: examples/relatorio-estoque-categoria/relatorio.rp -->
```text
/* Estoque por categoria e subcategoria: dois niveis de quebra em cascata.
   Cada nivel fecha com contagem, total e media do valor em estoque. */
input
   categoria     [string 12],
   subcategoria  [string 14],
   produto       [string 20],
   quantidade    [numeric 6],
   preco         [amount 10]

width 72
length 60
sort categoria, subcategoria

before report
   print 'ESTOQUE POR CATEGORIA' centered
   skip

before categoria
   print 'Categoria: ' + categoria in col 1

before subcategoria
   print 'Subcategoria: ' + subcategoria in col 3

detail
   print produto in col 5, quantidade in col 24 using '#####&', preco in col 32 using '###,##&.&&', quantidade * preco in col 46 using '#,###,##&.&&'

after subcategoria
   print 'Produtos:' in col 5, count(*) in col 24 using '#####&', 'Valor:' in col 32, total(quantidade * preco) in col 46 using '#,###,##&.&&'
   print 'Preco medio:' in col 5, avg(preco) in col 32 using '###,##&.&&'

after categoria
   print 'Itens na categoria:' in col 3, count(*) in col 24 using '#####&', total(quantidade * preco) in col 46 using '#,###,##&.&&'
   skip

after report
   print 'TOTAL EM ESTOQUE' in col 1, total(quantidade * preco) in col 46 using '#,###,##&.&&'
end
```

## Os dados

<!-- file: examples/relatorio-estoque-categoria/dados.txt -->
```text
Bebidas|Sucos|Suco de laranja|120|4.50
Bebidas|Sucos|Suco de uva|80|5.20
Bebidas|Refrigerantes|Cola 2L|200|7.90
Bebidas|Refrigerantes|Guarana 2L|150|6.80
Limpeza|Detergentes|Detergente neutro|300|2.30
Limpeza|Detergentes|Lava-loucas|90|8.75
Limpeza|Desinfetantes|Desinfetante pinho|60|9.40
Mercearia|Graos|Arroz 5kg|75|24.90
Mercearia|Graos|Feijao 1kg|110|8.60
Mercearia|Massas|Macarrao 500g|180|4.10
```

## Executar e o resultado

<!-- run -->
```sh
./rptgo examples/relatorio-estoque-categoria/relatorio.rp examples/relatorio-estoque-categoria/dados.txt
```
<!-- expect -->
```text


                           ESTOQUE POR CATEGORIA

  Categoria: Bebidas
    Subcategoria: Refrigerantes
      Cola 2L               200        7.90        1,580.00
      Guarana 2L            150        6.80        1,020.00
      Produtos:               2  Valor:            2,600.00
      Preco medio:                     7.35
    Subcategoria: Sucos
      Suco de laranja       120        4.50          540.00
      Suco de uva            80        5.20          416.00
      Produtos:               2  Valor:              956.00
      Preco medio:                     4.85
    Itens na categoria:       4                    3,556.00

  Categoria: Limpeza
    Subcategoria: Desinfetantes
      Desinfetante pinho     60        9.40          564.00
      Produtos:               1  Valor:              564.00
      Preco medio:                     9.40
    Subcategoria: Detergentes
      Detergente neutro     300        2.30          690.00
      Lava-loucas            90        8.75          787.50
      Produtos:               2  Valor:            1,477.50
      Preco medio:                     5.53
    Itens na categoria:       3                    2,041.50

  Categoria: Mercearia
    Subcategoria: Graos
      Arroz 5kg              75       24.90        1,867.50
      Feijao 1kg            110        8.60          946.00
      Produtos:               2  Valor:            2,813.50
      Preco medio:                    16.75
    Subcategoria: Massas
      Macarrao 500g         180        4.10          738.00
      Produtos:               1  Valor:              738.00
      Preco medio:                     4.10
    Itens na categoria:       3                    3,551.50

  TOTAL EM ESTOQUE                                 9,149.00
```

