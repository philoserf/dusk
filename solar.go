package dusk

import (
	"math"
	"time"
)

type Sun struct {
	Rise time.Time
	Noon time.Time
	Set  time.Time
}

/*
GetSolarMeanAnomaly()

@param J - the Ephemeris time or the number of centuries since J2000 epoch
@returns the non-uniform or anomalous apparent motion of the Sun along the plane of the ecliptic
@see EQ.47.3 p.338 of Meeus, Jean. 1991. Astronomical algorithms. Richmond, Va: Willmann-Bell
*/
func GetSolarMeanAnomaly(J float64) float64 {
	// applies modulo correction to the angle, and ensures always positive:
	M := math.Mod(357.5291092+(0.98560028*J), 360)

	if M < 0 {
		M += 360
	}

	return M
}

/*
GetSolarEquationOfCenter()

@param M - the mean solar anomaly for the Ephemeris time or the number of centuries since J2000 epoch
@returns the equation of center for the Sun
@see p.164 of Meeus, Jean. 1991. Astronomical algorithms. Richmond, Va: Willmann-Bell.
*/
func GetSolarEquationOfCenter(M float64) float64 {
	// applies modulo correction to the angle, and ensures always positive:
	return 1.9148*sinx(M) + 0.0200*sinx(2*M) + 0.0003*sinx(3*M)
}

/*
GetSolarEclipticLongitude()

@param M - the mean solar anomaly for the Ephemeris time or the number of centuries since J2000 epoch
@param C - the equation of center for the Sun
@returns the apparent Solar ecliptic longitude (in degrees)
*/
func GetSolarEclipticLongitude(M, C float64) float64 {
	// applies modulo correction to the angle, and ensures always positive:
	λ := math.Mod(M+C+180+102.9372, 360)

	if λ < 0 {
		λ += 360
	}

	return λ
}

/*
GetSolarTransitJulianDate()

@param J - the Ephemeris time or the number of centuries since J2000 epoch
@param M - the mean solar anomaly for the Ephemeris time or the number of centuries since J2000 epoch
@param λ - the ecliptic longitude of the Sun (in degrees)
@returns the Julian date for the local true solar transit (or solar noon).
*/
func GetSolarTransitJulianDate(J, M, λ float64) float64 {
	return 2451545.0 + J + 0.0053*sinx(M) - 0.0069*sinx(2*λ)
}

/*
GetSolarDeclination()

The declination of the Sun, δ☉, is the angle between the rays of the Sun and the plane of the Earth's equator.

@param λ - the ecliptic longitude of the Sun (in degrees)
@returns the declination of the Sun (in degrees)
@see https://gml.noaa.gov/grad/solcalc/glossary.html#solardeclination
*/
func GetSolarDeclination(λ float64) float64 {
	return asinx(sinx(λ) * sinx(23.44))
}

/*
GetSolarMeanLongitude()

@param J - the Ephemeris time or the number of centuries since J2000 epoch
@returns the mean longitude of the Sun
*/
func GetSolarMeanLongitude(J float64) float64 {
	// applies modulo correction to the angle, and ensures always positive:
	L := math.Mod(280.4665+36000.7698*J, 360)

	// correct for negative angles
	if L < 0 {
		L += 360
	}

	return L
}

/*
GetSolarHourAngle()

Observing the Sun from Earth, the solar hour angle is an expression of time, expressed in angular measurement,
usually degrees, from solar noon. At solar noon the hour angle is zero degrees, with the time before solar noon
expressed as negative degrees, and the local time after solar noon expressed as positive degrees.

@param δ - the ecliptic longitude of the Sun (in degrees)
@param latitude - is the latitude (south is negative, north is positive) in degrees of some observer on Earth
@param elevation - is the elevation (above sea level) in meters of some observer on Earth
@returns the solar hour angle for a given solar declination, of some observer on Earth
@see https://gml.noaa.gov/grad/solcalc/glossary.html#solardeclination
*/
func GetSolarHourAngle(δ, degreesBelowHorizon, latitude, elevation float64) float64 {
	// observations on a sea horizon needing an elevation-of-observer correction
	// (corrects for both apparent dip and terrestrial refraction):
	corr := -degreesBelowHorizon + -2.076*math.Sqrt(elevation)*1/60

	return acosx((sinx(-0.83-corr) - (sinx(latitude) * sinx(δ))) / (cosx(latitude) * cosx(δ)))
}

