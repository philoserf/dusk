# Dusk: Astronomical Calculations Library — Code Walkthrough

*2026-03-10T00:36:13Z by Showboat 0.6.1*
<!-- showboat-id: 4b2dcd3a-9851-4216-88af-c8b5f37e074e -->

## Overview

Dusk is a zero-dependency Go library for astronomical calculations. It computes sunrise/sunset, moonrise/moonset, lunar phase, twilight times, and celestial coordinate conversions using algorithms from Jean Meeus's *Astronomical Algorithms* (2nd ed., Willmann-Bell, 1998).

**Key design decisions:**

- All angles in **degrees** — trig helpers convert internally
- Zero-value `time.Time` signals "event did not occur"
- Sentinel errors `ErrCircumpolar` / `ErrNeverRises` for polar edge cases
- Single package, domain-focused files, zero external dependencies

The code has a clear dependency flow. We'll walk through it bottom-up: trig primitives → time/epoch foundations → coordinate conversions → solar calculations → lunar calculations → transit → twilight → string formatting.

### Project structure

```bash
find . -name "*.go" -not -name "*_test.go" | sort | while read f; do echo "$(wc -l < "$f" | tr -d " ") lines  $f"; done
```

```output
154 lines  ./coord.go
17 lines  ./doc.go
139 lines  ./epoch.go
167 lines  ./lunar_tables.go
257 lines  ./lunar.go
154 lines  ./solar.go
71 lines  ./stringer.go
141 lines  ./transit.go
40 lines  ./trig.go
66 lines  ./twilight.go
```

```bash
head -3 go.mod
```

```output
module github.com/philoserf/dusk/v2

go 1.24
```

Zero external dependencies — only `math`, `time`, `errors`, and `fmt` from the standard library. The entire library is ~1,200 lines of source code.

---

## Layer 1: Trig Primitives (`trig.go`)

Every astronomical formula in this library works in degrees, but Go's `math` package works in radians. `trig.go` bridges the gap with thin wrappers, plus angle-normalization utilities used everywhere.

```bash
sed -n "1,24p" trig.go
```

```output
package dusk

import "math"

const (
	degToRad = math.Pi / 180.0
	radToDeg = 180.0 / math.Pi
)

// clamp restricts x to [-1, 1] before passing it to asin/acos. This applies
// to any out-of-range value, not just those slightly outside [-1, 1] due to
// floating-point rounding.
func clamp(x float64) float64 { return math.Max(-1, math.Min(1, x)) }

func sinx(deg float64) float64    { return math.Sin(deg * degToRad) }
func cosx(deg float64) float64    { return math.Cos(deg * degToRad) }
func tanx(deg float64) float64    { return math.Tan(deg * degToRad) }
func asinx(x float64) float64     { return radToDeg * math.Asin(clamp(x)) }
func acosx(x float64) float64     { return radToDeg * math.Acos(clamp(x)) }
func atan2x(y, x float64) float64 { return radToDeg * math.Atan2(y, x) }

func sincosx(deg float64) (float64, float64) {
	return math.Sincos(deg * degToRad)
}
```

Key points:

- **`clamp`** guards `asinx`/`acosx` against domain errors. Floating-point accumulation in multi-step formulas (e.g., the cosine of hour angle in `solarHourAngle`) can produce values like 1.0000000000000002 that would return NaN from `math.Asin`. The clamp catches all out-of-range values, not just rounding artifacts.
- **`sincosx`** is a single-call optimization used in the lunar ecliptic position loop (60+ iterations per call).
- All wrappers are unexported — callers use degrees throughout and never touch radians.

### Angle normalization

```bash
sed -n "26,40p" trig.go
```

```output
func mod360(x float64) float64 {
	x = math.Mod(x, 360)
	if x < 0 {
		x += 360
	}
	return x
}

func mod24(x float64) float64 {
	x = math.Mod(x, 24)
	if x < 0 {
		x += 24
	}
	return x
}
```

`mod360` normalizes angles to [0, 360) and `mod24` normalizes hours to [0, 24). These appear throughout — ecliptic longitudes, right ascensions, sidereal times, and mean anomalies all need wrapping after arithmetic.

---

## Layer 2: Time and Epoch (`epoch.go`)

Astronomical algorithms operate on Julian dates (continuous day count since 4713 BC). `epoch.go` provides the bridge between Go `time.Time` values and Julian date arithmetic.

### Julian date conversion

```bash
sed -n "1,44p" epoch.go
```

```output
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
```

**Design note:** `JulianDate` uses `UnixNano()` for precision, which limits the valid range to ~1677–2262. This is a known limitation documented in the function and guarded by `ValidJulianDateRange`. Dates outside this range silently produce wrong results because Go's `UnixNano()` returns 0 for out-of-range times.

The two epoch constants anchor the conversions:
- `j1970` (2440587.5) — Julian date of the Unix epoch (1970-01-01 00:00 UTC)
- `j2000` (2451545.0) — Julian date of J2000.0 (2000-01-01 12:00 TT), the standard astronomical epoch

### Sidereal time

```bash
sed -n "46,70p" epoch.go
```

```output
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
```

Sidereal time is the angle between the vernal equinox and the local meridian — it tells you what right ascension is currently on your meridian. `greenwichMeanSiderealTime` uses Meeus eq. 12.4, and `LocalSiderealTime` adjusts for the observer's longitude.

