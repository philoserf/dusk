package dusk

import (
	"math"
	"time"
)

// the epoch of Unix time start i.e., 1 January 1970 00:00:00 UTC:
var J1970 = 2440587.5

// the epoch of Unix time start i.e., 1 January 2000 00:00:00 UTC:
var J2000 = 2451545.0

type JulianPeriod struct {
	/*
		The current Julian Date expressed as fractions of days
	*/
	JD float64
	/*
		The current Julian Date expressed as fractions of centuries
	*/
	T float64
}

/*
GetDatetimeZeroHour()

@param datetime - the datetime of the observer
@returns the datetime of the zero hour of the day
*/
func GetDatetimeZeroHour(datetime time.Time) time.Time {
	return time.Date(datetime.Year(), datetime.Month(), datetime.Day(), 0, 0, 0, 0, time.UTC)
}

/*
GetJulianDate()

@returns the Julian date i.e., the continuous count of days and fractions of day since the beginning of the Julian period
@see http://astro.vaporia.com/start/jd.html
*/
func GetJulianDate(datetime time.Time) float64 {
	// milliseconds elapsed since 1 January 1970 00:00:00 UTC up until now as an int64:
	time := datetime.UTC().UnixNano() / 1e6

	return float64(time)/86400000.0 + J1970
}

/*
GetUniversalTime()

@returns the universal time (UTC) for a given Julian date
*/
func GetUniversalTime(JD float64) time.Time {
	return time.Unix(0, int64((JD-J1970)*86400000.0*1e6)).UTC()
}

/*
GetLocalGreenwichSiderealTime

@param datetime - the datetime of the observer
@returns the local sidereal time relative to Greenwich, UK
*/
func GetGreenwichSiderealTime(datetime time.Time) float64 {
	JD := GetJulianDate(datetime)

	JD0 := GetJulianDate(time.Date(datetime.Year(), 1, 0, 0, 0, 0, 0, time.UTC))

	days := math.Floor(JD - JD0)

	T := (JD0 - 2415020.0) / 36525

	R := 6.6460656 + 2400.051262*T + 0.00002581*T*T

	B := 24.0 - R + float64(24*(datetime.Year()-1900))

	T0 := 0.0657098*days - B

	hr := float64(datetime.Hour())

	min := float64(datetime.Minute()) / 60.0

	sec := float64(datetime.Second()) / 3600.0

	ns := float64(datetime.Nanosecond()) / 3600000000.0

	UT := hr + min + sec + ns

	A := float64(UT) * 1.002737909

	GST := math.Mod(T0+A, 24)

	// correct for negative hour angles (24 hours is equivalent to 360°)
	if GST < 0 {
		GST += 24
	}

	return GST
}

/*
GetLocalSiderealTime()

@param datetime
@returns returns the local sidereal time, relative to some location's longitude
*/
func GetLocalSiderealTime(datetime time.Time, longitude float64) float64 {
	GST := GetGreenwichSiderealTime(datetime)

	d := (GST + longitude/15.0) / 24.0

	d = d - math.Floor(d)

	// correct for negative hour angles (24 hours is equivalent to 360°)
	if d < 0 {
		d += 1
	}

	return 24.0 * d
}

/*
GetCurrentJulianDayRelativeToJ2000()

@returns the number of Julian days between J2000 (i.e., 1 January 2000 00:00:00 UTC) and the the datetime, rounded up the the nearest integer
@see http://astro.vaporia.com/start/jd.html
*/
func GetCurrentJulianDayRelativeToJ2000(datetime time.Time) int {
	// get the Julian date:
	JD := GetJulianDate(datetime)

	// correction for the the fractional Julian Day for leap seconds and terrestrial time (TT):
	corr := 0.0008

	// calculate the current Julian day:
	n := math.Ceil(JD - 2451545.0 - corr)

	return int(n)
}

/*
GetFractionalJulianDayStandardEpoch()

@returns the total number of fractional dates elapsed since the standard epoch J2000.
@see p.136 of Lawrence, J.L. 2015. Celestial Calculations - A Gentle Introduction Yo Computational Astronomy. Cambridge, Ma: The MIT Press
*/
func GetFractionalJulianDaysSinceStandardEpoch(datetime time.Time) float64 {
	// get the Julian date:
	JD := GetJulianDate(datetime)

	// calculate the current Julian day:
	n := JD - 2451545.0

	return n
}

/*
GetCurrentJulianCenturyRelativeToJ2000()

@returns the number of Julian centuries between J2000 (i.e., 1 January 2000 00:00:00 UTC) and the the datetime, rounded up the the nearest integer
@see http://astro.vaporia.com/start/jd.html
*/
func GetCurrentJulianCenturyRelativeToJ2000(datetime time.Time) float64 {
	// get the Julian date:
	JD := GetJulianDate(datetime)

	// calculate the current Julian century as fractions of centuries:
	n := (JD - 2451545.0) / 36525

	return n
}

