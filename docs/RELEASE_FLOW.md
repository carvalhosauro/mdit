# Release flow

mdit ships via a `feat → stable → main` release train. Versions are derived
from Conventional Commits; releases are cut automatically on merge to `main`.

```
feat/*  --PR-->  stable  --auto rollout PR-->  main  --tag--> GitHub Release
        (CI)            (chore(release): vX.Y.Z)     (GoReleaser)
```

## Branches

| Branch | Role | How code lands |
|--------|------|----------------|
| `feat/*`, `fix/*` | work | PR into `stable` (**merge commit**) |
| `stable` | next-release staging | PRs only |
| `main` | released; tags live here | rollout PR only — no direct pushes |

## How it works

1. **Work** on a `feat/*` branch → PR into `stable`. CI runs (test/lint/vuln/build).
2. **Merge to `stable`** → `rollout.yml` computes the next version from the
   commits since the last tag (git-cliff) and opens/updates a single rollout PR
   `stable → main`, titled `chore(release): vX.Y.Z`, body = the release notes
   preview. Every further push to `stable` refreshes it.
3. **Merge the rollout PR into `main`** → `release.yml` recomputes the version
   (authoritative), creates and pushes the `vX.Y.Z` tag, and runs GoReleaser to
   publish a GitHub Release with cross-platform archives + checksums. Release
   notes come from git-cliff — there is **no committed CHANGELOG file**.

## Versioning

Conventional Commits decide the bump (git-cliff, `cliff.toml`):

- `feat:` → **minor**
- `fix:` / `perf:` / others → **patch**
- `!` or `BREAKING CHANGE:` → **major**
- `chore:` / merge commits → ignored (no bump, excluded from notes)

Because `feat → stable` uses merge commits, `cliff.toml` skips merge commits and
reads the individual conventional commits underneath.

## First release (bootstrap)

There are no tags yet, so the auto path deliberately **skips** (it will not
invent a `v0.1.0`). Cut the first release once, manually:

- GitHub → Actions → **release** → **Run workflow** → version `v0.1.0`.

After that, every rollout-PR merge auto-computes the next version.

## Hotfixes

`main` lags `stable`, so hotfixes still go `fix/* → stable → main` (via the
rollout PR). No direct commits to `main`. For an urgent fix, land it on `stable`
and merge the refreshed rollout PR immediately.

## Manual release

Actions → **release** → **Run workflow**, optionally passing an explicit
`version`. Leave it blank to auto-compute.

## Required repo settings (one-time)

- Create the `stable` branch from `main`.
- Branch protection on `main` **and** `stable`: require the `test`, `lint`, and
  `build` checks + 1 review; restrict direct pushes.
- Enable **merge commits** for `feat → stable` PRs.
- Allow GitHub Actions to push tags (default `GITHUB_TOKEN` + `contents: write`
  already suffices; the release job only pushes a tag, never a branch commit).
