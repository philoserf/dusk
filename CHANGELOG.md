# Changelog

## v3.0.0 — 2026-03-30

### Breaking changes

- **Module path** changed from `github.com/philoserf/dusk/v2` to `github.com/philoserf/dusk/v3`
- **`NewObserver` constructor** replaces direct struct construction — validates at creation, rejects NaN/Inf/nil
- **`Observer` fields unexported** — `Lat`/`Lon`/`Loc` → `lat`/`lon`/`loc`; use `NewObserver` to construct
- **Elevation removed** — `Observer.Elev` field deleted; elevation correction (~0.5' for typical altitudes) dropped from `solarHourAngle` for simplicity
- **Public API surface reduced** — removed `ObjectTransit`, `Transit`, `SolarPosition`, `LunarPosition`, `LunarEclipticPosition`, `EclipticToEquatorial`, `EquatorialToHorizontal`, `HourAngle`, `AngularSeparation`, `JulianDate`, `ValidJulianDateRange`, `LocalSiderealTime`, `ErrDateOutOfRange`
- **Coordinate types unexported** — `Equatorial`, `Horizontal`, `Ecliptic` → `equatorial`, `horizontal`, `ecliptic`

### Improvements

- `eclipticToEquatorial` now applies full nutation (both Δψ and Δε), improving RA accuracy by up to ~17"
- Observer validation (NaN/Inf rejection) happens once at construction, not repeated in every function call
- Date range validation (`validJulianDateRange`) at all public entry points
- Lunar table loops include `default` panic for unexpected M values (compile-time safety net)

### File consolidation

10 source files → 5: `dusk.go`, `solar.go`, `lunar.go`, `epoch.go`, `trig.go`. Twilight merged into solar, lunar tables merged into lunar, coordinate conversions merged into epoch, types/stringers/errors consolidated into dusk.

## v2.2.0 — 2026-03-30

### Bug fixes

- `DaysApprox` now uses a linear elongation-to-days formula matching the "days into lunation" documentation (#36)
- `solarHourAngle` no longer applies elevation correction to twilight calculations, per USNO convention (#39)
- Example tests handle errors from `time.LoadLocation` and computation functions instead of discarding (#43)

### Testing

- Fuzz tests assert error returns for out-of-range coordinates instead of silently skipping (#37)
- `TestLunarPhase` now asserts on `Angle` and `Elongation` ranges for all 8 phase test cases (#41)

### CI

- Enforce `gofumpt` formatting via `golangci-lint` formatters config (#38)

### Documentation

- `HourAngle` doc comment clarifies mixed-unit parameters (RA in degrees, LST in hours) (#40)
- `MoonriseMoonset` doc comment notes ~1-2 ms wall-clock cost per call (#42)

## v2.1.0 — 2026-03-10

### New features

- `fmt.Stringer` interface on all 8 exported result types (`SunEvent`, `MoonEvent`, `LunarPhaseInfo`, `Transit`, `TwilightEvent`, `Equatorial`, `Ecliptic`, `Horizontal`)
- `ValidJulianDateRange` helper and `ErrDateOutOfRange` sentinel for guarding `JulianDate` range (~1677–2262)

### Bug fixes

- `asinx`/`acosx` now clamp inputs to [-1, 1] via `clamp()`, preventing NaN from floating-point rounding
- `validateEquatorial` normalizes RA via `mod360` instead of rejecting values at 360.0; rejects NaN/Inf inputs

### Performance

- `ObjectTransit` transit maximum: replaced O(n) minute-by-minute scan with O(1) analytical solution (hour angle = 0)

### Refactoring

- Extracted `computeSolarParams` helper eliminating 3× duplication of the 6-step solar parameter sequence; `twilight()` reduced from 44 to 29 lines
- Renamed shadowed variable `F` → `frac` in `LunarPhase`

### Testing

- Fuzz tests for `SunriseSunset`, `LunarPhase`, `ObjectTransit`, `MoonriseMoonset`
- Benchmarks for `MoonriseMoonset`, `LunarEclipticPosition`, `SunriseSunset`, `ObjectTransit` (0 allocations)
- Polar twilight transition test (75°N, Nov 25→26)
- Negative elevation clamping, summer solstice solar position, GMST/julianCentury helper tests
- Southern hemisphere moonrise/moonset regression references
- Consistent `t.Run` subtests for `TestMod360`/`TestMod24`; table-driven `TestEclipticToEquatorial`

### Infrastructure

- `.golangci.yml` with explicit linter list (gocritic, revive, misspell, etc.); `captLocal` disabled for Meeus conventions
- 80% coverage threshold in CI (currently 99.7%); removed duplicate `go vet` step

### Documentation

- `Observer.Elev` doc comment: negative value clamping, scope (sunrise/sunset/twilight only)
- `gstToUT` precision note about J1900-epoch model accuracy

## v2.0.0 — 2026-03-06

Complete rewrite of the library. Zero external dependencies.

### Breaking changes

- **Module path** changed from `github.com/philoserf/dusk` to `github.com/philoserf/dusk/v2`
- **Removed `timezonemapper` dependency** — callers pass `*time.Location` explicitly via the new `Observer` struct
- **Renamed all exported functions** — `Get` prefix removed (e.g., `GetJulianDate` → `JulianDate`, `GetLocalSiderealTime` → `LocalSiderealTime`)
- **Replaced coordinate types** — `Coordinate`, `EquatorialCoordinate`, `EclipticCoordinate`, `HorizontalCoordinate` replaced by `Equatorial`, `Ecliptic`, `Horizontal` with named fields (`RA`/`Dec`, `Lon`/`Lat`, `Alt`/`Az`)
- **New `Observer` struct** — replaces separate `latitude`, `longitude` parameters; includes `Loc *time.Location` and `Elev float64`
- **New event structs** — `SunEvent`, `MoonEvent`, `TwilightEvent`, `Transit` replace raw time returns and ad-hoc structs
- **Removed exports** — `JulianPeriod`, `TemporalHorizontalCoordinate`, `TransitHorizontalCoordinate`, `GetEarthObliquity`, `GetUniversalTime`, `GetCurrentJulianPeriod`, `GetMeanSolarTime`, `GetDatetimeZeroHour`, `ConvertLocalSiderealTimeToGreenwichSiderealTime`, `ConvertGreenwichSiderealTimeToUniversalTime`, `GetAtmosphericRefraction`, `GetRelativeAirMass`, `GetApparentAltitude`, `GetArgumentOfLocalSiderealTimeForTransit`, `SunriseStatus`, `AboveHorizon`, `AtHorizon`, `BelowHorizon`, and others
- **Removed files** — `astrometry.go`, `coordinates.go`, `trigonometry.go`, `utils.go`, `lawrence.go` consolidated into domain-focused files

### New features

- `LunarPhase` — illumination, age in days, waxing/waning, and phase name
- `LunarEclipticPosition` — Meeus Chapter 47 ecliptic position with full periodic terms
- `MoonriseMoonset` — moonrise and moonset times via minute-by-minute altitude scan
- `ObjectTransit` — rise, set, and transit maximum for arbitrary equatorial coordinates
- `CivilTwilight`, `NauticalTwilight`, `AstronomicalTwilight` — twilight dusk/dawn pairs
- `ErrCircumpolar` and `ErrNeverRises` sentinel errors for polar edge cases
- `AngularSeparation` — robust atan2-based formula

### Improvements

- Meeus algorithms throughout (solar mean anomaly, lunar ecliptic, nutation, obliquity)
- Degree-based trig helpers (`sinx`, `cosx`, etc.) eliminate manual conversion
- `mod360`/`mod24` normalization helpers
- Zero-value `time.Time` convention for events that don't occur (circumpolar, never rises)
- High test coverage with table-driven tests and values from USNO, Stellarium, and Meeus
- Edge-case tests for equatorial, polar, and southern hemisphere observers

## v1.0.0

Initial release. Forked from [observerly/dusk](https://github.com/observerly/dusk).