/*
GetCurrentJulianPeriod()

@returns both the Julian date i.e., the continuous count of days and fractions of day since the beginning of the Julian period and the number of
Julian centuries between J2000 (i.e., 1 January 2000 00:00:00 UTC) and the the datetime, rounded up the the nearest integer
@see http://astro.vaporia.com/start/jd.html
*/
func GetCurrentJulianPeriod(datetime time.Time) JulianPeriod {
	// get the Julian date:
	JD := GetJulianDate(datetime)

	// calculate the current Julian date as fractions of centuries:
	T := (JD - 2451545.0) / 36525

	return JulianPeriod{
		JD: JD,
		T:  T,
	}
}

/*
GetMeanGreenwichSiderealTimeInDegrees()

@returns the mean sidereal time at Greenwich for the desired datetime (in degrees)
@see eq.12.4 p.88 of Meeus, Jean. 1991. Astronomical algorithms. Richmond, Va: Willmann - Bell.
*/
func GetMeanGreenwichSiderealTimeInDegrees(datetime time.Time) float64 {
	d := time.Date(datetime.Year(), datetime.Month(), datetime.Day(), 0, 0, 0, 0, time.UTC)

	julianPeriod := GetCurrentJulianPeriod(d)

	// get the Julian date:
	JD := julianPeriod.JD

	// the number of Julian centuries between J2000 (i.e., 1 January 2000 00:00:00 UTC) and the the datetime:
	T := julianPeriod.T

	// applies modulo correction to the angle, and ensures always positive:
	θ := math.Mod(280.46061837+(360.98564736629*(JD-2451545.0))+(0.000387933*T*T)-(T*T*T/38710000), 360)

	// correct for negative angles
	if θ < 0 {
		θ += 360
	}

	return θ
}

/*
GetApparentGreenwichSiderealTimeInDegrees()

@returns the apparent sidereal time at Greenwich for the desired datetime (in degrees)
@see eq.12.4 p.88 of Meeus, Jean. 1991. Astronomical algorithms. Richmond, Va: Willmann - Bell.
*/
func GetApparentGreenwichSiderealTimeInDegrees(datetime time.Time) float64 {
	θ := GetMeanGreenwichSiderealTimeInDegrees(datetime)

	J := GetCurrentJulianCenturyRelativeToJ2000(datetime)

	L := GetSolarMeanLongitude(J)

	l := GetLunarMeanLongitude(J)

	Ω := GetLunarLongitudeOfTheAscendingNode(J)

	ε := GetMeanObliquityOfTheEcliptic(J) + GetNutationInObliquityOfTheEcliptic(L, l, Ω)

	Δψ := GetNutationInLongitudeOfTheEcliptic(L, l, Ω)

	// applies a correction for the true vernal equinox:
	corr := Δψ * cosx(ε)

	// applies modulo correction to the angle, and ensures always positive:
	ϑ := math.Mod(θ+corr, 360)

	// correct for negative angles
	if ϑ < 0 {
		ϑ += 360
	}

	return ϑ
}

/*
GetMeanSolarTime()

@param datetime - the datetime of the observer (in UTC)
@param longitude - is the longitude (west is negative, east is positive) in degrees of some observer on Earth
@returns returns the mean solar time, relative to some observer's longitude on Earth
*/
func GetMeanSolarTime(datetime time.Time, longitude float64) float64 {
	// the number of Julian days between J2000 (i.e., 1 January 2000 00:00:00 UTC) and the the datetime:
	n := GetCurrentJulianDayRelativeToJ2000(datetime)

	return float64(n) - (longitude / 360)
}

/*
ConvertLocalSiderealTimeToGreenwichSiderealTime()

@param LST - the local sidereal time in hours (in decimal format)
@param longitude - is the longitude (west is negative, east is positive) in degrees of some observer on Earth
@returns returns the GST in hours (in decimal format)
*/
func ConvertLocalSiderealTimeToGreenwichSiderealTime(LST, longitude float64) float64 {
	GST := LST - longitude/15

	// correct for negative hour angles
	if GST < 0 {
		GST += 24
	}

	return GST
}

/*
ConvertGreenwichSiderealTimeToUniversalTime()

@param datetime - the datetime of the observer (in UTC)
@param GST - the GST in hours (in decimal format)
@returns the UT in hours (in decimal format)
*/
func ConvertGreenwichSiderealTimeToUniversalTime(datetime time.Time, GST float64) float64 {
	d := GetDatetimeZeroHour(datetime)

	JD := GetJulianDate(d)

	JD0 := GetJulianDate(time.Date(datetime.Year(), 1, 0, 0, 0, 0, 0, time.UTC))

	days := JD - JD0

	T := (JD0 - 2415020) / 36525

	R := 6.6460656 + 2400.051262*T + 0.00002581*T*T

	B := 24 - R + float64(24*(datetime.Year()-1900))

	T0 := (0.0657098 * days) - B

	// correct for negative hour angles
	if T0 < 0 {
		T0 += 24
	}

	// correct for hour angles greater than 24h
	if T0 > 24 {
		T0 -= 24
	}

	A := (GST - T0)

	// correct for negative hour angles
	if A < 0 {
		A += 24
	}

	return 0.997270 * A
}
