# Dusk

![Go version](https://img.shields.io/github/go-mod/go-version/philoserf/dusk/main?filename=go.mod&label=Go)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/philoserf/dusk)](https://pkg.go.dev/github.com/philoserf/dusk)
[![Go Report Card](https://goreportcard.com/badge/github.com/philoserf/dusk)](https://goreportcard.com/report/github.com/philoserf/dusk)
[![CI](https://github.com/philoserf/dusk/actions/workflows/ci.yml/badge.svg)](https://github.com/philoserf/dusk/actions/workflows/ci.yml)

Dusk is a Go library for astronomical calculations: twilight times, sunrise/sunset, lunar phase, moon position, and rise/set times for arbitrary celestial objects. Single external dependency.

```bash
go get github.com/philoserf/dusk
```

## Quick Start

Calculate astronomical twilight for Seattle, WA:

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/philoserf/dusk"
)

func main() {
	datetime := time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC)

	// Seattle, WA — longitude west is negative, latitude north is positive
	twilight, _, err := dusk.GetLocalAstronomicalTwilight(datetime, -122.3321, 47.6062, 58.0)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Sunset: %s\n", twilight.Until.Format(time.RFC3339))
	fmt.Printf("Sunrise: %s\n", twilight.From.Format(time.RFC3339))
	fmt.Printf("Dark sky: %s\n", twilight.Duration)
}
```

## Twilight

`GetLocalTwilight` computes rise and set times for a given sun angle below the horizon. Three convenience wrappers cover the standard definitions:

| Function                       | Sun angle | Use case                      |
| ------------------------------ | --------- | ----------------------------- |
| `GetLocalCivilTwilight`        | -6°       | Outdoor activities, headlamps |
| `GetLocalNauticalTwilight`     | -12°      | Horizon visible at sea        |
| `GetLocalAstronomicalTwilight` | -18°      | Deep-sky observation          |

All take `(datetime, longitude, latitude, elevation)` and return `(Twilight, *time.Location, error)`.

For a custom angle, call `GetLocalTwilight` directly with a fifth parameter `degreesBelowHorizon` (negative value, e.g., `-6.0`).

## Lunar

### Position

```go
datetime := time.Date(2025, 6, 21, 22, 0, 0, 0, time.UTC)

eq := dusk.GetLunarEquatorialPosition(datetime)
fmt.Printf("RA: %.4f°  Dec: %.4f°\n", eq.RightAscension, eq.Declination)
```

### Phase

```go
ec := dusk.GetLunarEclipticPosition(datetime)
phase := dusk.GetLunarPhase(datetime, -122.3321, ec)
fmt.Printf("Age: %.1f days  Illumination: %.1f%%\n", phase.Days, phase.Illumination)
```

## Solar

```go
eq := dusk.GetSolarEquatorialPosition(datetime)
fmt.Printf("Sun RA: %.4f°  Dec: %.4f°\n", eq.RightAscension, eq.Declination)
```

## Object Transit

Calculate rise, transit, and set times for any object given its equatorial coordinates:

```go
// Betelgeuse: RA 88.7929°, Dec 7.4071°
transit := dusk.GetObjectTransit(datetime, -122.3321, 47.6062, 58.0, 88.7929, 7.4071)

if transit.Rise != nil {
	fmt.Printf("Rise: %s\n", transit.Rise.Format(time.RFC3339))
}
if transit.Set != nil {
	fmt.Printf("Set: %s\n", transit.Set.Format(time.RFC3339))
}
```

## API Overview

The full API is documented at [pkg.go.dev](https://pkg.go.dev/github.com/philoserf/dusk). Key function groups:

| Domain      | Functions                                                                                               |
| ----------- | ------------------------------------------------------------------------------------------------------- |
| Twilight    | `GetLocalTwilight`, `GetLocalCivilTwilight`, `GetLocalNauticalTwilight`, `GetLocalAstronomicalTwilight` |
| Solar       | `GetSolarEquatorialPosition`, `GetSolarEclipticPosition`, `GetSolarMeanAnomaly`                         |
| Lunar       | `GetLunarEquatorialPosition`, `GetLunarEclipticPosition`, `GetLunarPhase`                               |
| Transit     | `GetObjectTransit`, `GetObjectTransitMaximaTime`                                                        |
| Coordinates | `ConvertEquatorialToHorizontal`, `ConvertEclipticToEquatorial`                                          |
| Astrometry  | `GetHourAngle`, `GetAngularSeparation`                                                                  |
| Epoch       | `GetJulianDate`, `GetLocalSiderealTime`, `GetGreenwichSiderealTime`                                     |

## Conventions

- All angles are in **degrees** (internal trig helpers handle radian conversion)
- Longitude: west negative, east positive
- Latitude: south negative, north positive
- Elevation: meters above mean sea level
- Two algorithm families: **Meeus** (primary) and **Lawrence** (functions suffixed `Lawrence`)

## License

Dusk is free software licensed under the GNU General Public License v3.0 (GPL-3.0). See [LICENSE](./LICENSE).

Originally created by [observerly](https://github.com/observerly). This fork includes bug fixes, correctness improvements, and structural changes.

| Attribution                                                           | License |
| --------------------------------------------------------------------- | ------- |
| [observerly/dusk](https://github.com/observerly/dusk)                 | GPL-3.0 |
| [zsefvlol/timezonemapper](https://github.com/zsefvlol/timezonemapper) | MIT     |
