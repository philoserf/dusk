package dusk

import (
	"math"
	"testing"
	"time"
)

func FuzzSunriseSunset(f *testing.F) {
	f.Add(40.7128, -74.006, int64(1710892800)) // NYC 2024-03-20
	f.Add(-33.87, 151.21, int64(1718928000))   // Sydney 2024-06-21
	f.Add(69.65, 18.96, int64(1718928000))     // Tromsø summer
	f.Add(-0.18, -78.47, int64(1710892800))    // Quito

	f.Fuzz(func(t *testing.T, lat, lon float64, unix int64) {
		date := time.Unix(unix, 0).UTC()
		if date.Year() < 1800 || date.Year() > 2200 {
			return
		}

		obs, err := NewObserver(lat, lon, time.UTC)
		if err != nil {
			return // invalid coordinates rejected by NewObserver
		}

		sun, err := SunriseSunset(date, obs)
		if err != nil {
			return // out-of-range dates are rejected, not asserted on
		}

		// Noon holds on every day at every latitude, polar ones included. That
		// is the invariant the error carrier used to hide: before v5 this whole
		// branch was unreachable above the Arctic circle.
		if sun.Noon.IsZero() {
			t.Error("Noon is the zero time")
		}

		if sun.Horizon != Crosses {
			if !sun.Rise.IsZero() || !sun.Set.IsZero() {
				t.Errorf("Horizon %v carries Rise %v and Set %v, want both zero", sun.Horizon, sun.Rise, sun.Set)
			}

			want := time.Duration(0)
			if sun.Horizon == StaysAbove {
				want = 24 * time.Hour
			}

			if sun.Duration != want {
				t.Errorf("Duration %v on a %v day, want %v", sun.Duration, sun.Horizon, want)
			}

			return
		}

		// A crossing SunEvent means the Sun rose and set. clamp (trig.go) keeps
		// NaN out of asin/acos by silently clamping, which its own comment
		// concedes can mask an upstream bug; these checks are the sweep that
		// fixed reference data cannot do.
		for _, e := range []struct {
			name string
			at   time.Time
		}{{"Rise", sun.Rise}, {"Noon", sun.Noon}, {"Set", sun.Set}} {
			if e.at.IsZero() {
				t.Errorf("%s is the zero time on a day with no error", e.name)
			}
		}

		// cmd/dusk renders the day by sorting events on the clock, so an
		// inversion would surface as a plausible report in the wrong order
		// rather than as a crash.
		if !sun.Noon.After(sun.Rise) {
			t.Errorf("Noon %v is not after Rise %v", sun.Noon, sun.Rise)
		}

		if !sun.Set.After(sun.Noon) {
			t.Errorf("Set %v is not after Noon %v", sun.Set, sun.Noon)
		}

		if sun.Duration <= 0 {
			t.Errorf("Duration %v is not positive on a day the Sun rises and sets", sun.Duration)
		}
	})
}

func FuzzLunarPhase(f *testing.F) {
	f.Add(int64(1704931200)) // 2024-01-11
	f.Add(int64(1706140800)) // 2024-01-25
	f.Add(int64(1710892800)) // 2024-03-20

	f.Fuzz(func(t *testing.T, unix int64) {
		date := time.Unix(unix, 0).UTC()
		if date.Year() < 1800 || date.Year() > 2200 {
			return
		}

		p, err := LunarPhase(date)
		if err != nil {
			return
		}

		if math.IsNaN(p.Illumination) {
			t.Error("NaN illumination")
		}

		if p.Illumination < 0 || p.Illumination > 100 {
			t.Errorf("illumination out of range: %f", p.Illumination)
		}
	})
}

func FuzzMoonriseMoonset(f *testing.F) {
	f.Add(40.7128, -74.006, int64(1705276800)) // NYC 2024-01-15
	f.Add(-33.87, 151.21, int64(1705276800))   // Sydney

	f.Fuzz(func(t *testing.T, lat, lon float64, unix int64) {
		date := time.Unix(unix, 0).UTC()
		if date.Year() < 1800 || date.Year() > 2200 {
			return
		}

		obs, err := NewObserver(lat, lon, time.UTC)
		if err != nil {
			return // invalid coordinates rejected by NewObserver
		}

		moon, err := MoonriseMoonset(date, obs)
		if err != nil {
			return
		}

		// MoonEvent is documented as the rise and set times "on a given day", so
		// a non-zero time outside the scanned local day is a defect. The scan
		// length comes from the gap between local midnights, which is the part a
		// DST boundary could plausibly get wrong.
		local := date.In(obs.loc)
		lo := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, obs.loc)
		hi := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, obs.loc)

		for _, e := range []struct {
			name string
			at   time.Time
		}{{"Rise", moon.Rise}, {"Set", moon.Set}} {
			if e.at.IsZero() {
				continue // the Moon need not rise or set on a given day
			}

			if e.at.Before(lo) || !e.at.Before(hi) {
				t.Errorf("%s %v falls outside the scanned local day [%v, %v)", e.name, e.at, lo, hi)
			}
		}
	})
}
