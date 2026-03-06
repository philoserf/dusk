package dusk

import (
	"errors"
	"testing"
	"time"
)

func TestObjectTransit(t *testing.T) {
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	// Sirius: RA=101.287°, Dec=-16.716° from NYC on 2024-01-15.
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	sirius := Equatorial{RA: 101.287, Dec: -16.716}
	obs := Observer{Lat: 40.7128, Lon: -74.006, Loc: nyc}

	tr, err := ObjectTransit(date, sirius, obs)
	if err != nil {
		t.Fatalf("ObjectTransit() returned error: %v", err)
	}

	tolerance := 10 * time.Minute

	// Object should rise and set (Dec > -(90-lat)).
	if tr.Rise.IsZero() {
		t.Fatal("expected Rise to be non-zero")
	}
	if tr.Set.IsZero() {
		t.Fatal("expected Set to be non-zero")
	}

	// Duration should be positive and less than 24 hours.
	if tr.Duration <= 0 {
		t.Errorf("Duration = %v, want > 0", tr.Duration)
	}
	if tr.Duration >= 24*time.Hour {
		t.Errorf("Duration = %v, want < 24h", tr.Duration)
	}

	// Maximum should be between Rise and Set.
	if tr.Maximum.Before(tr.Rise) || tr.Maximum.After(tr.Set) {
		t.Errorf("Maximum %v not between Rise %v and Set %v", tr.Maximum, tr.Rise, tr.Set)
	}

	// Rise should be before Set.
	if !tr.Rise.Before(tr.Set) {
		t.Errorf("Rise %v should be before Set %v", tr.Rise, tr.Set)
	}

	// Duration should match Set - Rise within tolerance.
	computedDuration := tr.Set.Sub(tr.Rise)
	if diff := tr.Duration - computedDuration; diff < -tolerance || diff > tolerance {
		t.Errorf("Duration %v != Set-Rise %v", tr.Duration, computedDuration)
	}
}

func TestObjectTransit_SouthernHemisphere(t *testing.T) {
	// Canopus (RA=95.988°, Dec=-52.696°) from Sydney on 2024-01-15.
	// From lat -33.87°, Canopus (Dec=-52.696°) should rise and set
	// because |Dec| < 90 - |lat| is false, but Dec and lat same sign
	// means the object is circumpolar or rises — it should be visible.
	loc, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	canopus := Equatorial{RA: 95.988, Dec: -52.696}
	obs := Observer{Lat: -33.87, Lon: 151.21, Loc: loc}

	tr, err := ObjectTransit(date, canopus, obs)

	// Canopus from Sydney: Dec=-52.696° and lat=-33.87°.
	// Both lat and dec are negative (southern), so the object is high in the sky.
	// It may be circumpolar from Sydney.
	if errors.Is(err, ErrCircumpolar) {
		t.Logf("Canopus is circumpolar from Sydney")
		return
	}
	if err != nil {
		t.Fatalf("ObjectTransit() returned error: %v", err)
	}

	if !tr.Rise.IsZero() && !tr.Set.IsZero() {
		if tr.Duration <= 0 {
			t.Errorf("Duration = %v, want > 0 when Rise and Set are non-zero", tr.Duration)
		}
		if !tr.Rise.Before(tr.Set) {
			t.Errorf("Rise %v should be before Set %v", tr.Rise, tr.Set)
		}
		t.Logf("Canopus from Sydney: rise=%v set=%v duration=%v", tr.Rise, tr.Set, tr.Duration)
	}
}

func TestObjectTransit_Circumpolar(t *testing.T) {
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	// Polaris: RA=37.95°, Dec=89.264° from NYC.
	// |Ar| > 1, so object never sets → circumpolar.
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	polaris := Equatorial{RA: 37.95, Dec: 89.264}
	obs := Observer{Lat: 40.7128, Lon: -74.006, Loc: nyc}

	_, err = ObjectTransit(date, polaris, obs)
	if !errors.Is(err, ErrCircumpolar) {
		t.Errorf("expected ErrCircumpolar for Polaris from NYC, got %v", err)
	}
}

func TestObjectTransit_NeverRises(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	// Object at Dec=-80° from lat=60°N: should never rise.
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	obj := Equatorial{RA: 180.0, Dec: -80.0}
	obs := Observer{Lat: 60.0, Lon: 10.0, Loc: loc}

	_, err = ObjectTransit(date, obj, obs)
	if !errors.Is(err, ErrNeverRises) {
		t.Errorf("expected ErrNeverRises for Dec=-80° from lat=60°N, got %v", err)
	}
}

func TestObjectTransit_NilLocation(t *testing.T) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	sirius := Equatorial{RA: 101.287, Dec: -16.716}
	_, err := ObjectTransit(date, sirius, Observer{Lat: 40.7128, Lon: -74.006})
	if err == nil {
		t.Error("ObjectTransit() with nil location should return error")
	}
}

func TestObjectTransit_InvalidObserverCoordinates(t *testing.T) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	sirius := Equatorial{RA: 101.287, Dec: -16.716}
	_, err := ObjectTransit(date, sirius, Observer{Lat: 0, Lon: 200, Loc: time.UTC})
	if err == nil {
		t.Error("expected error for invalid observer coordinates")
	}
}

func TestObjectTransit_InvalidEquatorial(t *testing.T) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	obs := Observer{Lat: 40.7128, Lon: -74.006, Loc: time.UTC}

	tests := []struct {
		name string
		eq   Equatorial
	}{
		{"RA too high", Equatorial{RA: 400, Dec: 0}},
		{"RA negative", Equatorial{RA: -1, Dec: 0}},
		{"Dec too high", Equatorial{RA: 0, Dec: 91}},
		{"Dec too low", Equatorial{RA: 0, Dec: -91}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ObjectTransit(date, tt.eq, obs)
			if err == nil {
				t.Error("expected error for invalid equatorial coordinates")
			}
		})
	}
}
