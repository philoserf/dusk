package dusk

import (
	"math"
	"testing"
	"time"
)

func TestLunarEclipticPosition(t *testing.T) {
	// Meeus p. 342: 1992-04-12 00:00 UTC
	dt := time.Date(1992, 4, 12, 0, 0, 0, 0, time.UTC)
	ec := LunarEclipticPosition(dt)

	tests := []struct {
		name    string
		got     float64
		want    float64
		epsilon float64
	}{
		{"longitude", ec.Lon, 133.162655, 0.001},
		{"latitude", ec.Lat, -3.229126, 0.001},
		{"distance", ec.Dist, 368409.7, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if math.Abs(tt.got-tt.want) > tt.epsilon {
				t.Errorf("%s = %f, want %f (±%f)", tt.name, tt.got, tt.want, tt.epsilon)
			}
		})
	}
}

func TestLunarPosition(t *testing.T) {
	// Meeus p. 342: 1992-04-12 00:00 UTC.
	// Expected equatorial coordinates (nutation-corrected): RA ~134.7°, Dec ~13.8°.
	dt := time.Date(1992, 4, 12, 0, 0, 0, 0, time.UTC)
	eq := LunarPosition(dt)

	if math.Abs(eq.RA-134.7) > 0.5 {
		t.Errorf("RA = %.4f, want ~134.7°", eq.RA)
	}
	if math.Abs(eq.Dec-13.8) > 0.5 {
		t.Errorf("Dec = %.4f, want ~13.8°", eq.Dec)
	}
}

func TestLunarPhase(t *testing.T) {
	tests := []struct {
		name       string
		date       time.Time
		wantLow    float64 // illumination lower bound
		wantHigh   float64 // illumination upper bound
		wantName   string  // expected phase name (empty to skip)
		wantWaxing bool
	}{
		{
			name:       "near new moon 2024-01-11",
			date:       time.Date(2024, 1, 11, 12, 0, 0, 0, time.UTC),
			wantLow:    0,
			wantHigh:   5,
			wantName:   "New Moon",
			wantWaxing: true,
		},
		{
			name:       "near first quarter 2024-01-18",
			date:       time.Date(2024, 1, 18, 3, 0, 0, 0, time.UTC),
			wantLow:    40,
			wantHigh:   60,
			wantName:   "First Quarter",
			wantWaxing: true,
		},
		{
			name:       "near full moon 2024-01-25",
			date:       time.Date(2024, 1, 25, 18, 0, 0, 0, time.UTC),
			wantLow:    95,
			wantHigh:   100,
			wantName:   "Full Moon",
			wantWaxing: false, // exact full moon was 17:54 UTC; by 18:00 elongation > 180°
		},
		{
			name:       "waxing crescent 2024-01-14",
			date:       time.Date(2024, 1, 14, 12, 0, 0, 0, time.UTC),
			wantLow:    5,
			wantHigh:   25,
			wantName:   "Waxing Crescent",
			wantWaxing: true,
		},
		{
			name:       "waxing gibbous 2024-01-21",
			date:       time.Date(2024, 1, 21, 12, 0, 0, 0, time.UTC),
			wantLow:    70,
			wantHigh:   90,
			wantName:   "Waxing Gibbous",
			wantWaxing: true,
		},
		{
			name:       "waning gibbous 2024-01-28",
			date:       time.Date(2024, 1, 28, 12, 0, 0, 0, time.UTC),
			wantLow:    70,
			wantHigh:   95,
			wantName:   "Waning Gibbous",
			wantWaxing: false,
		},
		{
			name:       "near last quarter 2024-02-02",
			date:       time.Date(2024, 2, 2, 23, 0, 0, 0, time.UTC),
			wantLow:    40,
			wantHigh:   60,
			wantName:   "Last Quarter",
			wantWaxing: false,
		},
		{
			name:       "waning crescent 2024-02-06",
			date:       time.Date(2024, 2, 6, 12, 0, 0, 0, time.UTC),
			wantLow:    5,
			wantHigh:   30,
			wantName:   "Waning Crescent",
			wantWaxing: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := LunarPhase(tt.date)

			if p.Illumination < tt.wantLow || p.Illumination > tt.wantHigh {
				t.Errorf("illumination = %.2f%%, want [%.0f, %.0f]",
					p.Illumination, tt.wantLow, tt.wantHigh)
			}

			if tt.wantName != "" && p.Name != tt.wantName {
				t.Errorf("name = %q, want %q", p.Name, tt.wantName)
			}

			if p.Waxing != tt.wantWaxing {
				t.Errorf("Waxing = %v, want %v", p.Waxing, tt.wantWaxing)
			}
		})
	}
}