Note the unit convention: GMST is computed in degrees (to stay in the library's degree convention), then `LocalSiderealTime` converts to hours via `/ 15.0`. This is the only exported function in `epoch.go`.

### Helper functions

```bash
sed -n "72,98p" epoch.go
```

```output
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
```

These helpers form the foundation for all date-based calculations:

- **`julianCentury`** — divides by 36525 (days per Julian century). Used by nutation, obliquity, and lunar calculations.
- **`julianDay`** — rounds to integer days since J2000.0. Used by `meanSolarTime` for the NOAA solar method.
- **`meanSolarTime`** — adjusts the day count for the observer's longitude.
- **`universalTimeFromJD`** — reverses `JulianDate`, converting back to `time.Time`. Used to produce sunrise/sunset/twilight times.
- **`datetimeZeroHour`** — extracts midnight UTC. Used to anchor sidereal time calculations to the start of the UT day.

### Nutation and obliquity

```bash
sed -n "100,140p" epoch.go
```

```output
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
```

Nutation is the Earth's axial wobble (~18.6 year cycle). Obliquity is the tilt of Earth's axis (~23.4°). These affect the transformation between ecliptic and equatorial coordinates.

The `nutationInObliquity` function uses a simplified 4-term model (Meeus p. 144). The full Meeus model has 63 terms, but the dominant terms captured here provide sub-arcminute accuracy sufficient for rise/set calculations.

Three mean-longitude functions (`solarMeanLongitude`, `lunarMeanLongitude`, `lunarAscendingNode`) are needed as arguments to the nutation formula. They appear again in the lunar position calculation.

---

## Layer 3: Coordinate Types and Conversions (`coord.go`)

`coord.go` defines the three coordinate systems, validation logic, and the transformations between them.

### Types and validation

```bash
sed -n "1,66p" coord.go
```

```output
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
	Lat float64 // latitude in degrees (north positive)
	Lon float64 // longitude in degrees (east positive)
	// Elev is the observer's elevation in meters above sea level.
	// Negative values are clamped to 0 (sea level).
	// Only affects sunrise/sunset and twilight; ignored by MoonriseMoonset
	// and ObjectTransit.
	Elev float64
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
// Non-finite RA or Dec values (NaN, ±Inf) are rejected.
func validateEquatorial(eq Equatorial) (Equatorial, error) {
	if math.IsNaN(eq.RA) || math.IsInf(eq.RA, 0) || math.IsNaN(eq.Dec) || math.IsInf(eq.Dec, 0) {
		return Equatorial{}, errInvalidEquatorial
	}
	eq.RA = mod360(eq.RA)
	if eq.Dec < -90 || eq.Dec > 90 {
		return Equatorial{}, errInvalidEquatorial
	}
	return eq, nil
}
```

Key design decisions:

- **`Observer` centralizes position** — latitude, longitude, elevation, and timezone in one struct. Every public function that produces local times takes an `Observer`.
- **`validateEquatorial` normalizes RA** via `mod360` rather than rejecting values at 360.0. This is defensive — intermediate calculations can produce RA slightly outside [0, 360). NaN/Inf values are explicitly rejected to prevent silent propagation.
- **`Observer.Elev`** only affects sunrise/sunset and twilight through the atmospheric refraction correction in `solarHourAngle`. Negative elevations are clamped to sea level. Moonrise and `ObjectTransit` ignore elevation entirely.
- **Split sentinel errors** — `ErrCircumpolar` and `ErrNeverRises` are separate sentinels rather than a single "polar" error, so callers can distinguish "the sun never sets" from "the sun never rises."

### Ecliptic to equatorial

```bash
sed -n "74,102p" coord.go
```

```output
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
```

This is the central coordinate transformation. The ecliptic plane (Earth's orbit around the Sun) is tilted relative to the equatorial plane (Earth's equator extended into space) by the obliquity angle (~23.4°). This function rotates from one to the other, applying the nutation correction for the wobble of Earth's axis.

The RA is normalized with `mod360` because `atan2x` can return negative values.

### Equatorial to horizontal

```bash
sed -n "104,139p" coord.go
```

```output
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
```

This converts sky coordinates (RA/Dec) to what you actually see: altitude above the horizon and compass direction (azimuth). The hour angle bridges the gap — it's how far the object has rotated past your meridian.

The pole/zenith guard (`cosAltCosLat < 1e-10`) prevents division by zero when the observer is at a geographic pole or the object is directly overhead. In those cases azimuth is undefined, so it defaults to 0.

### Angular separation

```bash
sed -n "141,155p" coord.go
```

```output
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
```

The `atan2` formula is more numerically stable than the simpler `acos(dot product)` approach, which loses precision near 0° and 180° separation. Parameter order is RA-first to match the `Equatorial{RA, Dec}` field order.

---

## Layer 4: Solar Calculations (`solar.go`)

Solar calculations are the core use case — sunrise, sunset, and solar noon. The implementation follows the NOAA solar calculator method (derived from Meeus).

### Shared solar parameters

```bash
sed -n "17,36p" solar.go
```

```output
// solarParams holds intermediate solar position values computed from a date
// and longitude. Used by SunriseSunset and twilight to avoid repeating the
// 6-step parameter sequence.
type solarParams struct {
	delta    float64 // solar declination (degrees)
	jTransit float64 // Julian date of solar transit (noon)
}

// computeSolarParams returns the solar declination and transit JD for a given
// date and observer longitude.
func computeSolarParams(date time.Time, lon float64) solarParams {
	J := meanSolarTime(date, lon)
	M := solarMeanAnomaly(J)
	C := solarEquationOfCenter(M)
	lambda := solarEclipticLongitude(M, C)
	T := julianCentury(date)
	delta := solarDeclination(lambda, T)
	jTransit := solarTransitJD(J, M, lambda)
	return solarParams{delta: delta, jTransit: jTransit}
}
```

`computeSolarParams` extracts the 6-step solar parameter sequence that was previously duplicated in `SunriseSunset`, `CivilTwilight`, `NauticalTwilight`, and `AstronomicalTwilight`. The pipeline is:

1. `meanSolarTime` → integer day offset for longitude
2. `solarMeanAnomaly` → M (where the Earth is in its orbit)
3. `solarEquationOfCenter` → C (correction for elliptical orbit)
4. `solarEclipticLongitude` → λ (Sun's position on the ecliptic)
5. `solarDeclination` → δ (how far the Sun is above/below the equatorial plane)
6. `solarTransitJD` → J_transit (Julian date of solar noon)

### Sunrise and sunset

```bash
sed -n "38,70p" solar.go
```

```output
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

	sp := computeSolarParams(date, obs.Lon)

	omega, err := solarHourAngle(sp.delta, 0, obs.Lat, obs.Elev)
	if err != nil {
		return SunEvent{}, err
	}

	Jrise := sp.jTransit - omega/360.0
	Jset := sp.jTransit + omega/360.0

	rise := universalTimeFromJD(Jrise).In(obs.Loc)
	noon := universalTimeFromJD(sp.jTransit).In(obs.Loc)
	set := universalTimeFromJD(Jset).In(obs.Loc)

	return SunEvent{
		Rise:     rise,
		Noon:     noon,
		Set:      set,
		Duration: set.Sub(rise),
	}, nil
}
```

The sunrise/sunset calculation is symmetric around solar noon (`jTransit`). The hour angle ω (in degrees) represents how far the Sun moves from the meridian between transit and rise/set. Dividing by 360 converts the angular displacement to a fraction of a day, which is added to/subtracted from the transit JD.

Depression = 0 tells `solarHourAngle` to use the standard −0.83° correction (atmospheric refraction + solar semidiameter).

### Solar hour angle and polar edge cases

```bash
sed -n "119,155p" solar.go
```

```output
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
```

This is the most subtle function in the library. The hour angle formula solves for when the Sun crosses a given altitude:

- **depression = 0** → sunrise/sunset: uses −0.83° (0.5° refraction + 0.267° semidiameter + 0.0583° "standard" correction), adjusted for elevation
- **depression > 0** → twilight: uses the depression angle directly (no refraction correction), per IAU/USNO convention

The elevation correction (`2.076 * √elev / 60`) accounts for the geometric dip of the horizon at altitude. Higher observers see further, so the Sun appears to rise earlier and set later.

**Polar edge cases:** When `cos(HA) < -1`, the Sun never reaches the depression angle from above (it's always above) → midnight sun / circumpolar. When `cos(HA) > 1`, it never reaches it from below → polar night / never rises.

### Solar position

```bash
sed -n "72,87p" solar.go
```

```output
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
```

`SolarPosition` differs from `computeSolarParams` in a key way: it uses continuous (fractional) Julian days instead of integer-rounded days. This gives precise Sun position at any instant, not just for a calendar date. It's used when you need the Sun's RA/Dec, not when computing rise/set times.

Note that the Sun lies on the ecliptic by definition (latitude = 0), so the ecliptic-to-equatorial conversion only needs longitude.

---

## Layer 5: Lunar Calculations (`lunar.go` + `lunar_tables.go`)

The Moon's orbit is far more complex than the Sun's. Where the solar calculations use a simple 3-term equation of center, the lunar position requires summing 60+ periodic terms from Meeus Chapter 47.

### Lunar ecliptic position

```bash
sed -n "35,98p" lunar.go
```

```output
// LunarEclipticPosition returns the geocentric ecliptic longitude, latitude
// (degrees), and distance (km) of the Moon for the given instant.
//
// The algorithm is from Meeus, Astronomical Algorithms, Chapter 47.
func LunarEclipticPosition(t time.Time) Ecliptic {
	T := julianCentury(t)

	D := lunarMeanElongation(T)
	Lp := lunarMeanLongitude(T)
	M := solarMeanAnomalyFromCentury(T)
	Mp := lunarMeanAnomaly(T)
	F := lunarArgumentOfLatitude(T)

	A1 := mod360(119.75 + 131.849*T)
	A2 := mod360(53.09 + 479264.29*T)
	A3 := mod360(313.45 + 481266.484*T)

	E := 1 - 0.002516*T - 0.0000074*T*T
	E2 := E * E

	// Additive corrections in units of 0.000001 degrees (Meeus p. 338).
	Sl := 3958*sinx(A1) + 1962*sinx(Lp-F) + 318*sinx(A2)
	Sr := 0.0
	Sb := -2235*sinx(Lp) + 382*sinx(A3) + 175*sinx(A1-F) + 175*sinx(A1+F) + 127*sinx(Lp-Mp) - 115*sinx(Lp+Mp)

	for i := range tableLongDist {
		r := &tableLongDist[i]
		arg := D*r.D + M*r.M + Mp*r.Mʹ + F*r.F
		sa, ca := sincosx(arg)

		switch r.M {
		case 0:
			Sl += r.Σl * sa
			Sr += r.Σr * ca
		case 1, -1:
			Sl += r.Σl * sa * E
			Sr += r.Σr * ca * E
		case 2, -2:
			Sl += r.Σl * sa * E2
			Sr += r.Σr * ca * E2
		}
	}

	for i := range tableLat {
		r := &tableLat[i]
		arg := D*r.D + M*r.M + Mp*r.Mʹ + F*r.F
		sb := sinx(arg)

		switch r.M {
		case 0:
			Sb += r.Σb * sb
		case 1, -1:
			Sb += r.Σb * sb * E
		case 2, -2:
			Sb += r.Σb * sb * E2
		}
	}

	return Ecliptic{
		Lon:  mod360(Lp + Sl/1e6),
		Lat:  Sb / 1e6,
		Dist: 385000.56 + Sr/1000,
	}
}
```

This is the most computationally intensive function in the library. Key details:

- **Five fundamental arguments** (D, L', M, M', F) describe the Moon's orbital parameters at time T.
- **Three auxiliary angles** (A1, A2, A3) provide additional corrections from Meeus.
- **E correction** accounts for the decreasing eccentricity of Earth's orbit over time. Terms involving the Sun's mean anomaly (M) are multiplied by E or E², depending on the power of M in the term.
- **`sincosx`** computes sin and cos simultaneously for efficiency — each table row needs both for longitude/distance terms.
- The summation loops iterate over Meeus Table 47.A (60 rows for longitude/distance) and Table 47.B (60 rows for latitude), stored in `lunar_tables.go`.
- Final values are scaled: longitude/latitude from millionths of a degree, distance from thousandths of a km.

### Lunar tables (`lunar_tables.go`)

```bash
sed -n "1,13p" lunar_tables.go && echo "..." && sed -n "14,19p" lunar_tables.go && echo "    ..." && echo "}" && echo "" && sed -n "91,96p" lunar_tables.go && echo "    ..." && echo "}"
```

```output
package dusk

// Meeus Table 47.A — Periodic terms for the longitude (Σl) and distance (Σr)
// of the Moon.
//
// See Meeus, Astronomical Algorithms, p. 339.
type lunarLongDistCoeff struct{ D, M, Mʹ, F, Σl, Σr float64 }

// Meeus Table 47.B — Periodic terms for the latitude (Σb) of the Moon.
//
// See Meeus, Astronomical Algorithms, p. 341.
type lunarLatCoeff struct{ D, M, Mʹ, F, Σb float64 }

...
var tableLongDist = [...]lunarLongDistCoeff{
	{0, 0, 1, 0, 6288774, -20905355},
	{2, 0, -1, 0, 1274027, -3699111},
	{2, 0, 0, 0, 658314, -2955968},
	{0, 0, 2, 0, 213618, -569925},

    ...
}

var tableLat = [...]lunarLatCoeff{
	{0, 0, 0, 1, 5128122},
	{0, 0, 1, 1, 280602},
	{0, 0, 1, -1, 277693},
	{2, 0, 0, -1, 173237},

    ...
}
```

```bash
awk "/tableLongDist/{a=1} a && /{/{c++} a && /}$/{print \"Table 47.A (longitude/distance): \" c-1 \" entries\"; a=0; c=0}" lunar_tables.go && awk "/tableLat/{a=1} a && /{/{c++} a && /}$/{print \"Table 47.B (latitude): \" c-1 \" entries\"; a=0; c=0}" lunar_tables.go
```

```output
Table 47.A (longitude/distance): 60 entries
Table 47.B (latitude): 60 entries
```

The tables are stored as fixed-size arrays (`[...]`), not slices — the compiler knows the exact size and can optimize accordingly. Each entry encodes the multipliers for the fundamental arguments (D, M, M', F) and the coefficient amplitudes. These are direct transcriptions from Meeus pages 339 and 341.

**Concern:** The table data is hand-transcribed from the book. Transcription errors would produce subtle position inaccuracies that might not be caught by tests unless compared against an independent implementation. The test suite validates against USNO and Stellarium values, which provides good confidence.

### Lunar phase

```bash
sed -n "100,137p" lunar.go
```

```output
// LunarPhase returns the lunar phase for the given instant.
//
// The phase angle uses the Meeus approach: solar ecliptic longitude from the
// mean-anomaly method, lunar ecliptic position from Chapter 47 tables.
func LunarPhase(t time.Time) LunarPhaseInfo {
	ec := LunarEclipticPosition(t)

	J := JulianDate(t) - j2000
	Msol := solarMeanAnomaly(J)
	C := solarEquationOfCenter(Msol)
	sunLon := solarEclipticLongitude(Msol, C)

	T := julianCentury(t)
	Mp := lunarMeanAnomaly(T)

	// elongation (0-360°, waxing = 0-180, waning = 180-360)
	d := acosx(cosx(ec.Lon-sunLon) * cosx(ec.Lat))
	if mod360(ec.Lon-sunLon) > 180 {
		d = 360 - d
	}

	// phase angle (Meeus p. 346)
	PA := 180 - d - 0.1468*((1-0.0549*sinx(Mp))/(1-0.0167*sinx(Msol)))*sinx(d)

	K := 100 * (1 + cosx(PA)) / 2

	frac := (1 - cosx(d)) / 2
	days := frac * lunarMonthDays

	return LunarPhaseInfo{
		Illumination: K,
		Elongation:   d,
		Angle:        PA,
		DaysApprox:   days,
		Waxing:       d < 180,
		Name:         lunarPhaseName(d),
	}
}
```

The phase calculation combines the Sun's ecliptic longitude (simple formula) with the Moon's ecliptic position (full Chapter 47 calculation). The elongation is the angular distance between Sun and Moon as seen from Earth:

- 0° = New Moon (Moon between Earth and Sun)
- 90° = First Quarter
- 180° = Full Moon (Earth between Moon and Sun)
- 270° = Last Quarter

The `Waxing` boolean (elongation < 180) lets callers distinguish waxing from waning phases. `DaysApprox` is symmetric — it gives days into the current half-cycle, not the overall lunation.

### Moonrise and moonset

```bash
sed -n "146,204p" lunar.go
```

```output
// MoonriseMoonset computes the moonrise and moonset times for the given date
// at the specified observer position and timezone.
//
// The algorithm scans minute-by-minute through the day to detect altitude
// sign changes. This is slow by design (~1440 ecliptic-position evaluations).
// Callers computing moonrise/moonset for many dates (e.g., monthly calendars)
// should expect proportional cost.
//
// Observer elevation (obs.Elev) is not used; it only affects sunrise/sunset
// and twilight calculations.
//
// An error is returned if obs.Loc is nil.
func MoonriseMoonset(date time.Time, obs Observer) (MoonEvent, error) {
	if err := validateObserver(obs); err != nil {
		return MoonEvent{}, err
	}

	localDate := date.In(obs.Loc)
	d := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, obs.Loc).UTC()
	nextMidnight := time.Date(localDate.Year(), localDate.Month(), localDate.Day()+1, 0, 0, 0, 0, obs.Loc).UTC()
	scanMinutes := int(nextMidnight.Sub(d).Minutes())

	var rise, set time.Time

	ec0 := LunarEclipticPosition(d)
	eq0 := EclipticToEquatorial(d, ec0.Lon, ec0.Lat)
	prevAlt := EquatorialToHorizontal(d, obs, eq0).Alt

	for i := 1; i <= scanMinutes; i++ {
		cur := d.Add(time.Duration(i) * time.Minute)

		ec := LunarEclipticPosition(cur)
		eq := EclipticToEquatorial(cur, ec.Lon, ec.Lat)
		hz := EquatorialToHorizontal(cur, obs, eq)

		if rise.IsZero() && hz.Alt > -lunarHorizonDepression && prevAlt <= -lunarHorizonDepression {
			rise = cur.In(obs.Loc)
		}
		if set.IsZero() && hz.Alt < -lunarHorizonDepression && prevAlt >= -lunarHorizonDepression {
			set = cur.In(obs.Loc)
		}
		if !rise.IsZero() && !set.IsZero() {
			break
		}

		prevAlt = hz.Alt
	}

	var dur time.Duration
	if !rise.IsZero() && !set.IsZero() && set.After(rise) {
		dur = set.Sub(rise)
	}

	return MoonEvent{
		Rise:     rise,
		Set:      set,
		Duration: dur,
	}, nil
}
```

Unlike the Sun (where rise/set can be computed analytically), moonrise/moonset requires brute-force scanning because the Moon moves significantly (~13°/day) during the day. The scan detects altitude crossing the horizon depression threshold (−0.833° for refraction + semidiameter).

Key details:

- **DST-safe scanning**: Uses local-midnight-to-local-midnight (`scanMinutes` adapts to 23h or 25h DST transition days), not a fixed 1440-minute window.
- **Early exit**: Breaks as soon as both rise and set are found.
- **Zero-value convention**: If the Moon doesn't rise or set during the day (possible near the poles or when the Moon is near the horizon for extended periods), the corresponding field is the zero `time.Time`.
- **Performance**: This is the slowest function in the library. Each minute evaluates the full Chapter 47 lunar position. Benchmarks show this is the dominant cost for any caller.

**Concern:** The minute-by-minute resolution means moonrise/moonset times have ±1 minute precision. This can differ from USNO by several minutes for the same reason. A bisection refinement step after detecting the crossing could improve precision without significant cost, but the current accuracy is adequate for most use cases.

---

## Layer 6: Object Transit (`transit.go`)

`ObjectTransit` computes rise, set, and transit maximum for any celestial object given its equatorial coordinates. Unlike the solar and lunar functions that compute positions internally, this takes pre-computed RA/Dec as input.

```bash
sed -n "1,85p" transit.go
```

```output
package dusk

import "time"

// Transit holds rise, set, and maximum transit times for a celestial object,
// along with the duration above the horizon.
// When using ObjectTransit, objects that never rise or set are reported via
// ErrCircumpolar or ErrNeverRises, and successful results always have
// non-zero Rise, Set, and Maximum times.
type Transit struct {
	Rise     time.Time
	Set      time.Time
	Maximum  time.Time
	Duration time.Duration
}

// ObjectTransit computes rise, set, and transit maximum for a celestial
// object at the given observer position on the specified date. Only the
// calendar date (year/month/day in UTC) is used; the time-of-day is ignored.
//
// Observer elevation (obs.Elev) is not used; it only affects sunrise/sunset
// and twilight calculations.
//
// Returns an error if obs.Loc is nil or coordinates are out of range.
// Returns ErrCircumpolar if the object never sets, or ErrNeverRises if
// it never rises at the given latitude.
func ObjectTransit(date time.Time, eq Equatorial, obs Observer) (Transit, error) {
	if err := validateObserver(obs); err != nil {
		return Transit{}, err
	}
	var err error
	if eq, err = validateEquatorial(eq); err != nil {
		return Transit{}, err
	}

	if err := objectTransitCheck(eq.Dec, obs.Lat); err != nil {
		return Transit{}, err
	}

	argLST := argOfLSTForTransit(obs.Lat, eq.Dec)

	// LST of rise and set in hours.
	riseLST := mod24(24 + eq.RA/15 - argLST/15)
	setLST := mod24(eq.RA/15 + argLST/15)

	// Convert LST → GST → UT.
	riseGST := lstToGST(riseLST, obs.Lon)
	setGST := lstToGST(setLST, obs.Lon)

	riseUT := gstToUT(date, riseGST)
	setUT := gstToUT(date, setGST)

	// Build time.Time values from UT hours on the given date (UTC midnight).
	midnight := datetimeZeroHour(date)

	rise := midnight.Add(time.Duration(riseUT * float64(time.Hour))).In(obs.Loc)
	set := midnight.Add(time.Duration(setUT * float64(time.Hour))).In(obs.Loc)

	// If set is before rise, push set to the next day.
	if set.Before(rise) {
		nextDay := midnight.AddDate(0, 0, 1)
		setUTNext := gstToUT(nextDay, setGST)
		set = nextDay.Add(time.Duration(setUTNext * float64(time.Hour))).In(obs.Loc)
	}

	// Transit maximum occurs when the hour angle is zero (LST = RA).
	transitLST := eq.RA / 15 // hours
	transitGST := lstToGST(transitLST, obs.Lon)
	transitUT := gstToUT(date, transitGST)
	maximum := midnight.Add(time.Duration(transitUT * float64(time.Hour))).In(obs.Loc)

	// If maximum is before rise, recompute on the next day.
	if maximum.Before(rise) {
		nextDay := midnight.AddDate(0, 0, 1)
		transitUTNext := gstToUT(nextDay, transitGST)
		maximum = nextDay.Add(time.Duration(transitUTNext * float64(time.Hour))).In(obs.Loc)
	}

	return Transit{
		Rise:     rise,
		Set:      set,
		Maximum:  maximum,
		Duration: set.Sub(rise),
	}, nil
}
```

The transit calculation uses the classical sidereal-time method:

1. **Check if the object rises/sets** — `objectTransitCheck` uses `tan(lat) × tan(dec)` to detect circumpolar/never-rises conditions.
2. **Compute the hour angle at rise/set** — `argOfLSTForTransit` returns `acos(-tan(lat) × tan(dec))`.
3. **Convert LST → GST → UT** — the chain goes from local sidereal time through Greenwich sidereal time to Universal Time.
4. **Transit maximum** — computed analytically as the moment when the hour angle is zero (LST = RA/15). This is an O(1) solution that replaced an earlier O(n) minute-by-minute scan.
5. **Day boundary handling** — if set or maximum falls before rise, it's pushed to the next day.

### GST to UT conversion

```bash
sed -n "113,141p" transit.go
```

```output
// gstToUT converts Greenwich sidereal time (hours) to Universal Time (hours)
// for the given date.
//
// This uses the Duffett-Smith / J1900-epoch algorithm rather than the Meeus
// J2000-based formulas used elsewhere in the library. The two epoch systems
// produce equivalent results; this particular algorithm is retained because it
// maps GST→UT directly without iterative inversion.
//
// Precision note: this J1900-era sidereal-time approximation (via the
// R/B/T0 terms) loses accuracy for dates far from 1900 due to the model
// and floating-point arithmetic. The resulting UT remains adequate within
// the library's valid JulianDate range (~1677–2262).
func gstToUT(datetime time.Time, GST float64) float64 {
	d := datetimeZeroHour(datetime)
	JD := JulianDate(d)
	// January 0 is Go's representation of December 31 of the prior year.
	JD0 := JulianDate(time.Date(datetime.Year(), 1, 0, 0, 0, 0, 0, time.UTC))
	days := JD - JD0
	T := (JD0 - 2415020) / 36525 // Duffett-Smith epoch (J1900)
	R := 6.6460656 + 2400.051262*T + 0.00002581*T*T
	B := 24 - R + float64(24*(datetime.Year()-1900))
	T0 := (0.0657098 * days) - B
	T0 = mod24(T0)
	A := GST - T0
	if A < 0 {
		A += 24
	}
	return 0.997270 * A
}
```

**Concern:** `gstToUT` uses a J1900-epoch algorithm (Duffett-Smith) while the rest of the library uses J2000-based Meeus formulas. This is intentional — the J1900 algorithm provides a direct (non-iterative) GST→UT conversion. The precision note documents that accuracy degrades for dates far from 1900, but remains adequate within the `JulianDate` valid range (~1677–2262).

The `0.997270` factor is the ratio of a solar day to a sidereal day (23h 56m 4.09s / 24h).

---

## Layer 7: Twilight (`twilight.go`)

Twilight calculations reuse the solar infrastructure with different depression angles.

```bash
cat twilight.go
```

```output
package dusk

import "time"

// TwilightEvent holds the dusk and dawn times of a twilight period.
// Dusk is tonight's boundary (sun passes below the depression angle).
// Dawn is tomorrow morning's boundary (sun passes above the depression angle).
// To get this morning's dawn, call with yesterday's date.
type TwilightEvent struct {
	Dusk     time.Time     // evening boundary (today)
	Dawn     time.Time     // morning boundary (tomorrow)
	Duration time.Duration // time from Dusk to Dawn (overnight period below the depression angle)
}

// CivilTwilight computes the evening civil twilight period (Sun 6° below the
// horizon) for the given date and observer position. Dusk is tonight's civil
// dusk; Dawn is tomorrow morning's civil dawn.
func CivilTwilight(date time.Time, obs Observer) (TwilightEvent, error) {
	return twilight(date, obs, 6)
}

// NauticalTwilight computes the evening nautical twilight period (Sun 12°
// below the horizon) for the given date and observer position.
func NauticalTwilight(date time.Time, obs Observer) (TwilightEvent, error) {
	return twilight(date, obs, 12)
}

// AstronomicalTwilight computes the evening astronomical twilight period (Sun
// 18° below the horizon) for the given date and observer position.
func AstronomicalTwilight(date time.Time, obs Observer) (TwilightEvent, error) {
	return twilight(date, obs, 18)
}

// twilight computes the twilight period for a given depression angle (positive
// degrees below the geometric horizon). Only the calendar date is used; the
// time-of-day is ignored. The returned Dusk is today's "set" at the depression
// angle (evening boundary) and Dawn is tomorrow's "rise" at the depression
// angle (morning boundary).
func twilight(date time.Time, obs Observer, depression float64) (TwilightEvent, error) {
	if err := validateObserver(obs); err != nil {
		return TwilightEvent{}, err
	}

	// Evening twilight: sunset at the given depression angle for today.
	sp := computeSolarParams(date, obs.Lon)
	omega, err := solarHourAngle(sp.delta, depression, obs.Lat, obs.Elev)
	if err != nil {
		return TwilightEvent{}, err
	}
	dusk := universalTimeFromJD(sp.jTransit + omega/360).In(obs.Loc)

	// Tomorrow's "rise" at this depression = twilight dawn.
	tomorrow := date.AddDate(0, 0, 1)
	sp2 := computeSolarParams(tomorrow, obs.Lon)
	omega2, err2 := solarHourAngle(sp2.delta, depression, obs.Lat, obs.Elev)
	if err2 != nil {
		return TwilightEvent{}, err2
	}
	dawn := universalTimeFromJD(sp2.jTransit - omega2/360).In(obs.Loc)

	return TwilightEvent{
		Dusk:     dusk,
		Dawn:     dawn,
		Duration: dawn.Sub(dusk),
	}, nil
}
```

Twilight is structurally identical to sunrise/sunset, just with a different depression angle:

| Type           | Depression | Meaning                                   |
| -------------- | ---------- | ----------------------------------------- |
| Civil          | 6°         | Enough light for outdoor activities       |
| Nautical       | 12°        | Horizon visible at sea; bright stars only |
| Astronomical   | 18°        | Sky fully dark for telescopes             |

The `twilight` function computes dusk (today's evening boundary) and dawn (tomorrow morning's boundary). This "overnight" convention means callers get a complete darkness period from a single call. To get this morning's dawn, call with yesterday's date.

**Design note:** `computeSolarParams` is called twice (today and tomorrow) because the Sun's declination changes slightly between days. At high latitudes near the solstices this difference can determine whether twilight occurs at all — the polar transition test verifies this at 75°N where civil twilight succeeds on November 25 but returns `ErrNeverRises` on November 26.

---

## Layer 8: String Formatting (`stringer.go`)

All eight exported result types implement `fmt.Stringer` for convenient display.

```bash
cat stringer.go
```

```output
package dusk

import (
	"fmt"
	"time"
)

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "--:--"
	}
	return t.Format("15:04")
}

// String returns a human-readable representation of the equatorial coordinates.
func (e Equatorial) String() string {
	return fmt.Sprintf("RA=%.3f° Dec=%.3f°", e.RA, e.Dec)
}

// String returns a human-readable representation of the horizontal coordinates.
func (h Horizontal) String() string {
	return fmt.Sprintf("Alt=%.3f° Az=%.3f°", h.Alt, h.Az)
}

// String returns a human-readable representation of the ecliptic coordinates.
func (e Ecliptic) String() string {
	return fmt.Sprintf("Lon=%.3f° Lat=%.3f° Dist=%.1fkm", e.Lon, e.Lat, e.Dist)
}

// String returns a human-readable representation of the sun event.
func (s SunEvent) String() string {
	return fmt.Sprintf("Rise=%s Noon=%s Set=%s Duration=%s",
		formatTime(s.Rise),
		formatTime(s.Noon),
		formatTime(s.Set),
		s.Duration)
}

// String returns a human-readable representation of the moon event.
func (m MoonEvent) String() string {
	return fmt.Sprintf("Rise=%s Set=%s Duration=%s",
		formatTime(m.Rise),
		formatTime(m.Set),
		m.Duration)
}

// String returns a human-readable representation of the lunar phase.
func (l LunarPhaseInfo) String() string {
	namePart := l.Name
	if namePart != "" {
		namePart += " "
	}
	return fmt.Sprintf("%s%.1f%% (day %.1f)", namePart, l.Illumination, l.DaysApprox)
}

// String returns a human-readable representation of the transit event.
func (t Transit) String() string {
	return fmt.Sprintf("Rise=%s Max=%s Set=%s Duration=%s",
		formatTime(t.Rise),
		formatTime(t.Maximum),
		formatTime(t.Set),
		t.Duration)
}

// String returns a human-readable representation of the twilight event.
func (tw TwilightEvent) String() string {
	return fmt.Sprintf("Dusk=%s Dawn=%s Duration=%s",
		formatTime(tw.Dusk),
		formatTime(tw.Dawn),
		tw.Duration)
}
```

The `formatTime` helper displays `--:--` for zero-value times (events that didn't occur), keeping the output readable rather than showing Go's zero time (`0001-01-01 00:00`).

Each Stringer is deliberately simple — HH:MM format for times, 3 decimal places for angles. They're designed for debugging and logging, not for user-facing display where callers would want control over formatting.

---

## Package Documentation (`doc.go`)

```bash
cat doc.go
```

```output
// Package dusk provides astronomical calculations: twilight times,
// sunrise/sunset, moonrise/moonset, lunar phase, and celestial
// coordinate conversions.
//
// All angles are in degrees. Time parameters use [time.Time].
// Functions that produce local times accept an [Observer] with a [*time.Location] field.
// Zero-value [time.Time] signals "event did not occur" (e.g., the Moon
// does not rise on a given day); check with [time.Time.IsZero].
//
// Two sentinel errors distinguish polar edge cases:
// [ErrCircumpolar] (object always above the horizon) and
// [ErrNeverRises] (object never rises).
//
// # References
//
//   - Meeus, Jean. Astronomical Algorithms. 2nd ed. Willmann-Bell, 1998.
package dusk
```

The package doc follows Go conventions: a single `doc.go` file with the package-level comment. It covers the three key conventions a caller needs to know (degrees, zero-value times, sentinel errors) and cites the primary reference.

---

## Testing and Quality

The test suite is comprehensive. Let's look at the coverage and test structure.

```bash
go test -count=1 -cover ./... 2>&1 | sed -n 's/.*\(coverage: [^ ]* of statements\).*/\1/p'
```

```output
coverage: 100.0% of statements
```

```bash
grep -c "func Test" *_test.go | grep -v ":0$"
```

```output
coord_test.go:5
epoch_test.go:6
lunar_test.go:7
solar_test.go:11
stringer_test.go:13
transit_test.go:8
trig_test.go:9
twilight_test.go:10
```

```bash
grep -c "func Fuzz" fuzz_test.go
```

```output
4
```

```bash
grep -c "func Benchmark" benchmark_test.go
```

```output
4
```

100% statement coverage across 69 test functions, 4 fuzz tests, and 4 benchmarks. The test suite uses:

- **Table-driven tests** — expected values from USNO, Stellarium, and Meeus
- **Edge-case coverage** — polar observers, equatorial observers, southern hemisphere, circumpolar objects, never-rises objects
- **Fuzz tests** — `SunriseSunset`, `LunarPhase`, `ObjectTransit`, `MoonriseMoonset` with random inputs to catch panics
- **Benchmarks** — `MoonriseMoonset`, `LunarEclipticPosition`, `SunriseSunset`, `ObjectTransit` (all 0 allocations)

### Linting

```bash
cat .golangci.yml
```

```output
version: "2"

linters:
  enable:
    - errcheck
    - govet
    - ineffassign
    - staticcheck
    - unused
    - copyloopvar
    - durationcheck
    - errorlint
    - gocritic
    - misspell
    - nilerr
    - prealloc
    - revive
    - unconvert
    - unparam
    - whitespace

  settings:
    gocritic:
      disabled-checks:
        - captLocal # Meeus convention: T (century), M (anomaly), J (day), etc.
```

```bash
golangci-lint run 2>&1
```

```output
0 issues.
```

The `captLocal` gocritic check is disabled because Meeus algorithms use single-letter uppercase variables by convention (T for Julian century, M for mean anomaly, J for Julian day, etc.). These are universally recognized in astronomical computing and match the textbook formulas, so flagging them as "capitalize local variable" would add noise.

---

## Concerns and Community Standards

### Adherence to Go conventions

- **Package structure**: Single package at the repo root — clean, simple, no unnecessary nesting.
- **Naming**: Exported names follow Go conventions (`SunriseSunset`, `LunarPhase`, `Observer`). Unexported helpers use descriptive names.
- **Error handling**: Sentinel errors with `errors.New`, checked via `errors.Is`. Functions return `(result, error)` pairs consistently.
- **Documentation**: All exported types and functions have godoc comments. `doc.go` provides the package overview.
- **Testing**: Table-driven tests, subtests with `t.Run`, fuzz tests, and benchmarks.
- **Zero dependencies**: No external modules — only the Go standard library.

### Potential concerns

1. **`JulianDate` silent failure** — Dates outside the int64 nanosecond range (~1677–2262) silently return wrong results. `ValidJulianDateRange` exists for callers to check, but `JulianDate` itself doesn't error. This is a deliberate trade-off for API simplicity (the function is called hundreds of times internally), but callers must remember to validate at the boundary.

2. **Moonrise precision** — The minute-by-minute scan gives ±1 minute accuracy. Higher precision would require either a bisection step or interpolation between the two bounding minutes. For most applications this is fine, but scientific users should be aware.

3. **Mixed epoch systems** — `gstToUT` uses a J1900-era algorithm while everything else uses J2000. This works correctly within the library's valid range but could confuse maintainers. The code documents the reason (direct non-iterative conversion).

4. **No `context.Context` support** — `MoonriseMoonset` takes ~0.5s per call. For server applications computing many dates, there's no way to cancel mid-computation. This is a minor concern for a library primarily used in CLI/batch contexts.

5. **Meeus table transcription** — The 120 rows of lunar coefficients are transcribed from the book. While validated against independent sources, any single-digit transcription error would produce subtle inaccuracies that might only show up for specific dates.

Overall, the library follows Go community standards well. The code is clean, well-tested, well-documented, and zero-dependency — it does one thing (astronomical calculations) and does it thoroughly.

