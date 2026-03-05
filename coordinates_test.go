package dusk

import (
	"math"
	"testing"
	"time"
)

func TestConvertEclipticCoordinateToEquatorialRA(t *testing.T) {
	// utilising the ecliptic position of the moon on the datetime provided:
	eq := ConvertEclipticCoordinateToEquatorial(d, EclipticCoordinate{Longitude: 133.162655, Latitude: -3.229126, Δ: 0})

	got := eq.RightAscension

	want := 134.683920

	if math.Abs(got-want) > 0.15 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestConvertEclipticCoordinateToEquatorialRAAlt(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, time.Month(1), 1, 0, 0, 0, 0, time.UTC)

	// utilising the ecliptic position of the moon on the datetime provided:
	eq := ConvertEclipticCoordinateToEquatorial(datetime, EclipticCoordinate{Longitude: 50.279952, Latitude: -2.981288, Δ: 0})

	got := eq.RightAscension

	want := 48.662544

	if math.Abs(got-want) > 0.15 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestConvertEclipticCoordinateToEquatorialDec(t *testing.T) {
	// utilising the ecliptic position of the moon on the datetime provided:
	ec := ConvertEclipticCoordinateToEquatorial(d, EclipticCoordinate{Longitude: 133.162655, Latitude: -3.229126, Δ: 0})

	got := ec.Declination

	want := 13.768368

	if math.Abs(got-want) > 0.15 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestConvertEclipticCoordinateToEquatorialDecAlt(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, time.Month(1), 1, 0, 0, 0, 0, time.UTC)

	// utilising the ecliptic position of the moon on the datetime provided:
	ec := ConvertEclipticCoordinateToEquatorial(datetime, EclipticCoordinate{Longitude: 50.279952, Latitude: -2.981288, Δ: 0})

	got := ec.Declination

	want := 14.941252

	if math.Abs(got-want) > 0.15 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestConvertEquatorialCoordinateTHorizontalAltitude(t *testing.T) {
	hz := ConvertEquatorialCoordinateToHorizontal(datetime, longitude, latitude, EquatorialCoordinate{RightAscension: 88.7929583, Declination: 7.4070639})

	got := hz.Altitude

	want := 72.800588

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestConvertEquatorialCoordinateTHorizontalAzimuth(t *testing.T) {
	hz := ConvertEquatorialCoordinateToHorizontal(datetime, longitude, latitude, EquatorialCoordinate{RightAscension: 88.7929583, Declination: 7.4070639})

	got := hz.Azimuth

	want := 134.396672

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}
