# bc-harness and documentation language rules

These rules live here, not in `CLAUDE.md` or `AGENTS.md`, because `/ai-context`
regenerates those two files and overwrites manual edits.

## Always use the bc-harness plugin

- Use the `bc-harness` plugin workflows for planning, specs and context docs:
  - `/bc-harness:plan` for any new feature, change or fix that needs a spec/plan.
  - `/bc-harness:init` (and its `init:*` commands) for project-level specs under `.spec/init/`.
  - `/bc-harness:ai-context` to generate or refresh `AGENTS.md`, `CLAUDE.md` and `docs/agents/*.md`.
- Do not hand-write or hand-edit files that these workflows own (`.spec/**`,
  `AGENTS.md`, `CLAUDE.md`, `docs/agents/*.md`). Re-run the matching command instead.
- Do not write application code during `/bc-harness:plan`; it only writes under
  `.spec/features/<slug>/`.

## Language: English for specs and documentation

- All specs are written in English: everything under `.spec/` (`SPEC.md`, `PLAN.md`,
  `PHASES.md`, `.spec/init/*`, contracts, clarification questions and answers).
- Everything under `docs/` is written in English, including `docs/agents/*.md`.
- When running `/bc-harness:plan`, `/bc-harness:init` or `/bc-harness:ai-context`, keep
  the output in English even if the developer's prompt is in Portuguese.
- Existing Portuguese (pt-BR) docs must be translated to English when they are next
  touched. Code identifiers and comments under `cmd/` and `internal/` are out of scope
  of this rule and keep following the existing code conventions.
