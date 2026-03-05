package dusk

import (
	"math"
	"testing"
	"time"
)

var datetime = time.Date(2021, 5, 14, 0, 0, 0, 0, time.UTC)

var longitude = -155.468094

func TestGetDatetimeZeroHour(t *testing.T) {
	got := GetDatetimeZeroHour(time.Date(2021, 5, 14, 12, 56, 18, 4, time.UTC))

	want := time.Date(2021, 5, 14, 0, 0, 0, 0, time.UTC)

	if got.String() != want.String() {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestGetJulianDate(t *testing.T) {
	got := GetJulianDate(datetime)

	want := 2459348.5

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetUniversalTime(t *testing.T) {
	got := GetUniversalTime(2459348.5)

	want := time.Date(2021, 5, 14, 0, 0, 0, 0, time.UTC)

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}

func TestGetGreenwichSiderealTime(t *testing.T) {
	got := GetGreenwichSiderealTime(datetime)

	want := 15.46396124

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetGreenwichSiderealTimeLawrence(t *testing.T) {
	datetime := time.Date(2010, 2, 7, 23, 30, 0, 0, time.UTC)

	got := GetGreenwichSiderealTime(datetime)

	want := 8.698091

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLocalSiderealTime(t *testing.T) {
	got := GetLocalSiderealTime(datetime, longitude)

	want := 5.099422

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetCurrentJulianDayJLLawrence(t *testing.T) {
	datetime := time.Date(2015, 2, 5, 12, 0, 0, 0, time.UTC)

	got := GetCurrentJulianDayRelativeToJ2000(datetime)

	want := 5514

	if got != want {
		t.Errorf("got %d, wanted %d", got, want)
	}
}

func TestGetFractionalJulianDayStandardEpoch(t *testing.T) {
	datetime := time.Date(2015, 2, 5, 17, 0, 0, 0, time.UTC)

	got := GetFractionalJulianDaysSinceStandardEpoch(datetime)

	want := 5514.208333

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetCurrentJulianDay(t *testing.T) {
	got := GetCurrentJulianDayRelativeToJ2000(datetime)

	want := 7804

	if got != want {
		t.Errorf("got %d, wanted %d", got, want)
	}
}

func TestGetCurrentJulianCentury(t *testing.T) {
	got := GetCurrentJulianCenturyRelativeToJ2000(datetime)

	want := 0.21364818617385353

	if got != want {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetCurrentJulianPeriodJulianDate(t *testing.T) {
	period := GetCurrentJulianPeriod(datetime)

	got := period.JD

	want := 2459348.5

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetCurrentJulianPeriodJulianCenturies(t *testing.T) {
	period := GetCurrentJulianPeriod(datetime)

	got := period.T

	want := 0.21364818617385353

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetMeanGreenwichSiderealTimeInDegrees(t *testing.T) {
	// For testing we need to specify a date because most calculations are
	// differential w.r.t a time component. We set it to the date provided
	// on p.89 of Meeus, Jean. 1991. Astronomical algorithms. Richmond,
	// Va: Willmann - Bell.:
	datetime := time.Date(1987, 4, 10, 19, 21, 0, 0, time.UTC)

	got := GetMeanGreenwichSiderealTimeInDegrees(datetime)

	want := 197.693195

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetApparentGreenwichSiderealTimeInDegrees(t *testing.T) {
	// For testing we need to specify a date because most calculations are
	// differential w.r.t a time component. We set it to the date provided
	// on p.88 of Meeus, Jean. 1991. Astronomical algorithms. Richmond,
	// Va: Willmann - Bell.:
	datetime := time.Date(1987, 4, 10, 0, 0, 0, 0, time.UTC)

	got := GetApparentGreenwichSiderealTimeInDegrees(datetime)

	want := 197.692600

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetApparentGreenwichSiderealTimeInDegreesBonus(t *testing.T) {
	// For testing we need to specify a date because most calculations are
	// differential w.r.t a time component. We set it to the date provided
	// on p.103 of Meeus, Jean. 1991. Astronomical algorithms. Richmond,
	// Va: Willmann - Bell.:
	datetime := time.Date(1988, 3, 20, 0, 0, 0, 0, time.UTC)

	got := GetApparentGreenwichSiderealTimeInDegrees(datetime)

	want := 177.741993

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetMeanSolarTime(t *testing.T) {
	got := GetMeanSolarTime(datetime, longitude)

	want := 7804.431856

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestConvertLocalSiderealTimeToGreenwichSiderealTime(t *testing.T) {
	got := ConvertLocalSiderealTimeToGreenwichSiderealTime(23.394722, 50.0)

	want := 20.061389

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestConvertGreenwichSiderealTimeToUniversalTime(t *testing.T) {
	datetime := time.Date(2010, 2, 7, 0, 0, 0, 0, time.UTC)

	got := ConvertGreenwichSiderealTimeToUniversalTime(datetime, 8.698056)

	want := 23.499977

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}
