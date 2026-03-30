# dusk

[![CI](https://github.com/philoserf/dusk/actions/workflows/ci.yml/badge.svg)](https://github.com/philoserf/dusk/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/philoserf/dusk/v3.svg)](https://pkg.go.dev/github.com/philoserf/dusk/v3)

A single, zero-dependency Go package for astronomical calculations — sunrise/sunset, moonrise/moonset, twilight, and lunar phase — based on Meeus's _Astronomical Algorithms_.

## Install

```bash
go get github.com/philoserf/dusk/v3
```

## Examples

### Sunrise and sunset

A complete program showing error handling and formatted output:

```go
package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/philoserf/dusk/v3"
)

func main() {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		log.Fatal(err)
	}

	obs, err := dusk.NewObserver(42.9634, -85.6681, loc)
	if err != nil {
		log.Fatal(err)
	}

	date := time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC)

	sun, err := dusk.SunriseSunset(date, obs)
	if err != nil {
		if errors.Is(err, dusk.ErrCircumpolar) {
			fmt.Println("Midnight sun — the sun does not set today.")
			return
		}
		if errors.Is(err, dusk.ErrNeverRises) {
			fmt.Println("Polar night — the sun does not rise today.")
			return
		}
		log.Fatal(err)
	}

	fmt.Printf("Sunrise:  %s\n", sun.Rise.Format(time.Kitchen))
	fmt.Printf("Noon:     %s\n", sun.Noon.Format(time.Kitchen))
	fmt.Printf("Sunset:   %s\n", sun.Set.Format(time.Kitchen))
	fmt.Printf("Daylight: %s\n", sun.Duration)
}
```

### Moonrise and moonset

The Moon may not rise or set on a given day. Use `IsZero()` to check, and `AboveHorizon` to determine whether the Moon was up at the start of the day:

```go
moon, err := dusk.MoonriseMoonset(date, obs)
if err != nil {
	log.Fatal(err)
}

if moon.Rise.IsZero() {
	if moon.AboveHorizon {
		fmt.Println("Moon was already up at midnight and does not set today.")
	} else {
		fmt.Println("Moon does not rise today.")
	}
} else {
	fmt.Printf("Moonrise: %s\n", moon.Rise.Format(time.Kitchen))
}

if !moon.Set.IsZero() {
	fmt.Printf("Moonset:  %s\n", moon.Set.Format(time.Kitchen))
}
```

### Lunar phase

All result types implement `fmt.Stringer`. Printing a `LunarPhaseInfo` value directly produces output like `Waxing Gibbous 67.3% (day 10.1)`:

```go
phase, err := dusk.LunarPhase(time.Date(2024, 1, 18, 3, 0, 0, 0, time.UTC))
if err != nil {
	log.Fatal(err)
}

fmt.Println(phase) // e.g., "Waxing Gibbous 67.3% (day 10.1)"
fmt.Printf("Illumination: %.1f%%  Waxing: %t\n", phase.Illumination, phase.Waxing)
```

### Civil twilight

Twilight functions return tonight's **Dusk** and tomorrow morning's **Dawn**. To get _this morning's_ dawn, call with yesterday's date:

```go
loc, err := time.LoadLocation("America/Los_Angeles")
if err != nil {
	log.Fatal(err)
}

obs, err := dusk.NewObserver(47.6062, -122.3321, loc)
if err != nil {
	log.Fatal(err)
}

date := time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC)

tw, err := dusk.CivilTwilight(date, obs)
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Dusk:           %s\n", tw.Dusk.Format(time.Kitchen))
fmt.Printf("Dawn:           %s\n", tw.Dawn.Format(time.Kitchen))
fmt.Printf("Night duration: %s\n", tw.NightDuration)
```

`NauticalTwilight` and `AstronomicalTwilight` follow the same signature.

### Polar error handling

At extreme latitudes, sunrise/sunset and twilight may be geometrically impossible. Use `errors.Is` to match the sentinel errors:

```go
loc, err := time.LoadLocation("Arctic/Longyearbyen")
if err != nil {
	log.Fatal(err)
}

obs, err := dusk.NewObserver(78.2, 15.6, loc) // Svalbard
if err != nil {
	log.Fatal(err)
}

midsummer := time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC)

_, err = dusk.SunriseSunset(midsummer, obs)
if errors.Is(err, dusk.ErrCircumpolar) {
	fmt.Println("Midnight sun — no sunset at this latitude today.")
}
if errors.Is(err, dusk.ErrNeverRises) {
	fmt.Println("Polar night — no sunrise at this latitude today.")
}
```

## API

### Solar

- `SunriseSunset(date, obs)` — sunrise, solar noon, sunset, and daylight duration

### Lunar

- `MoonriseMoonset(date, obs)` — moonrise/moonset times and whether the Moon was above the horizon at the start of the day
- `LunarPhase(date)` — illumination, elongation, approximate age, waxing/waning, phase angle, and name

### Twilight

- `CivilTwilight(date, obs)` — sun 6 degrees below horizon
- `NauticalTwilight(date, obs)` — sun 12 degrees below horizon
- `AstronomicalTwilight(date, obs)` — sun 18 degrees below horizon

### Observer

- `NewObserver(lat, lon, loc)` — create a validated observer from latitude, longitude, and timezone

### Result types

All result types implement `fmt.Stringer`:

- `SunEvent` — `Rise`, `Noon`, `Set` times and `Duration` (daylight)
- `MoonEvent` — `Rise`, `Set` times, `Duration`, and `AboveHorizon`
- `TwilightEvent` — `Dusk`, `Dawn` times and `NightDuration` (overnight darkness)
- `LunarPhaseInfo` — `Illumination`, `Elongation`, `Angle`, `DaysApprox`, `Waxing`, `Name`

### Errors

- `ErrCircumpolar` — object always above the horizon (e.g., midnight sun)
- `ErrNeverRises` — object never rises (e.g., polar night)

## Conventions

- All angles are in **degrees**.
- Longitude is **east-positive, west-negative** (e.g., New York is -74.006).
- `Observer` is constructed via `NewObserver`, which validates coordinates and rejects NaN/Inf.
- Functions that can fail return `error`. Two sentinel errors distinguish polar edge cases: `ErrCircumpolar` and `ErrNeverRises`.
- A **zero-value `time.Time`** signals "event did not occur" (e.g., the Moon does not rise on a given day). Check with `.IsZero()`.
- Twilight functions return tonight's **Dusk** and tomorrow morning's **Dawn**. To get this morning's dawn, call with yesterday's date.

## Accuracy

Sunrise/sunset times are typically within 1-2 minutes of USNO data. Moonrise/moonset uses a simplified Meeus approach with a minute-by-minute altitude scan and can differ from USNO by up to ~20 minutes. Lunar phase illumination is within 1-2% of published values. Lunar ecliptic position uses the full Meeus Chapter 47 periodic terms (100+ coefficients).

## Requirements

Go 1.24+. Zero dependencies.

## License

GPL-3.0. See [LICENSE](./LICENSE).

Originally created by [observerly](https://github.com/observerly/dusk). This fork includes bug fixes, algorithm improvements, and a complete rewrite.
