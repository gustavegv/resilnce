# Domain docs

## Before exploring the codebase

This repo uses one domain context shared by the web app and server.

- Read root `GLOSSARY.md` when present.
- Read decisions in `docs/adr/` relevant to the work.

When these files are absent, proceed silently. The domain-modeling
skill creates them as terms and decisions are resolved.

## Layout

- `GLOSSARY.md`: shared domain vocabulary.
- `docs/adr/`: architectural decision records.

## Vocabulary

Use glossary terms in issue titles, proposals, hypotheses, and tests.
For an undefined concept, reconsider the wording or note the gap
for domain-modeling.

## Decision conflicts

Explicitly identify any proposal that contradicts an existing ADR,
including the ADR reference and reason to reconsider it.
