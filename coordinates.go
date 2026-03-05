package dusk

import (
	"time"
)

type Coordinate struct {
	/*
		ϕ - the latitude in degrees, e.g., altitude, latitude, declination
	*/
	Latitude float64 `json:"latitude"`
	/*
		θ - the longitude in degrees, e.g., azimuth, right ascension, longitude
	*/
	Longitude float64 `json:"longitude"`
}

type EquatorialCoordinate struct {
	/*
		Right Ascension - the right ascension in degrees
	*/
	RightAscension float64 `json:"ra"`
	/*
		Declination - the declination in degrees
	*/
	Declination float64 `json:"dec"`
}

type EclipticCoordinate struct {
	/*
		Longitude - the longitude in degrees
	*/
	Longitude float64 `json:"longitude"`
	/*
		Latitude - the latitude in degrees
	*/
	Latitude float64 `json:"latitude"`
	/*
		Distance - the distance in km
	*/
	Δ float64 `json:"distance"`
}

type HorizontalCoordinate struct {
	/*
		altitude (a) or elevation
	*/
	Altitude float64 `json:"altitude"`
	/*
		azimuth (A)
	*/
	Azimuth float64 `json:"azimuth"`
}

type TemporalHorizontalCoordinate struct {
	/*
		datetime of horizontal observation
	*/
	Datetime time.Time `json:"datetime"`
	/*
		altitude (a) or elevation
	*/
	Altitude float64 `json:"altitude"`
	/*
		azimuth (A)
	*/
	Azimuth float64 `json:"azimuth"`
}

type TransitHorizontalCoordinate struct {
	/*
		datetime of horizontal observation
	*/
	Datetime time.Time `json:"datetime"`
	/*
		altitude (a) or elevation
	*/
	Altitude float64 `json:"altitude"`
	/*
		azimuth (A)
	*/
	Azimuth float64 `json:"azimuth"`
	/*
		Is this particular a Moon rise?
	*/
	IsRise bool `json:"isRise"`
	/*
		Is this particular a Moon set?
	*/
	IsSet bool `json:"isSet"`
}

/*
ConvertEclipticCoordinateToEquatorial()

@param datetime - the datetime of the observer (in UTC)
@param geocentric ecliptic coordinate of type EclipticCoordinate { λ, β, Λ }
@returns the converted equatorial coordinate { ra, dec }
@see eq13.3 & eq13.4 p.93 of Meeus, Jean. 1991. Astronomical algorithms. Richmond, Va: Willmann-Bell.
*/
func ConvertEclipticCoordinateToEquatorial(datetime time.Time, ec EclipticCoordinate) EquatorialCoordinate {
	J := GetCurrentJulianCenturyRelativeToJ2000(datetime)

	L := GetSolarMeanLongitude(J)

	l := GetLunarMeanLongitude(J)

	Ω := GetLunarLongitudeOfTheAscendingNode(J)

	ε := GetMeanObliquityOfTheEcliptic(J) + GetNutationInObliquityOfTheEcliptic(L, l, Ω)

	λ := ec.Longitude

	β := ec.Latitude

	α := atan2yx(sinx(λ)*cosx(ε)-tanx(β)*sinx(ε), cosx(λ))

	δ := asinx(sinx(β)*cosx(ε) + cosx(β)*sinx(ε)*sinx(λ))

	return EquatorialCoordinate{
		RightAscension: α,
		Declination:    δ,
	}
}

/*
ConvertEquatorialCoordinateToHorizontal()

@param datetime - the datetime of the observer (in UTC)
@param longitude - is the longitude (west is negative, east is positive) in degrees of some observer on Earth
@param latitude - is the latitude (south is negative, north is positive) in degrees of some observer on Earth
@param equatorial coordinate of type EquatorialCoordiate { ra, dec }
@returns the equivalent horizontal coordinate for the given observers position
@see eq13.5 and eq.6 p.93 of Meeus, Jean. 1991. Astronomical algorithms. Richmond, Va: Willmann-Bell.
*/
func ConvertEquatorialCoordinateToHorizontal(datetime time.Time, longitude, latitude float64, eq EquatorialCoordinate) HorizontalCoordinate {
	LST := GetLocalSiderealTime(datetime, longitude)

	ra := GetHourAngle(eq.RightAscension, LST)

	dec := eq.Declination

	alt := asinx(sinx(dec)*sinx(latitude) + cosx(dec)*cosx(latitude)*cosx(ra))

	az := acosx((sinx(dec) - sinx(alt)*sinx(latitude)) / (cosx(alt) * cosx(latitude)))

	return HorizontalCoordinate{
		Altitude: alt,
		Azimuth:  az,
	}
}
