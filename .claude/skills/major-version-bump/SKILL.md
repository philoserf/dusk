---
description: Bumping the module's major version (the /vN in the module path). Two config files break the gate loudly until they are hand-edited. Use when changing the major version or module path.
allowed-tools:
  - Read
  - Edit
  - Bash
---

# Major version bumps

The `/vN` in the module path is load-bearing in two config files that a `go mod edit` will not touch,
and both turn the gate red until they are updated by hand:

- `.golangci.yml` — the `depguard` allow-list names `github.com/philoserf/dusk/v4`. Under
  `list-mode: strict` the new path is not allowed, so every internal import is reported as forbidden
  (measured: 5 findings across `cmd/dusk` and `example_test.go`) — loudly, naming each import
- `coverage.ratchet` — the keys are full import paths. The ratchet reads the old packages as vanished
  and the new ones as appeared, so it fails even when coverage is unchanged; re-record with
  `task ratchet:update` once the path is right

Also update the imports in `cmd/dusk` and `example_test.go`, the root `CLAUDE.md`'s Architecture
line, and the README's badge, `go get`, and `go install` lines.
