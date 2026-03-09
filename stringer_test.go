package dusk

import (
	"testing"
	"time"
)

func TestEquatorialString(t *testing.T) {
	eq := Equatorial{RA: 101.287, Dec: -16.716}
	want := "RA=101.287° Dec=-16.716°"
	if got := eq.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestHorizontalString(t *testing.T) {
	h := Horizontal{Alt: 45.123, Az: 180.456}
	want := "Alt=45.123° Az=180.456°"
	if got := h.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEclipticString(t *testing.T) {
	ec := Ecliptic{Lon: 123.456, Lat: 1.234, Dist: 384400}
	want := "Lon=123.456° Lat=1.234° Dist=384400.0km"
	if got := ec.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSunEventString(t *testing.T) {
	loc := time.FixedZone("TEST", -5*3600)
	s := SunEvent{
		Rise:     time.Date(2024, 1, 15, 12, 1, 0, 0, time.UTC).In(loc),
		Noon:     time.Date(2024, 1, 15, 17, 30, 0, 0, time.UTC).In(loc),
		Set:      time.Date(2024, 1, 15, 22, 59, 0, 0, time.UTC).In(loc),
		Duration: 10*time.Hour + 58*time.Minute,
	}
	want := "Rise=07:01 Noon=12:30 Set=17:59 Duration=10h58m0s"
	if got := s.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSunEventString_Zero(t *testing.T) {
	s := SunEvent{}
	want := "Rise=--:-- Noon=--:-- Set=--:-- Duration=0s"
	if got := s.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMoonEventString_Zero(t *testing.T) {
	m := MoonEvent{}
	want := "Rise=--:-- Set=--:-- Duration=0s"
	if got := m.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMoonEventString(t *testing.T) {
	loc := time.FixedZone("TEST", 0)
	m := MoonEvent{
		Rise:     time.Date(2024, 1, 15, 8, 15, 0, 0, loc),
		Set:      time.Date(2024, 1, 15, 20, 30, 0, 0, loc),
		Duration: 12*time.Hour + 15*time.Minute,
	}
	want := "Rise=08:15 Set=20:30 Duration=12h15m0s"
	if got := m.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLunarPhaseInfoString_Zero(t *testing.T) {
	l := LunarPhaseInfo{}
	want := "0.0% (day 0.0)"
	if got := l.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLunarPhaseInfoString(t *testing.T) {
	l := LunarPhaseInfo{
		Illumination: 75.3,
		DaysApprox:   11.2,
		Name:         "Waxing Gibbous",
	}
	want := "Waxing Gibbous 75.3% (day 11.2)"
	if got := l.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTransitString_Zero(t *testing.T) {
	tr := Transit{}
	want := "Rise=--:-- Max=--:-- Set=--:-- Duration=0s"
	if got := tr.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTransitString(t *testing.T) {
	loc := time.FixedZone("TEST", 0)
	tr := Transit{
		Rise:     time.Date(2024, 1, 15, 18, 1, 0, 0, loc),
		Maximum:  time.Date(2024, 1, 15, 23, 0, 0, 0, loc),
		Set:      time.Date(2024, 1, 16, 4, 0, 0, 0, loc),
		Duration: 9*time.Hour + 59*time.Minute,
	}
	want := "Rise=18:01 Max=23:00 Set=04:00 Duration=9h59m0s"
	if got := tr.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTwilightEventString_Zero(t *testing.T) {
	tw := TwilightEvent{}
	want := "Dusk=--:-- Dawn=--:-- Duration=0s"
	if got := tw.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTwilightEventString(t *testing.T) {
	loc := time.FixedZone("TEST", 0)
	tw := TwilightEvent{
		Dusk:     time.Date(2024, 1, 15, 18, 30, 0, 0, loc),
		Dawn:     time.Date(2024, 1, 16, 6, 15, 0, 0, loc),
		Duration: 11*time.Hour + 45*time.Minute,
	}
	want := "Dusk=18:30 Dawn=06:15 Duration=11h45m0s"
	if got := tw.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
