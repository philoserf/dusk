package dusk

import (
	"math"
	"testing"
	"time"
)

func TestGetObliquityOfTheEclipticLawrence(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)

	T := GetCurrentJulianCenturyRelativeToJ2000(datetime)

	got := GetObliquityOfTheEclipticLawrence(T)

	want := 23.437992

	if math.Abs(got-want) > 0.0001 {
		t.Errorf("quad %f, wanted %f and difference %f", got, want, math.Abs(got-want))
	}
}

func TestGetLunarMeanAnomalyLawrence(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	got := GetLunarMeanAnomalyLawrence(datetime)

	want := 85.910642

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarEclipticPositionLawrenceX(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	Ωprime := GetLunarCorrectedEclipticLongitudeOfTheAscendingNode(datetime)

	λt := GetLunarTrueEclipticLongitude(datetime)

	got := cosx(λt - Ωprime)

	want := -0.638869

	if math.Abs(got-want) > 0.1 {
		t.Errorf("quad %f, wanted %f and difference %f", got, want, math.Abs(got-want))
	}
}

func TestGetLunarEclipticPositionLawrenceY(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	Ωprime := GetLunarCorrectedEclipticLongitudeOfTheAscendingNode(datetime)

	λt := GetLunarTrueEclipticLongitude(datetime)

	// the inclination of the Moon's orbit with respect to the ecliptic
	ι := 5.1453964

	got := sinx(λt-Ωprime) * cosx(ι)

	want := -0.766215

	if math.Abs(got-want) > 0.1 {
		t.Errorf("quad %f, wanted %f and difference %f", got, want, math.Abs(got-want))
	}
}

func TestGetLunarEclipticPositionLawrenceLongitudeQuadrant(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	Ωprime := GetLunarCorrectedEclipticLongitudeOfTheAscendingNode(datetime)

	λt := GetLunarTrueEclipticLongitude(datetime)

	// the inclination of the Moon's orbit with respect to the ecliptic
	ι := 5.1453964

	x := cosx(λt - Ωprime)

	y := sinx(λt-Ωprime) * cosx(ι)

	// utilise atan2yx to determine a quadrant adjustment for arctan
	got := math.Mod(atan2yx(y, x), 360)

	// correct for negative angles
	if got < 0 {
		got += 360
	}

	want := 230.178711

	if math.Abs(got-want) > 0.25 {
		t.Errorf("quad %f, wanted %f and difference %f", got, want, math.Abs(got-want))
	}
}

func TestGetLunarEclipticPositionLawrenceLongitude(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	ec := GetLunarEclipticPositionLawrence(datetime)

	got := ec.Longitude

	want := 65.059853

	if math.Abs(got-want) > 0.25 {
		t.Errorf("quad %f, wanted %f and difference %f", got, want, math.Abs(got-want))
	}
}

func TestGetLunarEclipticPositionLawrenceLatitude(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	ec := GetLunarEclipticPositionLawrence(datetime)

	got := ec.Latitude

	want := -3.956258

	if math.Abs(got-want) > 0.1 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarEquatorialPositionLawrenceRightAscension(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	eq := GetLunarEquatorialPositionLawrence(datetime)

	got := eq.RightAscension

	want := 63.856001

	if math.Abs(got-want) > 0.25 {
		t.Errorf("got %f, wanted %f and difference %f", got, want, math.Abs(got-want))
	}
}

func TestGetLunarEquatorialPositionLawrenceDeclination(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	eq := GetLunarEquatorialPositionLawrence(datetime)

	got := eq.Declination

	want := 17.245991

	if math.Abs(got-want) > 0.25 {
		t.Errorf("got %f, wanted %f and difference %f", got, want, math.Abs(got-want))
	}
}

func TestGetSolarMeanAnomalyLawrence(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 2, 5, 17, 0, 0, 0, time.UTC)

	got := GetSolarMeanAnomalyLawrence(datetime)

	want := 32.592589

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetSolarMeanAnomalyLawrenceAlt(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	got := GetSolarMeanAnomalyLawrence(datetime)

	want := 358.505618

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetSolarEquationOfCenterLawrence(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 2, 5, 17, 0, 0, 0, time.UTC)

	M := GetSolarMeanAnomalyLawrence(datetime)

	got := GetSolarEquationOfCenterLawrence(M)

	want := 1.031320

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetSolarEclipticLongitudeLawrence(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 2, 5, 17, 0, 0, 0, time.UTC)

	M := GetSolarMeanAnomalyLawrence(datetime)

	C := GetSolarEquationOfCenterLawrence(M)

	got := GetSolarEclipticLongitudeLawrence(M, C)

	want := 316.562255

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetSolarEclipticLongitudeLawrenceAlt(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	M := GetSolarMeanAnomalyLawrence(datetime)

	C := GetSolarEquationOfCenterLawrence(M)

	got := GetSolarEclipticLongitudeLawrence(M, C)

	want := 281.394034

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}
