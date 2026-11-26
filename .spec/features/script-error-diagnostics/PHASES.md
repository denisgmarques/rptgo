# Phases: script-error-diagnostics

Gerado por /plan a partir de PLAN.md — view executável para `./ralph.sh .spec/features/script-error-diagnostics/PHASES.md`.

## Phase 1: Baseline characterization tests

Antes de implementar, leia:
1. `.spec/features/script-error-diagnostics/SPEC.md` — requisitos RIGID que esta fase cobre
2. `.spec/features/script-error-diagnostics/PLAN.md` — decomposição completa, dependências e riscos

- [ ] T01 — Pin baseline behaviour with characterization tests
      Arquivos: `cmd/rptgo/regressao_script_test.go` (new)
      Mudança: with no production change, add tests that pass on the current working tree (which includes the uncommitted `-e` feature), with literal expected values captured from it: (1) `--check` on every `examples/*/*.rp` and `internal/report/testdata/golden/*.rp`; (2) keyword-named columns (`date`, `end`, `sort`, `detail`, `width` and others), including a last column after a comma, with stdout, `-o` content, stderr, exit code and `--check`; (3) valid sort keys: column, `<mes> mdy(data, 0)`, expression declared later in a `print`, `set` variable with `before` / `after`, `<g> g`, `today`, `hour`, `pageno`, `null`, `dow`, `mdy`, `index`, `substr`, and a named aggregate no key refers to; (4) invalid-value and field-count data errors on lines longer than 72 code points, indented and with tabs; (5) the static type check error for an arithmetic `if`. Identifiers and comments in pt-BR.
      Cobre: RF-07, RF-08, RNF-02
      Acceptance criteria: the test file covers the five groups with exact expected output; `--check` on each example and golden script exits 0 with stdout exactly `Script sem erros de sintaxe.\n`; data error expectations show the whole line with the marker from the first column capped at 40; no non-test file changes in this phase; `go test ./...` exits 0.
      Testes: `cmd/rptgo/regressao_script_test.go` — five table-driven groups

## Phase 2: Token positions and script error renderer

Antes de implementar, leia:
1. `.spec/features/script-error-diagnostics/SPEC.md` — requisitos RIGID que esta fase cobre
2. `.spec/features/script-error-diagnostics/PLAN.md` — decomposição completa, dependências e riscos

- [ ] T02 — Token positions, script error renderer and lexer errors
      Arquivos: `internal/report/lexer.go`, `internal/report/errors.go`, `internal/report/erro_script_test.go` (new)
      Mudança: `Token` gains the 0-based code point column and the source text as written; `Text` stays as today; track line start at every counted newline, and also count newlines consumed by a string backslash escape or `N[<newline>]`. Add a separate script error renderer (line, column, length, message); do not modify `comLinha`, `ErroLinhaScript` or `ErroLinhaDados`. Output: `linha N: msg`, `  N | excerpt`, padding as wide as `  N | ` plus one pad per printed code point before the word (tab where the line has a tab, else space; 3 spaces for a left `...`) and the `^` run. Strip a trailing `\r`; a line outside the source yields only the message. Window with n = line length, t = word length, c = column: whole line if n <= 72; else start = c - max(0, 72-t)/2 clamped to [0, n-72] when t < 72, start = c when t >= 72; end = min(start+72, n); `...` on each cut side; carets = max(1, min(t, end-c)). Use it for unrecognized character (that character), both string errors (opening quote to last non-blank of the opening line) and both comment errors (`/*` to last non-blank of the opening line); messages unchanged.
      Cobre: RF-03, RF-04, RF-05, RF-08, CT-01, RNF-01, RNF-03
      Acceptance criteria: for each lexer row of RF-03, with preceding text or indentation, the first `^` is under the word's first code point and the caret count equals its length subject to clipping; excerpts hold at most 72 code points of source, contain the word's start and show `...` only on cut sides; a word longer than the window starts the excerpt and carets do not pass its end; lines of 72 or fewer print whole; no `\r` is printed; tabs before the word are mirrored in the marker line and multibyte characters count as one position, also on cut lines; T01 tests and the `REFERENCIA.md` blocks still pass; `comLinha` is unchanged; `go test ./...` exits 0.
      Testes: `internal/report/erro_script_test.go` — lexer rows, window start/middle/end, long word, 72 vs 73, CRLF, tabs, multibyte (call `DefFonteScript`, restore with `t.Cleanup`)

## Phase 3: Parser error positions and messages

