package dusk

import (
	"math"
	"testing"
	"time"
)

// For testing we need to specify a date because most calculations are
// differential w.r.t a time component. We set it to the date provided
// on p.342 of Meeus, Jean. 1991. Astronomical algorithms.Richmond,
// Va: Willmann - Bell.:
var d = time.Date(1992, 4, 12, 0, 0, 0, 0, time.UTC)

var latitude = 19.798484

var elevation = 0.0

func TestGetSolarMeanAnomaly(t *testing.T) {
	J := GetMeanSolarTime(d, longitude)

	got := GetSolarMeanAnomaly(J)

	want := 98.561957

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetSolarEquationOfCenter(t *testing.T) {
	J := GetMeanSolarTime(d, longitude)

	M := GetSolarMeanAnomaly(J)

	got := GetSolarEquationOfCenter(M)

	want := 1.887301

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetSolarEclipticLongitude(t *testing.T) {
	J := GetMeanSolarTime(d, longitude)

	M := GetSolarMeanAnomaly(J)

	C := GetSolarEquationOfCenter(M)

	got := GetSolarEclipticLongitude(M, C)

	want := 23.386458

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetSolarTransit(t *testing.T) {
	J := GetMeanSolarTime(d, longitude)

	M := GetSolarMeanAnomaly(J)

	C := GetSolarEquationOfCenter(M)

	λ := GetSolarEclipticLongitude(M, C)

	got := GetSolarTransitJulianDate(J, M, λ)

	want := 2448725.432069

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetSolarDeclination(t *testing.T) {
	J := GetMeanSolarTime(d, longitude)

	M := GetSolarMeanAnomaly(J)

	C := GetSolarEquationOfCenter(M)

	λ := GetSolarEclipticLongitude(M, C)

	got := GetSolarDeclination(λ)

	want := 9.084711

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetSolarMeanLongitude(t *testing.T) {
	J := GetCurrentJulianCenturyRelativeToJ2000(d)

	got := GetSolarMeanLongitude(J)

	want := 20.448123

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetSolarHourAngle(t *testing.T) {
	J := GetMeanSolarTime(d, longitude)

	M := GetSolarMeanAnomaly(J)

	C := GetSolarEquationOfCenter(M)

	λ := GetSolarEclipticLongitude(M, C)

	δ := GetSolarDeclination(λ)

	got := GetSolarHourAngle(δ, 0, latitude, elevation)

	want := 94.195177

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetSunriseSunsetTimesRise(t *testing.T) {
	timezone, _ := time.LoadLocation("Pacific/Honolulu")

	sun, err := GetSunriseSunsetTimes(d, 0, longitude, latitude, elevation)
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	got := sun.Rise

	want := time.Date(1992, 4, 12, 6, 0o5, 23, 927740672, timezone)

	if got.String() != want.String() {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestGetSunriseSunsetTimesNoon(t *testing.T) {
	timezone, _ := time.LoadLocation("Pacific/Honolulu")

	sun, err := GetSunriseSunsetTimes(d, 0, longitude, latitude, elevation)
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	got := sun.Noon

	want := time.Date(1992, 4, 12, 12, 22, 10, 770278016, timezone)

	if got.String() != want.String() {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestGetSunriseSunsetTimesSet(t *testing.T) {
	timezone, _ := time.LoadLocation("Pacific/Honolulu")

	sun, err := GetSunriseSunsetTimes(d, 0, longitude, latitude, elevation)
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	got := sun.Set

	want := time.Date(1992, 4, 12, 18, 38, 57, 612815232, timezone)

	if got.String() != want.String() {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestGetSunriseSunsetTimesInUTCRise(t *testing.T) {
	timezone, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	sun := GetSunriseSunsetTimesInUTC(d, 0, longitude, latitude, elevation)

	got := sun.Rise.In(timezone)

	want := time.Date(1992, 4, 12, 6, 0o5, 23, 927740672, timezone)

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestGetSunriseSunsetTimesInUTCRiseWithOffsetHorizon(t *testing.T) {
	timezone, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	sun := GetSunriseSunsetTimesInUTC(d, -18, longitude, latitude, elevation)

	got := sun.Rise.In(timezone)

	want := time.Date(1992, 4, 12, 6, 0o5, 23, 927740672, timezone)

	if got.After(want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestGetSunriseSunsetTimesInUTCNoon(t *testing.T) {
	timezone, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	sun := GetSunriseSunsetTimesInUTC(d, 0, longitude, latitude, elevation)

	got := sun.Noon.In(timezone)

	want := time.Date(1992, 4, 12, 12, 22, 10, 770278016, timezone)

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestGetSunriseSunsetTimesInUTCSet(t *testing.T) {
	timezone, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	sun := GetSunriseSunsetTimesInUTC(d, 0, longitude, latitude, elevation)

	got := sun.Set.In(timezone)

	want := time.Date(1992, 4, 12, 18, 38, 57, 612815232, timezone)

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestGetSunriseSunsetTimesInUTCSetWithOffsetHorizon(t *testing.T) {
	timezone, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Errorf("got %q", err)
		return
	}

	sun := GetSunriseSunsetTimesInUTC(d, -18, longitude, latitude, elevation)

	got := sun.Set.In(timezone)

	want := time.Date(1992, 4, 12, 18, 38, 57, 612815232, timezone)

	if got.Before(want) {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestGetSolarEclipticPositionLongitude(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 2, 5, 17, 0, 0, 0, time.UTC)

	ec := GetSolarEclipticPosition(datetime)

	got := ec.Longitude

	want := 316.562255

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetSolarEclipticPositionLatitude(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 2, 5, 17, 0, 0, 0, time.UTC)

	ec := GetSolarEclipticPosition(datetime)

	got := ec.Latitude

	want := 0.0

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetSolarEquatorialPositionRightAscension(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 2, 5, 17, 0, 0, 0, time.UTC)

	eq := GetSolarEquatorialPosition(datetime)

	got := eq.RightAscension

	want := 319.017015

	if math.Abs(got-want) > 0.01 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetSolarEquatorialPositionDeclination(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 2, 5, 17, 0, 0, 0, time.UTC)

	eq := GetSolarEquatorialPosition(datetime)

	got := eq.Declination

	want := -15.872529

	if math.Abs(got-want) > 0.01 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}
