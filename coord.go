package dusk

import (
	"time"
)

// Coordinate conversions: ecliptic to equatorial, and equatorial to the
// observer's horizon. These sit one layer above everything in epoch.go, which
// they consume (julianCentury, the nutation and obliquity helpers, and
// localSiderealTime) and nothing in this file is consumed by. The only caller
// chain that reaches here is MoonriseMoonset's minute scan.
//
// See Meeus, Astronomical Algorithms, ch. 13.

// eclipticToEquatorial converts ecliptic coordinates (longitude, latitude in
// degrees) to equatorial coordinates using nutation-corrected obliquity and
// nutation in longitude.
//
// See Meeus, Astronomical Algorithms, eq. 13.3 & 13.4 p. 93.
func eclipticToEquatorial(t time.Time, lon, lat float64) equatorial {
	T := julianCentury(t)

	L := solarMeanLongitude(T)
	l := lunarMeanLongitude(T)
	omega := lunarAscendingNode(T)

	dpsi := nutationInLongitude(L, l, omega)
	lon += dpsi

	eps := meanObliquity(T) + nutationInObliquity(L, l, omega)

	ra := atan2x(sinx(lon)*cosx(eps)-tanx(lat)*sinx(eps), cosx(lon))
	dec := asinx(sinx(lat)*cosx(eps) + cosx(lat)*sinx(eps)*sinx(lon))

	return equatorial{
		ra:  mod360(ra),
		dec: dec,
	}
}

// altitudeOf returns the altitude in degrees of an equatorial position, for the
// given observer and time. Azimuth is deliberately not computed: the only caller
// is MoonriseMoonset's minute scan, which compares altitude against a horizon
// threshold. If a bearing is ever wanted, reinstate it against a live consumer
// and normalise it with mod360, as every other angle in this package is.
//
// See Meeus, Astronomical Algorithms, eq. 13.6 p. 93.
func altitudeOf(t time.Time, obs Observer, eq equatorial) float64 {
	lst := localSiderealTime(t, obs.lon)
	ha := hourAngle(eq.ra, lst)

	return asinx(sinx(eq.dec)*sinx(obs.lat) + cosx(eq.dec)*cosx(obs.lat)*cosx(ha))
}

// hourAngle computes the hour angle in degrees.
//
// Parameters use mixed units:
//   - ra: right ascension in degrees (0-360)
//   - lst: local sidereal time in hours (0-24)
//
// The conversion lst*15 is applied internally, so callers must not
// pre-convert LST to degrees.
func hourAngle(ra, lst float64) float64 {
	return mod360(lst*15 - ra)
}
