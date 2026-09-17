package dusk

import (
	"time"
)

// solarParams holds intermediate solar position values computed from a date
// and longitude. Used by SunriseSunset and twilight to avoid repeating the
// 6-step parameter sequence.
type solarParams struct {
	J        float64 // mean solar time: days since J2000 at the observer's longitude
	T        float64 // Julian centuries since J2000
	delta    float64 // solar declination at mean solar time (degrees)
	jTransit float64 // Julian date of solar transit (noon)
}

// computeSolarParams returns the solar declination and transit JD for a given
// date and observer longitude, along with the two time bases they were computed
// from so that declinationAt can re-solve on the same footing.
func computeSolarParams(date time.Time, lon float64) solarParams {
	J := meanSolarTime(date, lon)
	M := solarMeanAnomaly(J)
	C := solarEquationOfCenter(M)
	lambda := solarEclipticLongitude(M, C)
	T := julianCentury(date)
	delta := solarDeclination(lambda, T)
	jTransit := solarTransitJD(J, M, lambda)

	return solarParams{J: J, T: T, delta: delta, jTransit: jTransit}
}

// declinationAt returns the Sun's declination offsetDays away from the day's
// mean solar time. Both time bases move together, so the result sits on the
// same footing as sp.delta rather than a second convention.
func (sp solarParams) declinationAt(offsetDays float64) float64 {
	M := solarMeanAnomaly(sp.J + offsetDays)
	C := solarEquationOfCenter(M)
	lambda := solarEclipticLongitude(M, C)

	return solarDeclination(lambda, sp.T+offsetDays/36525.0)
}

// solarDay holds one day's solar geometry, already solved into instants.
// rise and set are meaningless unless horizon is Crosses; transit always holds.
type solarDay struct {
	rise    float64
	transit float64
	set     float64
	horizon Horizon
}

// solarCrossing resolves the observer's calendar day and solves the day's solar
// geometry at the given depression, returning transit and the two boundaries as
// Julian dates. SunriseSunset passes 0; Twilight passes its band.
//
// This is the whole of what the two share. Each turns the same three instants
// into its own result type, because they answer different questions about them.
func solarCrossing(date Date, obs Observer, depression float64) (solarDay, error) {
	err := validObserver(obs)
	if err != nil {
		return solarDay{}, err
	}

	// Build the day at UTC midnight, not in obs.loc: meanSolarTime applies the
	// observer's longitude itself, after julianDay has rounded, so handing it a
	// zone-adjusted instant would apply longitude twice. MoonriseMoonset does the
	// opposite for the opposite reason -- see lunar.go.
	day := date.at(time.UTC)

	err = validJulianDateRange(day)
	if err != nil {
		return solarDay{}, err
	}

	sp := computeSolarParams(day, obs.lon)

	omega, horizon := solarHourAngle(sp.delta, depression, obs.lat)
	if horizon != Crosses {
		return solarDay{transit: sp.jTransit, horizon: horizon}, nil
	}

	// One correction pass per boundary. The first estimate places both
	// symmetrically about transit, which assumes the Sun's declination is the
	// same in the morning as in the evening. Near an equinox it moves about
	// 0.4 degrees a day, so the afternoon half-day is genuinely shorter than the
	// morning one, and the mirrored construction cannot represent that at all.
	//
	// Re-solving each boundary against the declination at its own estimated
	// instant recovers most of the difference. It does not recover all of it:
	// the hour angle is still measured about a transit computed once for the
	// day, so the Sun's motion in right ascension between transit and the
	// boundary is unmodelled. Measured, that leaves about half the equinox skew
	// at high latitude, which is issue #114.
	return solarDay{
		rise:    sp.jTransit - refineOmega(sp, -omega/360.0, depression, obs.lat, omega)/360.0,
		transit: sp.jTransit,
		set:     sp.jTransit + refineOmega(sp, +omega/360.0, depression, obs.lat, omega)/360.0,
		horizon: Crosses,
	}, nil
}

// refineOmega re-solves the hour angle against the declination at the estimated
// boundary instant.
//
// A refined declination that puts the boundary out of reach is the polar limit
// arriving mid-correction, on a day whose first pass said the Sun does cross.
// Keep the first estimate there rather than reporting a state the day as a whole
// does not have -- the alternative is a discontinuity at the one latitude where
// the answer is already hardest to defend.
func refineOmega(sp solarParams, offsetDays, depression, lat, fallback float64) float64 {
	omega, horizon := solarHourAngle(sp.declinationAt(offsetDays), depression, lat)
	if horizon != Crosses {
		return fallback
	}

	return omega
}

