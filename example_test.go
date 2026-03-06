package dusk_test

import (
	"fmt"
	"time"

	"github.com/philoserf/dusk/v2"
)

func ExampleSunriseSunset() {
	loc, _ := time.LoadLocation("America/Chicago")
	date := time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC)
	obs := dusk.Observer{Lat: 42.9634, Lon: -85.6681, Elev: 188.0, Loc: loc}

	sun, _ := dusk.SunriseSunset(date, obs)
	fmt.Printf("Sunrise: %s\n", sun.Rise.Format("15:04"))
	fmt.Printf("Sunset:  %s\n", sun.Set.Format("15:04"))
	// Output:
	// Sunrise: 05:06
	// Sunset:  20:22
}

func ExampleLunarPhase() {
	t := time.Date(2024, 1, 25, 18, 0, 0, 0, time.UTC)
	phase := dusk.LunarPhase(t)
	fmt.Printf("Phase: %s\n", phase.Name)
	fmt.Printf("Illumination: %.0f%%\n", phase.Illumination)
	// Output:
	// Phase: Full Moon
	// Illumination: 100%
}

func ExampleCivilTwilight() {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	date := time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC)
	obs := dusk.Observer{Lat: 47.6062, Lon: -122.3321, Elev: 58.0, Loc: loc}

	tw, _ := dusk.CivilTwilight(date, obs)
	fmt.Printf("Dusk: %s\n", tw.Dusk.Format("15:04"))
	fmt.Printf("Dawn: %s\n", tw.Dawn.Format("15:04"))
	// Output:
	// Dusk: 21:49
	// Dawn: 04:33
}

func ExampleSolarPosition() {
	t := time.Date(2024, 6, 21, 12, 0, 0, 0, time.UTC)
	pos := dusk.SolarPosition(t)
	fmt.Printf("RA: %.1f°\n", pos.RA)
	fmt.Printf("Dec: %.1f°\n", pos.Dec)
	// Output:
	// RA: 90.2°
	// Dec: 23.4°
}

func ExampleMoonriseMoonset() {
	loc, _ := time.LoadLocation("America/New_York")
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, loc)
	obs := dusk.Observer{Lat: 40.7128, Lon: -74.0060, Loc: loc}

	evt, _ := dusk.MoonriseMoonset(date, obs)
	if !evt.Rise.IsZero() {
		fmt.Printf("Moonrise: %s\n", evt.Rise.Format("15:04"))
	}
	if !evt.Set.IsZero() {
		fmt.Printf("Moonset:  %s\n", evt.Set.Format("15:04"))
	}
	// Output:
	// Moonrise: 10:06
	// Moonset:  22:13
}

func ExampleObjectTransit() {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	date := time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC)
	obs := dusk.Observer{Lat: 47.6062, Lon: -122.3321, Elev: 58.0, Loc: loc}

	// Betelgeuse: RA 88.7929°, Dec 7.4071°
	betelgeuse := dusk.Equatorial{RA: 88.7929, Dec: 7.4071}
	tr, err := dusk.ObjectTransit(date, betelgeuse, obs)
	if err != nil {
		fmt.Printf("Error: %v\n", err) // e.g., ErrCircumpolar or ErrNeverRises
		return
	}
	fmt.Printf("Rise: %s\n", tr.Rise.Format("15:04"))
	// Output:
	// Rise: 06:31
}
