# Development

Developer workflow, tooling, and CI/CD for mdit.

## Quick start

```bash
make setup      # install dev tools (golangci-lint, lefthook, govulncheck) + git hooks
make check      # fmt-check + vet + lint + test — the local gate before pushing
make run ARGS=notes/   # run mdit against a vault
```

Run `make` with no target to list every task.

## Common tasks

| Command | What it does |
|---------|--------------|
| `make build` | Build `bin/mdit` with a version stamp (`git describe`) |
| `make run ARGS=...` | Run from source (`go run`) |
| `make test` | Run the suite |
| `make test-race` | Run with the race detector (needs cgo) |
| `make cover` / `cover-html` | Coverage total / HTML report |
| `make fmt` / `fmt-check` | Format / verify gofmt-clean |
| `make vet` / `lint` | `go vet` / golangci-lint |
| `make vuln` | `govulncheck` vulnerability scan |
| `make tidy` | `go mod tidy` + `verify` |
| `make check` | Fast local gate (mirrors required CI checks) |
| `make ci` | Full pipeline mirror (adds race, coverage, vuln) |

## Git hooks (lefthook)

`make hooks` (or `lefthook install`) wires up `lefthook.yml`:

- **pre-commit** — gofmt (auto-fix + restage), `go vet`, golangci-lint on staged Go files (parallel).
- **pre-push** — full `go test ./...`.

Bypass in a pinch with `LEFTHOOK=0 git commit ...` — CI still enforces everything.

## CI (`.github/workflows/ci.yml`)

Runs on pushes to `main` and on every PR. Jobs run in parallel; superseded runs
on the same ref are cancelled.

| Job | Purpose |
|-----|---------|
| `test` | `go test` + coverage on Linux/macOS/Windows matrix |
| `race` | race detector (Linux) |
| `lint` | gofmt check, `go vet`, golangci-lint |
| `vuln` | `govulncheck` |
| `build` | verify a version-stamped static binary builds |

## Releases (`.github/workflows/release.yml`)

Tag-driven with GoReleaser. Push a semver tag and CI builds cross-platform
archives + checksums and cuts a GitHub Release:

```bash
git tag v0.2.0 && git push origin v0.2.0
```

Dry-run locally: `goreleaser release --snapshot --clean`.

## Dependencies

Dependabot (`.github/dependabot.yml`) opens weekly grouped PRs for Go modules
and GitHub Actions. CI gates them like any other PR.

## Recommendations / roadmap

Implemented here: parallel CI, gofmt/vet/vuln gates, race + coverage, run-cancel
concurrency, tag-driven releases, dependabot, local↔CI parity via `make`.

Worth adding as the project grows:

- **Branch protection** on `main`: require the `test`, `lint`, and `build`
  checks + 1 review before merge.
- **Pin tool versions.** `latest` is used for portability; once confirmed, pin
  `golangci-lint`/`lefthook`/`govulncheck` (Makefile) and the lint action to
  exact tags so local == CI is reproducible. Dependabot then bumps them.
- **Coverage reporting** (Codecov/Coveralls) with a PR delta comment and a
  minimum threshold, instead of only uploading the artifact.
- **Merge queue** once contributor volume makes serialized merges worthwhile.
- **Expand linters.** `.golangci.yml` enables only errcheck + staticcheck; add
  `govet`, `ineffassign`, `unused`, `misspell`, `revive` incrementally.
- **SLSA / provenance + Cosign signing** of release artifacts for supply-chain
  integrity.
- **CodeQL** workflow for security scanning on a schedule.
