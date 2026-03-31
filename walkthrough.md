# Dusk Walkthrough

*2026-03-30T23:51:30Z by Showboat 0.6.1*
<!-- showboat-id: 0a2a474d-87b0-459b-b921-b32fc7805cf4 -->

## Overview

Dusk is a zero-dependency Go library for astronomical calculations: sunrise/sunset, moonrise/moonset, twilight times, and lunar phase. All algorithms derive from Jean Meeus's *Astronomical Algorithms* (2nd ed., 1998).

Install: `go get github.com/philoserf/dusk/v3`

The library is a single package with five source files totaling ~1,080 lines. No external dependencies — only the Go standard library (`math`, `time`, `errors`, `fmt`). All angles are in degrees; time values use `time.Time`. Zero-value `time.Time` in result structs signals "event did not occur" for a given day.

```bash
cat <<'HEREDOC'
github.com/philoserf/dusk/v3

dusk.go    177 lines  Package doc, Observer, event types, sentinel errors, Stringers
solar.go   211 lines  SunriseSunset, twilight, unexported solar helpers
lunar.go   437 lines  MoonriseMoonset, LunarPhase, Meeus Table 47.A/B coefficients
epoch.go   218 lines  Julian dates, sidereal time, nutation, obliquity, coordinate conversions
trig.go     40 lines  Degree-based trig wrappers, clamp, mod360/mod24
           ----
          1083 lines  total
HEREDOC
```

```output
github.com/philoserf/dusk/v3

dusk.go    177 lines  Package doc, Observer, event types, sentinel errors, Stringers
solar.go   211 lines  SunriseSunset, twilight, unexported solar helpers
lunar.go   437 lines  MoonriseMoonset, LunarPhase, Meeus Table 47.A/B coefficients
epoch.go   218 lines  Julian dates, sidereal time, nutation, obliquity, coordinate conversions
trig.go     40 lines  Degree-based trig wrappers, clamp, mod360/mod24
           ----
          1083 lines  total
```

## Types and Construction

The public API centers on `Observer` — a validated geographic position constructed via `NewObserver`. Fields are unexported so invalid observers cannot be created by struct literal. Three event types (`SunEvent`, `MoonEvent`, `TwilightEvent`) and one phase type (`LunarPhaseInfo`) hold calculation results. Two sentinel errors handle polar edge cases.

```bash
sed -n '28,41p' dusk.go
```

```output
// ErrCircumpolar is returned when a celestial object is circumpolar
// (always above the horizon) at the given latitude.
var ErrCircumpolar = errors.New("dusk: object is circumpolar (always above the horizon)")

// ErrNeverRises is returned when a celestial object never rises above
// the horizon at the given latitude.
var ErrNeverRises = errors.New("dusk: object never rises at this latitude")

var errNilLocation = errors.New("dusk: location must not be nil")

var errNonFiniteCoord = errors.New("dusk: coordinates must be finite (NaN and Inf are not allowed)")

var errInvalidCoord = errors.New("dusk: latitude must be in [-90, 90] and longitude in [-180, 180]")

```

Two sentinel errors are exported: `ErrCircumpolar` (midnight sun — object never sets) and `ErrNeverRises` (polar night — object never rises). Three unexported errors handle invalid input: nil location, non-finite coordinates, and out-of-range lat/lon. This validates once at construction time so every downstream function can trust the observer.

```bash
sed -n '51,73p' dusk.go
```

```output
// Observer represents a geographic position on Earth used as the viewpoint
// for all astronomical calculations.
type Observer struct {
	lat float64
	lon float64
	loc *time.Location
}

// NewObserver constructs an Observer after validating all inputs.
// lat must be in [-90, 90], lon in [-180, 180], and loc must not be nil.
// NaN and infinite values are rejected.
func NewObserver(lat, lon float64, loc *time.Location) (Observer, error) {
	if loc == nil {
		return Observer{}, errNilLocation
	}
	if math.IsNaN(lat) || math.IsInf(lat, 0) || math.IsNaN(lon) || math.IsInf(lon, 0) {
		return Observer{}, errNonFiniteCoord
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return Observer{}, errInvalidCoord
	}
	return Observer{lat: lat, lon: lon, loc: loc}, nil
}
```

Validation is strict: nil location, NaN/Inf, and out-of-range coordinates all fail. The `validObserver` guard (line 44) checks for nil `loc`, catching zero-value observers that bypass `NewObserver`.

```bash
sed -n '76,118p' dusk.go
```

