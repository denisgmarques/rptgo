# Phases: inline-script-option

Gerado por /plan a partir de PLAN.md — view executável para `./ralph.sh .spec/features/inline-script-option/PHASES.md`.

## Phase 1: CLI option and help text

Antes de implementar, leia:
1. `.spec/features/inline-script-option/SPEC.md` — requisitos RIGID que esta fase cobre
2. `.spec/features/inline-script-option/PLAN.md` — decomposição completa, dependências e riscos

- [ ] T01 — Accept `-e` / `--script` in the CLI, with tests
      Arquivos: `cmd/rptgo/main.go`, `cmd/rptgo/html.go`, `cmd/rptgo/script_texto_test.go` (new)
      Mudança: add script text plus a separate presence flag to `opcoes` (pt-BR names); add `case "-e", "--script"` in `analisaArgs` using `next()`; reject a repeated option (any spelling) and 2+ positionals there as pt-BR usage errors. In `executar`, with the flag set, skip the missing-script check, `os.ReadFile` and script `decodificar`; the only positional is the data file (absent or `-` = stdin); share the existing path from `report.DefFonteScript` onward. In `montaHTML`, with the flag set, `Script` is empty and the default title is the literal `rptgo` (not `nomePrograma`), `--title` overrides; never pass an empty path to `filepath.Base`. Add the second usage form to the package doc comment. Keep parsing in `internal/report` untouched and file-script behaviour unchanged.
      Cobre: RF-01, RF-02, RF-03, RF-04, RF-05, RF-06, RF-07, RF-08, CT-01, RNF-01, RNF-02, RNF-03
      Acceptance criteria: `-e` and `--script` give byte-identical stdout, `-o` content and exit code as a `.rp` file with the same text in text, csv, json and html (html exception only for `{{Script}}` and default `{{Title}}`), including multi-line TEXT under `--strict` and TEXT equal to an existing `.rp` path (file not read); the only positional is the data file, before or after the pair, with absent or `-` reading stdin, and a nonexistent one exits 1 with `rptgo: `; each of the 8 invocations of RF-03, with and without a leading `--check`, exits 2 with empty stdout, `rptgo: ` and the help hint on stderr, stdin unread and no `-o` file; script errors and `-e ""` match the file run's stderr and exit 1; `--check`/`-r` parse only the inline script and read no data; under latin-1/cp850 `São` in TEXT is written as `0xE3`/`0xC6` and data is decoded as today; html with `-e` renders `[{{Script}}]` as `[]` and default title `rptgo`; no existing expected value changes; `go test ./...` exits 0; no `require` in `go.mod` and no `go.sum`.
      Testes: `cmd/rptgo/script_texto_test.go` — table-driven cases for file equivalence, data source, usage errors (reader that records reads), script errors, `--check`, encoding and html globals
- [ ] T02 — Document the option in `--help` and sync the README help block
      Arquivos: `cmd/rptgo/ajuda.go`, `cmd/rptgo/ajuda_test.go`, `README.md` (only the `./rptgo --help` expect block)
      Mudança: keep the first usage line unchanged and add a second one for `-e TEXTO [ENTRADA.txt | -]`; add an `-e, --script` entry saying the text is the whole script, replaces `SCRIPT.rp`, is UTF-8 and is not affected by `--encoding`; reword `--encoding` to name what it applies to (`.rp` file, data, output) and `--title` so its default covers `-e` (`rptgo`); add one `-e` example using the program name; copy the new output into the marked README block; extend `TestAjudaListaTodasAsOpcoes`. New text in pt-BR; do not hand-edit `AGENTS.md` or `docs/agents/*`.
      Cobre: RF-09, RF-08, RNF-03
      Acceptance criteria: `--help` exits 0, its first line is unchanged, stdout contains `-e, --script` and states the inline text is UTF-8 and not affected by `--encoding`; no help statement says `--encoding` applies to the inline text; the README help block matches the output; `go test ./cmd/rptgo -run 'Readme|Ajuda'` exits 0.
      Testes: `cmd/rptgo/ajuda_test.go` — `TestAjudaListaTodasAsOpcoes`; `cmd/rptgo/readme_test.go` — `TestReadmeBlocosConferem`

## Phase 2: README examples

Antes de implementar, leia:
1. `.spec/features/inline-script-option/SPEC.md` — requisitos RIGID que esta fase cobre
2. `.spec/features/inline-script-option/PLAN.md` — decomposição completa, dependências e riscos

- [ ] T03 — Add verified `-e` examples to both READMEs
      Arquivos: `README.md`, `examples/conversao-formatos/README.md`
      Mudança: in `README.md`, replace the `<(echo ...)` command with a `-e` command in a `<!-- run -->` + `<!-- expect -->` pair and add the `-e` form to the unmarked usage block in "Linha de comando". In the example page, keep the heading "Sem criar o arquivo `.rp`" (anchor link), lead with `-e` in a marked pair, show piped data with `-e` in an unmarked block (run it manually to confirm the output), remove "No Windows, grave o script num arquivo.", add a shell-quoting note, and in "Opções de saída e codificação" say `--encoding` covers the `.rp` file, data and output while the `-e` text is always UTF-8. Marked commands take a data file under `examples/` (no pipes); delimited/CSV input needs all columns in the script, so use `--input json` with `clientes.json` for a short column list, or declare all six columns. Additions in pt-BR; no edits to `SAIDA-HTML.md`, `REFERENCIA.md`, `AGENTS.md` or `docs/agents/*`.
      Cobre: RF-09, RNF-03, RNF-04
      Acceptance criteria: both files contain at least one command using `-e` or `--script` in a `<!-- run -->` block followed by its `<!-- expect -->` block; no statement in them says `--encoding` applies to the inline text or that a script file is the only way to pass a script on Windows; the link `#sem-criar-o-arquivo-rp` resolves; `go test ./cmd/rptgo -run 'Readme|Ajuda'` and `go test ./...` exit 0; `AGENTS.md` and `docs/agents/*` are unchanged.
      Testes: `cmd/rptgo/readme_test.go` — `TestReadmeBlocosConferem`
