# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# Dusk

Go library for astronomical calculations: twilight, lunar phase, rise/set times.

Onboarding references: `THEORY.md` (Naur-style theory of the codebase) and `WALKTHROUGH.md` (linear code tour).
Both are hand-maintained prose — extend `THEORY.md` when a load-bearing idea changes, and regenerate
`WALKTHROUGH.md` with the `code-walkthrough` skill when the code it quotes moves. Its snippets are quoted
from the source by file and symbol, not by line range, and nothing in the gate checks them against the
files they came from: re-read it before tagging.

## Commands

Run `task --list` for the current set.

**CI runs exactly `task`.** Never add a check to CI that the local gate does not run,
and never add a tool to the gate without also installing it in the workflow. The Go
toolchain, golangci-lint and prettier are deliberately unpinned: a red gate on untouched
code is the signal working — answer the finding rather than pinning the tool.

Coverage is held by a **ratchet**, not a percentage: `coverage.ratchet` records the
count of uncovered statements per package, and `task ratchet` diffs the current counts
against it — so the gate fails in both directions and on a package appearing or
vanishing. A number that rises is lost coverage; one that falls is coverage to lock in
with `task ratchet:update`.

An integer rather than a percentage because a percentage holds still while a guarded
branch adds one covered statement and one uncovered, and it grows more forgiving as the
repository grows. The whole check is an awk program in `Taskfile.yml`; there is no tool
to maintain.

Reflowing blank lines splits coverage blocks, so a refactor can move these counts without
changing what the tests reach — read the diff before assuming a regression.

## Architecture

Library is a single package at the repo root, with a reference CLI under `cmd/dusk`. Zero external
dependencies — nothing outside the standard library, and the Meeus coefficient tables are transcribed
into the source rather than fetched. Module path: `github.com/philoserf/dusk/v4`.

| File        | Domain                                                                                                                                               |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| `dusk.go`   | Package doc, `Observer`/`NewObserver`, event types, `stringError`, all sentinels but one                                                             |
| `solar.go`  | `SunriseSunset`, civil/nautical/astronomical twilight, all unexported solar helpers                                                                  |
| `lunar.go`  | `MoonriseMoonset`, `LunarPhase`, unexported lunar helpers, Meeus Table 47.A/B coefficients                                                           |
| `epoch.go`  | Julian dates, sidereal time, nutation, obliquity — unexported apart from `ErrDateOutOfRange`, which lives beside the range check that returns it     |
| `coord.go`  | Coordinate conversions: `eclipticToEquatorial`, `altitudeOf`, `hourAngle`. Sits one layer above `epoch.go`; reached only by `MoonriseMoonset`'s scan |
| `trig.go`   | Degree-based trig wrappers, `clamp`, `mod360`/`mod24` normalization                                                                                  |
| `cmd/dusk/` | Reference CLI over the public API: `main.go` (flags, errors), `report.go` (assembly), `render.go` (text/JSON)                                        |

`cmd/dusk` is an executable specification, not a product: it calls every exported function, and every
documented edge case is reachable with a single flag. Its exit contract is part of that — polar geometry
is a result, so the report renders and exits 0; only a misused command line (`usage:`) and an
out-of-range date (`unsupported date:`) exit 1, told apart by the message rather than the status.

## Key Conventions

- All angles in **degrees** (trig helpers in `trig.go` handle conversion)
- Angle normalization via `mod360()` and `mod24()` helpers
- Longitude is **east-positive, west-negative** (New York is -74.006)
- Meeus algorithms preferred; `solarMeanAnomaly(J)` takes days, not centuries
- `Observer` constructed via `NewObserver` — validates once at creation, fields unexported
- **Callers must build dates in the observer's timezone.** Public entry points resolve the
  calendar day with `date.In(obs.loc)` and ignore time-of-day, so a `time.UTC` midnight
  selects the previous day for any observer west of Greenwich
- `LunarPhase` is the exception to the day regime: it takes an **instant**, not a day, and no
  `Observer` — phase is Sun-Earth-Moon geometry, so the observer is irrelevant
- Twilight functions return tonight's `Dusk` and **tomorrow morning's** `Dawn`. To get this
  morning's dawn, call with yesterday's date
- Zero-value `time.Time` signals "event did not occur" — check with `.IsZero()`
- `ErrCircumpolar` / `ErrNeverRises` for geometrically impossible events (polar)
- `error` returns for date out of range (validated at all public entry points)
- **Sentinel errors are `const`, not `var`** — declared as the unexported `stringError` string type
  in `dusk.go` so they cannot be reassigned. New sentinels follow that pattern, not `errors.New`
