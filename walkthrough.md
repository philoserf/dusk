# Dusk: Astronomical Calculations Library — Code Walkthrough

*2026-03-09T03:42:45Z by Showboat 0.6.1*
<!-- showboat-id: 5522f23e-041b-4abf-91ce-83ae538b0fc6 -->

## Overview

Dusk is a zero-dependency Go library for astronomical calculations. It computes sunrise/sunset, moonrise/moonset, lunar phase, twilight times, and celestial coordinate conversions using algorithms from Jean Meeus's *Astronomical Algorithms* (2nd ed., Willmann-Bell, 1998).

**Key design decisions:**

- All angles in **degrees** — trig helpers convert internally
- Zero-value `time.Time` signals "event did not occur"
- Sentinel errors `ErrCircumpolar` / `ErrNeverRises` for polar edge cases
- Single package, domain-focused files, zero external dependencies

The code has a clear dependency flow. We'll walk through it bottom-up: trig primitives → time/epoch foundations → coordinate conversions → solar calculations → lunar calculations → transit → twilight.

### Project structure

```bash
find . -name "*.go" -not -name "*_test.go" | sort | while read f; do echo "$(wc -l < "$f" | tr -d " ") lines  $f"; done
```

```output
144 lines  ./coord.go
17 lines  ./doc.go
114 lines  ./epoch.go
167 lines  ./lunar_tables.go
257 lines  ./lunar.go
140 lines  ./solar.go
142 lines  ./transit.go
35 lines  ./trig.go
83 lines  ./twilight.go
```

```bash
head -3 go.mod
```

```output
module github.com/philoserf/dusk/v2

go 1.24
```

Zero external dependencies — only `math`, `time`, and `errors` from the standard library. The entire library is ~1,100 lines of source code.

---

## Layer 1: Trig Primitives (`trig.go`)

The foundation. Every astronomical formula in the library works in degrees, but Go's `math` package uses radians. These wrappers eliminate conversion clutter throughout the codebase.

```bash
sed -n "1,19p" trig.go
```

```output
package dusk

import "math"

const (
	degToRad = math.Pi / 180.0
	radToDeg = 180.0 / math.Pi
)

func sinx(deg float64) float64    { return math.Sin(deg * degToRad) }
func cosx(deg float64) float64    { return math.Cos(deg * degToRad) }
func tanx(deg float64) float64    { return math.Tan(deg * degToRad) }
func asinx(x float64) float64     { return radToDeg * math.Asin(x) }
func acosx(x float64) float64     { return radToDeg * math.Acos(x) }
func atan2x(y, x float64) float64 { return radToDeg * math.Atan2(y, x) }

func sincosx(deg float64) (float64, float64) {
	return math.Sincos(deg * degToRad)
}
```

`sincosx` returns both sine and cosine in a single call — used in the lunar position loop where both are needed per term. This is a micro-optimization that reduces function call overhead over 60+ table terms.

The angle normalization helpers keep values in canonical ranges:

```bash
sed -n "21,35p" trig.go
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

`mod360` normalizes angles to [0, 360). `mod24` normalizes hours to [0, 24). Both handle negative values correctly — Go's `math.Mod` preserves the sign of the dividend, so the `if x < 0` branch is necessary.

**Concern: asinx/acosx domain safety.** Neither `asinx` nor `acosx` clamp inputs to [-1, 1]. Floating-point rounding could produce values like 1.0000000000000002, causing `NaN`. The library relies on mathematical proofs that valid Earth coordinates keep inputs in-bounds, but defensive clamping would be safer. [Issue #4](https://github.com/philoserf/dusk/issues/4) tracks this.

---

## Layer 2: Time and Epoch (`epoch.go`)

Converts between Go's `time.Time` and the Julian Date system used in astronomical formulas. This is the bridge between wall-clock time and celestial mechanics.

```bash
sed -n "8,19p" epoch.go
```

```output
const (
	j1970 = 2440587.5
	j2000 = 2451545.0
)

