package dusk

import (
	"testing"
	"time"
)

func TestGetDoesObjectRiseOrSetBetelgeuseNorthernHemisphere(t *testing.T) {
	got := GetDoesObjectRiseOrSet(EquatorialCoordinate{RightAscension: 88.7929583, Declination: 7.4070639}, 38.778132)

	want := true

	if got != want {
		t.Errorf("got %t, wanted %t", got, want)
	}
}

func TestGetDoesObjectRiseOrSetBetelgeuseSouthernHemisphere(t *testing.T) {
	got := GetDoesObjectRiseOrSet(EquatorialCoordinate{RightAscension: 88.7929583, Declination: 7.4070639}, -89.191006)

	want := false

	if got != want {
		t.Errorf("got %t, wanted %t", got, want)
	}
}

func TestGetDoesObjectRiseOrSetArcturusNorthernHemisphere(t *testing.T) {
	got := GetDoesObjectRiseOrSet(EquatorialCoordinate{RightAscension: 213.9153, Declination: 19.182409}, 38.778132)

	want := true

	if got != want {
		t.Errorf("got %t, wanted %t", got, want)
	}
}

func TestGetDoesObjectRiseOrSetArcturusSouthernHemisphere(t *testing.T) {
	got := GetDoesObjectRiseOrSet(EquatorialCoordinate{RightAscension: 213.9153, Declination: 19.182409}, -89.191006)

	want := false

	if got != want {
		t.Errorf("got %t, wanted %t", got, want)
	}
}

func TestGetObjectHorizontalCoordinatesForDay(t *testing.T) {
	datetime := time.Date(2022, 5, 14, 0, 0, 0, 0, time.UTC)

	location, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	got := GetObjectHorizontalCoordinatesForDay(datetime, EquatorialCoordinate{RightAscension: 88.7929583, Declination: 7.4070639}, -155.468094, 19.798484, location)

	if got[0].Datetime.String() != "2022-05-14 00:00:00 -1000 HST" {
		t.Errorf("got %v, wanted %v", got[0].Datetime, "2022-05-14 00:00:00 -1000 HST")
	}

	if got[1439].Datetime.String() != "2022-05-14 23:59:00 -1000 HST" {
		t.Errorf("got %v, wanted %v", got[1439].Datetime, "2022-05-14 23:59:00 -1000 HST")
	}

	if got[517].Datetime.String() != "2022-05-14 08:37:00 -1000 HST" && !got[517].IsRise {
		t.Errorf("We're expecting Betelgeuse to rise at 8:37am on 14th May 2022")
	}

	if got[1256].Datetime.String() != "2022-05-14 20:56:00 -1000 HST" && !got[1256].IsSet {
		t.Errorf("We're expecting Betelgeuse to set at 8:56pm on 14th May 2022")
	}
}

func TestGetObjectRiseObjectSetTimesInUTCLawrenceChapter5Exercise1(t *testing.T) {
	datetime := time.Date(2015, 6, 6, 0, 0, 0, 0, time.UTC)

	got := GetObjectRiseObjectSetTimesInUTC(datetime, EquatorialCoordinate{RightAscension: 90, Declination: -60}, 45.250132, -100.300288)

	if got.Rise != nil {
		t.Errorf("got %v, but expected the object to never rise for the given paramaters", got)
	}

	if got.Set != nil {
		t.Errorf("got %v, but expected the object to never set for the given parameters", got)
	}
}

func TestGetObjectRiseObjectSetTimesInUTCLawrenceChapter5Exercise2(t *testing.T) {
	datetime := time.Date(2015, 6, 6, 0, 0, 0, 0, time.UTC)

	got := GetObjectRiseObjectSetTimesInUTC(datetime, EquatorialCoordinate{RightAscension: 243.675000, Declination: 25.9613889}, 38.250132, -78.300288)

	rise := time.Date(2015, 6, 6, 20, 57, 48, 562000000, time.UTC)

	set := time.Date(2015, 6, 7, 11, 55, 55, 501000000, time.UTC)

	if got.Rise.String() != rise.String() {
		t.Errorf("got %v, wanted %v", *got.Rise, rise)
	}

	if got.Set.String() != set.String() {
		t.Errorf("got %v, wanted %v", *got.Set, set)
	}
}

func TestGetObjectRiseObjectSetTimesChapter5Exercise1(t *testing.T) {
	datetime := time.Date(2015, 6, 6, 0, 0, 0, 0, time.UTC)

	location, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	got := GetObjectRiseObjectSetTimes(datetime, EquatorialCoordinate{RightAscension: 90, Declination: -60}, 45.250132, -100.300288, location)

	if got.Rise != nil {
		t.Errorf("got %v, but expected the object to never rise for the given paramaters", got)
	}

	if got.Set != nil {
		t.Errorf("got %v, but expected the object to never set for the given parameters", got)
	}
}

