# Dusk Walkthrough

*2026-03-31T00:38:13Z by Showboat 0.6.1*
<!-- showboat-id: f8ca5a7c-29d4-4384-90dd-67a538a975c7 -->

## Overview

Dusk is a zero-dependency Go library for astronomical calculations: sunrise/sunset, civil/nautical/astronomical twilight, moonrise/moonset, and lunar phase. All algorithms derive from Jean Meeus's *Astronomical Algorithms* (2nd ed., 1998).

The library is a single package at module path `github.com/philoserf/dusk/v3`. Five source files, no external dependencies, ~650 lines of computation plus ~160 lines of Meeus coefficient tables.

### Public API surface

| Function               | Returns           | Purpose                              |
|------------------------|-------------------|--------------------------------------|
| `NewObserver`          | `Observer, error` | Construct a validated observer       |
| `SunriseSunset`        | `SunEvent, error` | Rise, noon, set, duration for a date |
| `CivilTwilight`        | `TwilightEvent, error` | 6° depression boundary          |
| `NauticalTwilight`     | `TwilightEvent, error` | 12° depression boundary         |
| `AstronomicalTwilight` | `TwilightEvent, error` | 18° depression boundary         |
| `MoonriseMoonset`      | `MoonEvent, error`| Rise/set via minute-by-minute scan   |
| `LunarPhase`           | `LunarPhaseInfo, error` | Phase at an exact instant      |

### File layout

```bash
wc -l dusk.go solar.go lunar.go epoch.go trig.go
```

```output
     204 dusk.go
     221 solar.go
     433 lunar.go
     221 epoch.go
      42 trig.go
    1121 total
```

| File       | Lines | Domain                                                   |
|------------|-------|----------------------------------------------------------|
| `trig.go`  | 42    | Degree-based trig wrappers, angle normalization          |
| `epoch.go` | 221   | Julian dates, sidereal time, nutation, coordinate xforms |
| `dusk.go`  | 204   | Types, Observer, sentinel errors, stringers              |
| `solar.go` | 221   | Sunrise/sunset, twilight, solar position helpers         |
| `lunar.go` | 433   | Moonrise/moonset, lunar phase, Meeus Table 47 coefficients |

The walkthrough follows the dependency chain bottom-up: `trig.go` → `epoch.go` → `dusk.go` → `solar.go` → `lunar.go`.

---

## Foundation: `trig.go`

All angles in dusk are in degrees. The trig helpers convert to radians internally so call sites read naturally (e.g., `sinx(delta)` instead of `math.Sin(delta * math.Pi / 180)`).

```bash
sed -n '1,42p' trig.go
```

```output
package dusk

import "math"

const (
	degToRad = math.Pi / 180.0
	radToDeg = 180.0 / math.Pi
)

// clamp restricts x to [-1, 1] before passing it to asin/acos. This prevents
// NaN from floating-point rounding in trig chains. Note: it also silently
// clamps genuinely wrong values (e.g., a miscalculated 1.3 → 1), which could
// mask upstream bugs. Correctness is validated by test coverage against Meeus
// and USNO reference data rather than runtime detection.
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

Key design choices:

- **`clamp`** guards `asinx`/`acosx` against NaN from floating-point rounding. The comment honestly notes this also silently clamps genuinely wrong values — correctness relies on test coverage against reference data, not runtime detection.
- **`sincosx`** wraps `math.Sincos` for the lunar position inner loop where both sin and cos of the same argument are needed (avoids computing the angle twice).
- **`mod360`/`mod24`** normalize angles and hours to their canonical range. The `if x < 0` branch handles Go's `math.Mod` returning negative remainders for negative inputs.

---

## Time Infrastructure: `epoch.go`

This file handles all time conversions: Julian dates, sidereal time, nutation, obliquity, and coordinate transformations (ecliptic ↔ equatorial ↔ horizontal).

### Julian date system

```bash
sed -n '1,43p' epoch.go
```

```output
package dusk

