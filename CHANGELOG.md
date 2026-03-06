# Changelog

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
