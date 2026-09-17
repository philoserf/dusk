package dusk

import (
	"errors"
	"testing"
	"time"
)

func TestSunriseSunset(t *testing.T) {
	t.Parallel()

	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	// NYC (40.7128°N, 74.006°W) on 2024-03-20 (vernal equinox).
	//
	// USNO reference, aa.usno.navy.mil/api/rstt/oneday: Rise 06:59, Upper Transit
	// 13:03, Set 19:09. Tolerance is 2 minutes because that is what README.md and
	// CLAUDE.md promise for sunrise/sunset; measured margin at the time of
	// writing is 16s on rise and 63s on set. USNO publishes to the minute.
	date := time.Date(2024, 3, 20, 0, 0, 0, 0, nyc)

	tolerance := 2 * time.Minute

	obs := mustObserver(t, 40.7128, -74.006, nyc)

	event, err := SunriseSunset(date, obs)
	if err != nil {
		t.Fatalf("SunriseSunset() returned error: %v", err)
	}

	wantRise := time.Date(2024, 3, 20, 6, 59, 0, 0, nyc)
	wantSet := time.Date(2024, 3, 20, 19, 9, 0, 0, nyc)

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
	t.Parallel()

	// Quito, Ecuador (lat≈0°): sunrise and sunset roughly 12h apart year-round.
	loc, err := time.LoadLocation("America/Guayaquil")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 6, 21, 0, 0, 0, 0, loc) // June solstice

	obs := mustObserver(t, -0.18, -78.47, loc)

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
	t.Parallel()

	// Sydney, Australia: June 21 is winter solstice — short day.
	loc, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 6, 21, 0, 0, 0, 0, loc)

	obs := mustObserver(t, -33.87, 151.21, loc)

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
	t.Parallel()

	// Tromsø, Norway (69.65°N) on June 21 — midnight sun, no sunrise/sunset.
	loc, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 6, 21, 0, 0, 0, 0, loc)
	obs := mustObserver(t, 69.65, 18.96, loc)

	_, err = SunriseSunset(date, obs)
	if !errors.Is(err, ErrCircumpolar) {
		t.Errorf("expected ErrCircumpolar for midnight sun at 69.65°N, got %v", err)
	}
}

func TestSunriseSunset_PolarNight(t *testing.T) {
	t.Parallel()

	// Tromsø, Norway (69.65°N) on December 21 — polar night, no sunrise/sunset.
	loc, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 12, 21, 0, 0, 0, 0, loc)
	obs := mustObserver(t, 69.65, 18.96, loc)

	_, err = SunriseSunset(date, obs)
	if !errors.Is(err, ErrNeverRises) {
		t.Errorf("expected ErrNeverRises for polar night at 69.65°N, got %v", err)
	}
}

func TestSolarMeanAnomaly(t *testing.T) {
	t.Parallel()

	// At the J2000.0 epoch the Sun's mean anomaly is 357.529 degrees.
	got := solarMeanAnomaly(0)
	if got < 357.0 || got >= 358.0 {
		t.Errorf("solarMeanAnomaly(0) = %f, want in [357, 358)", got)
	}
}

// ---------------------------------------------------------------------------
// Twilight tests: the three depression bands.
// ---------------------------------------------------------------------------

func TestCivilTwilight(t *testing.T) {
	t.Parallel()

	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	// NYC (40.7128°N, 74.006°W) on 2024-03-20 (vernal equinox).
	//
	// USNO reference, aa.usno.navy.mil/api/rstt/oneday: End Civil Twilight 19:36
	// on the 20th, Begin Civil Twilight 06:30 on the 21st -- Dawn is tomorrow
	// morning's, which is this package's convention. Measured margin at the time
	// of writing: dusk 43s, dawn 17s.
	date := time.Date(2024, 3, 20, 0, 0, 0, 0, nyc)
	tolerance := 2 * time.Minute

	obs := mustObserver(t, 40.7128, -74.006, nyc)

	event, err := CivilTwilight(date, obs)
	if err != nil {
		t.Fatalf("CivilTwilight() returned error: %v", err)
	}

	// Civil twilight dusk, about half an hour after the 7:09 PM sunset above.
	wantDusk := time.Date(2024, 3, 20, 19, 36, 0, 0, nyc)
	if diff := event.Dusk.Sub(wantDusk); diff < -tolerance || diff > tolerance {
		t.Errorf("Dusk = %v, want %v (±%v, diff=%v)", event.Dusk.Format("15:04:05"), wantDusk.Format("15:04"), tolerance, diff)
	}

	// Civil twilight dawn the following morning.
	wantDawn := time.Date(2024, 3, 21, 6, 30, 0, 0, nyc)
	if diff := event.Dawn.Sub(wantDawn); diff < -tolerance || diff > tolerance {
		t.Errorf("Dawn = %v, want %v (±%v, diff=%v)", event.Dawn.Format("15:04:05"), wantDawn.Format("15:04"), tolerance, diff)
	}

	if event.NightDuration <= 0 {
		t.Errorf("NightDuration = %v, want > 0", event.NightDuration)
	}
}