import (
	"math"
	"time"
)

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

// ErrDateOutOfRange is returned when a date falls outside the valid range
// for Julian date calculations (the int64 nanosecond bounds, approximately
// 1677-09-21 to 2262-04-11).
const ErrDateOutOfRange = errString("dusk: date outside valid range (1677-09-21 to 2262-04-11)")

// julianDateMin and julianDateMax are the bounds of the int64 UnixNano range.
var (
	julianDateMin = time.Unix(0, math.MinInt64).UTC()
	julianDateMax = time.Unix(0, math.MaxInt64).UTC()
)

// validJulianDateRange reports whether t falls within the valid range for
// [julianDate]. Returns nil if valid, [ErrDateOutOfRange] otherwise.
func validJulianDateRange(t time.Time) error {
	if t.Before(julianDateMin) || t.After(julianDateMax) {
		return ErrDateOutOfRange
	}
	return nil
}
```

The Julian date conversion anchors on `j1970` (Unix epoch in Julian days) and uses `UnixNano` for sub-second precision. The `j2000` constant (noon on 2000-01-01) is the standard astronomical epoch — most Meeus formulas measure time relative to it.

The valid range is bounded by Go's `int64` nanosecond representation (~1677–2262). Every public function calls `validJulianDateRange` before computing.

### Sidereal time and time helpers

```bash
sed -n '45,100p' epoch.go
```

```output
// greenwichMeanSiderealTime returns the mean sidereal time at Greenwich in
// degrees for the given instant.
//
// See Meeus, Astronomical Algorithms, eq. 12.4 p. 88.
func greenwichMeanSiderealTime(t time.Time) float64 {
	// T is computed from midnight UTC, not from t. This matches Meeus's
	// formulation: the polynomial terms use 0h UT for the date, while the
	// linear term (360.985… × (JD − J2000)) uses the full Julian date to
	// account for the fractional day. Do not "simplify" by passing t here.
	d := datetimeZeroHour(t)
	T := julianCentury(d)
	JD := julianDate(t)

	theta := 280.46061837 +
		360.98564736629*(JD-j2000) +
		0.000387933*T*T -
		T*T*T/38710000.0

	return mod360(theta)
}

// localSiderealTime returns the local sidereal time in hours for a given
// instant and observer longitude (east positive, west negative, in degrees).
func localSiderealTime(t time.Time, longitude float64) float64 {
	gst := greenwichMeanSiderealTime(t) // degrees
	lst := gst + longitude              // degrees
	return mod24(lst / 15.0)
}

// julianCentury returns the number of Julian centuries elapsed since J2000.0.
func julianCentury(t time.Time) float64 {
	return (julianDate(t) - j2000) / 36525.0
}