Antes de implementar, leia:
1. `.spec/features/script-error-diagnostics/SPEC.md` — requisitos RIGID que esta fase cobre
2. `.spec/features/script-error-diagnostics/PLAN.md` — decomposição completa, dependências e riscos

- [ ] T03 — Parser errors point at the offending word; missing `end`, invalid expression and comma hint
      Arquivos: `internal/report/parser.go`, `internal/report/erro_script_test.go`, `cmd/rptgo/erros_script_cli_test.go` (new), `REFERENCIA.md` (repeated section error block only)
      Mudança: route every parser error through one helper that takes the token index (captured before `nx()`); on EOF report the last real token; with no real token print `linha N: <message>` without source line. Mark per the RF-03 table: bad `<name>`, `numInt` token, the consumed token for column name, section target, `page` word and `set` name, the repeated section keyword, the repeated expression name, the non-integer `substr` bound (invalid range: the `substr` keyword), the time or date literal (wrap the `ParseData` error), otherwise the found token. Messages, quoting source text as written (strings keep their quotes, other words in `'...'`): `falta o 'end' que termina o script: chegou ao fim do script depois de 'X'` (no words: `...: o script não tem nenhuma palavra`); `falta 'end' do bloco 'begin' aberto na linha B: chegou ao fim do script depois de 'X'` (prefix and source line follow the last word); `expressão inválida: encontrado 'X'` or `expressão inválida: chegou ao fim do script depois de 'X'`; others unchanged. Extra comma hint, only when all hold: (a) `pInput` took as column name an identifier right after a `,`; (b) it is one of `input width length top bottom left separator sort before after header footer detail end`; (c) the parser error is on the token immediately after it (index + 1, EOF included, compared before the EOF fallback). Then append `; 'X' foi lido como nome de coluna do input; pode haver uma vírgula sobrando`. No new rejection. Update the repeated-`detail` block in `REFERENCIA.md` to `linha 4`, `  4 | detail`, `^^^^^^`.
      Cobre: RF-03, RF-05, RF-06, RF-07, CT-01, RNF-03, RNF-04
      Acceptance criteria: each parser row of RF-03 marks the stated word on its own line, including when it is last on its line; `detail` on lines 2 and 4 reports and shows line 4; EOF errors show the last word's line despite trailing newlines, blanks or comments; empty and comment-only scripts exit 1 without a source line; `print a, ,` marks the second `,`; the block-`end` message names the `begin` line; `input a,` + `end` reports line 2, quotes and shows `end` and has the hint; `input a,` + `sort a` + `end` has the hint; the hint is absent for `input a, width` followed by an error further on, for `input a, date`, for a keyword followed by `[` and for `input end`; stderr is byte-identical between `.rp` and `-e`; alignment holds under utf-8, latin-1 and cp850; T01 tests pass; `go test ./...` exits 0.
      Testes: `internal/report/erro_script_test.go` — parser rows, EOF cases, hint positive and negative; `cmd/rptgo/erros_script_cli_test.go` — file vs `-e`, encodings

## Phase 4: Column type check

Antes de implementar, leia:
1. `.spec/features/script-error-diagnostics/SPEC.md` — requisitos RIGID que esta fase cobre
2. `.spec/features/script-error-diagnostics/PLAN.md` — decomposição completa, dependências e riscos

- [ ] T04 — Reject unknown column types
      Arquivos: `internal/report/parser.go`, `internal/report/interp.go` (only if the type list is placed next to `tiposInt` / `tiposFlt`), `internal/report/erro_script_test.go`, `cmd/rptgo/erros_script_cli_test.go`
      Mudança: one ordered list of the 15 types (`string text numeric integer long int float amount double real huge date ldate time bool`) feeds both lookup and message. In `pInput`, after `[`, check the next token's `Text` before reading size and `]`. Message `tipo desconhecido 'X' na coluna 'C'; tipos válidos: <list>`, X as written, marker under X; at end of script say the end was reached (still naming C and the list) and mark the last word. `[10]` and `[]` are rejected; quoted `'numeric'` stays accepted; no `[...]` stays `string`. Layering: the check lives in `internal/report`; do not change `cmd/rptgo/entrada.go`.
      Cobre: RF-01, RF-03, RF-07, CT-01, RNF-03
      Acceptance criteria: `input nome, valor [numerc] end` exits 1 with empty stdout, stderr naming `valor`, `numerc` and the 15 types, marker under `numerc`; `--check` and `-r` give the same stderr and exit code without the OK message; with a nonexistent data file the type error is reported and stdin is not read; `input a [10] end` names `a` and `10`; the 15 types are accepted in lower and upper case, with and without size; T01 tests pass; `go test ./...` exits 0.
      Testes: `internal/report/erro_script_test.go` — invalid and valid types, EOF after `[`, `tiposInt` / `tiposFlt` contained in the list; `cmd/rptgo/erros_script_cli_test.go` — `--check`, `-r`, nonexistent data, stdin unread