```output
// of daylight for a single day.
type SunEvent struct {
	Rise     time.Time
	Noon     time.Time
	Set      time.Time
	Duration time.Duration
}

// MoonEvent holds the rise and set times for the Moon on a given day, along
// with the duration between rise and set.
type MoonEvent struct {
	Rise         time.Time     // zero value if the Moon does not rise
	Set          time.Time     // zero value if the Moon does not set
	Duration     time.Duration // zero if Rise or Set is missing
	AboveHorizon bool          // true if Moon was above the horizon at start of day
}

// TwilightEvent holds the dusk and dawn times of a twilight period.
// Dusk is tonight's boundary (sun passes below the depression angle).
// Dawn is tomorrow morning's boundary (sun passes above the depression angle).
// To get this morning's dawn, call with yesterday's date.
type TwilightEvent struct {
	Dusk          time.Time     // evening boundary (today)
	Dawn          time.Time     // morning boundary (tomorrow)
	NightDuration time.Duration // time from Dusk to Dawn (overnight darkness)
}

// LunarPhaseInfo describes the Moon's current phase.
type LunarPhaseInfo struct {
	Illumination float64 // percentage 0-100
	Elongation   float64 // degrees 0-360
	Angle        float64 // phase angle in degrees (may be negative per Meeus formula)
	DaysApprox   float64 // rough days into lunation (linear estimate from elongation)
	Waxing       bool    // true from New Moon to Full Moon (elongation 0-180)
	Name         string  // "New Moon", "Waxing Crescent", etc.
}

// equatorial represents right ascension and declination in degrees.
// Used internally for coordinate conversions.
type equatorial struct {
	ra  float64
	dec float64
}
```