// julianDay returns the number of days since J2000.0, rounded to the nearest
// integer (used for mean solar time).
func julianDay(t time.Time) int {
	JD := julianDate(t)
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

Notable detail: `greenwichMeanSiderealTime` computes `T` from midnight UTC, not from `t` itself. The comment warns against "simplifying" this — Meeus's formula requires the polynomial terms to use 0h UT while the linear term uses the full JD. This is a subtle correctness constraint.

`localSiderealTime` returns **hours** (0–24) while GMST works in **degrees** — the `/15.0` conversion happens at the boundary. This mixed-unit pattern recurs in `hourAngle`.

### Nutation, obliquity, and coordinate conversions

```bash
sed -n '102,178p' epoch.go
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

`eclipticToEquatorial` applies **full nutation** — both Δψ (longitude) and Δε (obliquity). This is the v3 improvement over v2. The nutation correction shifts the ecliptic longitude by Δψ before converting to RA/Dec, and uses nutation-corrected obliquity (mean + Δε).

**Intentional asymmetry**: `solarDeclination` in `solar.go` uses only mean obliquity (no nutation). This matches the NOAA simplified method for sunrise/sunset, where the ~17" nutation correction is negligible for the hour-angle calculation. The full nutation path here is used for lunar position and `solarPosition`, where accuracy matters more.

### Equatorial to horizontal conversion

```bash
sed -n '180,222p' epoch.go
```

```output
// equatorialToHorizontal converts equatorial coordinates to horizontal
// (altitude/azimuth) for the given observer position and time.
//
// See Meeus, Astronomical Algorithms, eq. 13.5 & 13.6 p. 93.
func equatorialToHorizontal(t time.Time, obs Observer, eq equatorial) horizontal {
	lst := localSiderealTime(t, obs.lon)
	ha := hourAngle(eq.ra, lst)

	alt := asinx(sinx(eq.dec)*sinx(obs.lat) + cosx(eq.dec)*cosx(obs.lat)*cosx(ha))

	cosAltCosLat := cosx(alt) * cosx(obs.lat)

	var az float64
	// Guard against division by zero at the poles (lat ±90) or zenith (alt 90).
	if math.Abs(cosAltCosLat) < 1e-10 {
		az = 0
	} else {
		az = acosx((sinx(eq.dec) - sinx(alt)*sinx(obs.lat)) / cosAltCosLat)
	}

	// acos gives 0..180; if sin(ha) > 0, object is west, so az = 360 - az
	if sinx(ha) > 0 {
		az = 360 - az
	}

	return horizontal{
		alt: alt,
		az:  az,
	}
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
```

The azimuth calculation has an explicit guard for division by zero at polar latitudes or zenith — `cosAltCosLat` can vanish when `lat = ±90` or `alt = 90`. The `sinx(ha) > 0` flip maps the `acos` output (0–180°) to the full 0–360° azimuth range.

The `hourAngle` function documents a mixed-unit interface: RA in degrees, LST in hours. The `*15` conversion is intentionally internal. This is a potential gotcha for anyone adding new callers.

---

## Public API Types: `dusk.go`

This file defines the package documentation, all public types, the `Observer` constructor, sentinel errors, and `String()` methods.

### Sentinel errors and Observer

```bash
sed -n '27,84p' dusk.go
```

```output
// errString is an immutable error type used for sentinel errors.
// Unlike errors.New, these can be declared as constants.
type errString string

func (e errString) Error() string { return string(e) }

// ErrCircumpolar is returned when a celestial object is circumpolar
// (always above the horizon) at the given latitude.
const ErrCircumpolar = errString("dusk: object is circumpolar (always above the horizon)")

// ErrNeverRises is returned when a celestial object never rises above
// the horizon at the given latitude.
const ErrNeverRises = errString("dusk: object never rises at this latitude")

// ErrNilLocation is returned when a nil *time.Location is passed to
// [NewObserver].
const ErrNilLocation = errString("dusk: location must not be nil")

// ErrNonFiniteCoord is returned when NaN or Inf coordinates are passed
// to [NewObserver].
const ErrNonFiniteCoord = errString("dusk: coordinates must be finite (NaN and Inf are not allowed)")

// ErrInvalidCoord is returned when latitude or longitude are outside
// the valid range in [NewObserver].
const ErrInvalidCoord = errString("dusk: latitude must be in [-90, 90] and longitude in [-180, 180]")

// validObserver returns an error if obs was not constructed via NewObserver
// (i.e., is a zero-value Observer with a nil location).
func validObserver(obs Observer) error {
	if obs.loc == nil {
		return ErrNilLocation
	}
	return nil
}

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
		return Observer{}, ErrNilLocation
	}
	if math.IsNaN(lat) || math.IsInf(lat, 0) || math.IsNaN(lon) || math.IsInf(lon, 0) {
		return Observer{}, ErrNonFiniteCoord
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return Observer{}, ErrInvalidCoord
	}
	return Observer{lat: lat, lon: lon, loc: loc}, nil
}
```

Design choices in the type system:

- **`errString` type** enables `const` sentinel errors. Unlike `errors.New` (which returns a pointer, requiring `var`), this pattern makes errors immutable — callers can't accidentally reassign them. Idiomatic for libraries that want `errors.Is` matching on sentinel values.
- **Unexported `Observer` fields** force construction through `NewObserver`, which validates once. Every public function then calls `validObserver` as a cheap nil-check to catch zero-value `Observer{}` that bypassed the constructor.
- **NaN/Inf checked before range** — the validation order in `NewObserver` matters: `NaN < -90` is false in IEEE 754, so the range check alone wouldn't catch NaN inputs.

### Result types and stringers

```bash
sed -n '104,205p' dusk.go
```

```output
// SunEvent holds the times of sunrise, solar noon, sunset, and the duration
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
	Rise         time.Time // zero value if the Moon does not rise
	Set          time.Time // zero value if the Moon does not set
	AboveHorizon bool      // true if Moon was above the horizon at start of day
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

