# gobank-products

The product contract and gobank's versioned products. What the package
does is in README.md, api.md, products.md and interest.md; this file is
how changes reach `main`.

## Code flow: level 1

`maturity.code: 1` in task-plus.yml (manual: software maturity). `main`
is protected on the Bytestone Forgejo and takes nothing but pull-request
merges and release tags. One branch per story; never commit on `main`.

1. Branch off `main`: `git checkout -b <story>`.
2. Work test-first on the branch and commit there. Fill CHANGELOG
   `[Unreleased]`; write `.tp-release-comment`.
3. Tell the user the version to pick. They run `tp release` on the
   branch, the prepare run: it rolls the changelog into the version
   heading, pushes the branch, opens the pull request and, with
   `merge_wait: 10m`, polls the forge for the merge. The user merges on
   the forge; the same run then fast-forwards `main`, re-runs `check`,
   tags and publishes.
4. If the wait times out, the pull request stays open and `tp release`
   on `main` after the merge is the tag run.

## Roadmap

ROADMAP.md is named epics of `- #N text` lines, each an open issue
labelled `roadmap` (task-plus ADR 0004). Add an item with
`tp roadmap add --epic <Epic> <title>` (the flag before the title); never
number, tick or stamp a line. An item is done when its issue is closed:
`tp release` moves it into the changelog under the version, or
`tp roadmap archive` on demand. `tp check` lints the file against the
forge.