func TestNauticalTwilight(t *testing.T) {
	t.Parallel()

	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 3, 20, 0, 0, 0, 0, nyc)

	obs := mustObserver(t, 40.7128, -74.006, nyc)

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

	if nautical.NightDuration <= 0 {
		t.Errorf("NightDuration = %v, want > 0", nautical.NightDuration)
	}
}

func TestAstronomicalTwilight(t *testing.T) {
	t.Parallel()

	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 3, 20, 0, 0, 0, 0, nyc)

	obs := mustObserver(t, 40.7128, -74.006, nyc)

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

	if astro.NightDuration <= 0 {
		t.Errorf("NightDuration = %v, want > 0", astro.NightDuration)
	}
}

func TestTwilight_Equatorial(t *testing.T) {
	t.Parallel()

	// Quito, Ecuador on 2024-03-20 (equinox).
	// Near the equator, twilight transitions are the fastest in the world.
	loc, err := time.LoadLocation("America/Guayaquil")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 3, 20, 0, 0, 0, 0, loc)

	obs := mustObserver(t, -0.18, -78.47, loc)

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
	if civil.NightDuration >= 12*time.Hour {
		t.Errorf("civil twilight NightDuration = %v, want < 12h near equator", civil.NightDuration)
	}

	if civil.NightDuration <= 0 {
		t.Errorf("civil twilight NightDuration = %v, want > 0", civil.NightDuration)
	}

	t.Logf("Quito civil twilight: dusk=%v dawn=%v duration=%v", civil.Dusk, civil.Dawn, civil.NightDuration)
}

func TestTwilight_PolarDay(t *testing.T) {
	t.Parallel()

	// Tromsø, Norway (69.65°N) on June 21 — no astronomical twilight during midnight sun.
	loc, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 6, 21, 0, 0, 0, 0, loc)
	obs := mustObserver(t, 69.65, 18.96, loc)

	_, err = AstronomicalTwilight(date, obs)
	if !errors.Is(err, ErrCircumpolar) {
		t.Errorf("expected ErrCircumpolar for astronomical twilight at 69.65°N midsummer, got %v", err)
	}
}

func TestTwilight_PolarNight(t *testing.T) {
	t.Parallel()

	// Near North Pole (87°N) on December 21 — deep polar night.
	// At this latitude the sun is far enough below the horizon that even
	// astronomical twilight (18° depression) does not occur.
	loc := time.UTC
	obs := mustObserver(t, 87.0, 0, loc)

	date := time.Date(2024, 12, 21, 0, 0, 0, 0, loc)

	_, err := AstronomicalTwilight(date, obs)
	if !errors.Is(err, ErrNeverRises) {
		t.Errorf("expected ErrNeverRises for astronomical twilight at 87°N midwinter, got %v", err)
	}
}

func TestNauticalTwilight_AbsoluteTime(t *testing.T) {
	t.Parallel()

	// NYC 2024-03-20 nautical twilight: dusk ~20:05 EDT, dawn ~05:55 EDT.
	//
	// These two are the only pins in this file that could NOT be corroborated
	// against USNO: aa.usno.navy.mil publishes civil twilight in its one-day
	// service and no nautical or astronomical equivalent, so the provenance of
	// 20:05 and 05:55 is unverified and they are treated as a regression pin.
	//
	// The tolerance is 4 minutes rather than the 2 the sunrise tests hold,
	// because a 12° depression amplifies declination error: measured margin at
	// the time of writing is 2m15s on dusk and 2m42s on dawn. That is twilight's
	// own tolerance, not a looser reading of sunrise's -- see CLAUDE.md.
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 3, 20, 0, 0, 0, 0, nyc)
	obs := mustObserver(t, 40.7128, -74.006, nyc)
	tolerance := 4 * time.Minute

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
	t.Parallel()

	// At 75°N, civil twilight succeeds on Nov 25 but fails on Nov 26,
	// exercising the polar-transition branch in twilight().
	loc := time.UTC
	obs := mustObserver(t, 75.0, 25.0, loc)

	before := time.Date(2024, 11, 25, 0, 0, 0, 0, loc)
	after := time.Date(2024, 11, 26, 0, 0, 0, 0, loc)

	_, err := CivilTwilight(before, obs)
	if err != nil {
		t.Fatalf("CivilTwilight(Nov 25, 75°N) should succeed, got %v", err)
	}

	_, err = CivilTwilight(after, obs)
	if !errors.Is(err, ErrNeverRises) {
		t.Fatalf("CivilTwilight(Nov 26, 75°N) should return ErrNeverRises, got %v", err)
	}
}