// horizontal represents altitude and azimuth in degrees.
// Used internally for coordinate conversions.
type horizontal struct {
	alt float64
	az  float64
}

// ecliptic represents ecliptic coordinates: longitude and latitude in degrees,
// and distance in kilometers.
// Used internally for lunar position calculations.
type ecliptic struct {
	lon  float64
	lat  float64
	dist float64
}

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
	return fmt.Sprintf("Rise=%s Set=%s AboveHorizon=%v",
		formatTime(m.Rise),
		formatTime(m.Set),
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

The type design follows a consistent pattern:

- **Public result structs** (`SunEvent`, `MoonEvent`, `TwilightEvent`, `LunarPhaseInfo`) with exported fields — callers read results directly, no getters needed.
- **Private coordinate structs** (`equatorial`, `horizontal`, `ecliptic`) — internal plumbing that callers never see.
- **Zero-value `time.Time` as "no event"** — e.g., the Moon may rise but not set on a given day. Callers check with `.IsZero()`. This is distinct from sentinel errors, which mean the geometry makes the event impossible.
- **`formatTime`** standardizes the `--:--` display for zero times across all `String()` methods.

`TwilightEvent.Dusk` is tonight, `Dawn` is tomorrow morning. To get *this* morning's dawn, call with yesterday's date — a subtle API contract documented in the struct comment.

---

## Solar Calculations: `solar.go`

This file implements `SunriseSunset`, the three twilight functions, and all unexported solar helpers. The algorithm follows the NOAA solar calculator (derived from Meeus).

### The solar parameter pipeline

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

The six-step pipeline: mean solar time → mean anomaly → equation of center → ecliptic longitude → declination → transit JD. This is factored into `computeSolarParams` so both `SunriseSunset` and `twilight` share the same computation without duplication.

### SunriseSunset — the primary solar function

```bash
sed -n '28,66p' solar.go
```

```output
// SunriseSunset computes sunrise, solar noon, and sunset for the given date
// and observer position. The observer must be constructed via [NewObserver].
// The date is converted to the observer's timezone to determine the local
// calendar day; the time-of-day is ignored.
// Output times are converted to the observer's timezone.
//
// The algorithm follows the NOAA solar calculator method (derived from Meeus,
// Astronomical Algorithms).
func SunriseSunset(date time.Time, obs Observer) (SunEvent, error) {
	if err := validObserver(obs); err != nil {
		return SunEvent{}, err
	}
	localDate := date.In(obs.loc)
	date = time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, time.UTC)
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

The flow:

1. Convert input to the observer's local calendar day, then to UTC midnight — this means passing `14:30 EST` and `22:00 EST` on the same date produce identical results.
2. Compute solar parameters (declination + transit JD) for that date.
3. Get the hour angle via `solarHourAngle` with `depression=0` — this is where polar errors (`ErrCircumpolar`, `ErrNeverRises`) surface.
4. Sunrise/sunset are symmetric around transit: `Jrise = jTransit − ω/360`, `Jset = jTransit + ω/360`.
5. Convert Julian dates back to `time.Time` in the observer's timezone.

### Solar hour angle — the polar edge-case gate

```bash
sed -n '115,149p' solar.go
```

```output
// solarHourAngle returns the hour angle in degrees for the Sun at the given
// declination, observer latitude, and depression angle (degrees below the
// geometric horizon, positive downward). For standard sunrise/sunset, pass
// depression = 0.
//
// For sunrise/sunset (depression=0), includes a -0.83 degree correction for
// atmospheric refraction and solar semidiameter. For twilight, uses the
// depression angle directly per IAU/USNO convention.
//
// Returns ErrCircumpolar when the Sun never sets (midnight sun) or
// ErrNeverRises when the Sun never rises (polar night) at this latitude
// and depression angle.
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

// solarTransitJD returns the Julian date of solar transit (solar noon).
func solarTransitJD(J, M, lambda float64) float64 {
	return j2000 + J + 0.0053*sinx(M) - 0.0069*sinx(2*lambda)
}
```

The `depression == 0` branch applies -0.83° for atmospheric refraction (~0.566°) plus solar semidiameter (~0.266°). For twilight, `depression` is passed directly as 6°, 12°, or 18° — no refraction correction, per IAU/USNO convention.

The polar detection is elegant: when `cosHA < -1`, the Sun never dips below the horizon (midnight sun); when `cosHA > 1`, it never rises (polar night). The `clamp` in `acosx` would silently hide these cases, so the explicit bounds check is essential.

### Twilight — reusing the solar pipeline

```bash
sed -n '155,221p' solar.go
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

// twilight computes the twilight period for a given depression angle (positive
// degrees below the geometric horizon). Only the calendar date is used; the
// time-of-day is ignored. The returned Dusk is today's "set" at the depression
// angle (evening boundary) and Dawn is tomorrow's "rise" at the depression
// angle (morning boundary).
//
// Because the calculation spans two calendar days (tonight and tomorrow morning),
// an error is returned if twilight cannot be computed for either day. Near polar
// transition dates (latitudes ~65–70°), today's dusk may exist but tomorrow's
// dawn may not (or vice versa). In that case the entire call returns an error;
// callers needing partial results should compute each boundary separately using
// the appropriate depression angle and [SunriseSunset]-style hour-angle logic.
func twilight(date time.Time, obs Observer, depression float64) (TwilightEvent, error) {
	if err := validObserver(obs); err != nil {
		return TwilightEvent{}, err
	}
	localDate := date.In(obs.loc)
	date = time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, time.UTC)
	if err := validJulianDateRange(date); err != nil {
		return TwilightEvent{}, err
	}

	// Evening twilight: sunset at the given depression angle for today.
	sp := computeSolarParams(date, obs.lon)
	omega, err := solarHourAngle(sp.delta, depression, obs.lat)
	if err != nil {
		return TwilightEvent{}, err
	}
	dusk := universalTimeFromJD(sp.jTransit + omega/360).In(obs.loc)

	// Tomorrow's "rise" at this depression = twilight dawn.
	tomorrow := date.AddDate(0, 0, 1)
	if err := validJulianDateRange(tomorrow); err != nil {
		return TwilightEvent{}, err
	}
	sp2 := computeSolarParams(tomorrow, obs.lon)
	omega2, err2 := solarHourAngle(sp2.delta, depression, obs.lat)
	if err2 != nil {
		return TwilightEvent{}, err2
	}
	dawn := universalTimeFromJD(sp2.jTransit - omega2/360).In(obs.loc)

	return TwilightEvent{
		Dusk:          dusk,
		Dawn:          dawn,
		NightDuration: dawn.Sub(dusk),
	}, nil
}
```

Twilight reuses the exact same `computeSolarParams` + `solarHourAngle` pipeline as `SunriseSunset`, just with a different depression angle. The three public functions are thin wrappers passing 6°, 12°, or 18°.

The key subtlety: twilight spans **two calendar days** — dusk tonight (`jTransit + ω/360`) and dawn tomorrow morning (`jTransit₂ − ω₂/360`). Near polar transition latitudes (~65–70°), one boundary may exist while the other doesn't. The function returns an error for the whole call rather than partial results — the doc comment explains why and suggests a workaround.

### Solar helper functions

```bash
sed -n '85,113p' solar.go
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