func TestMoonriseMoonset(t *testing.T) {
	// NYC 2024-01-15
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, loc)
	obs := Observer{Lat: 40.7128, Lon: -74.0060, Loc: loc}

	evt, err := MoonriseMoonset(date, obs)
	if err != nil {
		t.Fatal(err)
	}

	if evt.Rise.IsZero() {
		t.Error("expected non-zero rise time")
	}
	if evt.Set.IsZero() {
		t.Error("expected non-zero set time")
	}
	if evt.Duration <= 0 {
		t.Error("expected positive duration")
	}

	// Regression reference: algorithm-computed values for NYC 2024-01-15.
	// Note: simplified Meeus approach can differ from USNO by up to ~1.5h for the Moon.
	tolerance := 5 * time.Minute
	wantRise := time.Date(2024, 1, 15, 10, 6, 0, 0, loc)
	wantSet := time.Date(2024, 1, 15, 22, 13, 0, 0, loc)

	if diff := evt.Rise.Sub(wantRise); diff < -tolerance || diff > tolerance {
		t.Errorf("Rise = %v, want %v (±%v, diff=%v)", evt.Rise.Format("15:04"), wantRise.Format("15:04"), tolerance, diff)
	}
	if diff := evt.Set.Sub(wantSet); diff < -tolerance || diff > tolerance {
		t.Errorf("Set = %v, want %v (±%v, diff=%v)", evt.Set.Format("15:04"), wantSet.Format("15:04"), tolerance, diff)
	}

	t.Logf("rise=%v set=%v duration=%v", evt.Rise, evt.Set, evt.Duration)
}

func TestMoonriseMoonset_SouthernHemisphere(t *testing.T) {
	// Sydney, Australia on 2024-01-15.
	loc, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		t.Fatal(err)
	}

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, loc)
	obs := Observer{Lat: -33.87, Lon: 151.21, Loc: loc}

	evt, err := MoonriseMoonset(date, obs)
	if err != nil {
		t.Fatal(err)
	}

	if evt.Rise.IsZero() {
		t.Error("expected non-zero rise time for Sydney")
	}
	if evt.Set.IsZero() {
		t.Error("expected non-zero set time for Sydney")
	}
	if evt.Duration <= 0 {
		t.Error("expected positive duration for Sydney")
	}

	// Approximate reference times for Sydney 2024-01-15. The simplified Meeus
	// algorithm can differ from USNO by up to ~45 minutes for moonrise/set.
	// Library computes rise ~09:47, set ~23:06 AEDT.
	tolerance := 20 * time.Minute
	wantRise := time.Date(2024, 1, 15, 9, 50, 0, 0, loc)
	wantSet := time.Date(2024, 1, 15, 23, 0, 0, 0, loc)

	if diff := evt.Rise.Sub(wantRise); diff < -tolerance || diff > tolerance {
		t.Errorf("Rise = %v, want %v (±%v, diff=%v)", evt.Rise.Format("15:04"), wantRise.Format("15:04"), tolerance, diff)
	}
	if diff := evt.Set.Sub(wantSet); diff < -tolerance || diff > tolerance {
		t.Errorf("Set = %v, want %v (±%v, diff=%v)", evt.Set.Format("15:04"), wantSet.Format("15:04"), tolerance, diff)
	}

	t.Logf("Sydney moonrise=%v moonset=%v duration=%v", evt.Rise, evt.Set, evt.Duration)
}

func TestMoonriseMoonset_NilLocation(t *testing.T) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	_, err := MoonriseMoonset(date, Observer{Lat: 40.0, Lon: -74.0})
	if err == nil {
		t.Error("expected error for nil location, got nil")
	}
}

func TestMoonriseMoonset_InvalidCoordinates(t *testing.T) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	_, err := MoonriseMoonset(date, Observer{Lat: 91, Lon: 0, Loc: time.UTC})
	if err == nil {
		t.Error("expected error for invalid coordinates")
	}
}
