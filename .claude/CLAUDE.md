# Dusk

Go library for astronomical calculations: twilight, lunar phase, rise/set times.

## Commands

```bash
task              # Run all checks (fmt, vet, lint, test)
task fix          # Run all fixes (fmt, lint)
task test         # Run tests only
go test -v -run TestName ./...  # Run single test
```

## Architecture

Single package at the repo root. Zero dependencies. Module path: `github.com/philoserf/dusk/v3`.

| File       | Domain                                                                                     |
| ---------- | ------------------------------------------------------------------------------------------ |
| `dusk.go`  | Package doc, `Observer`/`NewObserver`, event types, sentinel errors, `String()` methods    |
| `solar.go` | `SunriseSunset`, civil/nautical/astronomical twilight, all unexported solar helpers        |
| `lunar.go` | `MoonriseMoonset`, `LunarPhase`, unexported lunar helpers, Meeus Table 47.A/B coefficients |
| `epoch.go` | Julian dates, sidereal time, nutation, obliquity, coordinate conversions (all unexported)  |
| `trig.go`  | Degree-based trig wrappers, `clamp`, `mod360`/`mod24` normalization                        |

## Key Conventions

- All angles in **degrees** (trig helpers in `trig.go` handle conversion)
- Angle normalization via `mod360()` and `mod24()` helpers
- Meeus algorithms preferred; `solarMeanAnomaly(J)` takes days, not centuries
- `Observer` constructed via `NewObserver` — validates once at creation, fields unexported
- Zero-value `time.Time` signals "event did not occur" — check with `.IsZero()`
- `ErrCircumpolar` / `ErrNeverRises` for geometrically impossible events (polar)
- `error` returns for date out of range (validated at public entry points, except `LunarPhase` — no error return since it cannot fail geometrically)
- Table-driven tests everywhere, expected values from USNO/Stellarium/Meeus

## Gotchas

- Moonrise/moonset iterates minute-by-minute (1440 iterations) — slow by design
- `solarHourAngle` returns `(float64, error)` — returns `ErrCircumpolar` (midnight sun) or `ErrNeverRises` (polar night)
- `solarHourAngle` takes `depression` (positive degrees below horizon) for twilight reuse; pass 0 for sunrise/sunset
- `LunarPhaseInfo.Waxing` distinguishes waxing (elongation 0-180) from waning; `DaysApprox` is a linear approximation
- `eclipticToEquatorial` applies full nutation (Δψ + Δε); `solarDeclination` uses mean obliquity only (intentional asymmetry — NOAA simplified method for sunrise/sunset)

## Dependencies

None. Zero external dependencies.

## CI

GitHub Actions on push to main and PR to main (skips markdown-only changes). Runs vet, golangci-lint, and tests with race detector. Go 1.24 on Ubuntu.
