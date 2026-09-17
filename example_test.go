package dusk_test

import (
	"fmt"
	"math"
	"time"

	"github.com/philoserf/dusk/v5"
)

func ExampleNewObserver() {
	// Valid observer
	obs, err := dusk.NewObserver(40.7128, -74.006, time.UTC)
	if err != nil {
		fmt.Println("unexpected error:", err)

		return
	}

	fmt.Println(obs)

	// Invalid: latitude out of range
	_, err = dusk.NewObserver(91, 0, time.UTC)
	fmt.Println(err)

	// Invalid: NaN
	_, err = dusk.NewObserver(math.NaN(), 0, time.UTC)
	fmt.Println(err)
	// Output:
	// 40.7128°, -74.0060° (UTC)
	// dusk: latitude must be in [-90, 90] and longitude in [-180, 180]
	// dusk: coordinates must be finite (NaN and Inf are not allowed)
}

func ExampleSunriseSunset_polar() {
	loc, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	obs, err := dusk.NewObserver(69.65, 18.96, loc)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	// Tromsø on June 21 — midnight sun
	date := dusk.Date{Year: 2024, Month: 6, Day: 21}

	summer, err := dusk.SunriseSunset(date, obs)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	if summer.Horizon == dusk.StaysAbove {
		fmt.Printf("Midnight sun — no sunrise or sunset, solar noon %s\n",
			summer.Noon.Round(time.Minute).Format("15:04"))
	}

	// Tromsø on December 21 — polar night
	date = dusk.Date{Year: 2024, Month: 12, Day: 21}

	winter, err := dusk.SunriseSunset(date, obs)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	// Noon is real on both days. Polar geometry is a property of the result,
	// not a failure to produce one, so the rest of the day is still reported.
	if winter.Horizon == dusk.StaysBelow {
		fmt.Printf("Polar night — sun never rises, solar noon %s\n",
			winter.Noon.Round(time.Minute).Format("15:04"))
	}
	// USNO for Tromsø: Upper Transit 12:46 on 2024-06-21. It publishes no transit
	// for 2024-12-21 (the Sun is continuously below the horizon), but it does
	// publish civil twilight at 09:32 and 13:53, and transit bisects them by
	// construction -- 11:42:30.
	//
	// Round before formatting. "15:04" truncates, and the library returns true
	// instants: 12:45:57 would print as 12:45, a minute below the reference it
	// is four seconds from.

	// Output:
	// Midnight sun — no sunrise or sunset, solar noon 12:46
	// Polar night — sun never rises, solar noon 11:42
}

func ExampleSunriseSunset() {
	// Grand Rapids, Michigan, which keeps Eastern time.
	loc, err := time.LoadLocation("America/Detroit")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	date := dusk.Date{Year: 2025, Month: 6, Day: 21}

	obs, err := dusk.NewObserver(42.9634, -85.6681, loc)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	sun, err := dusk.SunriseSunset(date, obs)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Printf("Sunrise: %s\n", sun.Rise.Round(time.Minute).Format("15:04"))
	fmt.Printf("Sunset:  %s\n", sun.Set.Round(time.Minute).Format("15:04"))
	// USNO publishes 06:04 and 21:25 for this date and place; the library
	// computes 06:03:46 and 21:25:09, margins of 14s and 9s. Both agree once
	// rounded -- truncating would report the sunrise as 06:03.

	// Output:
	// Sunrise: 06:04
	// Sunset:  21:25
}

func ExampleLunarPhase() {
	date := time.Date(2024, 1, 25, 18, 0, 0, 0, time.UTC)

	phase, err := dusk.LunarPhase(date)
	if err != nil {
		fmt.Printf("error: %v\n", err)

		return
	}

	fmt.Printf("Phase: %s\n", phase.Name)
	fmt.Printf("Illumination: %.0f%%\n", phase.Illumination)
	// Output:
	// Phase: Full Moon
	// Illumination: 100%
}

func ExampleTwilight() {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	date := dusk.Date{Year: 2025, Month: 6, Day: 21}

	obs, err := dusk.NewObserver(47.6062, -122.3321, loc)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	tw, err := dusk.Twilight(date, obs, 6)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	// Dawn first: both boundaries are on the queried day, in clock order.
	fmt.Printf("Dawn: %s\n", tw.Dawn.Round(time.Minute).Format("15:04"))
	fmt.Printf("Dusk: %s\n", tw.Dusk.Round(time.Minute).Format("15:04"))
	// USNO publishes 04:31 and 21:52 for this date and place; the library
	// computes 04:30:50 and 21:51:26, margins of 10s and 34s. Dawn agrees once
	// rounded; dusk is 34 seconds short of the minute and rounds down, which is
	// the reference's own rounding rather than an error here -- USNO publishes
	// to the minute too.

	// Output:
	// Dawn: 04:31
	// Dusk: 21:51
}

func ExampleMoonriseMoonset() {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	date := dusk.Date{Year: 2024, Month: 1, Day: 15}

	obs, err := dusk.NewObserver(40.7128, -74.0060, loc)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	evt, err := dusk.MoonriseMoonset(date, obs)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	if !evt.Rise.IsZero() {
		fmt.Printf("Moonrise: %s\n", evt.Rise.Round(time.Minute).Format("15:04"))
	}

	if !evt.Set.IsZero() {
		fmt.Printf("Moonset:  %s\n", evt.Set.Round(time.Minute).Format("15:04"))
	}
	// USNO publishes 10:11 and 22:07 for this date and place; the library computes
	// 10:10:53 and 22:06:59 -- seven seconds and one second from the reference.
	// Rounding is what lets that show: truncating printed both a minute early.

	// Output:
	// Moonrise: 10:11
	// Moonset:  22:07
}