// solarDeclination returns the Sun's declination in degrees from its ecliptic
// longitude and Julian century T since J2000.0.
func solarDeclination(lambda, T float64) float64 {
	eps := meanObliquity(T)
	return asinx(sinx(lambda) * sinx(eps))
}
```

Note the two mean anomaly functions: `solarMeanAnomaly(J)` takes **days** since J2000 (used in the NOAA sunrise/sunset path), while `solarMeanAnomalyFromCentury(T)` takes **centuries** (used in the lunar phase calculation). Same underlying formula, different time scales — the coefficients differ by the 36525 days/century factor.

`solarDeclination` uses `meanObliquity` only — no nutation. This is the intentional asymmetry noted earlier: the NOAA simplified method omits nutation for sunrise/sunset calculations where the ~17" correction is negligible.

---

## Lunar Calculations: `lunar.go`

The lunar module is the most complex part of dusk. It implements Meeus Chapter 47 (lunar position from periodic terms) and a minute-by-minute horizon scanner for moonrise/moonset.

### Lunar ecliptic position — Meeus Table 47

```bash
sed -n '17,80p' lunar.go
```

```output
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

		switch r.M {
		case 0:
			Sb += r.Σb * sb
		case 1, -1:
			Sb += r.Σb * sb * E
		case 2, -2:
			Sb += r.Σb * sb * E2
		}
	}

	return ecliptic{
		lon:  mod360(Lp + Sl/1e6),
		lat:  Sb / 1e6,
		dist: 385000.56 + Sr/1000,
	}
}
```

This is the computational core. Five fundamental arguments (D, Lp, M, Mp, F) feed two loops over Meeus's periodic term tables:

- **Table 47.A** (60 rows): longitude (Σl) and distance (Σr) corrections
- **Table 47.B** (60 rows): latitude (Σb) corrections

The `E` factor corrects for the eccentricity of Earth's orbit — terms involving the Sun's mean anomaly (M=±1) get scaled by E, and M=±2 terms by E². The `sincosx` call in the longitude/distance loop computes both sin and cos of the same argument in one pass.

The additive corrections (A1, A2, A3 terms) account for Venus perturbation, Jupiter perturbation, and Earth flattening — small but measurable effects on lunar position.

Final result: longitude and latitude in degrees (the `/1e6` converts from Meeus's millionth-of-a-degree units), distance in km (the `/1000` converts from meters).

### Moonrise/moonset — the minute-by-minute scanner

```bash
sed -n '137,210p' lunar.go
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
// The one-minute resolution means events shorter than one minute may not be
// detected. At polar or near-polar latitudes, the Moon can graze the horizon
// briefly enough to fall within a single scan step.
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
	// Construct in local time then convert to UTC so DST is handled:
	// spring-forward days are 23h, fall-back days are 25h.
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

		prevAlt = hz.alt
	}

	return MoonEvent{
		Rise:         rise,
		Set:          set,
		AboveHorizon: aboveAtStart,
	}, nil
}
```

Unlike `SunriseSunset` (which uses an analytical hour-angle formula), moonrise/moonset uses brute-force scanning. This is necessary because the Moon moves ~13° per day along the ecliptic — fast enough that the analytical one-shot approach used for the Sun doesn't work.

Key details:

- **DST-aware scan window**: local midnight to local midnight, not UTC midnight. `scanMinutes` is typically 1440 but can be 1380 (spring-forward) or 1500 (fall-back).
- **Horizon threshold**: `-lunarHorizonDepression` (0.833°) accounts for refraction + semidiameter, matching the same correction applied to the Sun.
- **Early termination**: the loop breaks as soon as both rise and set are found.
- **`AboveHorizon`**: records whether the Moon was above the threshold at the start of the day, letting callers distinguish "Moon was up all day" from "Moon never rose."
- **Each iteration** does a full ecliptic→equatorial→horizontal pipeline — ~1440 times per call. The doc comment notes ~1-2ms on M-series hardware.

### Lunar phase

```bash
sed -n '82,128p' lunar.go
```

```output
// LunarPhase returns the lunar phase at the given instant.
//
// Unlike SunriseSunset and MoonriseMoonset which use only the calendar date,
// LunarPhase uses the exact time — the phase changes continuously. The result
// depends on the UTC instant; timezone does not affect the calculation.
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

	// phase angle (Meeus p. 346)
	PA := 180 - d - 0.1468*((1-0.0549*sinx(Mp))/(1-0.0167*sinx(Msol)))*sinx(d)

	K := 100 * (1 + cosx(PA)) / 2

	days := d / 360 * lunarMonthDays

	return LunarPhaseInfo{
		Illumination: K,
		Elongation:   d,
		Angle:        PA,
		DaysApprox:   days,
		Waxing:       d < 180,
		Name:         lunarPhaseName(d),
	}, nil
}
```

`LunarPhase` is the only public function that uses the **exact instant** rather than just the calendar date. It computes:

1. **Elongation** — angular separation between Moon and Sun. The `mod360` check distinguishes waxing (0–180°) from waning (180–360°) since `acos` only returns 0–180°.
2. **Phase angle** — Meeus p. 346 formula with corrections for lunar and solar eccentricity.
3. **Illumination** — percentage from the phase angle via `(1 + cos(PA)) / 2`.
4. **DaysApprox** — linear estimate mapping elongation to days into the lunation. This is approximate by design (the actual Moon moves non-uniformly).

### Lunar helpers and phase naming

```bash
sed -n '214,263p' lunar.go
```

```output
// ---------------------------------------------------------------------------

