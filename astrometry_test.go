package dusk

import (
	"math"
	"testing"
)

func TestGetHourAngle(t *testing.T) {
	LST := GetLocalSiderealTime(datetime, longitude)

	got := GetHourAngle(88.7929583, LST)

	want := 347.698366

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetHourAngleBonus(t *testing.T) {
	LST := GetLocalSiderealTime(d, longitude)

	got := GetHourAngle(88.7929583, LST)

	want := 316.180845

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

var arcturus = Coordinate{Latitude: 19.1825, Longitude: 213.9154}

var spica = Coordinate{Latitude: -11.1614, Longitude: 201.2983}

var denebola = Coordinate{Latitude: 14.5720581, Longitude: 177.2649}

func TestGetAngularSeparationArcturusSpica(t *testing.T) {
	got := GetAngularSeparation(arcturus, spica)

	want := 32.793027

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetAngularSeparationSpicaDenebola(t *testing.T) {
	got := GetAngularSeparation(spica, denebola)

	want := 35.064334

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetAngularSeparationDenebolaArcturus(t *testing.T) {
	got := GetAngularSeparation(denebola, arcturus)

	want := 35.309668

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetAngularSeparationZero(t *testing.T) {
	coord1 := Coordinate{Latitude: 0, Longitude: 0}

	coord2 := Coordinate{Latitude: 0, Longitude: 0}

	got := GetAngularSeparation(coord1, coord2)

	want := 0.0

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}
