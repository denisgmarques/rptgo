# Phases: csv-json-output-docs

Gerado por /plan a partir de PLAN.md — view executável para `./ralph.sh .spec/features/csv-json-output-docs/PHASES.md`.

## Phase 1: Baseline safety net, CP850 table, and didactic examples

Antes de implementar, leia:
1. `.spec/features/csv-json-output-docs/SPEC.md` — requisitos RIGID que esta fase cobre
2. `.spec/features/csv-json-output-docs/PLAN.md` — decomposição completa, dependências e riscos

- [x] T01 — Golden baseline regression tests for current text-mode output
      Arquivos: `internal/report/golden_test.go`, `internal/report/testdata/golden/*.rp`, `internal/report/testdata/golden/*.txt`, `cmd/rptgo/main_test.go`
      Mudança: commit fixtures (pinning `RPT_NOW`) and tests capturing today's exact `Run()`/`executar()` output, before any production code changes.
      Cobre: RF-03, RF-07
      Acceptance criteria: `go test ./...` passes with the new golden tests committed and green against the unmodified current code.
      Testes: `internal/report/golden_test.go` — `TestGoldenTextoAtual`; `cmd/rptgo/main_test.go` — `TestExecutarPadraoAtual`
- [x] T02 — Add `--format <text|csv|json>` flag with validation
      Arquivos: `cmd/rptgo/main.go`
      Mudança: add `formato string` to `opcoes` (default `"text"`); add `case "--format":` in `analisaArgs` validating the value is `text`/`csv`/`json`, erroring like the existing unknown-flag branch otherwise.
      Cobre: RF-01, RF-02, RF-03, CT-01
      Acceptance criteria: no flag → `op.formato == "text"`; `--format csv`/`json`/`text` accepted; `--format xml` makes `executar()` exit 2 with `rptgo: ` on stderr.
      Testes: `cmd/rptgo/main_test.go` — `TestAnalisaArgsFormat`
- [x] T06 — CP850 code-page table
      Arquivos: `cmd/rptgo/cp850.go`
      Mudança: add `var cp850ParaRune [256]rune` with the standard public IBM/DOS CP850 byte→code-point table (sourced only from a public reference); derive `var cp850ParaByte = map[rune]byte{}` via `init()`.
      Cobre: RF-08, RF-09
      Acceptance criteria: all 256 table entries defined; every byte 0x80–0xFF where CP850 and latin-1 diverge produces a different rune than `rune(b)`; round-trip `cp850ParaByte[cp850ParaRune[b]] == b` holds for all 256 byte values.
      Testes: `cmd/rptgo/cp850_test.go` — `TestCP850Tabela256`, `TestCP850DivergeDeLatin1`, `TestCP850RoundTrip`
- [x] T12 — `examples/relatorio-vendas-regiao/`: 1 break level + total
      Arquivos: `examples/relatorio-vendas-regiao/relatorio.rp`, `examples/relatorio-vendas-regiao/dados.txt`
      Mudança: fictional "sales by region" `.rp` script (no `today`/`hour`) with one break level and an `after <level>` total/count accumulator, plus matching fictional dataset.
      Cobre: RF-10
      Acceptance criteria: `rptgo examples/relatorio-vendas-regiao/relatorio.rp examples/relatorio-vendas-regiao/dados.txt` exits 0 and its output contains at least one `after <level>` total line.
      Testes: `cmd/rptgo/examples_test.go` — `TestExemploVendasRegiao`
- [x] T13 — `examples/relatorio-estoque-categoria/`: 2+ cascading breaks + aggregate
      Arquivos: `examples/relatorio-estoque-categoria/relatorio.rp`, `examples/relatorio-estoque-categoria/dados.txt`
      Mudança: fictional "inventory by category" `.rp` script with 2+ sort names each with matching `before`/`after`, and at least one `after` section calling `count`/`total`/`avg`, plus matching fictional dataset.
      Cobre: RF-11
      Acceptance criteria: `rptgo examples/relatorio-estoque-categoria/relatorio.rp examples/relatorio-estoque-categoria/dados.txt` exits 0; script declares 2+ sort names with matching `before`/`after`; output contains at least one aggregate-derived value.
      Testes: `cmd/rptgo/examples_test.go` — `TestExemploEstoqueCategoria`

## Phase 2: Row representation refactor and form-feed/format validation

Antes de implementar, leia:
1. `.spec/features/csv-json-output-docs/SPEC.md` — requisitos RIGID que esta fase cobre
2. `.spec/features/csv-json-output-docs/PLAN.md` — decomposição completa, dependências e riscos

