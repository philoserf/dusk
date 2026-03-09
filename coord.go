package dusk

import (
	"errors"
	"math"
	"time"
)

// ErrCircumpolar is returned when a celestial object is circumpolar
// (always above the horizon) at the given latitude.
var ErrCircumpolar = errors.New("dusk: object is circumpolar (always above the horizon)")

// ErrNeverRises is returned when a celestial object never rises above
// the horizon at the given latitude.
var ErrNeverRises = errors.New("dusk: object never rises at this latitude")

var errNilLocation = errors.New("dusk: location must not be nil")

// Observer represents a geographic position on Earth.
type Observer struct {
	Lat  float64        // latitude in degrees (north positive)
	Lon  float64        // longitude in degrees (east positive)
	Elev float64        // elevation in meters above sea level; negative values are treated as sea level; only affects sunrise/sunset and twilight
	Loc  *time.Location // timezone
}

var (
	errInvalidCoord      = errors.New("dusk: latitude must be in [-90, 90] and longitude in [-180, 180]")
	errInvalidEquatorial = errors.New("dusk: Dec must be in [-90, 90]")
)

// validateObserver checks that the observer has a non-nil location and valid
// latitude/longitude ranges.
func validateObserver(obs Observer) error {
	if obs.Loc == nil {
		return errNilLocation
	}
	if obs.Lat < -90 || obs.Lat > 90 || obs.Lon < -180 || obs.Lon > 180 {
		return errInvalidCoord
	}
	return nil
}

// Equatorial represents right ascension and declination in degrees.
type Equatorial struct {
	RA  float64
	Dec float64
}

// validateEquatorial normalizes RA to [0, 360) via mod360 and checks that
// Dec is in [-90, 90]. Returns the normalized Equatorial or an error.
func validateEquatorial(eq Equatorial) (Equatorial, error) {
	eq.RA = mod360(eq.RA)
	if eq.Dec < -90 || eq.Dec > 90 {
		return Equatorial{}, errInvalidEquatorial
	}
	return eq, nil
}

// Horizontal represents altitude and azimuth in degrees.
type Horizontal struct {
	Alt float64
	Az  float64
}

// Ecliptic represents ecliptic coordinates: longitude and latitude in degrees,
// and distance in kilometers.
type Ecliptic struct {
	Lon  float64 // ecliptic longitude in degrees
	Lat  float64 // ecliptic latitude in degrees
	Dist float64 // distance in km (used for Moon)
}

// EclipticToEquatorial converts ecliptic coordinates (longitude, latitude in
// degrees) to equatorial coordinates using nutation-corrected obliquity.
//
// See Meeus, Astronomical Algorithms, eq. 13.3 & 13.4 p. 93.
func EclipticToEquatorial(t time.Time, lon, lat float64) Equatorial {
	T := julianCentury(t)

	L := solarMeanLongitude(T)
	l := lunarMeanLongitude(T)
	omega := lunarAscendingNode(T)

	eps := meanObliquity(T) + nutationInObliquity(L, l, omega)

	ra := atan2x(sinx(lon)*cosx(eps)-tanx(lat)*sinx(eps), cosx(lon))
	dec := asinx(sinx(lat)*cosx(eps) + cosx(lat)*sinx(eps)*sinx(lon))

	return Equatorial{
		RA:  mod360(ra),
		Dec: dec,
	}
}

// EquatorialToHorizontal converts equatorial coordinates to horizontal
// (altitude/azimuth) for the given observer position and time.
//
// See Meeus, Astronomical Algorithms, eq. 13.5 & 13.6 p. 93.
func EquatorialToHorizontal(t time.Time, obs Observer, eq Equatorial) Horizontal {
	lst := LocalSiderealTime(t, obs.Lon)
	ha := HourAngle(eq.RA, lst)

	alt := asinx(sinx(eq.Dec)*sinx(obs.Lat) + cosx(eq.Dec)*cosx(obs.Lat)*cosx(ha))

	cosAltCosLat := cosx(alt) * cosx(obs.Lat)

	var az float64
	// Guard against division by zero at the poles (lat ±90) or zenith (alt 90).
	if math.Abs(cosAltCosLat) < 1e-10 {
		az = 0
	} else {
		az = acosx((sinx(eq.Dec) - sinx(alt)*sinx(obs.Lat)) / cosAltCosLat)
	}

	// acos gives 0..180; if sin(ha) > 0, object is west, so az = 360 - az
	if sinx(ha) > 0 {
		az = 360 - az
	}

	return Horizontal{
		Alt: alt,
		Az:  az,
	}
}

// HourAngle computes the hour angle in degrees from right ascension (degrees)
// and local sidereal time (hours).
func HourAngle(ra, lst float64) float64 {
	return mod360(lst*15 - ra)
}

// AngularSeparation returns the angular distance in degrees between two
// positions given as (ra, dec) pairs in degrees. Works for any spherical
// coordinate system (equatorial, ecliptic, geographic).
//
// Uses the robust atan2 formula to avoid precision loss near 0 and 180 degrees.
func AngularSeparation(ra1, dec1, ra2, dec2 float64) float64 {
	dra := ra2 - ra1

	x := cosx(dec1)*sinx(dec2) - sinx(dec1)*cosx(dec2)*cosx(dra)
	y := cosx(dec2) * sinx(dra)
	z := sinx(dec1)*sinx(dec2) + cosx(dec1)*cosx(dec2)*cosx(dra)

	return atan2x(math.Sqrt(x*x+y*y), z)
}