- Table-driven tests everywhere, expected values from USNO/Stellarium/Meeus. Tolerances that
  reference data actually supports: **2 minutes** for sunrise/sunset, **2 minutes** for civil
  twilight, **4 minutes** for nautical and astronomical twilight (a 12-18° depression
  amplifies declination error, so twilight does **not** inherit sunrise's figure),
  **1 minute** for moonrise/moonset (measured margin is tens of seconds since the
  parallax-corrected threshold landed), **1-2%** for lunar illumination
- **A pin is either a reference value or a regression pin, and the comment says which.**
  USNO's one-day service publishes sunrise/sunset and civil twilight only, so the nautical
  pin in `solar_test.go` is uncorroborated and labelled as such. Record the measured margin
  beside a tolerance so the next reader can tell a tight test from a lucky one
- **Errors are checked on their own line**, never inline: `err := f()` then `if err != nil`,
  not `if err := f(); err != nil`. Enforced by `noinlineerr`, matching the other Go repos
- **Every test calls `t.Parallel()`**, top level and subtest. Enforced by `paralleltest`.
  Accumulating state across parallel subtests is a race — check table-wide properties in
  their own sequential test instead

## Lint posture

`.golangci.yml` runs `default: all` and disables only what fights this repo's deliberate design.
Each disable carries a **measured finding count and a reason** — the file's own rule, and the bar for
adding another: count the findings, read them, and write down why they are wrong here.

- `nolintlint` requires a **specific** linter and an **explanation**, and fails on an unused
  `//nolint` — a blanket directive will not pass the gate
- `depguard` is `list-mode: strict`, allowing only `$gostd` and this module's own path
- `_test.go` relaxes the linters that table-driven, white-box tests trip (`dupl`, `goconst`,
  `funlen`, `gocognit`, `cyclop`, `lll`, `gosec`, `noctx`, `testpackage`, `gochecknoglobals`)
- Terse Meeus notation (`T`, `M`, `Lp`, `Mp`, `h0`) is permitted by name in the `varnamelen`
  ignore list and by disabling `gocritic`'s `captLocal`; a new one-letter name needs an entry
- gofumpt and goimports run **inside** golangci-lint, which is the single definition of formatted
  this repo has **for Go**. A `PostToolUse` hook in `.claude/settings.json` also runs `gofumpt -w`
  on Go file writes — but the Brewfile does not install gofumpt, so on a fresh machine that hook
  fails rather than formats (exit 127), and `task lint` is what catches the formatting
- **prettier is the same thing for every file that is not Go** — Markdown, JSON and YAML — run
  by `task docs` and configured by
  `.prettierrc.json`. `embeddedLanguageFormatting: "off"` is the load-bearing setting: prettier's
  default rewrites source inside fenced blocks, and this repository's documents quote their own
  compiled examples. `.prettierignore` names what is out of scope and why — `.golangci.yml`
  (prettier explodes the commented `varnamelen` flow sequence) and `.issues/` (hidden by the
  global `core.excludesfile`, not by this repo's `.gitignore`, so git skips it and prettier
  would not)

## Gotchas

- Moonrise/moonset iterates minute-by-minute (1440 iterations) — slow by design
- `solarHourAngle` returns `(float64, error)` — returns `ErrCircumpolar` (midnight sun) or `ErrNeverRises` (polar night)
- `solarHourAngle` takes `depression` (positive degrees below horizon) for twilight reuse; pass 0 for sunrise/sunset
- `LunarPhaseInfo.Waxing` distinguishes waxing (elongation 0-180) from waning; `DaysApprox` is a linear approximation
- `eclipticToEquatorial` applies full nutation (Δψ + Δε); `solarDeclination` uses mean obliquity only (intentional asymmetry — NOAA simplified method for sunrise/sunset)
- **The Moon's horizon threshold is _positive_ and the Sun's is negative.** Meeus's lunar
  `h0 = 0.7275·π − 0.5667` is about **+0.125°**, because horizontal parallax outweighs
  refraction for the one body close enough for it to matter; the Sun's is −0.833°. The
  scan therefore compares `altitude > h0`, not `altitude > -h0`. Using 0.833 for both —
  which this code did until v4.1.0 — biases every moonrise early and every moonset late by
  5–12 minutes. `h0` is also recomputed per sample from the Moon's true distance, so it is
  a function, not a constant
- **A reported moon crossing is interpolated, and lands in `[cur−1m, cur)`.** That
  half-open interval is an invariant, not an implementation detail: it is what keeps the
  scan's closed upper bound from filing an event under the next calendar day, and it is
  what `FuzzMoonriseMoonset` asserts. `crossingInstant` interpolates on the difference
  `altitude − h0`, never on altitude against a fixed threshold
- **An unexported function with a test is invisible to `unused`.** The gate cannot tell you
  when the last production caller disappears — this bit twice, with `solarPosition` and
  then `lunarPosition`. After deleting or rerouting a call site, `grep` for the callee

## Major version bumps

The `/vN` in the module path breaks `.golangci.yml` and `coverage.ratchet` until both
are hand-edited — see the `major-version-bump` skill.

## Releases

No release automation. `CHANGELOG.md` is written by hand. **Tag last**, by hand, after the gate
is green and `WALKTHROUGH.md` has been re-verified against the current source.

## CI

One GitHub Actions job that installs the toolchain and runs `task`. No paths-ignore: a
gate that skips a documentation-only push cannot catch a broken example, and the examples
here compile.