// JulianDate returns the Julian date for a given time, i.e., the continuous
// count of days and fractions of day since the beginning of the Julian period.
// Uses UnixNano internally; valid for dates approximately 1677–2262.
func JulianDate(t time.Time) float64 {
	ms := t.UTC().UnixNano() / 1e6
	return float64(ms)/86400000.0 + j1970
}
```

Two epochs anchor the calculations:
- **J1970** (2440587.5): Julian date at Unix epoch (1970-01-01 00:00 UTC). Used to convert Unix timestamps to Julian dates.
- **J2000** (2451545.0): Julian date at J2000.0 (2000-01-01 12:00 UTC). The reference epoch for modern astronomical calculations.

`JulianDate` converts via milliseconds from `UnixNano()`. The integer division `/ 1e6` truncates sub-millisecond precision — acceptable for astronomical calculations where ±1ms is negligible. The `UnixNano` dependency limits the valid range to ~1677–2262 (int64 nanosecond bounds).

The reverse conversion:

```bash
sed -n "64,73p" epoch.go
```

```output
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

`universalTimeFromJD` reverses the Julian date conversion. The `(jd - j1970) * 86400000 * 1e6` converts fractional Julian days back to nanoseconds since Unix epoch.

### Sidereal Time

Sidereal time measures the Earth's rotation relative to the stars rather than the Sun. A sidereal day is ~23h 56m 4s. This is essential for converting between equatorial coordinates and local observation coordinates.

```bash
sed -n "25,45p" epoch.go
```

```output
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

`greenwichMeanSiderealTime` implements Meeus eq. 12.4 (p. 88). The dominant term `360.98564736629 * (JD - j2000)` captures the Earth's rotation — slightly faster than one revolution per day (360.985... degrees vs 360). Higher-order terms account for precession and nutation.

`LocalSiderealTime` offsets by longitude (east = positive = earlier in sidereal time) and converts from degrees to hours (÷15).

### Nutation and Obliquity

The Earth's axis wobbles. Nutation is the short-period oscillation; obliquity is the tilt angle of the ecliptic.

```bash
sed -n "79,114p" epoch.go
```

```output
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

The mean obliquity (~23.44°) is the tilt of Earth's equatorial plane relative to the ecliptic. It decreases slowly (~0.013° per century). The nutation correction `nutationInObliquity` adds small periodic wobbles (±9.2 arcseconds dominant term) driven by the Moon's ascending node, with the `/ 3600.0` converting arcseconds to degrees.

These helpers feed into `EclipticToEquatorial`, which needs the true obliquity (mean + nutation) to convert coordinates.

---

## Layer 3: Coordinate Systems (`coord.go`)

Astronomy uses three coordinate systems. Dusk converts between them:

- **Ecliptic** (Lon, Lat, Dist): measured along the plane of Earth's orbit. Natural output of orbital mechanics.
- **Equatorial** (RA, Dec): measured relative to the celestial equator. The standard "sky address" system.
- **Horizontal** (Alt, Az): measured from the observer's horizon. What you actually see — "how high" and "which direction."

### Types and Validation

```bash
sed -n "9,56p" coord.go
```

```output
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
	errInvalidEquatorial = errors.New("dusk: RA must be in [0, 360) and Dec in [-90, 90]")
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

// validateEquatorial checks that RA is in [0, 360) and Dec is in [-90, 90].
func validateEquatorial(eq Equatorial) error {
	if eq.RA < 0 || eq.RA >= 360 || eq.Dec < -90 || eq.Dec > 90 {
		return errInvalidEquatorial
	}
	return nil
}
```

The error strategy is clean: exported sentinels (`ErrCircumpolar`, `ErrNeverRises`) for domain-specific conditions that callers may want to match with `errors.Is`, unexported sentinels (`errNilLocation`, `errInvalidCoord`) for programming errors. All prefixed with `dusk:` for clear error provenance.

`Observer` centralizes the observation point — latitude, longitude, elevation, and timezone. Every public function that produces local times accepts an `Observer`.

**Convention:** RA is in degrees [0, 360), not the traditional hours [0, 24). This avoids a unit conversion in formulas but may surprise astronomers who expect RA in hours.

### Ecliptic → Equatorial Conversion

```bash
sed -n "72,92p" coord.go
```

