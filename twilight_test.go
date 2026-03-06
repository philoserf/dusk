package dusk

import (
	"testing"
	"time"
)

func TestGetLocalCivilTwilightFrom(t *testing.T) {
	timezone, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	twilight, location := GetLocalCivilTwilight(d, longitude, latitude, elevation, timezone)

	if timezone.String() != location.String() {
		t.Errorf("got %q, wanted %q", location.String(), timezone)
	}

	got := twilight.From

	want := time.Date(1992, 4, 12, 19, 0o4, 57, 329480960, timezone)

	if got.Before(want) {
		t.Errorf("got %q, wanted %q", got, want)
	}

	if want.After(got) {
		t.Errorf("got %q, wanted %q", got, want)
	}

	if !got.Equal(want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestGetLocalCivilTwilightUntil(t *testing.T) {
	timezone, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	twilight, location := GetLocalCivilTwilight(d, longitude, latitude, elevation, timezone)

	if timezone.String() != location.String() {
		t.Errorf("got %q, wanted %q", location.String(), timezone)
	}

	got := twilight.Until

	want := time.Date(1992, 4, 13, 5, 38, 34, 818866560, timezone)

	if got.Before(want) {
		t.Errorf("got %q, wanted %q", got, want)
	}

	if want.After(got) {
		t.Errorf("got %q, wanted %q", got, want)
	}

	if !got.Equal(want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestGetLocalCivilTwilightDuration(t *testing.T) {
	timezone, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	twilight, location := GetLocalCivilTwilight(d, longitude, latitude, elevation, timezone)

	if timezone.String() != location.String() {
		t.Errorf("got %q, wanted %q", location.String(), timezone)
	}

	got := twilight.Duration

	want := time.Duration(38017489385600)

	if got.Nanoseconds() != want.Nanoseconds() {
		t.Errorf("got %d, wanted %d", got.Nanoseconds(), want.Nanoseconds())
	}
}

func TestGetLocalNauticalTwilightFrom(t *testing.T) {
	timezone, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	twilight, location := GetLocalNauticalTwilight(d, longitude, latitude, elevation, timezone)

	if timezone.String() != location.String() {
		t.Errorf("got %q, wanted %q", location.String(), timezone)
	}

	got := twilight.From

	want := time.Date(1992, 4, 12, 19, 31, 11, 189139712, timezone)

	if got.Before(want) {
		t.Errorf("got %q, wanted %q", got, want)
	}

	if want.After(got) {
		t.Errorf("got %q, wanted %q", got, want)
	}

	if !got.Equal(want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestGetLocalNauticalTwilightUntil(t *testing.T) {
	timezone, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	twilight, location := GetLocalNauticalTwilight(d, longitude, latitude, elevation, timezone)

	if timezone.String() != location.String() {
		t.Errorf("got %q, wanted %q", location.String(), timezone)
	}

	got := twilight.Until

	want := time.Date(1992, 4, 13, 5, 12, 18, 311666304, timezone)

	if got.Before(want) {
		t.Errorf("got %q, wanted %q", got, want)
	}

	if want.After(got) {
		t.Errorf("got %q, wanted %q", got, want)
	}

	if !got.Equal(want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestGetLocalNauticalTwilightDuration(t *testing.T) {
	timezone, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	twilight, location := GetLocalNauticalTwilight(d, longitude, latitude, elevation, timezone)

	if timezone.String() != location.String() {
		t.Errorf("got %q, wanted %q", location.String(), timezone)
	}

	got := twilight.Duration

	want := time.Duration(34867122526592)

	if got.Nanoseconds() != want.Nanoseconds() {
		t.Errorf("got %d, wanted %d", got.Nanoseconds(), want.Nanoseconds())
	}
}

func TestGetLocalAstronomicalTwilightFrom(t *testing.T) {
	timezone, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	twilight, location := GetLocalAstronomicalTwilight(d, longitude, latitude, elevation, timezone)

	if timezone.String() != location.String() {
		t.Errorf("got %q, wanted %q", location.String(), timezone)
	}

	got := twilight.From

	want := time.Date(1992, 4, 12, 19, 57, 44, 24917760, timezone)

	if got.Before(want) {
		t.Errorf("got %q, wanted %q", got, want)
	}

	if want.After(got) {
		t.Errorf("got %q, wanted %q", got, want)
	}

	if !got.Equal(want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestGetLocalAstronomicalTwilightUntil(t *testing.T) {
	timezone, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	twilight, location := GetLocalAstronomicalTwilight(d, longitude, latitude, elevation, timezone)

	if timezone.String() != location.String() {
		t.Errorf("got %q, wanted %q", location.String(), timezone)
	}

	got := twilight.Until

	want := time.Date(1992, 4, 13, 4, 45, 42, 138791168, timezone)

	if got.Before(want) {
		t.Errorf("got %q, wanted %q", got, want)
	}

	if want.After(got) {
		t.Errorf("got %q, wanted %q", got, want)
	}

	if !got.Equal(want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestGetLocalAstronomicalTwilightDuration(t *testing.T) {
	timezone, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	twilight, location := GetLocalAstronomicalTwilight(d, longitude, latitude, elevation, timezone)

	if timezone.String() != location.String() {
		t.Errorf("got %q, wanted %q", location.String(), timezone)
	}

	got := twilight.Duration

	want := time.Duration(31678113873408)

	if got.Nanoseconds() != want.Nanoseconds() {
		t.Errorf("got %d, wanted %d", got.Nanoseconds(), want.Nanoseconds())
	}
}
