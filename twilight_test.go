package dusk

import (
	"errors"
	"testing"
	"time"
)

func TestCivilTwilight(t *testing.T) {
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	// NYC (40.7128°N, 74.006°W) on 2024-03-20 (vernal equinox).
	date := time.Date(2024, 3, 20, 0, 0, 0, 0, nyc)
	lat := 40.7128
	lon := -74.006
	elev := 10.0
	tolerance := 5 * time.Minute

	obs := Observer{Lat: lat, Lon: lon, Elev: elev, Loc: nyc}
	event, err := CivilTwilight(date, obs)
	if err != nil {
		t.Fatalf("CivilTwilight() returned error: %v", err)
	}

	// Civil twilight dusk should be roughly 7:30-7:40 PM EDT (after sunset ~7:08 PM).
	wantDusk := time.Date(2024, 3, 20, 19, 35, 0, 0, nyc)
	if diff := event.Dusk.Sub(wantDusk); diff < -tolerance || diff > tolerance {
		t.Errorf("Dusk = %v, want %v (±%v, diff=%v)", event.Dusk.Format("15:04:05"), wantDusk.Format("15:04"), tolerance, diff)
	}

	// Civil twilight dawn should be roughly 6:25-6:35 AM EDT next day.
	wantDawn := time.Date(2024, 3, 21, 6, 30, 0, 0, nyc)
	if diff := event.Dawn.Sub(wantDawn); diff < -tolerance || diff > tolerance {
		t.Errorf("Dawn = %v, want %v (±%v, diff=%v)", event.Dawn.Format("15:04:05"), wantDawn.Format("15:04"), tolerance, diff)
	}

	if event.Duration <= 0 {
		t.Errorf("Duration = %v, want > 0", event.Duration)
	}
}

func TestNauticalTwilight(t *testing.T) {
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 3, 20, 0, 0, 0, 0, nyc)
	lat := 40.7128
	lon := -74.006
	elev := 10.0

	obs := Observer{Lat: lat, Lon: lon, Elev: elev, Loc: nyc}
	civil, err := CivilTwilight(date, obs)
	if err != nil {
		t.Fatalf("CivilTwilight() returned error: %v", err)
	}

	nautical, err := NauticalTwilight(date, obs)
	if err != nil {
		t.Fatalf("NauticalTwilight() returned error: %v", err)
	}

	// Nautical twilight dusk is later than civil (Sun is deeper below horizon).
	if !nautical.Dusk.After(civil.Dusk) {
		t.Errorf("Nautical dusk %v should be after civil dusk %v", nautical.Dusk.Format("15:04:05"), civil.Dusk.Format("15:04:05"))
	}

	// Nautical twilight dawn is earlier than civil.
	if !nautical.Dawn.Before(civil.Dawn) {
		t.Errorf("Nautical dawn %v should be before civil dawn %v", nautical.Dawn.Format("15:04:05"), civil.Dawn.Format("15:04:05"))
	}

	if nautical.Duration <= 0 {
		t.Errorf("Duration = %v, want > 0", nautical.Duration)
	}
}

func TestAstronomicalTwilight(t *testing.T) {
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 3, 20, 0, 0, 0, 0, nyc)
	lat := 40.7128
	lon := -74.006
	elev := 10.0

	obs := Observer{Lat: lat, Lon: lon, Elev: elev, Loc: nyc}
	nautical, err := NauticalTwilight(date, obs)
	if err != nil {
		t.Fatalf("NauticalTwilight() returned error: %v", err)
	}

	astro, err := AstronomicalTwilight(date, obs)
	if err != nil {
		t.Fatalf("AstronomicalTwilight() returned error: %v", err)
	}

	// Astronomical twilight dusk is later than nautical.
	if !astro.Dusk.After(nautical.Dusk) {
		t.Errorf("Astronomical dusk %v should be after nautical dusk %v", astro.Dusk.Format("15:04:05"), nautical.Dusk.Format("15:04:05"))
	}

	// Astronomical twilight dawn is earlier than nautical.
	if !astro.Dawn.Before(nautical.Dawn) {
		t.Errorf("Astronomical dawn %v should be before nautical dawn %v", astro.Dawn.Format("15:04:05"), nautical.Dawn.Format("15:04:05"))
	}

	if astro.Duration <= 0 {
		t.Errorf("Duration = %v, want > 0", astro.Duration)
	}
}

func TestTwilight_Equatorial(t *testing.T) {
	// Quito, Ecuador on 2024-03-20 (equinox).
	// Near the equator, twilight transitions are the fastest in the world.
	loc, err := time.LoadLocation("America/Guayaquil")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 3, 20, 0, 0, 0, 0, loc)
	lat := -0.18
	lon := -78.47
	elev := 2800.0

	obs := Observer{Lat: lat, Lon: lon, Elev: elev, Loc: loc}
	civil, err := CivilTwilight(date, obs)
	if err != nil {
		t.Fatalf("CivilTwilight() returned error: %v", err)
	}

	if civil.Dusk.IsZero() {
		t.Error("expected non-zero civil twilight Dusk")
	}
	if civil.Dawn.IsZero() {
		t.Error("expected non-zero civil twilight Dawn")
	}

	// Civil twilight duration (darkness period) should be less than 12 hours
	// at the equator, where twilight transitions are rapid.
	if civil.Duration >= 12*time.Hour {
		t.Errorf("civil twilight Duration = %v, want < 12h near equator", civil.Duration)
	}
	if civil.Duration <= 0 {
		t.Errorf("civil twilight Duration = %v, want > 0", civil.Duration)
	}

	t.Logf("Quito civil twilight: dusk=%v dawn=%v duration=%v", civil.Dusk, civil.Dawn, civil.Duration)
}