func TestGetObjectRiseObjectSetTimesChapter5Exercise2(t *testing.T) {
	timezone, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	datetime := time.Date(2015, 6, 6, 0, 0, 0, 0, time.UTC)

	got := GetObjectRiseObjectSetTimes(datetime, EquatorialCoordinate{RightAscension: 243.675000, Declination: 25.9613889}, 38.250132, -78.300288, timezone)

	rise := time.Date(2015, 6, 6, 16, 57, 48, 562000000, timezone)

	set := time.Date(2015, 6, 7, 7, 55, 55, 501000000, timezone)

	if rise.After(set) {
		t.Errorf("the object must rise before it sets")
	}

	if got.Rise.String() != rise.String() {
		t.Errorf("got %v, wanted %v", *got.Rise, rise)
	}

	if got.Set.String() != set.String() {
		t.Errorf("got %v, wanted %v", *got.Set, set)
	}
}

func TestGetObjectTransitMaximaTime(t *testing.T) {
	datetime := time.Date(2015, 6, 6, 0, 0, 0, 0, time.UTC)

	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	transit := GetObjectRiseObjectSetTimes(datetime, EquatorialCoordinate{RightAscension: 243.675000, Declination: 25.9613889}, 38.250132, -78.300288, location)

	got := GetObjectTransitMaximaTime(datetime, EquatorialCoordinate{RightAscension: 243.675000, Declination: 25.9613889}, 38.250132, -78.300288, location)

	if got.Before(*transit.Rise) || got.After(*transit.Set) {
		t.Errorf("maxima time must be between rise and set")
	}
}

func TestGetObjectTransitMaximaTimeNoRiseNoSet(t *testing.T) {
	datetime := time.Date(2015, 6, 6, 0, 0, 0, 0, time.UTC)

	location, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	got := GetObjectTransitMaximaTime(datetime, EquatorialCoordinate{RightAscension: 90, Declination: -60}, 45.250132, -100.300288, location)

	if got == nil {
		t.Errorf("expected the object to reach a maxima for the given paramaters")
	}
}

func TestGetObjectTransit(t *testing.T) {
	timezone, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	datetime := time.Date(2015, 6, 6, 0, 0, 0, 0, time.UTC)

	got := GetObjectTransit(datetime, EquatorialCoordinate{RightAscension: 243.675000, Declination: 25.9613889}, 38.250132, -78.300288, timezone)

	rise := time.Date(2015, 6, 6, 16, 57, 48, 562000000, timezone)

	set := time.Date(2015, 6, 7, 7, 55, 55, 501000000, timezone)

	if rise.After(set) {
		t.Errorf("the object must rise before it sets")
	}

	if got.Rise.String() != rise.String() {
		t.Errorf("got %v, wanted %v", *got.Rise, rise)
	}

	if got.Set.String() != set.String() {
		t.Errorf("got %v, wanted %v", *got.Set, set)
	}

	if got.Maximum == nil {
		t.Errorf("got %v, wanted a maxima time", got)
	}

	if got.Maximum.Before(*got.Rise) || got.Maximum.After(*got.Set) {
		t.Errorf("maxima time must be between rise and set")
	}
}

func TestGetObjectTransitForBetelgeuseAtHonolulu(t *testing.T) {
	timezone, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	datetime := time.Date(2022, 5, 14, 0, 0, 0, 0, time.UTC)

	got := GetObjectTransit(datetime, EquatorialCoordinate{RightAscension: 88.7929583, Declination: 7.4070639}, 19.798484, -155.300288, timezone)

	if got.Rise.After(*got.Set) {
		t.Errorf("the object must rise before it sets")
	}

	if got.Maximum == nil {
		t.Errorf("got %v, wanted a maxima time", got)
	}

	if got.Maximum.Before(*got.Rise) || got.Maximum.After(*got.Set) {
		t.Errorf("maxima time must be between rise and set")
	}
}

func TestGetObjectTransitNoRiseNoSetNoMaximum(t *testing.T) {
	datetime := time.Date(2015, 6, 6, 0, 0, 0, 0, time.UTC)

	location, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	got := GetObjectTransit(datetime, EquatorialCoordinate{RightAscension: 90, Declination: -60}, 45.250132, -100.300288, location)

	if got.Rise != nil {
		t.Errorf("got %v, but expected the object to never rise for the given paramaters", got)
	}

	if got.Set != nil {
		t.Errorf("got %v, but expected the object to never set for the given parameters", got)
	}

	if got.Maximum != nil {
		t.Errorf("got %v, but expected the object to never reach a maxima above the horizon for the given paramaters", got)
	}
}
