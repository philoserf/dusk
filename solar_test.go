package dusk

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestSunriseSunset(t *testing.T) {
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	// NYC (40.7128°N, 74.006°W) on 2024-03-20 (vernal equinox).
	// USNO data: Sunrise ~6:57 AM EDT, Sunset ~7:11 PM EDT.
	date := time.Date(2024, 3, 20, 0, 0, 0, 0, nyc)
	lat := 40.7128
	lon := -74.006
	elev := 10.0

	tolerance := 3 * time.Minute

	obs := Observer{Lat: lat, Lon: lon, Elev: elev, Loc: nyc}
	event, err := SunriseSunset(date, obs)
	if err != nil {
		t.Fatalf("SunriseSunset() returned error: %v", err)
	}

	wantRise := time.Date(2024, 3, 20, 6, 57, 0, 0, nyc)
	wantSet := time.Date(2024, 3, 20, 19, 8, 0, 0, nyc)

	if diff := event.Rise.Sub(wantRise); diff < -tolerance || diff > tolerance {
		t.Errorf("Sunrise = %v, want %v (±%v, diff=%v)", event.Rise.Format("15:04:05"), wantRise.Format("15:04"), tolerance, diff)
	}

	if diff := event.Set.Sub(wantSet); diff < -tolerance || diff > tolerance {
		t.Errorf("Sunset = %v, want %v (±%v, diff=%v)", event.Set.Format("15:04:05"), wantSet.Format("15:04"), tolerance, diff)
	}

	// Noon should be between rise and set.
	if event.Noon.Before(event.Rise) || event.Noon.After(event.Set) {
		t.Errorf("Noon %v not between Rise %v and Set %v", event.Noon, event.Rise, event.Set)
	}

	// Duration should be positive and roughly 12 hours near the equinox.
	if event.Duration < 11*time.Hour || event.Duration > 13*time.Hour {
		t.Errorf("Duration = %v, expected ~12h near equinox", event.Duration)
	}
}

func TestSunriseSunset_Equatorial(t *testing.T) {
	// Quito, Ecuador (lat≈0°): sunrise and sunset roughly 12h apart year-round.
	loc, err := time.LoadLocation("America/Guayaquil")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 6, 21, 0, 0, 0, 0, loc) // June solstice
	lat := -0.18
	lon := -78.47
	elev := 2800.0 // Quito elevation in meters

	obs := Observer{Lat: lat, Lon: lon, Elev: elev, Loc: loc}
	event, err := SunriseSunset(date, obs)
	if err != nil {
		t.Fatalf("SunriseSunset() returned error: %v", err)
	}

	if event.Rise.IsZero() {
		t.Fatal("expected non-zero Rise")
	}
	if event.Set.IsZero() {
		t.Fatal("expected non-zero Set")
	}

	// Near the equator, day length should be close to 12 hours year-round.
	if event.Duration < 11*time.Hour+30*time.Minute || event.Duration > 12*time.Hour+30*time.Minute {
		t.Errorf("Duration = %v, want 11h30m–12h30m near equator", event.Duration)
	}

	// Sunrise and sunset should be roughly 12 hours apart.
	gap := event.Set.Sub(event.Rise)
	wantGap := 12 * time.Hour
	if diff := gap - wantGap; diff < -30*time.Minute || diff > 30*time.Minute {
		t.Errorf("Set-Rise = %v, want ~12h (±30m)", gap)
	}
}

func TestSunriseSunset_SouthernHemisphere(t *testing.T) {
	// Sydney, Australia: June 21 is winter solstice — short day.
	loc, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 6, 21, 0, 0, 0, 0, loc)
	lat := -33.87
	lon := 151.21
	elev := 58.0

	obs := Observer{Lat: lat, Lon: lon, Elev: elev, Loc: loc}
	event, err := SunriseSunset(date, obs)
	if err != nil {
		t.Fatalf("SunriseSunset() returned error: %v", err)
	}

	if event.Rise.IsZero() {
		t.Fatal("expected non-zero Rise")
	}
	if event.Set.IsZero() {
		t.Fatal("expected non-zero Set")
	}

	// Winter solstice in Sydney: day length should be less than 11 hours.
	if event.Duration >= 11*time.Hour {
		t.Errorf("Duration = %v, want < 11h for Sydney winter solstice", event.Duration)
	}
	if event.Duration <= 8*time.Hour {
		t.Errorf("Duration = %v, want > 8h (sanity check)", event.Duration)
	}
}

func TestSunriseSunset_PolarDay(t *testing.T) {
	// Tromsø, Norway (69.65°N) on June 21 — midnight sun, no sunrise/sunset.
	loc, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 6, 21, 0, 0, 0, 0, loc)
	obs := Observer{Lat: 69.65, Lon: 18.96, Elev: 0, Loc: loc}

	_, err = SunriseSunset(date, obs)
	if !errors.Is(err, ErrCircumpolar) {
		t.Errorf("expected ErrCircumpolar for midnight sun at 69.65°N, got %v", err)
	}
}

