package dusk

import (
	"math"
	"time"
)

// SunEvent holds the times of sunrise, solar noon, sunset, and the duration
// of daylight for a single day.
type SunEvent struct {
	Rise     time.Time
	Noon     time.Time
	Set      time.Time
	Duration time.Duration
}

// SunriseSunset computes sunrise, solar noon, and sunset for the given date
// and observer position.
//
// Longitude is east-positive, west-negative. An error is returned if obs.Loc is nil.
//
// The algorithm follows the NOAA solar calculator method (derived from Meeus,
// Astronomical Algorithms).
func SunriseSunset(date time.Time, obs Observer) (SunEvent, error) {
	if err := validateObserver(obs); err != nil {
		return SunEvent{}, err
	}

	J := meanSolarTime(date, obs.Lon)
	M := solarMeanAnomaly(J)
	C := solarEquationOfCenter(M)
	lambda := solarEclipticLongitude(M, C)
	T := julianCentury(date)
	delta := solarDeclination(lambda, T)

	Jtransit := solarTransitJD(J, M, lambda)

	omega, err := solarHourAngle(delta, 0, obs.Lat, obs.Elev)
	if err != nil {
		return SunEvent{}, err
	}

	Jrise := Jtransit - omega/360.0
	Jset := Jtransit + omega/360.0

	rise := universalTimeFromJD(Jrise).In(obs.Loc)
	noon := universalTimeFromJD(Jtransit).In(obs.Loc)
	set := universalTimeFromJD(Jset).In(obs.Loc)

	return SunEvent{
		Rise:     rise,
		Noon:     noon,
		Set:      set,
		Duration: set.Sub(rise),
	}, nil
}

// SolarPosition returns the equatorial coordinates (RA, Dec) of the Sun for
// a given instant, using the Meeus mean anomaly + equation of center method.
//
// Unlike SunriseSunset (which rounds J to an integer for the NOAA method),
// this function uses continuous Julian days for precise position at any instant.
func SolarPosition(t time.Time) Equatorial {
	JD := JulianDate(t)
	J := JD - j2000

	M := solarMeanAnomaly(J)
	C := solarEquationOfCenter(M)
	lambda := solarEclipticLongitude(M, C)

	// The Sun lies on the ecliptic (latitude = 0).
	return EclipticToEquatorial(t, lambda, 0)
}

// solarMeanAnomaly returns the Sun's mean anomaly in degrees.
// J is the number of days since J2000.0.
func solarMeanAnomaly(J float64) float64 {
	return mod360(357.5291092 + 0.98560028*J)
}

// solarMeanAnomalyFromCentury returns the Sun's mean anomaly in degrees.
// T is Julian centuries since J2000.0.
func solarMeanAnomalyFromCentury(T float64) float64 {
	return mod360(357.5291092 + 35999.0503*T)
}

// solarEquationOfCenter returns the equation of center in degrees for a given
// solar mean anomaly M (in degrees).
func solarEquationOfCenter(M float64) float64 {
	return 1.9148*sinx(M) + 0.0200*sinx(2*M) + 0.0003*sinx(3*M)
}

// solarEclipticLongitude returns the Sun's ecliptic longitude in degrees.
func solarEclipticLongitude(M, C float64) float64 {
	return mod360(M + C + 180 + 102.9372)
}

// solarDeclination returns the Sun's declination in degrees from its ecliptic
// longitude and Julian century T since J2000.0.
func solarDeclination(lambda, T float64) float64 {
	eps := meanObliquity(T)
	return asinx(sinx(lambda) * sinx(eps))
}

// solarHourAngle returns the hour angle in degrees for the Sun at the given
// declination, observer latitude, elevation (meters), and depression angle
// (degrees below the geometric horizon, positive downward). For standard
// sunrise/sunset, pass depression = 0. For civil twilight, pass 6, etc.
//
// For sunrise/sunset (depression=0), includes a −0.83° correction for
// atmospheric refraction and solar semidiameter. For twilight, uses the
// depression angle directly per IAU/USNO convention.
//
// Returns ErrCircumpolar when the Sun never sets (midnight sun) or
// ErrNeverRises when the Sun never rises (polar night) at this latitude
// and depression angle.
func solarHourAngle(delta, depression, lat, elev float64) (float64, error) {
	elevCorr := 2.076 * math.Sqrt(math.Max(0, elev)) / 60
	var h0 float64
	if depression == 0 {
		h0 = -(0.83 - elevCorr)
	} else {
		h0 = -(depression - elevCorr)
	}
	num := sinx(h0) - sinx(lat)*sinx(delta)
	den := cosx(lat) * cosx(delta)
	cosHA := num / den
	if cosHA < -1 {
		return 0, ErrCircumpolar
	}
	if cosHA > 1 {
		return 0, ErrNeverRises
	}
	return acosx(cosHA), nil
}

// solarTransitJD returns the Julian date of solar transit (solar noon).
func solarTransitJD(J, M, lambda float64) float64 {
	return j2000 + J + 0.0053*sinx(M) - 0.0069*sinx(2*lambda)
}
