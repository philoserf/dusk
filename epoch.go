package dusk

import (
	"errors"
	"math"
	"time"
)

const (
	j1970 = 2440587.5
	j2000 = 2451545.0
)

// JulianDate returns the Julian date for a given time, i.e., the continuous
// count of days and fractions of day since the beginning of the Julian period.
//
// Uses UnixNano internally, which limits the valid range to the int64
// nanosecond bounds (approximately 1677-09-21 to 2262-04-11). Dates outside
// this range silently produce incorrect results because UnixNano returns 0.
// Use [ValidJulianDateRange] to check before calling.
func JulianDate(t time.Time) float64 {
	ms := t.UTC().UnixNano() / 1e6
	return float64(ms)/86400000.0 + j1970
}

// ErrDateOutOfRange is returned when a date falls outside the valid range
// for [JulianDate] (the int64 nanosecond bounds, approximately
// 1677-09-21 to 2262-04-11).
var ErrDateOutOfRange = errors.New("dusk: date outside valid range (1677-09-21 to 2262-04-11) for JulianDate")

// julianDateMin and julianDateMax are the bounds of the int64 UnixNano range.
var (
	julianDateMin = time.Unix(0, math.MinInt64).UTC()
	julianDateMax = time.Unix(0, math.MaxInt64).UTC()
)

// ValidJulianDateRange reports whether t falls within the valid range for
// [JulianDate]. Returns nil if valid, [ErrDateOutOfRange] otherwise.
func ValidJulianDateRange(t time.Time) error {
	if t.Before(julianDateMin) || t.After(julianDateMax) {
		return ErrDateOutOfRange
	}
	return nil
}

// greenwichMeanSiderealTime returns the mean sidereal time at Greenwich in
// degrees for the given instant.
//
// See Meeus, Astronomical Algorithms, eq. 12.4 p. 88.
func greenwichMeanSiderealTime(t time.Time) float64 {
	// Midnight UTC for the date.
	d := datetimeZeroHour(t)
	T := julianCentury(d)
	JD := JulianDate(t)

	theta := 280.46061837 +
		360.98564736629*(JD-j2000) +
		0.000387933*T*T -
		T*T*T/38710000.0

	return mod360(theta)
}

// LocalSiderealTime returns the local sidereal time in hours for a given
// instant and observer longitude (east positive, west negative, in degrees).
func LocalSiderealTime(t time.Time, longitude float64) float64 {
	gst := greenwichMeanSiderealTime(t) // degrees
	lst := gst + longitude              // degrees
	return mod24(lst / 15.0)
}

// julianCentury returns the number of Julian centuries elapsed since J2000.0.
func julianCentury(t time.Time) float64 {
	return (JulianDate(t) - j2000) / 36525.0
}

// julianDay returns the number of days since J2000.0, rounded to the nearest
// integer (used for mean solar time).
func julianDay(t time.Time) int {
	JD := JulianDate(t)
	return int(math.Round(JD - j2000))
}

// meanSolarTime returns the mean solar time for a given instant and longitude.
func meanSolarTime(t time.Time, longitude float64) float64 {
	return float64(julianDay(t)) - longitude/360.0
}

// universalTimeFromJD converts a Julian date back to a time.Time in UTC.
func universalTimeFromJD(jd float64) time.Time {
	return time.Unix(0, int64((jd-j1970)*86400000.0*1e6)).UTC()
}

// datetimeZeroHour returns midnight UTC for the given date.
func datetimeZeroHour(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

// ---------------------------------------------------------------------------
// Nutation and obliquity helpers (Meeus ch. 22, 25)
// ---------------------------------------------------------------------------

// meanObliquity returns the mean obliquity of the ecliptic in degrees.
//
// T is Julian centuries since J2000.0.
// See Meeus, Astronomical Algorithms, p. 147.
func meanObliquity(T float64) float64 {
	return 23.4392917 - 0.0130041667*T - 0.00000016667*T*T + 0.0000005027778*T*T*T
}

// nutationInObliquity returns Δε (nutation in obliquity) in degrees.
//
// See Meeus p. 144.
func nutationInObliquity(L, l, omega float64) float64 {
	return (9.20*cosx(omega) + 0.57*cosx(2*L) + 0.1*cosx(2*l) - 0.09*cosx(2*omega)) / 3600.0
}

// solarMeanLongitude returns the Sun's mean longitude in degrees for Julian
// centuries T since J2000.0.
func solarMeanLongitude(T float64) float64 {
	return mod360(280.4665 + 36000.7698*T)
}

// lunarMeanLongitude returns the Moon's mean longitude in degrees for Julian
// centuries T since J2000.0.
//
// See Meeus eq. 47.1 p. 338.
func lunarMeanLongitude(T float64) float64 {
	return mod360(218.3164477 + 481267.88123421*T - 0.0015786*T*T + T*T*T/538841.0 - T*T*T*T/65194000.0)
}

// lunarAscendingNode returns the longitude of the Moon's ascending node in
// degrees for Julian centuries T since J2000.0.
//
// See Meeus p. 144.
func lunarAscendingNode(T float64) float64 {
	return mod360(125.04452 - 1934.136261*T + 0.0020708*T*T + T*T*T/450000.0)
}
