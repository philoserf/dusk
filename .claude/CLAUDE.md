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

Single package at the repo root. Zero dependencies. Module path: `github.com/philoserf/dusk/v2`.

| File              | Domain                                                                   |
| ----------------- | ------------------------------------------------------------------------ |
| `trig.go`         | Degree-based trig wrappers, `clamp`, `mod360`/`mod24` normalization      |
| `epoch.go`        | Julian dates, sidereal time, nutation, obliquity, `ValidJulianDateRange` |
| `coord.go`        | Coordinate types, validation, ecliptic/equatorial/horizontal conversions |
| `solar.go`        | Sun position, sunrise/sunset, `computeSolarParams` shared helper         |
| `lunar.go`        | Moon position (Meeus tables), moonrise/moonset, phase                    |
| `lunar_tables.go` | Meeus Table 47.A/B coefficients                                          |
| `transit.go`      | Object rise/set/transit (analytical maximum) for equatorial coordinates  |
| `twilight.go`     | Civil, nautical, astronomical twilight (uses `computeSolarParams`)       |
| `stringer.go`     | `fmt.Stringer` implementations for all exported result types             |
| `doc.go`          | Package documentation                                                    |

## Key Conventions

- All angles in **degrees** (trig helpers in `trig.go` handle conversion)
- Angle normalization via `mod360()` and `mod24()` helpers
- Meeus algorithms preferred; `solarMeanAnomaly(J)` takes days, not centuries
- `*time.Location` parameter for functions that produce local times
- Zero-value `time.Time` signals "event did not occur" — check with `.IsZero()`
- `(float64, bool)` returns for calculations that may not apply
- `error` returns for bad input (nil location, out-of-range coordinates)
- Table-driven tests everywhere, expected values from USNO/Stellarium/Meeus

## Gotchas

- `Transit` uses zero-value `time.Time` (not pointers) — check `.IsZero()` not `!= nil`
- Moonrise/moonset iterates minute-by-minute (1440 iterations) — slow by design
- `solarHourAngle` returns `(float64, error)` — returns `ErrCircumpolar` (midnight sun) or `ErrNeverRises` (polar night)
- `solarHourAngle` takes `depression` (positive degrees below horizon) for twilight reuse; pass 0 for sunrise/sunset
- `AngularSeparation` takes `(ra1, dec1, ra2, dec2)` — RA-first to match `Equatorial{RA, Dec}` field order
- `LunarPhaseInfo.Waxing` distinguishes waxing (elongation 0-180) from waning; `DaysApprox` is symmetric
- `Observer.Elev` only affects sunrise/sunset and twilight — not moonrise or `ObjectTransit`

## Dependencies

None. Zero external dependencies.

## CI

GitHub Actions on push to main and PR to main (skips markdown-only changes). Runs vet, golangci-lint, and tests with race detector. Go 1.24 on Ubuntu.