Four result types, each with a `String()` method. Key design choices: `MoonEvent.AboveHorizon` tracks whether the Moon was up at midnight (needed because moonrise/moonset don't always bracket a day neatly). `TwilightEvent` spans overnight — Dusk is today's evening, Dawn is tomorrow morning. `LunarPhaseInfo.DaysApprox` is a linear estimate from elongation, not a precise lunation clock.

Three unexported coordinate types (`equatorial`, `horizontal`, `ecliptic`) flow through the internal pipeline.

## Foundation: Trig and Epoch

Two files form the foundation everything else builds on. `trig.go` wraps Go's radian-based `math` functions to work in degrees. `epoch.go` handles Julian dates, sidereal time, nutation, obliquity, and coordinate conversions.

```bash
sed -n '1,40p' trig.go
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

The entire file is 40 lines. Every trig function takes and returns degrees — the `x` suffix is the naming convention (`sinx`, `cosx`, etc.). `clamp` guards `asinx`/`acosx` against floating-point values slightly outside [-1, 1] that would produce NaN. `sincosx` is an optimization for the lunar tables where both sin and cos of the same angle are needed. `mod360` and `mod24` normalize angles and hours to their canonical ranges.

```bash
sed -n '9,43p' epoch.go
```

```output
const (
	j1970 = 2440587.5
	j2000 = 2451545.0
)

// julianDate returns the Julian date for a given time, i.e., the continuous
// count of days and fractions of day since the beginning of the Julian period.
//
// Uses UnixNano internally, which limits the valid range to the int64
// nanosecond bounds (approximately 1677-09-21 to 2262-04-11). Dates outside
// this range silently produce incorrect results because UnixNano returns 0.
// Use [validJulianDateRange] to check before calling.
func julianDate(t time.Time) float64 {
	ms := t.UTC().UnixNano() / 1e6
	return float64(ms)/86400000.0 + j1970
}

// errDateOutOfRange is returned when a date falls outside the valid range
// (the int64 nanosecond bounds, approximately 1677-09-21 to 2262-04-11).
var errDateOutOfRange = errors.New("dusk: date outside valid range (1677-09-21 to 2262-04-11)")

// julianDateMin and julianDateMax are the bounds of the int64 UnixNano range.
var (
	julianDateMin = time.Unix(0, math.MinInt64).UTC()
	julianDateMax = time.Unix(0, math.MaxInt64).UTC()
)

// validJulianDateRange reports whether t falls within the valid range for
// [julianDate]. Returns nil if valid, [errDateOutOfRange] otherwise.
func validJulianDateRange(t time.Time) error {
	if t.Before(julianDateMin) || t.After(julianDateMax) {
		return errDateOutOfRange
	}
	return nil
}
```

Two epoch constants anchor everything: `j1970` (Unix epoch as Julian date) and `j2000` (the J2000.0 standard epoch used by Meeus). `julianDate` converts via `UnixNano`, which limits valid dates to roughly 1677–2262. The `validJulianDateRange` guard runs at the top of every public function.

```bash
sed -n '99,175p' epoch.go
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

// nutationInLongitude returns Δψ (nutation in longitude) in degrees.
//
// See Meeus p. 144.
func nutationInLongitude(L, l, omega float64) float64 {
	return (-17.20*sinx(omega) - 1.32*sinx(2*L) - 0.23*sinx(2*l) + 0.21*sinx(2*omega)) / 3600.0
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

// ---------------------------------------------------------------------------
// Coordinate conversions (moved from coord.go)
// ---------------------------------------------------------------------------

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
```

`eclipticToEquatorial` is the key coordinate conversion — used by both solar and lunar pipelines. It applies full nutation correction (Δψ to longitude, Δε to obliquity) per Meeus ch. 13. Note the intentional asymmetry: `solarDeclination` in solar.go uses mean obliquity only (the NOAA simplified method for sunrise/sunset), while this function uses true obliquity. The lunar pipeline always flows through `eclipticToEquatorial` for higher precision.

`equatorialToHorizontal` (lines 181–206) completes the chain: equatorial → altitude/azimuth for a specific observer. This is what the moonrise scanner checks each minute.

## Solar Calculations

The solar pipeline follows the NOAA method (derived from Meeus). The flow is: `computeSolarParams` → `solarHourAngle` → rise/set Julian dates → `time.Time` in the observer's timezone. Twilight reuses the same pipeline with a different depression angle.

```bash
sed -n '7,26p' solar.go
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

The six-step pipeline: mean solar time → mean anomaly → equation of center → ecliptic longitude → declination → transit JD. These two outputs (declination and transit time) are all that `SunriseSunset` and `twilight` need to compute rise/set.

```bash
sed -n '37,65p' solar.go
```

```output
func SunriseSunset(date time.Time, obs Observer) (SunEvent, error) {
	if err := validObserver(obs); err != nil {
		return SunEvent{}, err
	}
	if err := validJulianDateRange(date); err != nil {
		return SunEvent{}, err
	}

	sp := computeSolarParams(date, obs.lon)

	omega, err := solarHourAngle(sp.delta, 0, obs.lat)
	if err != nil {
		return SunEvent{}, err
	}

	Jrise := sp.jTransit - omega/360.0
	Jset := sp.jTransit + omega/360.0

	rise := universalTimeFromJD(Jrise).In(obs.loc)
	noon := universalTimeFromJD(sp.jTransit).In(obs.loc)
	set := universalTimeFromJD(Jset).In(obs.loc)

	return SunEvent{
		Rise:     rise,
		Noon:     noon,
		Set:      set,
		Duration: set.Sub(rise),
	}, nil
}
```

The core formula: sunrise = transit - hourAngle/360, sunset = transit + hourAngle/360. The hour angle is symmetric around solar noon. `solarHourAngle` with depression=0 applies the standard -0.83 degree correction (atmospheric refraction + solar semidiameter).

```bash
sed -n '126,143p' solar.go
```

```output
func solarHourAngle(delta, depression, lat float64) (float64, error) {
	var h0 float64
	if depression == 0 {
		h0 = -0.83
	} else {
		h0 = -depression
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

This is the polar edge-case gate. When `cosHA < -1`, the sun never dips below the threshold — circumpolar (midnight sun). When `cosHA > 1`, it never rises above it — polar night. The depression parameter enables twilight reuse: pass 6 for civil, 12 for nautical, 18 for astronomical.

```bash
sed -n '154,172p' solar.go
```

```output
// CivilTwilight computes the evening civil twilight period (Sun 6 degrees below the
// horizon) for the given date and observer position. Dusk is tonight's civil
// dusk; Dawn is tomorrow morning's civil dawn.
func CivilTwilight(date time.Time, obs Observer) (TwilightEvent, error) {
	return twilight(date, obs, 6)
}

// NauticalTwilight computes the evening nautical twilight period (Sun 12
// degrees below the horizon) for the given date and observer position.
func NauticalTwilight(date time.Time, obs Observer) (TwilightEvent, error) {
	return twilight(date, obs, 12)
}

// AstronomicalTwilight computes the evening astronomical twilight period (Sun
// 18 degrees below the horizon) for the given date and observer position.
func AstronomicalTwilight(date time.Time, obs Observer) (TwilightEvent, error) {
	return twilight(date, obs, 18)
}

```

Three thin wrappers over a single `twilight` function, differing only in the depression angle. The unexported `twilight` (lines 178–211) computes today's dusk and tomorrow's dawn by running the solar pipeline twice — once for today, once for tomorrow — producing the overnight span.

## Lunar Calculations

The lunar code is the largest file (437 lines), dominated by the Meeus Table 47.A/B coefficient data. Two public functions: `LunarPhase` for illumination/phase name, and `MoonriseMoonset` for rise/set times via minute-by-minute scanning.

```bash
sed -n '96,132p' lunar.go
```

```output
	return ecliptic{
		lon:  mod360(Lp + Sl/1e6),
		lat:  Sb / 1e6,
		dist: 385000.56 + Sr/1000,
	}
}

// LunarPhase returns the lunar phase at the given instant.
//
// Unlike SunriseSunset and MoonriseMoonset which use only the calendar date,
// LunarPhase uses the exact time — the phase changes continuously.
//
// The phase angle uses the Meeus approach: solar ecliptic longitude from the
// mean-anomaly method, lunar ecliptic position from Chapter 47 tables.
//
// An error is returned if the date is out of the valid Julian date range.
func LunarPhase(date time.Time) (LunarPhaseInfo, error) {
	if err := validJulianDateRange(date); err != nil {
		return LunarPhaseInfo{}, err
	}

	ec := lunarEclipticPosition(date)

	J := julianDate(date) - j2000
	Msol := solarMeanAnomaly(J)
	C := solarEquationOfCenter(Msol)
	sunLon := solarEclipticLongitude(Msol, C)

	T := julianCentury(date)
	Mp := lunarMeanAnomaly(T)

	// elongation (0-360°, waxing = 0-180, waning = 180-360)
	d := acosx(cosx(ec.lon-sunLon) * cosx(ec.lat))
	if mod360(ec.lon-sunLon) > 180 {
		d = 360 - d
	}

```

`LunarPhase` combines the solar and lunar positions to compute elongation (the Moon-Sun angular separation). The elongation drives everything: illumination via the phase angle (Meeus p. 346), waxing/waning from the 0–180/180–360 split, phase name from 45-degree bins, and days-into-lunation as a linear proportion of the 29.53-day synodic month. Note that `LunarPhase` takes no `Observer` — phase is the same for everyone on Earth.

```bash
sed -n '157,214p' lunar.go
```

```output
// MoonriseMoonset computes the moonrise and moonset times for the given date
// at the specified observer position and timezone.
// The date is converted to the observer's timezone to determine the local
// calendar day, then the function scans that local day (midnight to midnight)
// for rise/set events. This means the same time.Time can produce different
// results for observers in different timezones.
//
// The algorithm scans minute-by-minute through the day to detect altitude
// sign changes. This is slow by design (~1440 ecliptic-position evaluations).
// A single call takes approximately 1-2 ms on modern hardware (Apple M-series
// or equivalent; see BenchmarkMoonriseMoonset). Callers computing
// moonrise/moonset for many dates (e.g., a 30-day calendar ≈ 30-60 ms)
// should expect proportional cost and may benefit from caching or
// parallelization.
//
// An error is returned if the date is out of the valid Julian date range.
func MoonriseMoonset(date time.Time, obs Observer) (MoonEvent, error) {
	if err := validObserver(obs); err != nil {
		return MoonEvent{}, err
	}
	if err := validJulianDateRange(date); err != nil {
		return MoonEvent{}, err
	}

	localDate := date.In(obs.loc)
	d := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, obs.loc).UTC()
	nextMidnight := time.Date(localDate.Year(), localDate.Month(), localDate.Day()+1, 0, 0, 0, 0, obs.loc).UTC()
	if err := validJulianDateRange(d); err != nil {
		return MoonEvent{}, err
	}
	if err := validJulianDateRange(nextMidnight); err != nil {
		return MoonEvent{}, err
	}
	scanMinutes := int(nextMidnight.Sub(d).Minutes())

	var rise, set time.Time

	ec0 := lunarEclipticPosition(d)
	eq0 := eclipticToEquatorial(d, ec0.lon, ec0.lat)
	prevAlt := equatorialToHorizontal(d, obs, eq0).alt
	aboveAtStart := prevAlt > -lunarHorizonDepression

	for i := 1; i <= scanMinutes; i++ {
		cur := d.Add(time.Duration(i) * time.Minute)

		ec := lunarEclipticPosition(cur)
		eq := eclipticToEquatorial(cur, ec.lon, ec.lat)
		hz := equatorialToHorizontal(cur, obs, eq)

		if rise.IsZero() && hz.alt > -lunarHorizonDepression && prevAlt <= -lunarHorizonDepression {
			rise = cur.In(obs.loc)
		}
		if set.IsZero() && hz.alt < -lunarHorizonDepression && prevAlt >= -lunarHorizonDepression {
			set = cur.In(obs.loc)
		}
		if !rise.IsZero() && !set.IsZero() {
			break
		}
```

Unlike the Sun (which has a clean hour-angle formula), the Moon moves fast enough that its position changes significantly during a single day. So `MoonriseMoonset` brute-forces it: scan every minute from local midnight to local midnight, computing the full ecliptic → equatorial → horizontal pipeline each time. That is ~1,440 iterations per call.

Key details: the scan uses `scanMinutes` derived from the actual midnight-to-midnight span (handles DST transitions where a day is 23 or 25 hours). It records the Moon's altitude at midnight to set `AboveHorizon`. The early-break on line 197 exits as soon as both rise and set are found. The 0.833-degree depression threshold (`lunarHorizonDepression`) accounts for refraction and the Moon's semidiameter.

```bash
sed -n '22,85p' lunar.go
```

```output
		switch r.M {
		case 0, 1, -1, 2, -2:
		default:
			panic(fmt.Sprintf("dusk: tableLat[%d] has unexpected M value %v", i, r.M))
		}
	}
}

// lunarHorizonDepression accounts for atmospheric refraction (~0.566°) and
// the Moon's mean semidiameter (~0.25°) when detecting moonrise/moonset.
const lunarHorizonDepression = 0.833

// ---------------------------------------------------------------------------
// Exported functions
// ---------------------------------------------------------------------------

// lunarEclipticPosition returns the geocentric ecliptic longitude, latitude
// (degrees), and distance (km) of the Moon for the given instant.
//
// The algorithm is from Meeus, Astronomical Algorithms, Chapter 47.
func lunarEclipticPosition(t time.Time) ecliptic {
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

```

This is the computational heart of the library — Meeus Chapter 47. Five fundamental arguments (D, Lp, M, Mp, F) plus three additive corrections (A1, A2, A3) feed into two summation loops over the coefficient tables. The `E` factor corrects for the eccentricity of Earth's orbit — terms involving the Sun's mean anomaly (M=±1) are scaled by E, and M=±2 terms by E². The `sincosx` optimization computes both sin and cos in one call since Table 47.A needs both for longitude (Σl) and distance (Σr).

The two tables (`tableLongDist` with 60 rows, `tableLat` with 60 rows) account for most of the file's 437 lines. The coefficients are in units of 0.000001 degrees — divided by 1e6 at the end.

## Stringers

All four result types implement `fmt.Stringer`. A shared `formatTime` helper renders `time.Time` as `HH:MM` or `--:--` for zero values.

```bash
sed -n '136,177p' dusk.go
```

```output
// formatTime formats a time as "HH:MM", or "--:--" for the zero value.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "--:--"
	}
	return t.Format("15:04")
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
	return fmt.Sprintf("Rise=%s Set=%s Duration=%s AboveHorizon=%v",
		formatTime(m.Rise),
		formatTime(m.Set),
		m.Duration,
		m.AboveHorizon)
}

