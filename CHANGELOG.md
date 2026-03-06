# Changelog

## v2.0.0

### Breaking Changes

- All local-time functions now accept a `*time.Location` parameter instead of deriving timezone from latitude/longitude
- Removed error returns from functions where timezone lookup was the only error source:
  - `GetSunriseSunsetTimes` returns `Sun` (was `(Sun, error)`)
  - `GetLocalTwilight` returns `(Twilight, *time.Location)` (was `(Twilight, time.Location, error)`)
  - `GetLocalCivilTwilight`, `GetLocalNauticalTwilight`, `GetLocalAstronomicalTwilight` return `(Twilight, *time.Location)` (was `(Twilight, time.Location, error)`)
  - `GetObjectHorizontalCoordinatesForDay` returns `[]TransitHorizontalCoordinate` (was `([]TransitHorizontalCoordinate, error)`)
  - `GetObjectRiseObjectSetTimes` returns `*Transit` (was `(*Transit, error)`)
  - `GetObjectTransitMaximaTime` returns `*time.Time` (was `(*time.Time, error)`)
  - `GetObjectTransit` returns `*Transit` (was `(*Transit, error)`)
  - `GetLunarHorizontalCoordinatesForDay` returns `[]TransitHorizontalCoordinate` (was `([]TransitHorizontalCoordinate, error)`)

### Removed

- `zsefvlol/timezonemapper` dependency — the library now has zero external dependencies

## v1.0.0

Initial fork from [observerly/dusk](https://github.com/observerly/dusk). Bug fixes, correctness improvements, and structural changes.
