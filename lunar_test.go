package dusk

import (
	"math"
	"testing"
	"time"
)

func TestGetLunarMeanLongitude(t *testing.T) {
	J := GetCurrentJulianCenturyRelativeToJ2000(d)

	got := GetLunarMeanLongitude(J)

	want := 134.290182

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarMeanEclipticLongitude(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	got := GetLunarMeanEclipticLongitude(datetime)

	want := 59.716785

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarTrueEclipticLongitude(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	got := GetLunarTrueEclipticLongitude(datetime)

	want := 65.164007

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarMeanEclipticLongitudeOfTheAscendingNode(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	got := GetLunarMeanEclipticLongitudeOfTheAscendingNode(datetime)

	want := 194.877008

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarCorrectedEclipticLongitudeOfTheAscendingNode(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	got := GetLunarCorrectedEclipticLongitudeOfTheAscendingNode(datetime)

	want := 194.881180

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarMeanElongation(t *testing.T) {
	J := GetCurrentJulianCenturyRelativeToJ2000(d)

	got := GetLunarMeanElongation(J)

	want := 113.842304

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarMeanAnomaly(t *testing.T) {
	J := GetCurrentJulianCenturyRelativeToJ2000(d)

	got := GetLunarMeanAnomaly(J)

	want := 5.150833

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarTrueAnomaly(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	got := GetLunarTrueAnomaly(datetime)

	want := 6.302889

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarArgumentOfLatitude(t *testing.T) {
	J := GetCurrentJulianCenturyRelativeToJ2000(d)

	got := GetLunarArgumentOfLatitude(J)

	want := 219.889721

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarHorizontalLongitude(t *testing.T) {
	J := GetCurrentJulianCenturyRelativeToJ2000(d)

	M := GetLunarMeanAnomaly(J)

	L := GetLunarMeanLongitude(J)

	got := GetLunarHorizontalLongitude(M, L)

	want := 134.854795

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarHorizontalLatitude(t *testing.T) {
	J := GetCurrentJulianCenturyRelativeToJ2000(d)

	F := GetLunarArgumentOfLatitude(J)

	got := GetLunarHorizontalLatitude(F)

	want := 356.711352

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarLongitudeOfTheAscendingNode(t *testing.T) {
	// For testing we need to specify a date because most calculations are
	// differential w.r.t a time component. We set it to the date provided
	// on p.148 of Meeus, Jean. 1991. Astronomical algorithms.Richmond,
	// Va: Willmann - Bell.:
	d := time.Date(1987, 4, 10, 0, 0, 0, 0, time.UTC)

	J := GetCurrentJulianCenturyRelativeToJ2000(d)

	got := GetLunarLongitudeOfTheAscendingNode(J)

	want := 11.253083

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarAnnualEquationCorrection(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	M := GetSolarMeanAnomalyLawrence(datetime)

	got := GetLunarAnnualEquationCorrection(M)

	want := -0.004845

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarEvectionCorrection(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	M := GetLunarMeanAnomalyLawrence(datetime)

	λ := GetLunarMeanEclipticLongitude(datetime)

	Msol := GetSolarMeanAnomalyLawrence(datetime)

	Csol := GetSolarEquationOfCenterLawrence(Msol)

	λsol := GetSolarEclipticLongitudeLawrence(Msol, Csol)

	got := GetLunarEvectionCorrection(M, λ, λsol)

	want := -0.237282

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarMeanAnomalyCorrection(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 2, 3, 0, 0, 0, time.UTC)

	M := GetLunarMeanAnomalyLawrence(datetime)

	λ := GetLunarMeanEclipticLongitude(datetime)

	Msol := GetSolarMeanAnomalyLawrence(datetime)

	Csol := GetSolarEquationOfCenterLawrence(Msol)

	λsol := GetSolarEclipticLongitudeLawrence(Msol, Csol)

	Ae := GetLunarAnnualEquationCorrection(M)

	Eν := GetLunarEvectionCorrection(M, λ, λsol)

	got := GetLunarMeanAnomalyCorrection(M, Msol, Ae, Eν)

	want := 85.497682

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarEquatorialPositionRightAscension(t *testing.T) {
	eq := GetLunarEquatorialPosition(datetime)

	got := eq.RightAscension

	want := 76.239624

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarEquatorialPositionDeclination(t *testing.T) {
	eq := GetLunarEquatorialPosition(datetime)

	got := eq.Declination

	want := 23.598793

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarEclipticPositionLongitude(t *testing.T) {
	ec := GetLunarEclipticPosition(d)

	got := ec.Longitude

	want := 133.162655

	if math.Abs(got-want) > 0.15 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarEclipticPositionLongitudeAlt(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)

	ec := GetLunarEclipticPosition(datetime)

	got := ec.Longitude

	want := 50.604878

	if math.Abs(got-want) > 0.15 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarEclipticPositionLatitude(t *testing.T) {
	ec := GetLunarEclipticPosition(d)

	got := ec.Latitude

	want := -3.229126

	if math.Abs(got-want) > 0.1 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarEclipticPositionLatitudeAlt(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)

	ec := GetLunarEclipticPosition(datetime)

	got := ec.Latitude

	want := -2.981288

	if math.Abs(got-want) > 0.1 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarEclipticPositionDistance(t *testing.T) {
	ec := GetLunarEclipticPosition(d)

	got := ec.Δ

	want := 368409.684786

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarHorizontalParallax(t *testing.T) {
	ec := GetLunarEclipticPosition(d)

	got := GetLunarHorizontalParallax(ec.Δ)

	want := 0.991990

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarHourAngle(t *testing.T) {
	ec := GetLunarEclipticPosition(datetime)

	eq := GetLunarEquatorialPosition(datetime)

	π := GetLunarHorizontalParallax(ec.Δ)

	got := GetLunarHourAngle(eq.Declination, latitude, 0, π)

	want := 98.942949

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetEclipticLongitudeInXHours(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)

	M := GetLunarMeanAnomalyLawrence(datetime)

	λ := GetLunarMeanEclipticLongitude(datetime)

	Msol := GetSolarMeanAnomalyLawrence(datetime)

	Csol := GetSolarEquationOfCenterLawrence(Msol)

	λsol := GetSolarEclipticLongitudeLawrence(Msol, Csol)

	Ae := GetLunarAnnualEquationCorrection(M)

	Eν := GetLunarEvectionCorrection(M, λ, λsol)

	Ca := GetLunarMeanAnomalyCorrection(M, Msol, Ae, Eν)

	ec := GetLunarEclipticPosition(datetime)

	got := GetLunarEclipticLongitudeInXHours(ec.Longitude, Ca, 12)

	want := 57.438144

	if math.Abs(got-want) > 0.15 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetEclipticLatitudeInXHours(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)

	ec := GetLunarEclipticPosition(datetime)

	Ωprime1 := GetLunarCorrectedEclipticLongitudeOfTheAscendingNode(datetime)

	λt1 := GetLunarTrueEclipticLongitude(datetime)

	got := GetLunarEclipticLatitudeInXHours(ec.Latitude, Ωprime1, λt1, 12)

	want := -3.470089

	if math.Abs(got-want) > 0.1 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarTransitJulianDate(t *testing.T) {
	eq := GetLunarEquatorialPosition(d)

	ϑ := GetApparentGreenwichSiderealTimeInDegrees(d)

	got := GetLunarTransitJulianDate(datetime, eq.RightAscension, longitude, ϑ)

	want := 2459348.890048

	if math.Abs(got-want) > 0.00001 {
		t.Errorf("got %f, wanted %f", got, want)
	}
}

func TestGetLunarHorizontalCoordinatesForDayCorrectLength(t *testing.T) {
	location, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	horizontalCoordinates := GetLunarHorizontalCoordinatesForDay(datetime, longitude, latitude, location)

	if len(horizontalCoordinates) != 1440 {
		t.Errorf("there is not enough horizontal coordinates for the day, expected 1440")
	}
}

func TestGetLunarHorizontalCoordinatesForDayCorrectStartTime(t *testing.T) {
	location, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	horizontalCoordinates := GetLunarHorizontalCoordinatesForDay(datetime, longitude, latitude, location)

	d := time.Date(datetime.Year(), datetime.Month(), datetime.Day(), 0, 0, 0, 0, location)

	if horizontalCoordinates[0].Datetime.String() != d.String() {
		t.Errorf("the start date for the day after timezone adjustments is wrong")
	}
}

func TestGetLunarHorizontalCoordinatesForDayCorrectEndTime(t *testing.T) {
	location, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	horizontalCoordinates := GetLunarHorizontalCoordinatesForDay(datetime, longitude, latitude, location)

	d := time.Date(datetime.Year(), datetime.Month(), datetime.Day(), 23, 59, 0, 0, location)

	if horizontalCoordinates[len(horizontalCoordinates)-1].Datetime.String() != d.String() {
		t.Errorf("the end date for the day after timezone adjustments is wrong")
	}
}

func TestGetLunarHorizontalCoordinatesForDay20210506(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2021, 5, 6, 0, 0, 0, 0, time.UTC)

	location, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	horizontalCoordinates := GetLunarHorizontalCoordinatesForDay(datetime, longitude, latitude, location)

	if horizontalCoordinates[181].Datetime.String() == "2021-05-06 03:01:00 -1000 HST" && !horizontalCoordinates[181].IsRise {
		t.Errorf("We're expecting the Moon to rise at 3:01am on 6th May 2021")
	}

	if horizontalCoordinates[897].Datetime.String() == "2021-05-06 14:57:00 -1000 HST" && !horizontalCoordinates[897].IsSet {
		t.Errorf("We're expecting the Moon to set at 14:57pm on 6th May 2021")
	}
}

func TestGetLunarHorizontalCoordinatesForDay20210521(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2021, 5, 21, 0, 0, 0, 0, time.UTC)

	location, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	horizontalCoordinates := GetLunarHorizontalCoordinatesForDay(datetime, longitude, latitude, location)

	if horizontalCoordinates[859].Datetime.String() == "2021-05-21 14:19:00 -1000 HST" && !horizontalCoordinates[859].IsRise {
		t.Errorf("We're expecting the Moon to rise at 14:19pm on 21st May 2021")
	}

	if horizontalCoordinates[134].Datetime.String() == "2021-05-21 02:14:00 -1000 HST" && !horizontalCoordinates[134].IsSet {
		t.Errorf("We're expecting the Moon to set at 2:14am on 21st May 2021")
	}
}

func TestGetLunarPhase(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)

	got := GetLunarPhase(datetime, 78, EclipticCoordinate{Longitude: 50.279952, Latitude: -2.981288, Δ: 0})

	age := 129.966690

	angle := 49.924934

	days := 24.24562692920683

	fraction := 0.82

	illumination := 82.0

	if math.Abs(got.Age-age) > 0.1 {
		t.Errorf("got %f, wanted %f", got.Age, age)
	}

	if got.Age < 0 || got.Age > 360 {
		t.Errorf("got %f, wanted a Lunar age (in degrees) to be between 0° and 360°", got.Age)
	}

	if math.Abs(got.Angle-angle) > 0.1 {
		t.Errorf("got %f, wanted %f", got.Angle, angle)
	}

	if got.Angle < 0 || got.Angle > 360 {
		t.Errorf("got %f, wanted a Lunar phase angle to be between 0° and 360°", got.Angle)
	}

	if math.Abs(got.Days-days) > 0.1 {
		t.Errorf("got %f, wanted %f", got.Days, days)
	}

	if got.Days < 0 || got.Days > LUNAR_MONTH_IN_DAYS {
		t.Errorf("got %f, wanted a Lunar age (in days) to be between 0 and 29.5 days", got.Angle)
	}

	if math.Abs(got.Fraction-fraction) > 0.1 {
		t.Errorf("got %f, wanted %f", got.Fraction, fraction)
	}

	if got.Fraction < 0 || got.Fraction > 1 {
		t.Errorf("got %f, but wanted an phase fraction value to be between 0 and 1", got.Fraction)
	}

	if math.Abs(got.Illumination-illumination) > 0.2 {
		t.Errorf("got %f, wanted %f", got.Illumination, illumination)
	}

	if got.Illumination < 0 || got.Illumination > 100 {
		t.Errorf("got %f, but wanted an illumination value to be between 0 and 100%%", got.Illumination)
	}
}

func TestGetMoonriseMoonsetTimes20210506(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2021, 5, 6, 0, 0, 0, 0, time.UTC)

	location, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	moon := GetMoonriseMoonsetTimes(datetime, longitude, latitude, location)

	if moon.Rise.String() != "2021-05-06 03:01:00 -1000 HST" {
		t.Errorf("We're expecting the Moon to rise at 3:01am on 6th May 2021")
	}

	if moon.Set.String() != "2021-05-06 14:57:00 -1000 HST" {
		t.Errorf("We're expecting the Moon to set at 14:57pm on 6th May 2021")
	}
}

func TestGetMoonriseMoonsetTimes20210521(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2021, 5, 21, 0, 0, 0, 0, time.UTC)

	location, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	moon := GetMoonriseMoonsetTimes(datetime, longitude, latitude, location)

	if moon.Rise.String() != "2021-05-21 14:19:00 -1000 HST" {
		t.Errorf("We're expecting the Moon to rise at 14:19pm on 21st May 2021")
	}

	if moon.Set.String() != "2021-05-21 02:14:00 -1000 HST" {
		t.Errorf("We're expecting the Moon to set at 2:14am on 21st May 2021")
	}
}

func TestGetMoonriseMoonsetTimesInUTC20210506(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2021, 5, 6, 0, 0, 0, 0, time.UTC)

	location, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	moon := GetMoonriseMoonsetTimesInUTC(datetime, longitude, latitude, location)

	if moon.Rise.String() != "2021-05-06 13:01:00 +0000 UTC" {
		t.Errorf("We're expecting the Moon to rise at 13:01pm on 6th May 2021")
	}

	if moon.Set.String() != "2021-05-07 00:57:00 +0000 UTC" {
		t.Errorf("We're expecting the Moon to set at 0:57am on 7th May 2021")
	}
}

func TestGetMoonriseMoonsetTimesInUTC20210521(t *testing.T) {
	// Date of observation:
	datetime := time.Date(2021, 5, 21, 0, 0, 0, 0, time.UTC)

	location, err := time.LoadLocation("Pacific/Honolulu")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	moon := GetMoonriseMoonsetTimesInUTC(datetime, longitude, latitude, location)

	if moon.Rise.String() != "2021-05-22 00:19:00 +0000 UTC" {
		t.Errorf("We're expecting the Moon to rise at 0:19am on 22nd May 2021")
	}

	if moon.Set.String() != "2021-05-21 12:14:00 +0000 UTC" {
		t.Errorf("We're expecting the Moon to set at 12:14pm on 21st May 2021")
	}
}
