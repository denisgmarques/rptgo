# Backlog

Known issues and open decisions, none scheduled yet. Behaviour was observed on 2026-11-25 at commit
`a4895bd`. When an item is picked up and needs a spec, use `/bc-harness:plan`
(see `.claude/rules/bc-harness.md`), and remove it from this file once it is done.

Commands assume a fresh build: `go build -o rptgo ./cmd/rptgo`.

| # | Item | Type | Priority |
|---|---|---|---|
| B1 | Text wrapping counts bytes instead of characters | Bug | High |
| B2 | Impossible dates are accepted and roll over | Bug | High |
| B3 | `header` and `footer` errors after page 1 are silently ignored | Bug | High |
| B4 | Other evaluation errors in `sort` keys are silently ignored | Decision | Medium |
| B5 | Very large sizes crash with out of memory | Bug | Low |
| B6 | `REFERENCIA.md` describes the `&` mask position backwards | Docs | Low |
| B7 | `--help` does not list `TRIADSEP` | Docs | Low |
| B8 | `SAIDA-HTML.md` does not cover `-e` | Docs | Low |
| B9 | pt-BR docs versus the English documentation rule | Decision | Low |
| B10 | Two Go files are not `gofmt` clean | Chore | Low |

## B1. Text wrapping counts bytes instead of characters

```sh
printf 'ação ação\n' | ./rptgo -e "input s detail print s using '9' end"
printf 'São João\n'  | ./rptgo -e "input s detail print s using '2' end"
```

- **Today:** the first command prints two lines, although `ação ação` has 9 characters. The second
  splits accented letters in half and writes invalid UTF-8.
- **Expected:** the width is counted in characters and no character is split.
- **Where:** `QuebraTexto` in `internal/report/format.go`.
- **Note:** the output of existing reports with accented wrapped text will change.

## B2. Impossible dates are accepted and roll over

```sh
printf '31/02/2026\n' | ./rptgo -e 'input d [ldate] detail print d end'
```

- **Today:** prints `03/03/2026` with exit code 0.
- **Expected:** an invalid value error showing the record, as for other invalid input.
- **Where:** date parsing in `internal/report/dates.go`.

## B3. `header` and `footer` errors after page 1 are silently ignored

```sh
printf '1|1\n2|1\n3|1\n4|0\n5|0\n' | ./rptgo -e "input n [numeric], z [numeric] top margin 0 bottom margin 0 length 3 header print 'h', 10 / z detail print n end"
```

- **Today:** on the page where `z` is 0 the header prints only `h`, with exit code 0. The same
  happens with `footer`.
- **Expected:** the division by zero stops the report, as it does in `detail`.
- **Where:** the calls to `exec` in `fechaPagina` and `novaPagina`, `internal/report/interp.go`.

## B4. Other evaluation errors in `sort` keys are silently ignored

```sh
printf '1|0\n' | ./rptgo --format csv -e 'input n [numeric], z [numeric] sort n / z end'
```

- **Today:** the key becomes null and the run succeeds. Only wrong value types (such as `dow` on a
  number) stop the report.
- **To decide:** reporting every error is safer, but scripts that run today would start failing. One
  known case is `sort d substr[1-2]` where `d` is a date.
- **Where:** `Interp.chave` in `internal/report/interp.go`.

## B5. Very large sizes crash with out of memory

```sh
printf 'x\n' | ./rptgo -e 'input a detail print a in col 999999999999 end'
printf 'x\n' | ./rptgo -e 'input a detail print 999999999999[x] end'
printf 'x\n' | ./rptgo -e 'input a detail skip 999999999999 end'
```

- **Today:** the program dies with a Go `out of memory` trace.
- **Expected:** an error message with a documented upper limit. A column can come from the data, so
  this does not require an unusual script.

## B6. `REFERENCIA.md` describes the `&` mask position backwards

- **Today:** the mask table says `&` also applies to the `#` positions on its **left**. The program
  applies it to the positions on its **right**: `##&##` formats 7 as two blanks followed by `007`.
- **Expected:** fix the sentence and add a verified example.

## B7. `--help` does not list `TRIADSEP`

- **Today:** the program reads `TRIADSEP`, but the environment variable list in the help omits it.
- **Where:** `cmd/rptgo/ajuda.go`; the help block in `README.md` must be updated with it.

## B8. `SAIDA-HTML.md` does not cover `-e`

- **Today:** the page does not say that with `-e` the `{{Script}}` variable is empty and the default
  `{{Title}}` is `rptgo`.

## B9. pt-BR docs versus the English documentation rule

- **Today:** `.claude/rules/bc-harness.md` says existing pt-BR docs are translated to English when
  next touched. `README.md`, `REFERENCIA.md` and the example pages were edited and kept in pt-BR.
- **To decide:** translate them, or change the rule so user-facing docs stay in pt-BR.

## B10. Two Go files are not `gofmt` clean

- **Today:** `gofmt -l cmd internal` lists `internal/report/interp.go` and
  `internal/report/lexer.go`.
