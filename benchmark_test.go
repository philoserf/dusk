package dusk

import (
	"testing"
	"time"
)

// benchEDT is a fixed UTC-5 zone used by benchmarks to avoid depending on the
// system timezone database.
var benchEDT = time.FixedZone("EDT", -5*3600)

func BenchmarkMoonriseMoonset(b *testing.B) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, benchEDT)
	obs := Observer{lat: 40.7128, lon: -74.006, loc: benchEDT}

	for b.Loop() {
		MoonriseMoonset(date, obs) //nolint:errcheck
	}
}

func BenchmarkSunriseSunset(b *testing.B) {
	date := time.Date(2024, 3, 20, 0, 0, 0, 0, benchEDT)
	obs := Observer{lat: 40.7128, lon: -74.006, loc: benchEDT}

	for b.Loop() {
		SunriseSunset(date, obs) //nolint:errcheck
	}
}