```output
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

This is the Meeus eq. 13.3/13.4 rotation — a coordinate system rotation by the obliquity angle `eps`. The true obliquity (mean + nutation correction) accounts for the current tilt of Earth's axis. The `atan2x` for RA handles quadrant ambiguity correctly; `asinx` for Dec is safe because the formula's output is bounded by [-1, 1] for valid ecliptic coordinates.

This function is called by both `SolarPosition` and `LunarPosition` — it's the shared bridge between orbital mechanics (ecliptic) and sky observation (equatorial).

### Equatorial → Horizontal Conversion

```bash
sed -n "94,123p" coord.go
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
```

This answers "where in the sky is this object right now?" for a specific observer.

The hour angle (`ha`) is the key: it measures how far the object is from the observer's meridian. Local sidereal time minus RA gives the angular distance west of the meridian.

The altitude formula (`asinx(sin(Dec)sin(Lat) + cos(Dec)cos(Lat)cos(ha))`) is standard spherical trigonometry. Note the division-by-zero guard at line 108: when the observer is at a pole or the object is at zenith, `cos(alt) * cos(lat)` approaches zero. The code defensively sets azimuth to 0° rather than producing `NaN`.

The `sin(ha) > 0` test disambiguates azimuth: `acos` returns [0°, 180°], but azimuth is [0°, 360°]. When the hour angle is positive (object is west of meridian), the true azimuth is `360 - az`.

### Angular Separation

```bash
sed -n "131,144p" coord.go
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

This uses the `atan2` form of the Vincenty formula rather than the simpler `acos(dot product)`. The `acos` version loses precision when objects are very close together (near 0°) or nearly opposite (near 180°). The `atan2` version is numerically stable across the full range. Good practice.

---

## Layer 4: Solar Calculations (`solar.go`)

The Sun's position and rise/set times. This is the most-used public API.

```bash
sed -n "75,96p" solar.go
```

```output
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
```

The solar position pipeline:

1. **Mean anomaly** (`M`): where the Sun *would* be if Earth's orbit were circular. ~0.986°/day.
2. **Equation of center** (`C`): correction for orbital eccentricity. The dominant term (1.9148°) accounts for Earth's slightly elliptical orbit.
3. **Ecliptic longitude** (`λ`): true position along the ecliptic. The `+ 180 + 102.9372` shifts from anomaly to longitude (180° because anomaly is measured from perihelion, plus the argument of perihelion ~102.9°).

Note: two mean anomaly functions exist — `solarMeanAnomaly(J)` takes *days* since J2000 (used in sunrise/sunset), while `solarMeanAnomalyFromCentury(T)` takes *centuries* (used in lunar phase where `T` is already computed). Same formula, different time units.

### The Hour Angle — Where Polar Edge Cases Live

```bash
sed -n "105,135p" solar.go
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
```

This is the most subtle function in the library. It determines *when* the Sun crosses a specific altitude threshold.

**The −0.83° correction** (for `depression == 0` only): accounts for atmospheric refraction (~0.566°) bending light around the horizon, plus the Sun's angular semidiameter (~0.266°). The Sun is visually "on the horizon" when its geometric center is actually 0.83° below it. For twilight, the IAU convention uses the geometric depression angle directly — no refraction/semidiameter correction.

**Elevation correction**: `2.076 * sqrt(elev) / 60` — at higher elevations, you can see slightly farther past the geometric horizon. Negative elevations are clamped to 0 via `math.Max`.

**Polar edge cases**: when `cosHA < -1`, the Sun never reaches the threshold (midnight sun / circumpolar). When `cosHA > 1`, it never rises above it (polar night). This is the only function that returns errors in the solar calculation chain.

**Design choice**: `solarHourAngle` is shared between sunrise/sunset and twilight by parameterizing the depression angle. This avoids code duplication but requires understanding the `depression == 0` special case.

### Sunrise/Sunset — The Main Public API

```bash
sed -n "17,56p" solar.go
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
```

The full pipeline in action:

1. `meanSolarTime` → integer Julian day adjusted for longitude
2. `solarMeanAnomaly` → `solarEquationOfCenter` → `solarEclipticLongitude` → `solarDeclination`: Sun's position
3. `solarTransitJD`: Julian date of solar noon (highest point)
4. `solarHourAngle`: how far (in time) sunrise/sunset are from noon
5. `Jtransit ± omega/360`: Julian dates of rise and set
6. `universalTimeFromJD` → `.In(obs.Loc)`: back to wall-clock time

The symmetry is elegant: sunrise is transit minus half-arc, sunset is transit plus half-arc. The `omega/360` converts the hour angle from degrees to fraction of a day.

---

## Layer 5: Lunar Calculations (`lunar.go` + `lunar_tables.go`)

The Moon is far more complex than the Sun. Its orbit has large perturbations from the Sun, Jupiter, and Earth's oblateness. Meeus Chapter 47 uses over 120 periodic terms.

### Meeus Coefficient Tables