func TestSunriseSunset_PolarNight(t *testing.T) {
	// Tromsø, Norway (69.65°N) on December 21 — polar night, no sunrise/sunset.
	loc, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 12, 21, 0, 0, 0, 0, loc)
	obs := Observer{Lat: 69.65, Lon: 18.96, Elev: 0, Loc: loc}

	_, err = SunriseSunset(date, obs)
	if !errors.Is(err, ErrNeverRises) {
		t.Errorf("expected ErrNeverRises for polar night at 69.65°N, got %v", err)
	}
}

func TestSunriseSunset_NilLocation(t *testing.T) {
	date := time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)
	_, err := SunriseSunset(date, Observer{Lat: 40.0, Lon: -74.0})
	if err == nil {
		t.Error("SunriseSunset() with nil location should return error")
	}
}

func TestSunriseSunset_InvalidCoordinates(t *testing.T) {
	loc := time.UTC
	date := time.Date(2024, 3, 20, 0, 0, 0, 0, loc)

	tests := []struct {
		name string
		obs  Observer
	}{
		{"latitude too high", Observer{Lat: 91, Lon: 0, Loc: loc}},
		{"latitude too low", Observer{Lat: -91, Lon: 0, Loc: loc}},
		{"longitude too high", Observer{Lat: 0, Lon: 181, Loc: loc}},
		{"longitude too low", Observer{Lat: 0, Lon: -181, Loc: loc}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SunriseSunset(date, tt.obs)
			if err == nil {
				t.Error("expected error for invalid coordinates")
			}
		})
	}
}

func TestSunriseSunset_NegativeElevation(t *testing.T) {
	// Negative elevation should be clamped to sea level (math.Max(0, elev)).
	// Results with Elev: -100 should match Elev: 0 exactly.
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)
	base := Observer{Lat: 40.7128, Lon: -74.006, Elev: 0, Loc: nyc}
	neg := Observer{Lat: 40.7128, Lon: -74.006, Elev: -100, Loc: nyc}

	evtBase, err := SunriseSunset(date, base)
	if err != nil {
		t.Fatalf("SunriseSunset(Elev=0) error: %v", err)
	}
	evtNeg, err := SunriseSunset(date, neg)
	if err != nil {
		t.Fatalf("SunriseSunset(Elev=-100) error: %v", err)
	}

	if !evtBase.Rise.Equal(evtNeg.Rise) {
		t.Errorf("Rise differs: Elev=0 %v, Elev=-100 %v", evtBase.Rise, evtNeg.Rise)
	}
	if !evtBase.Set.Equal(evtNeg.Set) {
		t.Errorf("Set differs: Elev=0 %v, Elev=-100 %v", evtBase.Set, evtNeg.Set)
	}
}

func TestSolarPosition(t *testing.T) {
	// Near the vernal equinox: RA ~0°, Dec ~0°.
	dt := time.Date(2024, 3, 20, 12, 0, 0, 0, time.UTC)
	pos := SolarPosition(dt)

	// RA should be near 0° (or 360°). Handle wrap-around.
	ra := pos.RA
	if ra > 180 {
		ra -= 360
	}
	if math.Abs(ra) > 5 {
		t.Errorf("SolarPosition() RA = %f°, want near 0° (±5°) at vernal equinox", pos.RA)
	}

	if math.Abs(pos.Dec) > 2 {
		t.Errorf("SolarPosition() Dec = %f°, want near 0° (±2°) at vernal equinox", pos.Dec)
	}
}

func TestSolarPosition_SummerSolstice(t *testing.T) {
	// Summer solstice 2024-06-20: RA ~90°, Dec ~+23.44°.
	dt := time.Date(2024, 6, 20, 12, 0, 0, 0, time.UTC)
	pos := SolarPosition(dt)

	if math.Abs(pos.RA-90) > 2 {
		t.Errorf("SolarPosition() RA = %f°, want near 90° (±2°) at summer solstice", pos.RA)
	}
	if math.Abs(pos.Dec-23.44) > 1 {
		t.Errorf("SolarPosition() Dec = %f°, want near 23.44° (±1°) at summer solstice", pos.Dec)
	}
}

func TestSolarMeanAnomaly(t *testing.T) {
	tests := []struct {
		name    string
		J       float64
		wantMin float64
		wantMax float64
	}{
		{
			name:    "J2000 epoch (J=0)",
			J:       0,
			wantMin: 357.0,
			wantMax: 358.0,
		},
		{
			name:    "large positive day count wraps via mod360",
			J:       36525, // 100 years
			wantMin: 0,
			wantMax: 360,
		},
		{
			name:    "negative day count wraps via mod360",
			J:       -365,
			wantMin: 0,
			wantMax: 360,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := solarMeanAnomaly(tt.J)
			if got < tt.wantMin || got >= tt.wantMax {
				t.Errorf("solarMeanAnomaly(%f) = %f, want in [%f, %f)", tt.J, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}
