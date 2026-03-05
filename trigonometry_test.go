package dusk

import (
	"math"
	"testing"
)

func TestSinX(t *testing.T) {
	got := sinx(45)

	want := 0.70710678118

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestCosX(t *testing.T) {
	got := cosx(45)

	want := 0.70710678118

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestSinCosX(t *testing.T) {
	sinGot, cosGot := sincosx(45)

	sinWant := 0.70710678118

	cosWant := 0.70710678118

	if math.Abs(sinGot-sinWant) > 0.00001 {
		t.Errorf("got %f, wanted %f", sinGot, sinWant)
	}

	if math.Abs(cosGot-cosWant) > 0.00001 {
		t.Errorf("got %f, wanted %f", cosWant, cosWant)
	}
}

func TestTanX(t *testing.T) {
	got := tanx(45)

	want := 1.0

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestASinX(t *testing.T) {
	got := asinx(45)

	want := 0.903339111

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestACosX(t *testing.T) {
	got := acosx(45)

	want := 0.667457216

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestATanX(t *testing.T) {
	got := atanx(45)

	want := 88.726970

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestATan2YX(t *testing.T) {
	got := atan2yx(45, 45)

	want := 45.0

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}
