# Dusk

Go library for astronomical calculations: twilight, lunar phase, rise/set times.

## Commands

```bash
go test -count=1 ./...        # Run all tests
go test -v -run TestName ./... # Run single test
go vet ./...                   # Static analysis
```

## Architecture

Single package at the repo root.

| File              | Domain                                                          |
| ----------------- | --------------------------------------------------------------- |
| `epoch.go`        | Julian dates, sidereal time, epoch conversions                  |
| `solar.go`        | Sun position, hour angle, sunrise/sunset                        |
| `lunar.go`        | Moon position (Meeus tables), moonrise/moonset, phase           |
| `lawrence.go`     | Alternative algorithms from Lawrence textbook                   |
| `transit.go`      | Object rise/set/transit for arbitrary equatorial coordinates    |
| `twilight.go`     | Civil, nautical, astronomical twilight wrappers                 |
| `coordinates.go`  | Coordinate types and ecliptic/equatorial/horizontal conversions |
| `astrometry.go`   | Hour angle, angular separation                                  |
| `utils.go`        | Obliquity, nutation, refraction, air mass                       |
| `trigonometry.go` | Degree-based trig wrappers (sinx, cosx, etc.)                   |

## Key Conventions

- All angles in **degrees** (trig helpers in `trigonometry.go` handle conversion)
- Negative angle correction: every `math.Mod(x, 360)` must be followed by `if x < 0 { x += 360 }`
- Two algorithm families coexist:
  - **Meeus** (e.g., `GetLunarEclipticPosition`): parameters in Julian **centuries**
  - **Lawrence** (e.g., `GetSolarMeanAnomalyLawrence`): parameters in Julian **days**
- `GetSolarMeanAnomaly()` uses the daily-rate coefficient (0.98560028) — pass **days**, not centuries

## Gotchas

- `time.Time` pointers in `Transit` struct — always nil-check both `Rise` and `Set`
- `GetLunarHorizontalCoordinatesForDay` allocates 1442 entries and iterates minute-by-minute — slow by design
- Twilight `degreesBelowHorizon` parameter is negative (e.g., -6 for civil) despite the name
- Tests use package-level vars `d`, `datetime`, `longitude`, `latitude`, `elevation` defined across `solar_test.go` and `epoch_test.go`

## Dependencies

Single external dep: `github.com/zsefvlol/timezonemapper` (lat/lng to IANA timezone)

## CI

GitHub Actions on PR to main (skips markdown-only changes). Matrix: Go 1.20-1.22 on Ubuntu.
