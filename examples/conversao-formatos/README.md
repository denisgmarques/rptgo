# Converter entre texto delimitado, CSV e JSON

[← Exemplos](../../README.md#exemplos) · [Referência da linguagem](../../REFERENCIA.md)

O `rptgo` também serve de conversor de formatos: `--input` diz como ler os dados e `--format` como
escrevê-los. Não há layout para descrever, mas o **script é obrigatório**: ele traz a lista de
colunas (`input`) e termina com `end`. Ele pode ser escrito na própria linha de comando, [sem criar
arquivo](#sem-criar-o-arquivo-rp).

| De | Para | Opções |
|---|---|---|
| texto delimitado | CSV | `--format csv` |
| texto delimitado | JSON | `--format json` |
| CSV | JSON | `--input csv --format json` |
| JSON | CSV | `--input json --format csv` |
| CSV ou JSON | texto delimitado | `--input csv` ou `json`, com `--format csv --csv-delimiter '\|' --csv-no-header` |

## O script

Basta declarar as colunas:

<!-- file: examples/conversao-formatos/colunas.rp -->
```text
/* Conversao entre formatos: so a lista de colunas, sem layout. */
input
   id       [numeric],
   nome     [string],
   cidade   [string],
   cadastro [ldate],
   ativo    [bool],
   limite   [float]
end
```

O tipo de cada coluna decide como o valor sai:

| Tipo no `input` | Entrada | No JSON | No CSV |
|---|---|---|---|
| `string` (ou sem tipo) | `0012` | `"0012"` | `0012` |
| `numeric` (inteiro: a fração é descartada) | `42` | `42` | `42` |
| `float` | `1500.50` | `1500.5` | `1500.5` |
| `date`, `ldate` | `15/01/2026` | `"2026-01-15"` | `2026-01-15` |
| `bool` | `yes` | `true` | `true` |
| qualquer um, com o campo vazio | | `null` | vazio |

Declare como `string` o que deve sair igual ao que entrou (códigos com zeros à esquerda, CEP, telefone)
e como `float` os valores com casas decimais. Um valor inválido para o tipo, como `abc` numa coluna
numérica, interrompe a conversão com erro.

## De texto delimitado para CSV e JSON

Um registro por linha, com os campos separados por `|`:

<!-- file: examples/conversao-formatos/clientes.txt -->
```text
1|Ana Souza|Curitiba|15/01/2026|yes|1500.50
2|Bruno Lima|São Paulo|04/07/2025|no|-250
3|Carla Dias|Recife, PE||yes|
```

<!-- run -->
```sh
./rptgo --format csv \
  examples/conversao-formatos/colunas.rp examples/conversao-formatos/clientes.txt
```
<!-- expect -->
```text
id,nome,cidade,cadastro,ativo,limite
1,Ana Souza,Curitiba,2026-01-15,true,1500.5
2,Bruno Lima,São Paulo,2025-07-04,false,-250
3,Carla Dias,"Recife, PE",,true,
```

<!-- run -->
```sh
./rptgo --format json \
  examples/conversao-formatos/colunas.rp examples/conversao-formatos/clientes.txt
```
<!-- expect -->
```text
[
{"id":1,"nome":"Ana Souza","cidade":"Curitiba","cadastro":"2026-01-15","ativo":true,"limite":1500.5},
{"id":2,"nome":"Bruno Lima","cidade":"São Paulo","cadastro":"2025-07-04","ativo":false,"limite":-250},
{"id":3,"nome":"Carla Dias","cidade":"Recife, PE","cadastro":null,"ativo":true,"limite":null}
]
```

Para outro separador, use a instrução `separator ';'` no script ou a variável de ambiente
`SEPARATOR=';'`. O texto delimitado não interpreta aspas: se os campos vêm entre aspas, leia com
`--input csv --input-delimiter` e o caractere.

## De CSV para JSON

<!-- file: examples/conversao-formatos/clientes.csv -->
```csv
id,nome,cidade,cadastro,ativo,limite
1,Ana Souza,Curitiba,2026-01-15,true,1500.5
2,Bruno Lima,São Paulo,2025-07-04,false,-250
3,Carla Dias,"Recife, PE",,true,
```

<!-- run -->
```sh
./rptgo --input csv --format json \
  examples/conversao-formatos/colunas.rp examples/conversao-formatos/clientes.csv
```
<!-- expect -->
```text
[
{"id":1,"nome":"Ana Souza","cidade":"Curitiba","cadastro":"2026-01-15","ativo":true,"limite":1500.5},
{"id":2,"nome":"Bruno Lima","cidade":"São Paulo","cadastro":"2025-07-04","ativo":false,"limite":-250},
{"id":3,"nome":"Carla Dias","cidade":"Recife, PE","cadastro":null,"ativo":true,"limite":null}
]
```

O delimitador (`;`, `,`, tab ou `|`) é detectado; `--input-delimiter` força um. A primeira linha é
tratada como cabeçalho quando o primeiro campo é o nome da primeira coluna do `input`; se não for,
use `--header sim`.

## De JSON para CSV

A entrada é um array de objetos, ou um objeto por linha:

<!-- file: examples/conversao-formatos/clientes.json -->
```json
[
{"id":1,"nome":"Ana Souza","cidade":"Curitiba","cadastro":"2026-01-15","ativo":true,"limite":1500.5},
{"id":2,"nome":"Bruno Lima","cidade":"São Paulo","cadastro":"2025-07-04","ativo":false,"limite":-250},
{"id":3,"nome":"Carla Dias","cidade":"Recife, PE","cadastro":null,"ativo":true,"limite":null}
]
```

<!-- run -->
```sh
./rptgo --input json --format csv \
  examples/conversao-formatos/colunas.rp examples/conversao-formatos/clientes.json
```
<!-- expect -->
```text
id,nome,cidade,cadastro,ativo,limite
1,Ana Souza,Curitiba,2026-01-15,true,1500.5
2,Bruno Lima,São Paulo,2025-07-04,false,-250
3,Carla Dias,"Recife, PE",,true,
```

## De volta para texto delimitado

O texto delimitado é um CSV com outro delimitador e sem cabeçalho:

<!-- run -->
```sh
./rptgo --input json --format csv --csv-delimiter '|' --csv-no-header \
  examples/conversao-formatos/colunas.rp examples/conversao-formatos/clientes.json
```
<!-- expect -->
```text
1|Ana Souza|Curitiba|2026-01-15|true|1500.5
2|Bruno Lima|São Paulo|2025-07-04|false|-250
3|Carla Dias|Recife, PE||true|
```

Um campo que contenha o próprio delimitador, aspas ou quebra de linha, ou que comece com branco, sai
entre aspas. Datas saem como `AAAA-MM-DD`, que o `rptgo` lê de volta.

## Escolher colunas e ordenar

- **CSV e texto delimitado** são lidos por posição: o `input` precisa declarar **todas** as colunas,
  na ordem do arquivo. Os nomes das chaves ou colunas na saída são os do `input`.
- **JSON** é lido por nome: o `input` pode ter só algumas chaves, em qualquer ordem. É também o jeito
  de tirar colunas de um CSV: converta para JSON e, num segundo comando, escolha as que quer.
- Uma linha `sort limite desc` no script ordena a saída.

## Sem criar o arquivo `.rp`

A opção `-e` (ou `--script`) recebe o script inteiro como texto. O único argumento que sobra é o
arquivo de dados:

<!-- run -->
```sh
./rptgo --input json --format csv -e 'input nome, limite [float] sort limite desc end' \
  examples/conversao-formatos/clientes.json
```
<!-- expect -->
```text
nome,limite
Ana Souza,1500.5
Bruno Lima,-250
Carla Dias,
```

Com os dados vindo de outro comando:

```sh
printf '1;Ana\n2;Beto\n' | ./rptgo --input csv --format json -e 'input id [numeric], nome end'
```

```text
[
{"id":1,"nome":"Ana"},
{"id":2,"nome":"Beto"}
]
```

As aspas seguem as regras do seu terminal. Quando o script tem aspas simples, como em
`separator '|'`, envolva o texto em aspas duplas:

<!-- run -->
```sh
./rptgo --format csv -e "input id, nome, cidade, cadastro, ativo, limite separator '|' end" \
  examples/conversao-formatos/clientes.txt
```
<!-- expect -->
```text
id,nome,cidade,cadastro,ativo,limite
1,Ana Souza,Curitiba,15/01/2026,yes,1500.50
2,Bruno Lima,São Paulo,04/07/2025,no,-250
3,Carla Dias,"Recife, PE",,yes,
```

O texto do `-e` aceita tudo o que um arquivo `.rp` aceita, inclusive quebras de linha e seções de
relatório.

## Opções de saída e codificação

- CSV: `--csv-delimiter`, `--csv-quote`, `--csv-decimal` e `--csv-no-header` (veja `./rptgo --help`).
- CSV do Excel brasileiro: `--input csv --input-decimal auto --encoding latin-1`.
- `--encoding` vale para o arquivo `.rp`, os dados **e** a saída; o texto do `-e` é sempre UTF-8.
  Como JSON costuma ser lido em UTF-8, converta a saída:
  `... --encoding latin-1 ... | iconv -f latin1 -t utf-8`.
- `-o arquivo` grava o resultado num arquivo.
