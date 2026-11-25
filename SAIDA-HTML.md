# Saída HTML e caixas

[← README](README.md) · [Referência da linguagem](REFERENCIA.md) · [Exemplo da fatura](examples/invoice-html/README.md)

`--format html` junta as páginas do relatório a um modelo HTML, para imprimir numa jato de tinta
ou salvar em PDF pelo navegador, sem mudar o script `.rp`. O RPT continua cuidando da
ordenação, das quebras, dos totais e do alinhamento; o modelo só cuida da moldura (logo,
cabeçalho, rodapé, cores).

```
./rptgo --format html --box --template examples/invoice-html/template.html \
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
