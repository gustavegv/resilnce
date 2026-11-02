## Agent skills

### Issue tracker

Issues and specs live in GitHub Issues for `gustavegv/resilnce`.
Before tracker operations, read `docs/agents/issue-tracker.md`.

### Triage labels

Five default labels map the canonical triage roles.
Before triaging, read `docs/agents/triage-labels.md`.

### Domain docs

Single-context: root `GLOSSARY.md` and `docs/adr/`, shared by all apps.
Before exploring the codebase, read `docs/agents/domain.md`.

## Svelte

For any creation, edit, review, or analysis of `*.svelte`, `*.svelte.ts`,
or `*.svelte.js` files:

- MUST use `$svelte-code-writer` for writing or designing svelte code.
- MUST use `$svelte-core-bestpractices` for discovering good way to work in Svelte.
- Prefer the `svelte-file-editor` subagent for `.svelte` files when available.
- Use modern Svelte 5 syntax and runes; avoid legacy syntax unless required by
  the existing codebase.