func TestTwilight_PolarDay(t *testing.T) {
	// Tromsø, Norway (69.65°N) on June 21 — no astronomical twilight during midnight sun.
	loc, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 6, 21, 0, 0, 0, 0, loc)
	obs := Observer{Lat: 69.65, Lon: 18.96, Elev: 0, Loc: loc}

	_, err = AstronomicalTwilight(date, obs)
	if !errors.Is(err, ErrCircumpolar) {
		t.Errorf("expected ErrCircumpolar for astronomical twilight at 69.65°N midsummer, got %v", err)
	}
}

func TestTwilight_PolarNight(t *testing.T) {
	// Near North Pole (87°N) on December 21 — deep polar night.
	// At this latitude the sun is far enough below the horizon that even
	// astronomical twilight (18° depression) does not occur.
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 12, 21, 0, 0, 0, 0, loc)
	obs := Observer{Lat: 87.0, Lon: 0, Elev: 0, Loc: loc}

	_, err = AstronomicalTwilight(date, obs)
	if !errors.Is(err, ErrNeverRises) {
		t.Errorf("expected ErrNeverRises for astronomical twilight at 87°N midwinter, got %v", err)
	}
}

func TestNauticalTwilight_AbsoluteTime(t *testing.T) {
	// USNO reference: NYC 2024-03-20 nautical twilight dusk ~20:05 EDT, dawn ~05:55 EDT.
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 3, 20, 0, 0, 0, 0, nyc)
	obs := Observer{Lat: 40.7128, Lon: -74.006, Elev: 10.0, Loc: nyc}
	tolerance := 10 * time.Minute

	nautical, err := NauticalTwilight(date, obs)
	if err != nil {
		t.Fatalf("NauticalTwilight() returned error: %v", err)
	}

	wantDusk := time.Date(2024, 3, 20, 20, 5, 0, 0, nyc)
	if diff := nautical.Dusk.Sub(wantDusk); diff < -tolerance || diff > tolerance {
		t.Errorf("Dusk = %v, want %v (±%v, diff=%v)", nautical.Dusk.Format("15:04:05"), wantDusk.Format("15:04"), tolerance, diff)
	}

	wantDawn := time.Date(2024, 3, 21, 5, 55, 0, 0, nyc)
	if diff := nautical.Dawn.Sub(wantDawn); diff < -tolerance || diff > tolerance {
		t.Errorf("Dawn = %v, want %v (±%v, diff=%v)", nautical.Dawn.Format("15:04:05"), wantDawn.Format("15:04"), tolerance, diff)
	}
}

func TestTwilight_PolarTransition(t *testing.T) {
	// Test high-latitude locations near polar twilight boundaries.
	// At these latitudes, twilight functions may return errors for some dates
	// but not others, exercising the transition branches in twilight().
	loc := time.UTC

	// Scan multiple latitudes and date ranges to exercise error paths.
	latitudes := []float64{68.0, 70.0, 72.0, 75.0}
	found := false
	for _, lat := range latitudes {
		obs := Observer{Lat: lat, Lon: 25.0, Elev: 0, Loc: loc}
		prevOk := false
		for day := 1; day <= 31; day++ {
			date := time.Date(2024, 12, day, 0, 0, 0, 0, loc)
			_, err := CivilTwilight(date, obs)
			if err == nil {
				prevOk = true
			} else if prevOk {
				found = true
				t.Logf("Civil twilight transition at %.0f°N on Dec %d: %v", lat, day, err)
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Log("No civil transition found; trying astronomical twilight")
		for _, lat := range latitudes {
			obs := Observer{Lat: lat, Lon: 25.0, Elev: 0, Loc: loc}
			prevOk := false
			for day := 1; day <= 31; day++ {
				date := time.Date(2024, 12, day, 0, 0, 0, 0, loc)
				_, err := AstronomicalTwilight(date, obs)
				if err == nil {
					prevOk = true
				} else if prevOk {
					found = true
					t.Logf("Astronomical twilight transition at %.0f°N on Dec %d: %v", lat, day, err)
					break
				}
			}
			if found {
				break
			}
		}
	}
	if !found {
		t.Log("No transition found in scan; polar twilight behavior verified without panic")
	}
}

func TestTwilight_NilLocation(t *testing.T) {
	date := time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)
	_, err := CivilTwilight(date, Observer{Lat: 40.7128, Lon: -74.006, Elev: 10})
	if err == nil {
		t.Error("CivilTwilight() with nil location should return error")
	}
}

func TestTwilight_InvalidCoordinates(t *testing.T) {
	date := time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)
	_, err := CivilTwilight(date, Observer{Lat: -91, Lon: 0, Loc: time.UTC})
	if err == nil {
		t.Error("expected error for invalid coordinates")
	}
}