// String returns a human-readable representation of the lunar phase.
func (l LunarPhaseInfo) String() string {
	namePart := l.Name
	if namePart != "" {
		namePart += " "
	}
	return fmt.Sprintf("%s%.1f%% (day %.1f)", namePart, l.Illumination, l.DaysApprox)
}

// String returns a human-readable representation of the twilight event.
func (tw TwilightEvent) String() string {
	return fmt.Sprintf("Dusk=%s Dawn=%s NightDuration=%s",
		formatTime(tw.Dusk),
		formatTime(tw.Dawn),
		tw.NightDuration)
}
```

## Concerns

**Intentional trade-offs (documented, not bugs):**

- **Moonrise performance:** The minute-by-minute scan (~1,440 ecliptic position evaluations per call) is slow by design. Documented in the function comment with benchmark guidance. A polynomial interpolation approach would be faster but harder to verify against reference data.
- **Nutation asymmetry:** `solarDeclination` uses mean obliquity (NOAA simplified method), while `eclipticToEquatorial` applies full nutation. This is intentional — sunrise/sunset accuracy is dominated by refraction uncertainty, not nutation.
- **DaysApprox linearity:** The days-into-lunation estimate is a linear scaling of elongation. Actual lunation is non-uniform, so this is approximate by design.

**Observations:**

- **Table validation in `init()`:** The Meeus coefficient tables are validated at package load time to ensure all M values are in the expected set {0, ±1, ±2}. This replaces runtime panics that were previously in the computation loop, keeping the library panic-free while still catching table corruption early.
- **No `context.Context`:** Public functions do I/O-free computation, so the absence of context parameters is reasonable. If the moonrise scanner were ever made interruptible, context would be the mechanism.

