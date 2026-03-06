# dusk

[![CI](https://github.com/philoserf/dusk/actions/workflows/ci.yml/badge.svg)](https://github.com/philoserf/dusk/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/philoserf/dusk/v2.svg)](https://pkg.go.dev/github.com/philoserf/dusk/v2)

A single, zero-dependency Go package for astronomical calculations — sunrise/sunset, moonrise/moonset, twilight, lunar phase, and celestial coordinate conversions — based on Meeus's _Astronomical Algorithms_.

```bash
go get github.com/philoserf/dusk/v2
```

## Examples

### Sunrise and sunset

```go
package main

import (
	"fmt"
	"time"

	"github.com/philoserf/dusk/v2"
)

func main() {
	loc, _ := time.LoadLocation("America/Chicago")
	date := time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC)
	obs := dusk.Observer{Lat: 42.9634, Lon: -85.6681, Elev: 188.0, Loc: loc}

	sun, _ := dusk.SunriseSunset(date, obs)
	fmt.Printf("Sunrise: %s\n", sun.Rise.Format(time.Kitchen))
	fmt.Printf("Noon:    %s\n", sun.Noon.Format(time.Kitchen))
	fmt.Printf("Sunset:  %s\n", sun.Set.Format(time.Kitchen))
}
```

### Lunar phase

```go
phase := dusk.LunarPhase(time.Date(2024, 1, 18, 3, 0, 0, 0, time.UTC))
fmt.Printf("%s (%.0f%% illuminated, waxing: %t)\n", phase.Name, phase.Illumination, phase.Waxing)
```

### Civil twilight

```go
loc, _ := time.LoadLocation("America/Los_Angeles")
date := time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC)
obs := dusk.Observer{Lat: 47.6062, Lon: -122.3321, Elev: 58.0, Loc: loc}

tw, _ := dusk.CivilTwilight(date, obs)
fmt.Printf("Dusk:  %s\n", tw.Dusk.Format(time.Kitchen))
fmt.Printf("Dawn:  %s\n", tw.Dawn.Format(time.Kitchen))
```

### Object transit

```go
loc, _ := time.LoadLocation("America/Los_Angeles")
date := time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC)
obs := dusk.Observer{Lat: 47.6062, Lon: -122.3321, Elev: 58.0, Loc: loc}

// Betelgeuse: RA 88.7929°, Dec 7.4071°
betelgeuse := dusk.Equatorial{RA: 88.7929, Dec: 7.4071}
tr, err := dusk.ObjectTransit(date, betelgeuse, obs)
if err != nil {
	fmt.Println(err) // e.g., ErrCircumpolar or ErrNeverRises
	return
}
fmt.Printf("Rise:    %s\n", tr.Rise.Format(time.Kitchen))
fmt.Printf("Maximum: %s\n", tr.Maximum.Format(time.Kitchen))
fmt.Printf("Set:     %s\n", tr.Set.Format(time.Kitchen))
```

## API

### Solar

- `SunriseSunset` — sunrise, solar noon, sunset, and daylight duration
- `SolarPosition` — equatorial coordinates of the Sun

### Lunar

- `LunarPhase` — illumination, elongation, approximate age, waxing/waning, phase angle, and name
- `LunarEclipticPosition` — ecliptic longitude, latitude, and distance (km)
- `LunarPosition` — equatorial coordinates of the Moon
- `MoonriseMoonset` — moonrise and moonset times

### Twilight

- `CivilTwilight` — sun 6° below horizon
- `NauticalTwilight` — sun 12° below horizon
- `AstronomicalTwilight` — sun 18° below horizon

### Transit

- `ObjectTransit` — rise, set, transit maximum, and above-horizon duration for arbitrary equatorial coordinates

### Coordinates

- `EclipticToEquatorial` — ecliptic to equatorial conversion
- `EquatorialToHorizontal` — equatorial to horizontal conversion
- `HourAngle` — local hour angle from right ascension
- `AngularSeparation` — angular distance between two positions

### Epoch

- `JulianDate` — Julian date from `time.Time`
- `LocalSiderealTime` — LST for a given longitude

## Conventions

- All angles are in **degrees**. `Equatorial.RA` is in degrees (0–360); divide by 15 to convert to hours.
- `Observer.Lon` is **east-positive, west-negative** (e.g., New York is −74.006).
- Functions that can fail return `error`. Two sentinel errors distinguish polar edge cases: `ErrCircumpolar` (object always above the horizon) and `ErrNeverRises` (object never rises).
- A **zero-value `time.Time`** signals "event did not occur" (e.g., the Moon does not rise on a given day). Check with `.IsZero()`.
- Twilight functions return tonight's **Dusk** and tomorrow morning's **Dawn**. To get this morning's dawn, call with yesterday's date.

## Accuracy

Sunrise/sunset times are typically within 1–2 minutes of USNO data. Moonrise/moonset uses a simplified Meeus approach with a minute-by-minute altitude scan and can differ from USNO by up to several minutes. Lunar ecliptic position uses the full Meeus Chapter 47 periodic terms (100+ coefficients).

## Requirements

Go 1.24+. Zero dependencies.

## License

GPL-3.0. See [LICENSE](./LICENSE).

Originally created by [observerly](https://github.com/observerly/dusk). This fork includes bug fixes, algorithm improvements, and a complete v2 rewrite.
