package dusk

import (
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
	// CLAUDE.md promise for sunrise/sunset; measured margin after the v5.1.0
	// independent-boundary fix is 4s on rise and 41s on set, from 16s and 63s
	// before it. USNO publishes to the minute.
	date := Date{2024, 3, 20}

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

func TestSunriseSunset_HighLatitudeEquinox(t *testing.T) {
	t.Parallel()

	// Oslo (59.91°N, 10.75°E) on 2026-09-17, eleven days before the September
	// equinox. This test exists because the suite had no absolute solar pin
	// above 40.7°N at all, and that is exactly where the mirrored-hour-angle
	// error lived: at NYC's latitude it is under a minute, so every assertion
	// passed while README.md's accuracy claim was false further north.
	//
	// USNO reference, aa.usno.navy.mil/api/rstt/oneday: Rise 06:49, Upper
	// Transit 13:12, Set 19:32.
	//
	// Measured margin: 47s on rise, 17s on transit, 107s on set. Before v5.1.0
	// solved the two boundaries independently it was 3s, 17s and **148s** --
	// note that rise was *better* then. The mirrored construction put the whole
	// error on sunset, which left sunrise accidentally accurate; correcting the
	// geometry distributes it. The remaining asymmetry is issue #114.
	oslo, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := Date{2026, 9, 17}

	tolerance := 2 * time.Minute

	obs := mustObserver(t, 59.91, 10.75, oslo)

	event, err := SunriseSunset(date, obs)
	if err != nil {
		t.Fatalf("SunriseSunset() returned error: %v", err)
	}

	for _, tt := range []struct {
		name string
		got  time.Time
		want time.Time
	}{
		{"Rise", event.Rise, time.Date(2026, 9, 17, 6, 49, 0, 0, oslo)},
		{"Noon", event.Noon, time.Date(2026, 9, 17, 13, 12, 0, 0, oslo)},
		{"Set", event.Set, time.Date(2026, 9, 17, 19, 32, 0, 0, oslo)},
	} {
		if diff := tt.got.Sub(tt.want); diff < -tolerance || diff > tolerance {
			t.Errorf("%s = %v, want %v (±%v, diff=%v)",
				tt.name, tt.got.Format("15:04:05"), tt.want.Format("15:04"), tolerance, diff)
		}
	}

	// The afternoon half-day is genuinely shorter than the morning near an
	// equinox, because declination moves about 0.4° a day. USNO has 22980s and
	// 22800s here, a skew of −180s. A mirrored construction reports 0 by
	// definition, which is what this assertion exists to forbid: it fails if
	// anyone reintroduces the symmetry, and it does not pretend the current
	// −86s is the whole story.
	morning := event.Noon.Sub(event.Rise)
	afternoon := event.Set.Sub(event.Noon)

	if skew := afternoon - morning; skew > -30*time.Second {
		t.Errorf("afternoon − morning = %v, want a clearly negative skew near the September equinox", skew)
	}
}

func TestSunriseSunset_Equatorial(t *testing.T) {
	t.Parallel()

	// Quito, Ecuador (lat≈0°): sunrise and sunset roughly 12h apart year-round.
	loc, err := time.LoadLocation("America/Guayaquil")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := Date{2024, 6, 21} // June solstice

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

	date := Date{2024, 6, 21}

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

	date := Date{2024, 6, 21}
	obs := mustObserver(t, 69.65, 18.96, loc)

	event, err := SunriseSunset(date, obs)
	if err != nil {
		t.Fatalf("SunriseSunset() returned error: %v", err)
	}

	if event.Horizon != StaysAbove {
		t.Errorf("Horizon = %v, want StaysAbove for midnight sun at 69.65°N", event.Horizon)
	}

	if !event.Rise.IsZero() || !event.Set.IsZero() {
		t.Errorf("Rise = %v, Set = %v, want both zero under the midnight sun", event.Rise, event.Set)
	}

	// Neither of these was reachable while polar geometry arrived as an error:
	// transit happens whether or not the Sun sets, and a day with no night is
	// 24 hours of daylight.
	if event.Noon.IsZero() {
		t.Error("Noon is zero, want solar transit under the midnight sun")
	}

	if event.Duration != 24*time.Hour {
		t.Errorf("Duration = %v, want 24h under the midnight sun", event.Duration)
	}
}

func TestSunriseSunset_PolarNight(t *testing.T) {
	t.Parallel()

	// Tromsø, Norway (69.65°N) on December 21 — polar night, no sunrise/sunset.
	loc, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := Date{2024, 12, 21}
	obs := mustObserver(t, 69.65, 18.96, loc)

	event, err := SunriseSunset(date, obs)
	if err != nil {
		t.Fatalf("SunriseSunset() returned error: %v", err)
	}

	if event.Horizon != StaysBelow {
		t.Errorf("Horizon = %v, want StaysBelow for polar night at 69.65°N", event.Horizon)
	}

	if !event.Rise.IsZero() || !event.Set.IsZero() {
		t.Errorf("Rise = %v, Set = %v, want both zero through the polar night", event.Rise, event.Set)
	}

	if event.Noon.IsZero() {
		t.Error("Noon is zero, want solar transit through the polar night")
	}

	if event.Duration != 0 {
		t.Errorf("Duration = %v, want 0 through the polar night", event.Duration)
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
	// USNO reference, aa.usno.navy.mil/api/rstt/oneday: Begin Civil Twilight
	// 06:31 and End Civil Twilight 19:36, both on the 20th. Both are real USNO
	// values for the reported day -- re-derived when v5 made Dawn and Dusk
	// same-day, not carried over from the v4 pins.
	//
	// Measured margin after v5.1.0's independent-boundary fix: 47s on dawn and
	// 21s on dusk, from 24s and 43s before it. Twilight moves because v5.0.0
	// gave it the same solarCrossing sunrise uses.
	date := Date{2024, 3, 20}
	tolerance := 2 * time.Minute

	obs := mustObserver(t, 40.7128, -74.006, nyc)

	event, err := Twilight(date, obs, 6)
	if err != nil {
		t.Fatalf("Twilight() returned error: %v", err)
	}

	// Civil twilight dusk, about half an hour after the 7:09 PM sunset above.
	wantDusk := time.Date(2024, 3, 20, 19, 36, 0, 0, nyc)
	if diff := event.Dusk.Sub(wantDusk); diff < -tolerance || diff > tolerance {
		t.Errorf("Dusk = %v, want %v (±%v, diff=%v)", event.Dusk.Format("15:04:05"), wantDusk.Format("15:04"), tolerance, diff)
	}

	// Civil twilight dawn the same morning, ahead of the 6:59 sunrise.
	wantDawn := time.Date(2024, 3, 20, 6, 31, 0, 0, nyc)
	if diff := event.Dawn.Sub(wantDawn); diff < -tolerance || diff > tolerance {
		t.Errorf("Dawn = %v, want %v (±%v, diff=%v)", event.Dawn.Format("15:04:05"), wantDawn.Format("15:04"), tolerance, diff)
	}

	// Both boundaries belong to the reported day, and bracket it.
	if !event.Dawn.Before(event.Dusk) {
		t.Errorf("Dawn %v should precede Dusk %v on the same day", event.Dawn, event.Dusk)
	}
}

func TestNauticalTwilight(t *testing.T) {
	t.Parallel()

	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := Date{2024, 3, 20}

	obs := mustObserver(t, 40.7128, -74.006, nyc)

	civil, err := Twilight(date, obs, 6)
	if err != nil {
		t.Fatalf("Twilight() returned error: %v", err)
	}

	nautical, err := Twilight(date, obs, 12)
	if err != nil {
		t.Fatalf("Twilight() returned error: %v", err)
	}

	// Nautical twilight dusk is later than civil (Sun is deeper below horizon).
	if !nautical.Dusk.After(civil.Dusk) {
		t.Errorf("Nautical dusk %v should be after civil dusk %v", nautical.Dusk.Format("15:04:05"), civil.Dusk.Format("15:04:05"))
	}

	// Nautical twilight dawn is earlier than civil.
	if !nautical.Dawn.Before(civil.Dawn) {
		t.Errorf("Nautical dawn %v should be before civil dawn %v", nautical.Dawn.Format("15:04:05"), civil.Dawn.Format("15:04:05"))
	}

	if !nautical.Dawn.Before(nautical.Dusk) {
		t.Errorf("Dawn %v should precede Dusk %v on the same day", nautical.Dawn, nautical.Dusk)
	}
}

func TestAstronomicalTwilight(t *testing.T) {
	t.Parallel()

	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := Date{2024, 3, 20}

	obs := mustObserver(t, 40.7128, -74.006, nyc)

	nautical, err := Twilight(date, obs, 12)
	if err != nil {
		t.Fatalf("Twilight() returned error: %v", err)
	}

	astro, err := Twilight(date, obs, 18)
	if err != nil {
		t.Fatalf("Twilight() returned error: %v", err)
	}

	// Astronomical twilight dusk is later than nautical.
	if !astro.Dusk.After(nautical.Dusk) {
		t.Errorf("Astronomical dusk %v should be after nautical dusk %v", astro.Dusk.Format("15:04:05"), nautical.Dusk.Format("15:04:05"))
	}

	// Astronomical twilight dawn is earlier than nautical.
	if !astro.Dawn.Before(nautical.Dawn) {
		t.Errorf("Astronomical dawn %v should be before nautical dawn %v", astro.Dawn.Format("15:04:05"), nautical.Dawn.Format("15:04:05"))
	}

	if !astro.Dawn.Before(astro.Dusk) {
		t.Errorf("Dawn %v should precede Dusk %v on the same day", astro.Dawn, astro.Dusk)
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

	date := Date{2024, 3, 20}

	obs := mustObserver(t, -0.18, -78.47, loc)

	civil, err := Twilight(date, obs, 6)
	if err != nil {
		t.Fatalf("Twilight() returned error: %v", err)
	}

	if civil.Dusk.IsZero() {
		t.Error("expected non-zero civil twilight Dusk")
	}

	if civil.Dawn.IsZero() {
		t.Error("expected non-zero civil twilight Dawn")
	}

	// The equator is where twilight is briefest: the Sun crosses the 6° band
	// almost vertically, so dawn leads sunrise by around twenty minutes rather
	// than the hour it takes at high latitude. Asserting that gap is what this
	// test is for -- the same-day event makes it directly measurable.
	sun, err := SunriseSunset(date, obs)
	if err != nil {
		t.Fatalf("SunriseSunset() returned error: %v", err)
	}

	lead := sun.Rise.Sub(civil.Dawn)
	if lead <= 0 || lead > 30*time.Minute {
		t.Errorf("civil dawn leads sunrise by %v, want a positive gap under 30m near the equator", lead)
	}

	trail := civil.Dusk.Sub(sun.Set)
	if trail <= 0 || trail > 30*time.Minute {
		t.Errorf("civil dusk trails sunset by %v, want a positive gap under 30m near the equator", trail)
	}

	t.Logf("Quito civil twilight: dawn=%v dusk=%v (lead %v, trail %v)", civil.Dawn, civil.Dusk, lead, trail)
}

func TestTwilight_RefinementAtThePolarLimit(t *testing.T) {
	t.Parallel()

	// 65.5°N, civil twilight, 2025-05-13. The first pass says the Sun crosses
	// 6° below the horizon; re-solving the evening boundary against the
	// declination at its own estimated instant says it does not, because a day
	// this close to the limit is gaining length fast enough to cross it inside
	// the correction.
	//
	// Found by search rather than by guess -- it is the earliest such day at any
	// latitude and band the suite covers. The branch it exercises is the one
	// deliberate approximation in solarCrossing: keep the first estimate rather
	// than report a state the day as a whole does not have, because the
	// alternative is a boundary that vanishes and reappears across one degree of
	// latitude.
	obs := mustObserver(t, 65.5, 0, time.UTC)

	event, err := Twilight(Date{2025, 5, 13}, obs, 6)
	if err != nil {
		t.Fatalf("Twilight() returned error: %v", err)
	}

	if event.Horizon != Crosses {
		t.Fatalf("Horizon = %v, want Crosses -- the refinement must not overturn the day's own answer", event.Horizon)
	}

	if event.Dawn.IsZero() || event.Dusk.IsZero() {
		t.Fatalf("dawn = %v, dusk = %v, want both real", event.Dawn, event.Dusk)
	}

	if !event.Dawn.Before(event.Dusk) {
		t.Errorf("dawn %v should precede dusk %v", event.Dawn, event.Dusk)
	}

	// The fallback keeps the symmetric estimate for the boundary it could not
	// refine, so that one stays mirrored about transit while the other does not.
	// Asserting the day still makes sense is the point; asserting which side got
	// the fallback would pin an implementation detail.
	if d := event.Dusk.Sub(event.Dawn); d <= 0 || d > 24*time.Hour {
		t.Errorf("dusk − dawn = %v, want a sane span", d)
	}
}

func TestTwilight_PolarDay(t *testing.T) {
	t.Parallel()

	// Tromsø, Norway (69.65°N) on June 21 — no astronomical twilight during midnight sun.
	loc, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := Date{2024, 6, 21}
	obs := mustObserver(t, 69.65, 18.96, loc)

	event, err := Twilight(date, obs, 18)
	if err != nil {
		t.Fatalf("Twilight() returned error: %v", err)
	}

	if event.Horizon != StaysAbove {
		t.Errorf("Horizon = %v, want StaysAbove for astronomical twilight at 69.65°N midsummer", event.Horizon)
	}
}

func TestTwilight_PolarNight(t *testing.T) {
	t.Parallel()

	// Near North Pole (87°N) on December 21 — deep polar night.
	// At this latitude the sun is far enough below the horizon that even
	// astronomical twilight (18° depression) does not occur.
	loc := time.UTC
	obs := mustObserver(t, 87.0, 0, loc)

	date := Date{2024, 12, 21}

	event, err := Twilight(date, obs, 18)
	if err != nil {
		t.Fatalf("Twilight() returned error: %v", err)
	}

	if event.Horizon != StaysBelow {
		t.Errorf("Horizon = %v, want StaysBelow for astronomical twilight at 87°N midwinter", event.Horizon)
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
	// Both values are carried over from v4 unchanged, which is the point. When
	// TwilightEvent stopped spanning two days, Dusk stayed where it was and Dawn
	// moved to the day it actually describes: the 05:55 pin was always the
	// morning AFTER the queried date, so it is now Twilight(D+1).Dawn. Asserting
	// it that way makes this test a direct check of the migration -- the same
	// instant reached by a call that names the right day.
	//
	// The tolerance is 4 minutes rather than the 2 the sunrise tests hold,
	// because a 12° depression amplifies declination error. That is twilight's
	// own tolerance, not a looser reading of sunrise's -- see CLAUDE.md.
	//
	// Measured margin after v5.1.0's independent-boundary fix: 2m41s on dusk and
	// 3m07s on dawn, from 2m15s and 2m42s before it. **The margin grew because
	// the library moved, and USNO cannot say which of the two is now right** --
	// it publishes no nautical band. Every USNO-checkable quantity improved in
	// that release, so the likeliest reading is that these pins were never very
	// good. If one ever breaches 4 minutes, discard the pin rather than widening
	// the tolerance; a value with no provenance has earned no protection.
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := Date{2024, 3, 20}
	obs := mustObserver(t, 40.7128, -74.006, nyc)
	tolerance := 4 * time.Minute

	nautical, err := Twilight(date, obs, 12)
	if err != nil {
		t.Fatalf("Twilight() returned error: %v", err)
	}

	wantDusk := time.Date(2024, 3, 20, 20, 5, 0, 0, nyc)
	if diff := nautical.Dusk.Sub(wantDusk); diff < -tolerance || diff > tolerance {
		t.Errorf("Dusk = %v, want %v (±%v, diff=%v)", nautical.Dusk.Format("15:04:05"), wantDusk.Format("15:04"), tolerance, diff)
	}

	next, err := Twilight(Date{date.Year, date.Month, date.Day + 1}, obs, 12)
	if err != nil {
		t.Fatalf("Twilight(tomorrow) returned error: %v", err)
	}

	wantDawn := time.Date(2024, 3, 21, 5, 55, 0, 0, nyc)
	if diff := next.Dawn.Sub(wantDawn); diff < -tolerance || diff > tolerance {
		t.Errorf("Dawn = %v, want %v (±%v, diff=%v)", next.Dawn.Format("15:04:05"), wantDawn.Format("15:04"), tolerance, diff)
	}
}

func TestTwilight_PolarTransition(t *testing.T) {
	t.Parallel()

	// At 75°N the civil band closes between 2024-11-26 and 2024-11-27.
	//
	// A same-day event is symmetric about solar transit, so both boundaries
	// exist or neither does -- the transition can only land BETWEEN days, and
	// that is what this pins. Under the v4 two-day event the Nov 26 call failed,
	// but it failed on Nov 27's geometry: Nov 26 has a real 19-minute civil
	// window that the whole-call error threw away along with it.
	loc := time.UTC
	obs := mustObserver(t, 75.0, 25.0, loc)

	last := Date{2024, 11, 26}
	gone := Date{2024, 11, 27}

	event, err := Twilight(last, obs, 6)
	if err != nil {
		t.Fatalf("Twilight(Nov 26, 75°N, 6) should succeed, got %v", err)
	}

	if event.Horizon != Crosses {
		t.Fatalf("Nov 26 Horizon = %v, want Crosses", event.Horizon)
	}

	if event.Dawn.IsZero() || event.Dusk.IsZero() {
		t.Fatalf("Nov 26 should carry both boundaries, got dawn=%v dusk=%v", event.Dawn, event.Dusk)
	}

	// The window is narrow and closing: minutes, not hours.
	window := event.Dusk.Sub(event.Dawn)
	if window <= 0 || window > time.Hour {
		t.Errorf("Nov 26 civil window = %v, want a positive span under an hour", window)
	}

	next, err := Twilight(gone, obs, 6)
	if err != nil {
		t.Fatalf("Twilight() returned error: %v", err)
	}

	if next.Horizon != StaysBelow {
		t.Fatalf("Nov 27 Horizon = %v, want StaysBelow", next.Horizon)
	}

	if !next.Dawn.IsZero() || !next.Dusk.IsZero() {
		t.Errorf("Nov 27 dawn = %v, dusk = %v, want both zero", next.Dawn, next.Dusk)
	}
}