// lunarMeanElongation returns the Moon's mean elongation in degrees.
//
// T is Julian centuries since J2000.0.
// See Meeus eq. 47.2 p. 338.
func lunarMeanElongation(T float64) float64 {
	return mod360(297.8501921 + 445267.1114034*T - 0.0018819*T*T + T*T*T/545868 - T*T*T*T/113065000)
}

// lunarMeanAnomaly returns the Moon's mean anomaly in degrees.
//
// T is Julian centuries since J2000.0.
// See Meeus eq. 47.4 p. 338.
func lunarMeanAnomaly(T float64) float64 {
	return mod360(134.9633964 + 477198.8675055*T + 0.0087414*T*T + T*T*T/69699 - T*T*T*T/14712000)
}

// lunarArgumentOfLatitude returns the Moon's argument of latitude in degrees.
//
// T is Julian centuries since J2000.0.
// See Meeus eq. 47.5 p. 338.
func lunarArgumentOfLatitude(T float64) float64 {
	return mod360(93.272095 + 483202.0175233*T - 0.0036539*T*T + T*T*T/3526000 - T*T*T*T/863310000)
}

// lunarPhaseName returns the common name for the lunar phase based on the
// elongation angle in degrees.
func lunarPhaseName(age float64) string {
	age = mod360(age)

	switch {
	case age < 22.5 || age >= 337.5:
		return "New Moon"
	case age < 67.5:
		return "Waxing Crescent"
	case age < 112.5:
		return "First Quarter"
	case age < 157.5:
		return "Waxing Gibbous"
	case age < 202.5:
		return "Full Moon"
	case age < 247.5:
		return "Waning Gibbous"
	case age < 292.5:
		return "Last Quarter"
	default:
		return "Waning Crescent"
	}
}
```

The lunar fundamental arguments are 4th-degree polynomials in T (Julian centuries) — higher precision than the solar helpers because the Moon's orbit has more significant perturbation terms.

`lunarPhaseName` divides the 360° elongation cycle into eight 45° bins. The `New Moon` bin wraps around 0°/360° (handling both `< 22.5` and `>= 337.5`).

### Meeus coefficient tables

```bash
sed -n '265,282p' lunar.go && echo '...' && echo "tableLongDist: 155 entries (showing first 4)" && sed -n '280,284p' lunar.go && echo '...' && echo "tableLat: 61 entries (showing first 4)" && sed -n '358,362p' lunar.go
```

```output
// ---------------------------------------------------------------------------
// Meeus Table 47.A/B coefficients (moved from lunar_tables.go)
// ---------------------------------------------------------------------------

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
...
tableLongDist: 155 entries (showing first 4)
var tableLongDist = [...]lunarLongDistCoeff{
	{0, 0, 1, 0, 6288774, -20905355},
	{2, 0, -1, 0, 1274027, -3699111},
	{2, 0, 0, 0, 658314, -2955968},
	{0, 0, 2, 0, 213618, -569925},
...
tableLat: 61 entries (showing first 4)
	{0, 0, 0, 1, 5128122},
	{0, 0, 1, 1, 280602},
	{0, 0, 1, -1, 277693},
	{2, 0, 0, -1, 173237},

```

The two coefficient tables are transcribed directly from Meeus pp. 339–341:

- **`tableLongDist`** (60 rows): each row has 4 argument multipliers (D, M, M', F) plus Σl (longitude, in 0.000001°) and Σr (distance, in 0.001 km).
- **`tableLat`** (60 rows): same argument structure, with Σb (latitude, in 0.000001°).

These are declared as fixed-size arrays (`[...]`), not slices — the compiler enforces the exact count and they live in static memory. The coefficient values are integers in the source but typed as `float64` to avoid per-iteration casts in the inner loop.

---

## Concerns

### Code quality

1. **Mixed time-unit interfaces** — `hourAngle` takes RA in degrees but LST in hours. `solarMeanAnomaly` takes days while `solarMeanAnomalyFromCentury` takes centuries. Both are documented, but a type alias or wrapper could prevent misuse at the call site.

2. **`clamp` masks bugs silently** — The comment on `clamp` honestly acknowledges this trade-off. A genuinely wrong value (e.g., 1.3 from an upstream bug) gets silently clamped to 1.0. A debug-build assertion or metric could help catch this without affecting production.

3. **`MoonriseMoonset` performance** — 1440 full ecliptic→equatorial→horizontal evaluations per call is correct but expensive. The doc comment benchmarks this transparently. A coarse-then-fine approach (scan every 10 minutes, then refine) could reduce work ~10×, but the current ~1-2ms per call is adequate for typical use.

### Community standards adherence

4. **Error handling** — Exemplary. Every public function validates inputs, returns errors rather than panicking, and uses sentinel errors with `const` for immutability. The `errString` pattern is idiomatic for Go libraries.

5. **Documentation** — Thorough. Package doc, function docs with Meeus page references, parameter constraints, edge-case behavior, and performance characteristics are all documented. The doc comments follow Go conventions (start with function name, use `[Type]` cross-references).

6. **Test infrastructure** — Table-driven tests with USNO/Stellarium/Meeus reference values (not shown in this walkthrough but verified via `task test`). Race detector enabled in CI.

7. **Zero dependencies** — The library has no external dependencies, which is ideal for a leaf library. All math is self-contained.

8. **API design** — The v3 API follows the "make zero values useful" principle where possible (zero `time.Time` = no event), but correctly rejects zero-value `Observer` since a nil location would panic. Constructor validation at `NewObserver` is the right pattern.

### Deferred-by-design items

9. **`DaysApprox` linearity** — The linear elongation-to-days mapping is acknowledged as approximate. A more accurate model would require solving Kepler's equation for the Moon, which is out of scope for this library's precision goals.

10. **Nutation asymmetry** — `solarDeclination` (NOAA path) omits nutation while `eclipticToEquatorial` (lunar path) applies it. This is intentional and documented, but could surprise contributors who expect consistency. The CLAUDE.md gotchas section captures this.