- [x] T03 — Reject `--format csv|json` + `--form-feed` at validation time
      Arquivos: `cmd/rptgo/main.go`
      Mudança: at the end of `analisaArgs`, if `op.formato` is `csv`/`json` and `op.formFeed` is true, return an error before any script/data I/O happens.
      Cobre: RF-05, CT-02
      Acceptance criteria: `rptgo --format csv --form-feed script.rp dados.txt` (any flag order) exits 2, stderr matches `rptgo: `, stdout is empty, and no `-o` file is written.
      Testes: `cmd/rptgo/main_test.go` — `TestFormatFormFeedRejeitado`
- [x] T04 — Replace `Row` map with a compact slice + shared column index
      Arquivos: `internal/report/interp.go`
      Mudança: `type Row = map[string]Value` becomes `type Row = []Value` (positional, matching `it.S.Input` order); add `Interp.colIdx map[string]int` built once in `NovoInterp`; update `converte` to build positional slices; update `ev`'s `ColExpr` case to resolve the name via `colIdx` before indexing `it.row`.
      Cobre: RF-07
      Acceptance criteria: for every T01 golden fixture, `Run()`'s output after this change is byte-for-byte identical to the T01 baseline.
      Testes: `internal/report/interp_test.go` — `TestRowCompactoSaidaIdentica`

## Phase 3: Sorted-rows accessor, CP850 wiring, and memory benchmark

Antes de implementar, leia:
1. `.spec/features/csv-json-output-docs/SPEC.md` — requisitos RIGID que esta fase cobre
2. `.spec/features/csv-json-output-docs/PLAN.md` — decomposição completa, dependências e riscos
3. `cmd/rptgo/cp850.go` — tabela CP850 já implementada na Fase 1 (T06)

- [x] T05 — Export `Interp.LinhasOrdenadas()` (no event execution)
      Arquivos: `internal/report/interp.go`
      Mudança: add `func (it *Interp) LinhasOrdenadas() []Row { it.ordena(); return it.rows }` — calls only `ordena()`, never `exec`/`Run` or anything touching events/pagination state.
      Cobre: RF-04
      Acceptance criteria: for a script with 2 sort levels and side-effecting `before`/`after` sections, calling `LinhasOrdenadas()` returns rows in `sort`-clause order while `it.vars` stays empty and `it.pageno` stays 0.
      Testes: `internal/report/interp_test.go` — `TestLinhasOrdenadasSemEventos`
- [x] T07 — Wire `cp850` into `decodificar`/`codificar`
      Arquivos: `cmd/rptgo/main.go`
      Mudança: add `case "cp850":` to both `decodificar` and `codificar`, alongside the existing `latin-1` case; decode via `cp850ParaRune`, encode via `cp850ParaByte` with `'?'` fallback on miss.
      Cobre: RF-08, RF-09, CT-03
      Acceptance criteria: running `executar()` with `--encoding cp850` round-trips every byte 0x00–0xFF through `decodificar`→`codificar` back to the original byte.
      Testes: `cmd/rptgo/main_test.go` — `TestEncodingCP850`
- [x] T11 — Synthetic 200k-record dataset generator + memory benchmark
      Arquivos: `internal/report/testdata/gerador200k_test.go`, `internal/report/bench_test.go`
      Mudança: fixed-seed `math/rand` generator producing 200,000 records matching the T12 example's `input` schema; `BenchmarkConverte200k` measuring `AllocedBytesPerOp`/`runtime.MemStats` around `Interp.converte`.
      Cobre: RNF-01
      Acceptance criteria: running `go test -bench=Converte200k -benchmem ./internal/report/...` once before T04's commit and once after shows a measurable reduction in bytes/allocs per op for the new `Row` representation versus the old one.
      Testes: `internal/report/bench_test.go` — `BenchmarkConverte200k`

## Phase 4: CSV and JSON serializers

Antes de implementar, leia:
1. `.spec/features/csv-json-output-docs/SPEC.md` — requisitos RIGID que esta fase cobre
2. `.spec/features/csv-json-output-docs/PLAN.md` — decomposição completa, dependências e riscos

- [x] T08 — CSV serializer
      Arquivos: `cmd/rptgo/csv.go`
      Mudança: `escreveCSV(w io.Writer, s *report.Script, rows []report.Row) error` using `encoding/csv.Writer` (default comma, never `s.Separator`); header from `s.Input` names; per-column formatting by `col.Type` (date/ldate → bare ISO-8601 `YYYY-MM-DD` bypassing `DATEFMT`/`CENTURY_CUTOFF`; int/float families → decimal text; nil → `""`); never reads `s.Width`/`s.Length`/margins.
      Cobre: RF-01, RF-04, RF-06
      Acceptance criteria: N records → N+1 lines (header + N rows); header equals declared column names in order; fields containing `,`/`"`/`\n` are RFC 4180-quoted; every `date`/`ldate` field matches `^\d{4}-\d{2}-\d{2}$` regardless of `DATEFMT`/`CENTURY_CUTOFF`.
      Testes: `cmd/rptgo/csv_test.go` — `TestEscreveCSVHeaderEContagem`, `TestEscreveCSVQuoting`, `TestEscreveCSVDatasISO`