// SunriseSunset computes sunrise, solar noon, and sunset for the given calendar
// day and observer position. The observer must be constructed via [NewObserver].
// Output times are in the observer's timezone.
//
// The algorithm follows the NOAA solar calculator method (derived from Meeus,
// Astronomical Algorithms).
func SunriseSunset(date Date, obs Observer) (SunEvent, error) {
	day, err := solarCrossing(date, obs, 0)
	if err != nil {
		return SunEvent{}, err
	}

	noon := universalTimeFromJD(day.transit).In(obs.loc)

	// Transit is defined on every day at every latitude, so Noon is always set
	// -- including through the polar night, when the Sun reaches its highest
	// point below the horizon and there is no rise or set to report.
	if day.horizon != Crosses {
		return SunEvent{
			Noon:     noon,
			Duration: daylightOf(day.horizon),
			Horizon:  day.horizon,
		}, nil
	}

	rise := universalTimeFromJD(day.rise).In(obs.loc)
	set := universalTimeFromJD(day.set).In(obs.loc)

	return SunEvent{
		Rise:     rise,
		Noon:     noon,
		Set:      set,
		Duration: set.Sub(rise),
		Horizon:  Crosses,
	}, nil
}

// daylightOf gives the length of a day with no sunrise in it: a full day under
// the midnight sun, and none at all through the polar night.
func daylightOf(horizon Horizon) time.Duration {
	if horizon == StaysAbove {
		return 24 * time.Hour
	}

	return 0
}

// solarMeanAnomaly returns the Sun's mean anomaly in degrees.
// J is the number of days since J2000.0.
func solarMeanAnomaly(J float64) float64 {
	return mod360(357.5291092 + 0.98560028*J)
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
// declination, observer latitude, and depression angle (degrees below the
// geometric horizon, positive downward). For standard sunrise/sunset, pass
// depression = 0.
//
// For sunrise/sunset (depression=0), includes a -0.8333 degree correction for
// atmospheric refraction and solar semidiameter, the value Meeus and USNO use. For twilight, uses the
// depression angle directly per IAU/USNO convention.
//
// The second return says whether the Sun reaches the queried altitude at all.
// When it does not, the hour angle is meaningless and returned as zero: callers
// must branch on the [Horizon] rather than use it.
func solarHourAngle(delta, depression, lat float64) (float64, Horizon) {
	var h0 float64
	if depression == 0 {
		h0 = -0.8333
	} else {
		h0 = -depression
	}

	num := sinx(h0) - sinx(lat)*sinx(delta)
	den := cosx(lat) * cosx(delta)

	cosHA := num / den
	if cosHA < -1 {
		return 0, StaysAbove
	}

	if cosHA > 1 {
		return 0, StaysBelow
	}

	return acosx(cosHA), Crosses
}

// solarTransitJD returns the Julian date of solar transit (solar noon).
func solarTransitJD(J, M, lambda float64) float64 {
	return j2000 + J + 0.0053*sinx(M) - 0.0069*sinx(2*lambda)
}

// ---------------------------------------------------------------------------
// Twilight: one parameterised band, sharing solarCrossing with sunrise and
// sunset above.
// ---------------------------------------------------------------------------

// Twilight computes the morning and evening boundaries of a twilight band for
// the given date and observer position. depression is degrees below the
// geometric horizon, positive downward: 6 for civil, 12 for nautical and 18 for
// astronomical, the IAU/USNO values. The observer must be constructed via
// [NewObserver].
//
// Dawn and Dusk are both on date, the same day [SunriseSunset] reports. They
// are symmetric about solar transit, so either both exist or neither does:
// through a polar day or night both are zero and [TwilightEvent.Horizon] says
// which way.
//
// Passing depression = 0 gives sunrise and sunset, refraction included, which
// is what [SunriseSunset] returns.
func Twilight(date Date, obs Observer, depression float64) (TwilightEvent, error) {
	day, err := solarCrossing(date, obs, depression)
	if err != nil {
		return TwilightEvent{}, err
	}

	if day.horizon != Crosses {
		return TwilightEvent{Horizon: day.horizon}, nil
	}

	return TwilightEvent{
		Dawn:    universalTimeFromJD(day.rise).In(obs.loc),
		Dusk:    universalTimeFromJD(day.set).In(obs.loc),
		Horizon: Crosses,
	}, nil
}