## Phase 5: Sort name and function check

Antes de implementar, leia:
1. `.spec/features/script-error-diagnostics/SPEC.md` — requisitos RIGID que esta fase cobre
2. `.spec/features/script-error-diagnostics/PLAN.md` — decomposição completa, dependências e riscos

- [ ] T05 — Reject unknown names, unknown functions and aggregates in `sort`
      Arquivos: `internal/report/ordenacao.go` (new), `internal/report/parser.go`, `internal/report/interp.go` (comment in `chave` only), `internal/report/tipos.go` (comment in `tipoDeV` only, if inaccurate), `internal/report/erro_script_test.go`, `cmd/rptgo/erros_script_cli_test.go`
      Mudança: side table in the parser, not in the AST (`registraNome` uses `reflect.DeepEqual`): `ePrim` appends each column reference and function call (name, call flag, token index) in source order, excluding `null`, `today`, `hour`, `pageno` without `(`; each sort key and registered named expression keeps its range (first registration wins; `<g> g` registers nothing). In `Parse`, after the `input` check, `coletarVariaveis` and `s.Nomeadas`, before `verificaNomes` / `verificaTipos`, check keys in order: calls must be `dow`, `mdy`, `index`; aggregates give `agregados não podem ser usados no sort: 'f'`; other calls `função desconhecida no sort: 'f'`; a column reference is valid as input column or `set` variable, else expand the named expression once (visited set), else `nome desconhecido no sort: 'n'`. Mark the token where the name is written. Do not change evaluation in `interp.go`.
      Cobre: RF-02, RF-03, RF-07, CT-01, RNF-03
      Acceptance criteria: `sort idades` exits 1 naming and marking `idades`; unknown names nested, in `desc` or second keys, in function arguments or in referenced named expressions are reported; `sort <x> x` reports `x`; `foo(a)` and `count`, `total`, `avg`, `min`, `max` (including `count(*)`), direct or through a named expression, are reported at the function name, aggregates saying they cannot be used in `sort`; same stderr and exit code with `--check` and `-r`; for text, csv, json and html a nonexistent data file or stdin is not read and no `-o` file is created; every valid form pinned by T01 keeps its output; an aggregate no key refers to stays accepted; self-referencing names do not loop; `go test ./...` exits 0.
      Testes: `internal/report/erro_script_test.go` — invalid and valid sort forms; `cmd/rptgo/erros_script_cli_test.go` — `--check`, no data read, no `-o`, four formats

## Phase 6: Reference documentation and final verification

Antes de implementar, leia:
1. `.spec/features/script-error-diagnostics/SPEC.md` — requisitos RIGID que esta fase cobre
2. `.spec/features/script-error-diagnostics/PLAN.md` — decomposição completa, dependências e riscos

- [ ] T06 — Reference documentation and final verification
      Arquivos: `REFERENCIA.md`, `README.md` (only if a statement contradicts RF-01 to RF-06)
      Mudança: in pt-BR, section 3 states that an unlisted type is an error; section 4 states that unknown names, unknown functions and aggregates in `sort` are errors before data is read; section 12 gets one verified `<!-- script -->` + `<!-- data -->` + `<!-- error -->` example for each new error and describes the marker under the word, the 72 character excerpt and the extra comma hint. Re-read the section 12 introduction and "Outros erros frequentes"; grep both files for statements about types, `sort` and errors. Do not edit data error blocks, translate existing text, or touch `AGENTS.md` / `docs/agents/*`.
      Cobre: RF-08, RNF-01, RNF-02, RNF-03, RNF-04, RNF-05
      Acceptance criteria: `REFERENCIA.md` has a passing error example for the unknown type and for an unknown sort name; no statement in `README.md` or `REFERENCIA.md` contradicts RF-01 to RF-06; the invalid-value and field-count blocks are unedited; `go test ./cmd/rptgo -run 'Readme|Referencia'` and `go test ./...` exit 0; `go.mod` has no `require` and no `go.sum` exists; `AGENTS.md` and `docs/agents/*` are unchanged.
      Testes: `cmd/rptgo/readme_test.go` — `TestReferenciaExemplos`, `TestReadmeBlocosConferem`
