package dusk

import (
	"testing"
	"time"
)

func BenchmarkMoonriseMoonset(b *testing.B) {
	loc, _ := time.LoadLocation("America/New_York")
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, loc)
	obs := Observer{Lat: 40.7128, Lon: -74.006, Loc: loc}

	for b.Loop() {
		MoonriseMoonset(date, obs) //nolint:errcheck
	}
}

func BenchmarkLunarEclipticPosition(b *testing.B) {
	dt := time.Date(1992, 4, 12, 0, 0, 0, 0, time.UTC)

	for b.Loop() {
		LunarEclipticPosition(dt)
	}
}

func BenchmarkSunriseSunset(b *testing.B) {
	loc, _ := time.LoadLocation("America/New_York")
	date := time.Date(2024, 3, 20, 0, 0, 0, 0, loc)
	obs := Observer{Lat: 40.7128, Lon: -74.006, Elev: 10, Loc: loc}

	for b.Loop() {
		SunriseSunset(date, obs) //nolint:errcheck
	}
}

func BenchmarkObjectTransit(b *testing.B) {
	loc, _ := time.LoadLocation("America/New_York")
	date := time.Date(2024, 3, 20, 0, 0, 0, 0, loc)
	obs := Observer{Lat: 40.7128, Lon: -74.006, Loc: loc}
	eq := Equatorial{RA: 100, Dec: 20}

	for b.Loop() {
		ObjectTransit(date, eq, obs) //nolint:errcheck
	}
}