/*
GetSunriseSunsetTimes()

@param datetime - the datetime of the observer (in localtime)
@param longitude - is the longitude (west is negative, east is positive) in degrees of some observer on Earth
@param latitude - is the latitude (south is negative, north is positive) in degrees of some observer on Earth
@param elevation - is the elevation (above sea level) in meters of some observer on Earth
@param location - the timezone location for the observer (e.g., from time.LoadLocation)
@returns the rise, noon and set for the Sun, in localtime
*/
func GetSunriseSunsetTimes(datetime time.Time, degreesBelowHorizon, longitude, latitude, elevation float64, location *time.Location) Sun {
	if location == nil {
		location = time.UTC
	}

	sun := GetSunriseSunsetTimesInUTC(datetime, degreesBelowHorizon, longitude, latitude, elevation)

	return Sun{
		Rise: sun.Rise.In(location),
		Noon: sun.Noon.In(location),
		Set:  sun.Set.In(location),
	}
}

/*
GetSunriseSunsetTimesInUTC()

@param datetime - the datetime of the observer (in UTC)
@param longitude - is the longitude (west is negative, east is positive) in degrees of some observer on Earth
@param latitude - is the latitude (south is negative, north is positive) in degrees of some observer on Earth
@param elevation - is the elevation (above sea level) in meters of some observer on Earth
@returns the rise, noon and set for the Sun, in UTC (*not local time)
*/
func GetSunriseSunsetTimesInUTC(datetime time.Time, degreesBelowHorizon, longitude, latitude, elevation float64) Sun {
	J := GetMeanSolarTime(datetime, longitude)

	M := GetSolarMeanAnomaly(J)

	C := GetSolarEquationOfCenter(M)

	λ := GetSolarEclipticLongitude(M, C)

	δ := GetSolarDeclination(λ)

	ω := GetSolarHourAngle(δ, degreesBelowHorizon, latitude, elevation)

	h := ω / 360

	J_transit := GetSolarTransitJulianDate(J, M, λ)

	J_rise := J_transit - h

	J_set := J_transit + h

	sun := Sun{
		Rise: GetUniversalTime(J_rise),
		Noon: GetUniversalTime(J_transit),
		Set:  GetUniversalTime(J_set),
	}

	return sun
}

/*
GetSolarEclipticPosition()

@param datetime - the datetime of the observer (in UTC)
@returns the geocentric ecliptic coodinate (λ - geocentric longitude, β - geocentric latidude) of the Sun.
*/
func GetSolarEclipticPosition(datetime time.Time) EclipticCoordinate {
	M := GetSolarMeanAnomalyLawrence(datetime)

	C := GetSolarEquationOfCenterLawrence(M)

	λ := GetSolarEclipticLongitudeLawrence(M, C)

	return EclipticCoordinate{
		Longitude: λ,
		Latitude:  0,
	}
}

/*
GetSolarEquatorialPosition()

@param datetime - the datetime of the observer (in UTC)
@returns the Solar equatorial position (right ascension & declination) in degrees:
*/
func GetSolarEquatorialPosition(datetime time.Time) EquatorialCoordinate {
	T := GetCurrentJulianCenturyRelativeToJ2000(datetime)

	ε := GetObliquityOfTheEclipticLawrence(T)

	ec := GetSolarEclipticPosition(datetime)

	// trigoneometric functions handle the correct degrees and radians conversions:
	ra := atan2yx(sinx(ec.Longitude)*cosx(ε)-tanx(ec.Latitude)*sinx(ε), cosx(ec.Longitude))

	// trigoneometric functions handle the correct degrees and radians conversions:
	dec := asinx(sinx(ec.Latitude)*cosx(ε) + cosx(ec.Latitude)*sinx(ε)*sinx(ec.Longitude))

	// correct for negative angles
	if ra < 0 {
		ra += 360
	}

	return EquatorialCoordinate{
		RightAscension: ra,
		Declination:    dec,
	}
}