- [x] T09 — JSON serializer
      Arquivos: `cmd/rptgo/json.go`
      Mudança: `escreveJSON(w io.Writer, s *report.Script, rows []report.Row) error` building each object manually in `s.Input` declared order (never a generic `map[string]any` + `json.Marshal`, which would mis-render `time.Time` as RFC 3339 and reorder keys alphabetically); nil → `null`; date/ldate → quoted ISO string; numeric → bare `json.Marshal(int64|float64)`; bool (defensive) → `json.Marshal(bool)`; never reads layout clauses.
      Cobre: RF-02, RF-04, RF-06
      Acceptance criteria: output is valid JSON, top-level array of exactly N objects with keys exactly matching `s.Input` names; every nil field is `null`; every date/ldate field is a string matching `^\d{4}-\d{2}-\d{2}$`; every numeric field unmarshals as a JSON number, never a string.
      Testes: `cmd/rptgo/json_test.go` — `TestEscreveJSONEstrutura`, `TestEscreveJSONTipos`

## Phase 5: Format dispatch integration

Antes de implementar, leia:
1. `.spec/features/csv-json-output-docs/SPEC.md` — requisitos RIGID que esta fase cobre
2. `.spec/features/csv-json-output-docs/PLAN.md` — decomposição completa, dependências e riscos

- [x] T10 — Wire `--format csv/json` dispatch into `executar()`
      Arquivos: `cmd/rptgo/main.go`
      Mudança: after `report.NovoInterp(...)`, branch on `op.formato`: `"text"` keeps the exact existing `it.Run()` sequence untouched; `"csv"`/`"json"` call `it.LinhasOrdenadas()` then `escreveCSV`/`escreveJSON` into a buffer, then feed the result through the same existing crlf-conversion + `codificar` + stdout/`-o` tail as the text path.
      Cobre: RF-01, RF-02, RF-03, RF-04, RF-06, CT-01
      Acceptance criteria: every T01 golden fixture run with no `--format`/`--format text` is byte-for-byte identical to the T01 baseline; `--format csv`/`json` produce correct content with exit code 0; a script with side-effecting `before`/`after` sections and a `sort` clause emits csv/json rows in `sort` order with no event-only content.
      Testes: `cmd/rptgo/main_test.go` — `TestExecutarFormatoTextoInalterado`, `TestExecutarFormatoCSV`, `TestExecutarFormatoJSON`, `TestExecutarFormatoOrdemESort`

## Phase 6: README flag/encoding reference

Antes de implementar, leia:
1. `.spec/features/csv-json-output-docs/SPEC.md` — requisitos RIGID que esta fase cobre
2. `.spec/features/csv-json-output-docs/PLAN.md` — decomposição completa, dependências e riscos

- [x] T14 — README: `--format`/`--encoding cp850` reference + "Estrutura" + close resolved gaps
      Arquivos: `README.md`
      Mudança: document `--format text|csv|json` and `cp850` in `## Uso`; add `cp850.go`/`csv.go`/`json.go` to `## Estrutura`; remove the two now-resolved bullets from `## O que ainda não está aqui`, keeping the still-true third bullet.
      Cobre: CT-01, CT-03
      Acceptance criteria: `README.md`'s "O que ainda não está aqui" section no longer mentions CSV/JSON output or the CP850 gap; `## Uso` documents both new flag values.
      Testes: none (prose change); verified by manual review of the diff

## Phase 7: README "Primeiros passos"

Antes de implementar, leia:
1. `.spec/features/csv-json-output-docs/SPEC.md` — requisitos RIGID que esta fase cobre
2. `.spec/features/csv-json-output-docs/PLAN.md` — decomposição completa, dependências e riscos

- [x] T15 — README "Primeiros passos" section + reproducibility test
      Arquivos: `README.md`, `cmd/rptgo/readme_test.go`
      Mudança: add a "## Primeiros passos" section showing the exact command line for the T12 example and its exact expected output (or exact first/last lines if long).
      Cobre: RF-12
      Acceptance criteria: copy-pasting the documented command against the committed `examples/relatorio-vendas-regiao/` files reproduces the documented output exactly, with no manual edits.
      Testes: `cmd/rptgo/readme_test.go` — `TestReadmePrimeirosPassos`