```bash
echo "=== Table 47.A: Longitude and Distance ===" && head -18 lunar_tables.go && echo "..." && echo "" && echo "=== Table 47.B: Latitude ===" && sed -n "91,101p" lunar_tables.go && echo "..." && echo "" && echo "Table 47.A entries: $(grep -c "^	{" lunar_tables.go | head -1)" && echo "Table 47.B entries: $(sed -n "/tableLat/,\$p" lunar_tables.go | grep -c "^	{")"
```

```output
=== Table 47.A: Longitude and Distance ===
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

var tableLongDist = [...]lunarLongDistCoeff{
	{0, 0, 1, 0, 6288774, -20905355},
	{2, 0, -1, 0, 1274027, -3699111},
	{2, 0, 0, 0, 658314, -2955968},
	{0, 0, 2, 0, 213618, -569925},
...

=== Table 47.B: Latitude ===
var tableLat = [...]lunarLatCoeff{
	{0, 0, 0, 1, 5128122},
	{0, 0, 1, 1, 280602},
	{0, 0, 1, -1, 277693},
	{2, 0, 0, -1, 173237},

	{2, 0, -1, 1, 55413},
	{2, 0, -1, -1, 46271},
	{2, 0, 0, 1, 32573},
	{0, 0, 2, 1, 17198},

...

Table 47.A entries: 120
Table 47.B entries: 60
```

