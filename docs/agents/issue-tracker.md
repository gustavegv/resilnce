# Issue tracker: GitHub

Issues and specs live in GitHub Issues for `gustavegv/resilnce`.
Use the `gh` CLI from this repo.

## Conventions

- Create: `gh issue create --title "..." --body-file <file>`.
- Read: `gh issue view <number> --json number,title,body,labels,comments`.
- List: `gh issue list --state open --json number,title,body,labels,assignees`.
  Add label and state filters as needed.
- Comment: `gh issue comment <number> --body-file <file>`.
- Edit body: `gh issue edit <number> --body-file <file>`.
- Apply or remove labels: `gh issue edit <number> --add-label "..."`
  or `--remove-label "..."`.
- Close: post the explanation as a comment, then `gh issue close <number>`.
  Post comments separately even when an issue is already closed.
- For multiline bodies, write the exact Markdown to a temporary file
  and use `--body-file`.

Before the first issue creation in a session, check
`gh repo view --json nameWithOwner`. Target `gustavegv/resilnce`;
confirm with the user if another repository is returned.

## Parent issues and blockers

Use GitHub's native sub-issues and dependencies when available.
Check installed CLI help before using newer flags.

For older CLI versions, use `gh api`:

- Link a child: POST `repos/gustavegv/resilnce/issues/<parent>/sub_issues`
  with `-F sub_issue_id=<child-database-id>`.
- Add a blocker: POST
  `repos/gustavegv/resilnce/issues/<child>/dependencies/blocked_by`
  with `-F issue_id=<blocker-database-id>`.

Get database IDs with
`gh api repos/gustavegv/resilnce/issues/<number> --jq .id`.

If unavailable, put `Part of #<parent>` and `Blocked by: #<number>`
at the top of the child body, and list children in the parent's task list.
A ticket is startable when every blocker is closed.

## Pull requests as a triage surface

**PRs as a request surface: no.**

## Skill terminology

“Publish to the issue tracker” means create a GitHub issue.
“Fetch the relevant ticket” means read the GitHub issue.

## Wayfinding operations

- Map: one issue labelled `wayfinder:map`, with Notes,
  Decisions-so-far, and Fog sections.
- Children: linked sub-issues labelled `wayfinder:<type>`, where type is
  research, prototype, grilling, or task.
- Frontier: open children in map order, excluding assigned tickets
  and tickets with open blockers.
- Claim: assign the ticket with `gh issue edit <number> --add-assignee @me`.
- Resolve: comment with the answer, close the ticket, then append a
  gist and link to the map's Decisions-so-far section.