Each row is a periodic term: the four integers (D, M, M', F) are multipliers for the Moon's mean elongation, Sun's mean anomaly, Moon's mean anomaly, and Moon's argument of latitude. The Σl/Σr/Σb values are amplitudes in units of 0.000001° (longitude/latitude) or 0.001 km (distance).

The largest longitude term (6,288,774 × 10⁻⁶ ≈ 6.29°) is the Moon's equation of center — the dominant effect of its elliptical orbit. The tables are stored as fixed-size arrays (`[...]`) rather than slices, avoiding heap allocation.

### Lunar Ecliptic Position

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

The heart of lunar computation. Key implementation details:

**The E factor**: Earth's orbital eccentricity is decreasing over time. Terms involving the Sun's mean anomaly (M) must be multiplied by E (or E² for terms with |M|=2). This is a Meeus-specific correction that many simpler implementations miss.

**Table iteration**: uses `range` with pointer-to-element (`&tableLongDist[i]`) to avoid copying the struct on each iteration. The `sincosx` call computes both sin and cos of the argument in one operation — used because longitude terms need sin and distance terms need cos of the same argument.

**Output scaling**: Σl and Σb are accumulated in units of 10⁻⁶ degrees, then divided by 1e6. Σr is in units of 0.001 km, divided by 1000. The base lunar distance (385,000.56 km) is the mean Earth-Moon distance.

### Lunar Phase

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

	F := (1 - cosx(d)) / 2
	days := F * lunarMonthDays

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

Phase combines solar and lunar positions:

1. **Elongation** (`d`): angular separation between Moon and Sun. 0° = New Moon, 180° = Full Moon. The `mod360` check distinguishes waxing (Moon east of Sun, 0-180°) from waning (180-360°).
2. **Phase angle** (`PA`): geometric angle at the Moon between Sun and Earth. The 0.1468 correction term accounts for the Moon's and Earth's orbital eccentricities.
3. **Illumination** (`K`): percentage of the Moon's disk that's lit, from the phase angle via `(1 + cos(PA)) / 2`.
4. **Days approximate**: fractional position in the 29.53-day lunation cycle.

The `Waxing` boolean is simply `d < 180` — the first half of elongation is waxing.

### Moonrise/Moonset — The Brute Force Approach

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

Unlike `SunriseSunset` which uses an analytical formula, moonrise/moonset uses brute-force minute-by-minute scanning. Why?

The Moon moves ~13° per day (vs the Sun's ~1°), so its position changes significantly during a single day. Simple analytical formulas assume constant declination, which is a reasonable approximation for the Sun but not the Moon. The scan detects altitude sign changes across the -0.833° horizon depression threshold.

**DST awareness**: the scan runs from local midnight to next local midnight, computing `scanMinutes` from the actual time difference. This correctly handles 23-hour (spring forward) and 25-hour (fall back) days.

**Performance**: each minute requires a full `LunarEclipticPosition` → `EclipticToEquatorial` → `EquatorialToHorizontal` chain, evaluating ~120 periodic terms. This is deliberate — the doc comment warns callers about the cost. The early `break` when both rise and set are found helps.

**Concern**: the ±1 minute precision is acceptable for most uses but could be improved with bisection between detected crossings. [Issue #6](https://github.com/philoserf/dusk/issues/6) proposes benchmarks to quantify the cost.

---

## Layer 6: Object Transit (`transit.go`)

Computes rise/set/transit for arbitrary equatorial coordinates — any star, planet, or deep-sky object. Unlike the solar and lunar functions which compute positions internally, this takes pre-computed coordinates as input.

```bash
sed -n "76,94p" transit.go
```

```output
// objectTransitCheck returns nil if an object at the given declination rises
// and sets at the given latitude, ErrCircumpolar if it never sets, or
// ErrNeverRises if it never rises.
func objectTransitCheck(dec, lat float64) error {
	product := tanx(lat) * tanx(dec)
	if product >= 1 {
		return ErrCircumpolar
	}
	if product <= -1 {
		return ErrNeverRises
	}
	return nil
}

// argOfLSTForTransit returns the argument of LST for transit in degrees:
// acos(-tan(lat) * tan(dec)).
func argOfLSTForTransit(lat, dec float64) float64 {
	return acosx(-tanx(lat) * tanx(dec))
}
```

The circumpolar test: `tan(lat) * tan(dec) >= 1` means the object never dips below the horizon (always visible). `<= -1` means it never rises above it. This guards the subsequent `acos(-tan(lat)*tan(dec))` call, which would produce `NaN` for out-of-domain inputs.

### GST ↔ UT Conversion

```bash
sed -n "102,125p" transit.go
```

```output
// gstToUT converts Greenwich sidereal time (hours) to Universal Time (hours)
// for the given date.
//
// This uses the Duffett-Smith / J1900-epoch algorithm rather than the Meeus
// J2000-based formulas used elsewhere in the library. The two epoch systems
// produce equivalent results; this particular algorithm is retained because it
// maps GST→UT directly without iterative inversion.
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

**Concern: mixed epoch systems.** This function uses the Duffett-Smith J1900-epoch algorithm while the rest of the library uses J2000-based Meeus formulas. The comment explains why — GST→UT inversion is non-trivial with the Meeus formulas and would require iteration. The J1900 algorithm gives a direct closed-form solution.

The magic constant 2415020 is the Julian date of J1900.0 (1899-12-31 12:00 UTC). The `0.997270` factor converts sidereal time intervals to solar time intervals (a sidereal day is shorter). The `January 0` trick (`time.Date(year, 1, 0, ...)`) is Go's way of saying December 31 of the prior year — giving the Julian date at the start of the year.

**Known limitation**: the J1900 epoch loses floating-point precision for dates far from 1900. For the library's valid range (~1677–2262), this is acceptable.

### The Full Transit Pipeline

```bash
sed -n "17,74p" transit.go
```

```output
// ObjectTransit computes rise, set, and transit maximum for a celestial
// object at the given observer position on the specified date. Only the
// calendar date (year/month/day in UTC) is used; the time-of-day is ignored.
//
// Observer elevation (obs.Elev) is not used; it only affects sunrise/sunset
// and twilight calculations. Transit maximum precision is ±1 minute.
//
// Returns an error if obs.Loc is nil or coordinates are out of range.
// Returns ErrCircumpolar if the object never sets, or ErrNeverRises if
// it never rises at the given latitude.
func ObjectTransit(date time.Time, eq Equatorial, obs Observer) (Transit, error) {
	if err := validateObserver(obs); err != nil {
		return Transit{}, err
	}
	if err := validateEquatorial(eq); err != nil {
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

	// Find transit maximum by minute-by-minute scan from rise to set.
	maximum := findTransitMaximum(rise, set, obs, eq)

	return Transit{
		Rise:     rise,
		Set:      set,
		Maximum:  maximum,
		Duration: set.Sub(rise),
	}, nil
}
```

The transit pipeline converts between four time systems:
1. **LST** (Local Sidereal Time): when the object rises/sets for *this* observer
2. **GST** (Greenwich Sidereal Time): LST adjusted to Greenwich
3. **UT** (Universal Time): GST converted to solar time
4. **Local time**: UT in the observer's timezone

The `set.Before(rise)` check handles objects that set after midnight UTC — common for evening observations.

`findTransitMaximum` uses the same minute-by-minute scan approach as moonrise/moonset, finding the moment of peak altitude. This gives ±1 minute precision.

---

## Layer 7: Twilight (`twilight.go`)

The simplest domain file — it reuses the entire solar pipeline with different depression angles.

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
	J := meanSolarTime(date, obs.Lon)
	M := solarMeanAnomaly(J)
	C := solarEquationOfCenter(M)
	lambda := solarEclipticLongitude(M, C)
	T := julianCentury(date)
	delta := solarDeclination(lambda, T)
	omega, err := solarHourAngle(delta, depression, obs.Lat, obs.Elev)
	if err != nil {
		return TwilightEvent{}, err
	}
	h := omega / 360
	jTransit := solarTransitJD(J, M, lambda)

	// Today's "set" at this depression = twilight dusk.
	dusk := universalTimeFromJD(jTransit + h).In(obs.Loc)

	// Tomorrow's "rise" at this depression = twilight dawn.
	tomorrow := date.AddDate(0, 0, 1)
	J2 := meanSolarTime(tomorrow, obs.Lon)
	M2 := solarMeanAnomaly(J2)
	C2 := solarEquationOfCenter(M2)
	lambda2 := solarEclipticLongitude(M2, C2)
	T2 := julianCentury(tomorrow)
	delta2 := solarDeclination(lambda2, T2)
	omega2, err2 := solarHourAngle(delta2, depression, obs.Lat, obs.Elev)
	if err2 != nil {
		return TwilightEvent{}, err2
	}
	h2 := omega2 / 360
	jTransit2 := solarTransitJD(J2, M2, lambda2)

	dawn := universalTimeFromJD(jTransit2 - h2).In(obs.Loc)

	return TwilightEvent{
		Dusk:     dusk,
		Dawn:     dawn,
		Duration: dawn.Sub(dusk),
	}, nil
}
```

Three depression angles define the three standard twilight types:
- **Civil** (6°): enough light for outdoor activities without artificial lighting
- **Nautical** (12°): horizon line no longer visible at sea
- **Astronomical** (18°): sky dark enough for faint astronomical observations

The `twilight` function runs the solar pipeline twice — once for today's dusk (transit + hour angle) and once for tomorrow's dawn (transit − hour angle). The `Duration` is the overnight dark period between them.

**Design note**: the dusk/dawn pairing is evening-centric. A call for March 8 returns tonight's dusk and tomorrow morning's dawn. To get *this morning's* dawn, you'd call with yesterday's date. This is well-documented in the `TwilightEvent` type.

**Concern: code duplication.** The solar pipeline (mean anomaly → equation of center → ecliptic longitude → declination → hour angle) is repeated inline here rather than being extracted into a helper. This is a minor readability issue — the repeated block is 8 lines, and extracting it would add abstraction for only two call sites.

---

## Summary of Concerns

| # | Concern | Severity | Reference |
|---|---------|----------|-----------|
| 1 | `asinx`/`acosx` lack domain clamping for float rounding | Medium | [Issue #4](https://github.com/philoserf/dusk/issues/4) |
| 2 | No fuzz tests for unusual input combinations | Medium | [Issue #5](https://github.com/philoserf/dusk/issues/5) |
| 3 | No benchmarks for performance-critical paths | Medium | [Issue #6](https://github.com/philoserf/dusk/issues/6) |
| 4 | Mixed J1900/J2000 epoch systems in `gstToUT` | Low | Documented in code |
| 5 | Twilight duplicates solar pipeline inline | Low | 2 call sites only |
| 6 | `MoonriseMoonset` ±1 min precision, no bisection refinement | Low | Documented in doc comment |
| 7 | RA in degrees, not hours — potential surprise for astronomers | Low | Consistent within library |

## Community Standards Adherence

| Standard | Status |
|----------|--------|
| Go module with semantic versioning (v2) | ✓ |
| Package doc comment (`doc.go`) | ✓ |
| All exported symbols documented | ✓ |
| Sentinel errors follow Go conventions | ✓ |
| Input validation at public API boundary | ✓ |
| Zero external dependencies | ✓ |
| Table-driven tests | ✓ |
| CI with vet + lint + race detector | ✓ |
| License file (GPL-3.0) | ✓ |
| No `init()` functions | ✓ |
| No global mutable state | ✓ |
| Unexported helpers keep public API clean | ✓ |

The library follows Go community standards closely. The code is clean, well-documented, and idiomatically structured.

