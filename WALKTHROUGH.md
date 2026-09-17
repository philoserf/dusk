# dusk Walkthrough

A linear tour of `github.com/philoserf/dusk/v4` — a zero-dependency Go library for
sunrise, twilight, moonrise and lunar phase, plus the reference CLI that consumes it.

This document is hand-maintained prose. Its snippets are quoted from the source by **file
and symbol**, never by line range, and nothing in the gate checks them: re-read it before
tagging. Its companion is `THEORY.md`, which covers why the code is shaped this way rather
than how it runs.

## Overview

The library answers four questions about one place on one day:

| Question                         | Entry point                                                 |
| -------------------------------- | ----------------------------------------------------------- |
| When does the Sun rise and set?  | `SunriseSunset`                                             |
| When does the sky get dark?      | `CivilTwilight`, `NauticalTwilight`, `AstronomicalTwilight` |
| When does the Moon rise and set? | `MoonriseMoonset`                                           |
| What phase is the Moon in?       | `LunarPhase`                                                |

Everything else is unexported. There are no external dependencies — not even for the Meeus
coefficient tables, which are transcribed into the source rather than fetched.

Two conventions run through the whole library and explain most of what looks odd at first:

- **Every angle is in degrees.** The trig wrappers in `trig.go` convert at the boundary, so
  no formula ever contains a radian conversion.
- **Longitude is east-positive.** New York is `-74.006`.

## Architecture

Six files, one package, at the repository root:

```
trig.go     degree trig, clamping, angle normalisation   — depends on nothing
epoch.go    Julian dates, sidereal time, nutation        — depends on trig
coord.go    ecliptic → equatorial → horizon              — depends on epoch
dusk.go     Observer, result types, sentinel errors      — the public vocabulary
solar.go    SunriseSunset, the three twilight bands
lunar.go    MoonriseMoonset, LunarPhase, Meeus ch. 47
cmd/dusk/   the reference CLI: main.go, report.go, render.go
```

The dependency arrow runs strictly downward, and the tour follows it: angles, then time,
then coordinates, then the vocabulary, then the two astronomical halves, then the CLI.

`coord.go` is worth noting up front because it was carved out of `epoch.go` in v4.1.0. The
two files held different layers — `epoch.go` is consumed by everything, `coord.go` by
exactly one call chain — and no linear reading order could keep them together.

## Layer one: angles

`trig.go` is the whole of the numerical foundation. Go's `math` package works in radians;
every formula in Meeus is in degrees; so the conversion happens once, here.

`trig.go` — the wrappers

```go
func sinx(deg float64) float64    { return math.Sin(deg * degToRad) }
func cosx(deg float64) float64    { return math.Cos(deg * degToRad) }
func tanx(deg float64) float64    { return math.Tan(deg * degToRad) }
func asinx(x float64) float64     { return radToDeg * math.Asin(clamp(x)) }
func acosx(x float64) float64     { return radToDeg * math.Acos(clamp(x)) }
func atan2x(y, x float64) float64 { return radToDeg * math.Atan2(y, x) }
```

The `x` suffix means "in degrees". Note that `asinx` and `acosx` route through `clamp`,
which is the most load-bearing six lines in the file:

`trig.go` — `clamp`

```go
// clamp restricts x to [-1, 1] before passing it to asin/acos. This prevents
// NaN from floating-point rounding in trig chains. Note: it also silently
// clamps genuinely wrong values (e.g., a miscalculated 1.3 → 1), which could
// mask upstream bugs. Correctness is validated by test coverage against Meeus
// and USNO reference data rather than runtime detection.
func clamp(x float64) float64 { return math.Max(-1, math.Min(1, x)) }
```

The comment is unusually candid, and it should be read as a standing caveat rather than an
apology. A long trig chain can produce `1.0000000000000002`, which `math.Asin` answers with
NaN; clamping kills that. But it equally silences a genuinely wrong `1.3`. The library
accepts that trade and pays for it with reference-data tests — and, since v4.1.0, with
fuzz targets that actually assert, which is the only thing here that sweeps the input
space rather than sampling fixed points.

Angles are normalised by two helpers that appear constantly:

`trig.go` — `mod360`

```go
func mod360(x float64) float64 {
	x = math.Mod(x, 360)
	if x < 0 {
		x += 360
	}

	return x
}
```

`mod24` is the same function for hours. Go's `math.Mod` keeps the sign of its argument, so
the explicit `+= 360` is what makes the range half-open `[0, 360)` rather than
`(-360, 360)`.

## Layer two: time

Everything astronomical is a function of the Julian date, so `epoch.go` starts there.

`epoch.go` — `julianDate`

```go
// julianDate returns the Julian date for a given time, i.e., the continuous
// count of days and fractions of day since the beginning of the Julian period.
//
// Uses UnixNano internally, which limits the valid range to the int64
// nanosecond bounds (approximately 1677-09-21 to 2262-04-11). UnixNano's
// result is undefined outside that range: it wraps to an arbitrary value
// rather than to zero or any recognizable sentinel, so dates outside it
// silently produce incorrect results.
// Use [validJulianDateRange] to check before calling.
func julianDate(t time.Time) float64 {
	ms := t.UTC().UnixNano() / 1e6

	return float64(ms)/86400000.0 + j1970
}
```

The implementation is three lines; the doc comment is nine, and the comment is the
interesting part. `UnixNano` overflows int64 outside roughly 1677–2262, and — this is the
trap — it **does not** signal that. It wraps to an arbitrary value. A date outside the
range produces a confident, wrong answer.

So the range check is a separate, explicit call, and every public entry point makes it:

`epoch.go` — `validJulianDateRange`

```go
// validJulianDateRange reports whether t falls within the valid range for
// [julianDate]. Returns nil if valid, [ErrDateOutOfRange] otherwise.
func validJulianDateRange(t time.Time) error {
	if t.Before(julianDateMin) || t.After(julianDateMax) {
		return ErrDateOutOfRange
	}

	return nil
}
```

`julianDateMin`/`julianDateMax` are derived from `math.MinInt64`/`math.MaxInt64` rather
than written as dates, so they cannot drift from the bound they describe.

### Three ways of counting from J2000

The same epoch is counted three ways, and mixing them up is the single easiest error to
make in this codebase:

`epoch.go` — `julianCentury`, `julianDay`, `meanSolarTime`

```go
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
```

- `julianCentury` returns a **continuous** count in centuries. Every Meeus polynomial takes
  this.
- `julianDay` returns a **rounded integer** day number. Only the NOAA solar path uses it.
- `meanSolarTime` takes that rounded day and applies the observer's longitude.

The rounding in `julianDay` is why `SunriseSunset` must be handed a UTC midnight rather
than a local one — more on that when we reach `solar.go`.

There is a fourth form, and it is the one that catches people: `solarMeanAnomaly` takes
**days**, not centuries. Until v4.1.0 there were two functions for it, one per unit; the
duplicate is gone, and `CLAUDE.md` now records the unit as a convention.

### Sidereal time

Sidereal time is where the Earth's rotation enters. It is needed only by the horizon
conversion, which is needed only by the Moon.

`epoch.go` — `greenwichMeanSiderealTime`

```go
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
```

The embedded warning is a real one. `T` comes from **midnight UTC** while `JD` comes from
the actual instant. That asymmetry looks like a bug and is not: Meeus's formulation uses
0h UT for the slow polynomial terms and the true Julian date for the fast linear term.
Passing `t` to both — the obvious "simplification" — introduces an error that grows through
the day.

`localSiderealTime` then adds the longitude and converts to hours:

`epoch.go` — `localSiderealTime`

```go
// localSiderealTime returns the local sidereal time in hours for a given
// instant and observer longitude (east positive, west negative, in degrees).
func localSiderealTime(t time.Time, longitude float64) float64 {
	gst := greenwichMeanSiderealTime(t) // degrees
	lst := gst + longitude              // degrees

	return mod24(lst / 15.0)
}
```

### Nutation and obliquity

The rest of `epoch.go` is Meeus's periodic corrections for the wobble of the Earth's axis:

`epoch.go` — `meanObliquity`

```go
// meanObliquity returns the mean obliquity of the ecliptic in degrees.
//
// T is Julian centuries since J2000.0.
// See Meeus, Astronomical Algorithms, p. 147.
func meanObliquity(T float64) float64 {
	return 23.4392917 - 0.0130041667*T - 0.00000016667*T*T + 0.0000005027778*T*T*T
}
```

`nutationInLongitude` (Δψ) and `nutationInObliquity` (Δε) follow the same shape — a handful
of terms in arcseconds, divided by 3600 to reach degrees. They are consumed by exactly one
function, in the next file.

## Layer three: coordinates

`coord.go` converts between the three frames the library needs. It sits one layer above
`epoch.go` and is reached by exactly one call chain — the Moon's minute scan — which is why
it is its own file.

`coord.go` — `eclipticToEquatorial`

```go
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

Note that this applies the **full** nutation correction: Δψ shifts the longitude, Δε the
obliquity. That is a deliberate asymmetry with the solar path, which uses `meanObliquity`
alone. The Sun's declination feeds a 1–2 minute sunrise calculation where nutation is
below the noise floor; the Moon's position feeds a minute-by-minute altitude scan where it
is not.

`coord.go` — `altitudeOf`

```go
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
```

This function used to return an `altitude, azimuth` pair. The azimuth was computed on all
~1441 iterations of every moonrise scan and read by nothing; v4.1.0 deleted it, along with
the two-field struct that existed only to carry it back. The doc comment records what to
do if a bearing is ever wanted, including the `mod360` the deleted version was missing.

`coord.go` — `hourAngle`

```go
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

The mixed units are the trap the comment names: `ra` in degrees, `lst` in **hours**. The
`*15` is inside, so a caller who "helpfully" converts LST to degrees first gets an answer
fifteen times too large.

## Layer four: the vocabulary

`dusk.go` holds the public surface: the `Observer`, the four result types, and the sentinel
errors.

### Sentinel errors as constants

```go
// stringError is an immutable error type used for sentinel errors.
// Unlike errors.New, these can be declared as constants.
type stringError string

func (e stringError) Error() string { return string(e) }
```

`errors.New` returns a `*errorString`, which is a variable and can be reassigned by any
package that imports this one. A string-typed constant cannot. Every sentinel here follows
that pattern:

```go
// ErrCircumpolar is returned when a celestial object is circumpolar
// (always above the horizon) at the given latitude.
const ErrCircumpolar = stringError("dusk: object is circumpolar (always above the horizon)")
```

`ErrDateOutOfRange` is the one exception to the file layout — it lives in `epoch.go`,
beside the range check that returns it.

### The Observer

`dusk.go` — `NewObserver`

```go
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

Fields are unexported and validated once, at construction. The NaN check is separate from
the range check because `NaN < -90` is false — a NaN latitude would slide through a bounds
test unnoticed.

The zero-value `Observer` is therefore invalid by construction, and every entry point
checks for it:

`dusk.go` — `validObserver`

```go
// validObserver returns an error if obs was not constructed via NewObserver
// (i.e., is a zero-value Observer with a nil location).
func validObserver(obs Observer) error {
	if obs.loc == nil {
		return ErrNilLocation
	}

	return nil
}
```

### Two ways of saying "did not happen"

This is the distinction most worth internalising before reading further.

`dusk.go` — `MoonEvent`

```go
// MoonEvent holds the rise and set times for the Moon on a given day, along
// with whether the Moon was already above the horizon when that day began.
//
// There is deliberately no duration field. On a day when the Moon is up at
// midnight, Set precedes Rise, so Set.Sub(Rise) is negative; MoonEvent.Duration
// was removed in v3.0.0 for exactly that reason. Callers needing an interval
// should handle that case themselves.
type MoonEvent struct {
	Rise         time.Time // zero value if the Moon does not rise
	Set          time.Time // zero value if the Moon does not set
	AboveHorizon bool      // true if Moon was above the horizon at start of day
}
```

- A **zero `time.Time`** means the event did not occur on this particular day, which is
  ordinary. A lunar day runs about 24h50m, so the Moon routinely rises without setting
  before midnight.
- A **sentinel error** means the geometry forbids the event at this latitude —
  `ErrCircumpolar` for midnight sun, `ErrNeverRises` for polar night.

`MoonriseMoonset` never returns either sentinel; it says "up all day" with
`AboveHorizon` plus two zero times. A consumer that checks only the times disagrees with
the one that checks the flag — a bug `cmd/dusk` had and fixed in v4.0.0.

The third asymmetry is in `TwilightEvent`:

`dusk.go` — `TwilightEvent`

```go
// TwilightEvent holds the dusk and dawn times of a twilight period.
// Dusk is tonight's boundary (sun passes below the depression angle).
// Dawn is tomorrow morning's boundary (sun passes above the depression angle).
// To get this morning's dawn, call with yesterday's date.
type TwilightEvent struct {
	Dusk          time.Time     // evening boundary (today)
	Dawn          time.Time     // morning boundary (tomorrow)
	NightDuration time.Duration // time from Dusk to Dawn (overnight darkness)
}
```

`Dusk` is tonight's; `Dawn` is **tomorrow morning's**. To get this morning's dawn you call
with yesterday's date. This is the single most easily missed detail in the contract, and
the CLI does it explicitly to demonstrate it.

## Layer five: the Sun

The solar path follows the NOAA solar calculator, which is Meeus simplified for the
1–2 minute accuracy sunrise actually supports.

### Resolving "the day"

`solar.go` — `SunriseSunset`

```go
// SunriseSunset computes sunrise, solar noon, and sunset for the given date
// and observer position. The observer must be constructed via [NewObserver].
// The date is converted to the observer's timezone to determine the local
// calendar day; the time-of-day is ignored.
// Output times are converted to the observer's timezone.
//
// The algorithm follows the NOAA solar calculator method (derived from Meeus,
// Astronomical Algorithms).
func SunriseSunset(date time.Time, obs Observer) (SunEvent, error) {
	err := validObserver(obs)
	if err != nil {
		return SunEvent{}, err
	}

	localDate := date.In(obs.loc)

	// Rebuild the day at UTC midnight, not in obs.loc: meanSolarTime applies the
	// observer's longitude itself, after julianDay has rounded, so handing it a
	// zone-adjusted instant would apply longitude twice. MoonriseMoonset does the
	// opposite for the opposite reason -- see lunar.go.
	date = time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, time.UTC)

	err = validJulianDateRange(date)
	if err != nil {
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

The day is rebuilt at **UTC** midnight even though the calendar day came from the
observer's zone. The comment explains why, and since v4.1.0 it names the other convention
so the two no longer read as a contradiction: `meanSolarTime` applies the longitude itself,
after `julianDay` has rounded. Handing it a zone-adjusted instant would apply longitude
twice.

### The six-step parameter chain

Sunrise and twilight share the same preamble, factored into one struct:

`solar.go` — `computeSolarParams`

```go
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

Mean anomaly → equation of centre → ecliptic longitude → declination, plus the transit
time. Each step is a one-line function:

`solar.go` — the chain

```go
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
```

`solarDeclination` uses `meanObliquity` alone — no nutation. That is the intentional
asymmetry with `eclipticToEquatorial` noted earlier.

### Turning declination into two times

`solar.go` — `solarHourAngle`

```go
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
```

This is where the polar cases are decided, and the sign convention is worth pausing on.
`depression` is **positive downward**. For ordinary sunrise the caller passes `0` and the
function substitutes `-0.83`, which is refraction plus the Sun's semidiameter — the Sun is
_seen_ to rise while geometrically below the horizon. For twilight the depression is used
directly.

`cosHA` outside `[-1, 1]` is not an error in the arithmetic; it is the geometry saying the
Sun never reaches that altitude. Below `-1` it is always above: `ErrCircumpolar`. Above
`1` it never gets there: `ErrNeverRises`.

Sunrise and sunset are then symmetric about transit — `jTransit ∓ omega/360`.

### Twilight, which does it differently

`solar.go` — `twilight` (the tail)

```go
	// Evening twilight: sunset at the given depression angle for today.
	sp := computeSolarParams(date, obs.lon)

	omega, err := solarHourAngle(sp.delta, depression, obs.lat)
	if err != nil {
		return TwilightEvent{}, err
	}

	dusk := universalTimeFromJD(sp.jTransit + omega/360).In(obs.loc)

	// Tomorrow's "rise" at this depression = twilight dawn.
	tomorrow := date.AddDate(0, 0, 1)
```

Two calls to `computeSolarParams`, for two different days. That is what makes `Dawn`
tomorrow morning's: it is literally tomorrow's sunrise, computed at a depression angle.

The consequence is that a single failure fails the whole call. Near a polar transition,
tonight's dusk can exist while tomorrow's dawn does not, and the function returns an error
rather than a partial result. The doc comment says so and points callers at the workaround.

## Layer six: the Moon

The lunar path shares almost nothing with the solar one. It is full Meeus Chapter 47 —
sixty longitude/distance terms and sixty latitude terms — and it finds rise and set by
brute-force scanning rather than by solving.

### The position series

`lunar.go` — `lunarEclipticPosition` (the argument setup)

```go
	T := julianCentury(t)

	D := lunarMeanElongation(T)
	Lp := lunarMeanLongitude(T)
	// Meeus gives the Sun's mean anomaly per century here; solarMeanAnomaly takes
	// days, which is the unit CLAUDE.md fixes for this quantity. julianCentury is
	// days/36525, so scaling T back recovers the same argument.
	M := solarMeanAnomaly(T * 36525)
	Mp := lunarMeanAnomaly(T)
	F := lunarArgumentOfLatitude(T)
```

Five fundamental angles. Every one of the 120 periodic terms is a linear combination of
them. The comment on `M` is a v4.1.0 addition, recording why a centuries-argument formula
is being fed a days-argument function — the two spellings of one formula were collapsed.

`lunar.go` — the eccentricity correction and the longitude loop

```go
	E := 1 - 0.002516*T - 0.0000074*T*T
	E2 := E * E
```

```go
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
```

`E` corrects for the slow change in the Earth's orbital eccentricity, and Meeus applies it
to terms involving the Sun's mean anomaly — once for `M = ±1`, twice for `M = ±2`. The
`switch` is that rule. `sincosx` returns both at once because every term needs the sine for
longitude and the cosine for distance.

The struct field names use the actual Meeus symbols — `Mʹ`, `Σl`, `Σr`, `Σb` — which Go
permits and which makes the table checkable against the book:

`lunar.go` — `lunarLongDistCoeff`

```go
// Meeus Table 47.A — Periodic terms for the longitude (Σl) and distance (Σr)
// of the Moon.
//
// See Meeus, Astronomical Algorithms, p. 339.
type lunarLongDistCoeff struct{ D, M, Mʹ, F, Σl, Σr float64 }
```

### Phase

`lunar.go` — `LunarPhase` (the geometry)

```go
	// elongation (0-360°, waxing = 0-180, waning = 180-360)
	d := acosx(cosx(ec.lon-sunLon) * cosx(ec.lat))
	if mod360(ec.lon-sunLon) > 180 {
		d = 360 - d
	}

	// phase angle (Meeus p. 346)
	PA := 180 - d - 0.1468*((1-0.0549*sinx(Mp))/(1-0.0167*sinx(Msol)))*sinx(d)

	K := 100 * (1 + cosx(PA)) / 2
```

`acosx` returns `0..180`, which cannot distinguish waxing from waning — both are the same
angular separation. The `mod360` test reflects the second half of the cycle into
`180..360`.

`PA` is computed and then consumed one line later. It used to be published as
`LunarPhaseInfo.Angle` and was removed in v4.0.0 as unused; the README advertised it for
one release longer, which v4.1.0 fixed.

`LunarPhase` is the exception to the day regime: it takes an **instant**, not a day, and no
`Observer`. Phase is Sun–Earth–Moon geometry, so where you stand does not matter.

### The horizon threshold, and why it is positive

This is the part v4.1.0 changed most.

`lunar.go` — `moonAltitudeAboveHorizon`

```go
// moonAltitudeAboveHorizon returns how far the Moon's centre is above the
// altitude at which it is seen to rise or set, in degrees. Positive means up.
//
// The threshold is Meeus's lunar h0 (Astronomical Algorithms, ch. 15 p. 102):
//
//	h0 = 0.7275*pi - 0.5667
//
// where pi is the equatorial horizontal parallax, about 0.951 degrees at mean
// distance. Note the sign: unlike the Sun's -0.8333, the lunar h0 is *positive*
// (about +0.125 degrees). The Moon is close enough that parallax outweighs
// refraction, so its centre is above the geometric horizon when it is seen to
// rise -- not below it, as the Sun's is.
//
// h0 is recomputed per sample rather than fixed, because the distance varies by
// about +/-0.05 degrees of h0 between perigee and apogee. The distance is free:
// lunarEclipticPosition already returns it.
func moonAltitudeAboveHorizon(t time.Time, obs Observer) float64 {
	ec := lunarEclipticPosition(t)
	eq := eclipticToEquatorial(t, ec.lon, ec.lat)
	h0 := 0.7275*asinx(earthRadiusKm/ec.dist) - 0.5667

	return altitudeOf(t, obs, eq) - h0
}
```

Until v4.1.0 this was a fixed constant of `0.833` — the **Sun's** value — and the
comparison was `alt > -0.833`. The Moon is the one body close enough that horizontal
parallax dominates: at mean distance it is about 0.951°, and Meeus's `h0 = 0.7275π − 0.5667`
comes out to roughly **+0.125°**. The sign is the opposite of the Sun's. The old threshold
was about 0.96° of altitude wrong, which biased every rise early and every set late by
5–12 minutes.

The distance needed for the parallax was already being computed and thrown away, so
recomputing `h0` per sample also picks up the perigee/apogee variation for free.

### The minute scan

`lunar.go` — `MoonriseMoonset` (the scan)

```go
	scanMinutes := int(nextMidnight.Sub(d).Minutes())

	var rise, set time.Time

	prevDiff := moonAltitudeAboveHorizon(d, obs)
	aboveAtStart := prevDiff > 0

	for i := 1; i <= scanMinutes; i++ {
		cur := d.Add(time.Duration(i) * time.Minute)

		curDiff := moonAltitudeAboveHorizon(cur, obs)

		if rise.IsZero() && curDiff > 0 && prevDiff <= 0 {
			rise = crossingInstant(cur, prevDiff, curDiff).In(obs.loc)
		}

		if set.IsZero() && curDiff < 0 && prevDiff >= 0 {
			set = crossingInstant(cur, prevDiff, curDiff).In(obs.loc)
		}

		if !rise.IsZero() && !set.IsZero() {
			break
		}

		prevDiff = curDiff
	}
```

Because `h0` now varies per sample, the scan tracks the **difference** `altitude − h0`
rather than an altitude against a fixed constant. A sign change in that difference is a
crossing.

`scanMinutes` is computed from the gap between two local midnights, so it is 1380 on a
spring-forward day and 1500 on a fall-back day. That is why `MoonriseMoonset` builds its
day in `obs.loc` and converts, the exact opposite of what `SunriseSunset` does — and each
site now names the other.

The loop bound is `<=`, which samples the next midnight itself. That used to attribute a
crossing to the wrong calendar day, about 0.17% of the time. The repair was not to change
the bound but to stop reporting the sample:

`lunar.go` — `crossingInstant`

```go
// crossingInstant interpolates the moment the Moon crossed its horizon
// threshold, between the sample one minute before cur and cur itself. prevDiff
// and curDiff are moonAltitudeAboveHorizon at those two instants and must
// straddle zero.
//
// The result lies in [cur-1m, cur) -- strictly inside the scanned day even on
// the final iteration, which is what keeps a closed upper bound from attributing
// an event to the wrong calendar day. It also removes the one-minute
// quantization of reporting the sample instant rather than the crossing.
func crossingInstant(cur time.Time, prevDiff, curDiff float64) time.Time {
	frac := prevDiff / (prevDiff - curDiff)

	return cur.Add(-time.Minute + time.Duration(frac*float64(time.Minute)))
}
```

`frac` lands in `[0, 1)` for both directions — at a rise `prevDiff ≤ 0 < curDiff`, so
numerator and denominator are both negative; at a set both are positive. The result is
therefore always in `[cur−1m, cur)`, strictly inside the scanned day even on the final
iteration, which makes the closed upper bound harmless. It also removes the one-minute
quantization that the old code conceded.

Measured against published USNO values after both changes, over 28 rise/set events from
55°S to 64°N across all four seasons, the largest disagreement is **32 seconds** — and USNO
publishes only to the minute.

## The reference CLI

`cmd/dusk` is an executable specification, not a product. It calls every exported function,
and every documented edge case is reachable with a single flag.

### The date hazard, paid for at the call site

`cmd/dusk/main.go` — `parseDate`

```go
// parseDate parses --date in the observer's zone, defaulting to today there.
func parseDate(arg string, loc *time.Location) (time.Time, error) {
	if arg == "" {
		return time.Now().In(loc), nil
	}

	// Parsed without a zone, then anchored at midday in the observer's. A few
	// zones move their clocks at midnight - America/Santiago in September,
	// America/Havana in March - and there 00:00 does not exist, so
	// ParseInLocation resolves it to 23:00 the day before and the whole report
	// silently comes out for the previous day. Midday exists everywhere; the
	// library ignores the time of day and keeps only the calendar date.
	day, err := time.Parse(dateLayout, arg)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: --date %q is not YYYY-MM-DD: %w", errUsage, arg, err)
	}

	return time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, loc), nil
}
```

Two hazards, one line. The date must be parsed in the **observer's** zone, because the
library resolves the day with `date.In(obs.loc)` — a UTC midnight lands on the previous day
for anyone west of Greenwich. And it is anchored at **midday**, not midnight, because a few
zones do not have a midnight on some days and Go resolves the missing hour backwards.

### Turning errors back into values

`cmd/dusk/report.go` — `horizonState` and `stateOf`

```go
// horizonState says whether a crossing happened, or why the geometry forbade
// it. The library signals this with two sentinel errors; carrying it as a value
// lets the renderer compose its own prose instead of parsing ours.
type horizonState int

// stateOf maps the library's sentinels onto horizonState. Any other error is a
// genuine failure and is reported as not-a-state.
func stateOf(err error) (horizonState, bool) {
	switch {
	case errors.Is(err, dusk.ErrCircumpolar):
		return stateStaysAbove, true
	case errors.Is(err, dusk.ErrNeverRises):
		return stateStaysBelow, true
	default:
		return stateCrosses, false
	}
}
```

The library signals polar geometry with errors; the renderer wants to compose prose about
it. Converting to a value at the boundary means the renderer never parses an error string.
Anything that is _not_ one of the two sentinels is a real failure and is reported as such.

### Calling each band twice

`cmd/dusk/report.go` — `twilightReports` (the loop body)

```go
		// Yesterday's call supplies this morning's dawn and nothing else. Its
		// state is discarded: on a polar transition day it describes a night
		// the report is not about, and letting it stand printed "twilight
		// never arrives" above tonight's real dusk time.
		morning, _, err := callTwilight(band.fn, yesterday, obs, band.name)
		if err != nil {
			return nil, err
		}

		report.Dawn = toSecond(morning.Dawn)
```

This is the library's asymmetric `TwilightEvent` being paid for explicitly. The discarded
state is a real bug fixed in v4.0.0: seeding tonight's state from yesterday's call printed
"civil, nautical and astronomical twilight never arrives" directly above a real civil dusk.

### Rendering a day in the order it is lived

`cmd/dusk/render.go` — `timeline`

```go
// timeline collects every event that actually happened and orders them by the
// clock. A zero time means the event did not occur today and is simply absent.
func timeline(report Report) []dayEvent {
	var events []dayEvent

	// A sunset at 00:03 belongs to the day after the one being reported. It
	// sorts to the end correctly, but a bare "00:03" under a "19:00" reads as
	// a mistake, so the day it lands on is said out loud.
	day := report.date.Day()

	add := func(at time.Time, label string) {
		if at.IsZero() {
			return
		}

		if at.Day() != day {
			label += " (" + at.Format("2 Jan") + ")"
		}

		events = append(events, dayEvent{at: at, label: label})
	}

	add(report.Sun.Rise, "Sunrise")
	add(report.Sun.Noon, "Solar noon")
	add(report.Sun.Set, "Sunset")

	for _, band := range report.Twilight {
		add(band.Dawn, band.Name+" dawn")
		add(band.Dusk, band.Name+" dusk")
	}

	add(report.Moon.Rise, "Moonrise")
	add(report.Moon.Set, "Moonset")

	slices.SortFunc(events, func(a, b dayEvent) int { return a.at.Compare(b.at) })

	return events
}
```

The library returns results grouped by the call that produced them. A day is not lived that
way, so every event is collected with its label and sorted on the clock. The day-suffix
branch handles a sunset at 00:03 that belongs to the following morning.

`cmd/dusk/render.go` — `wrapAt`

```go
// wrapAt breaks a sentence onto lines no longer than width, so a condition
// reads as a paragraph rather than one long row. Every line carries indent,
// including the first, so width means the same thing on all of them -- a caller
// that prepends the indent itself would give the first line that much more room
// than the continuations beneath it.
func wrapAt(text string, width int, indent string) string {
	var (
		out  strings.Builder
		line int
	)

	for i, word := range strings.Fields(text) {
		// Runes, not bytes: a degree sign is two bytes and one column.
		runcount := utf8.RuneCountInString(word)

		switch {
		case i == 0:
			out.WriteString(indent + word)

			line = utf8.RuneCountInString(indent) + runcount
		case line+1+runcount > width:
			out.WriteString("\n" + indent + word)
			line = utf8.RuneCountInString(indent) + runcount
		default:
			out.WriteString(" " + word)

			line += 1 + runcount
		}
	}

	return out.String()
}
```

The indent is entirely this function's business, including on the first line. Until v4.1.0
the caller prepended it _and_ passed it in, so the first line got two columns more than its
continuations and the right edge was ragged by exactly the indent width.

## Running it

A mid-latitude summer day — everything happens, in order:

Transcript of `go run ./cmd/dusk --lat 42.9634 --lon -85.6681 --tz America/Detroit --date 2025-06-21`:

```text
Saturday 21 June 2025
42.9634°N  85.6681°W  ·  America/Detroit

  02:42   Moonrise
  03:45   Astronomical dawn
  04:42   Nautical dawn
  05:28   Civil dawn
  06:03   Sunrise
  13:44   Solar noon
  17:28   Moonset
  21:25   Sunset
  22:00   Civil dusk
  22:46   Nautical dusk
  23:43   Astronomical dusk

  Daylight   15h21m
  Dark        4h02m  (astronomical, tonight)
  Moon       Waning Crescent, 19%
```

Polar night at Tromsø. The Sun never rises, but twilight still arrives, and the report says
both:

Transcript of `go run ./cmd/dusk --lat 69.6492 --lon 18.9553 --tz Europe/Oslo --date 2025-12-21`:

```text
Sunday 21 December 2025
69.6492°N  18.9553°E  ·  Europe/Oslo

  The sun does not rise today (polar night). Twilight still
  reaches civil depth around midday.
  The moon neither rises nor sets today.

  06:28   Astronomical dawn
  07:46   Nautical dawn
  09:31   Civil dawn
  13:53   Civil dusk
  15:37   Nautical dusk
  16:56   Astronomical dusk

  Dark       13h33m  (astronomical, tonight)
  Moon       New Moon, 2%
```

The same place in midsummer. Note the moonset printed **above** the moonrise — the Moon was
already up at midnight, which is what `AboveHorizon` exists to say:

Transcript of `go run ./cmd/dusk --lat 69.6492 --lon 18.9553 --tz Europe/Oslo --date 2025-06-21`:

```text
Saturday 21 June 2025
69.6492°N  18.9553°E  ·  Europe/Oslo

  The sun does not set today (midnight sun).
  The sun never drops 6° below the horizon tonight, so civil,
  nautical and astronomical twilight never arrives.
  The moon is already up at midnight, so today's moonset precedes
  its moonrise.

  19:08   Moonset
  22:49   Moonrise

  Moon       Waning Crescent, 21%
```

Both polar reports exit **0**. Polar geometry is a result, not a failure. Only a misused
command line (`usage:`) and an out-of-range date (`unsupported date:`) exit 1, and they are
told apart by the message rather than the status.

## How it is held together

`task` is the whole gate, and CI runs exactly `task` — no more, no less. The rule is
symmetric: never add a check to CI the local gate does not run, and never add a tool to the
gate without installing it in the workflow.

Three parts are worth knowing about:

**The coverage ratchet.** `coverage.ratchet` records the count of **uncovered statements**
per package, and the gate diffs current counts against it. It fails in both directions — a
number that rises is lost coverage, one that falls is coverage to lock in with
`task ratchet:update`. An integer rather than a percentage, because a percentage holds
still while a guarded branch adds one covered statement and one uncovered, and it grows
more forgiving as the repository grows. Reflowing blank lines splits coverage blocks, so a
refactor can move these counts without changing what the tests reach.

**The lint posture.** `.golangci.yml` runs `default: all` and disables only what fights this
repository's deliberate design — each disable carrying a measured finding count and a
reason. `depguard` is `list-mode: strict`, allowing only the standard library and this
module's own path, which is how "zero dependencies" is enforced rather than merely stated.

**Two formatters, one job each.** gofumpt and goimports run _inside_ golangci-lint, which is
the single definition of formatted for Go. Prettier is the same thing for everything that is
not Go, with `embeddedLanguageFormatting: "off"` — the load-bearing setting, because
prettier's default rewrites source inside fenced blocks, and this document quotes its own
compiled examples.

**The fuzz targets.** Since v4.1.0 all three assert rather than merely failing to panic:
ordering and positive duration for the Sun, and for the Moon that any non-zero time falls
inside the scanned local day. That last invariant is exactly what the scan got wrong before
the boundary fix, which is why the two had to land in that order.

## What this pass turned up

Two observations from reading the source against the prose, neither large enough to change
the code:

- **`lunarPosition` was left without a production caller** by the parallax fix, because
  `moonAltitudeAboveHorizon` needs the distance that `lunarPosition` discards and so calls
  `lunarEclipticPosition` and `eclipticToEquatorial` itself. That is the same shape #73 and
  #74 removed one release earlier, and `golangci-lint` does not catch it — `unused` sees the
  test call and is satisfied. Deleted in #101, with its Meeus p. 342 reference value
  re-pointed at the composition the scan actually performs.
- **The `// Exported functions` banner in `lunar.go` sits above `lunarEclipticPosition`,
  which is unexported.** It was accurate when the file was laid out and is not now;
  `LunarPhase` and `MoonriseMoonset` are the exported pair, separated by two unexported
  helpers. Left alone: it is a one-line inaccuracy in a comment, and the file is otherwise
  settled.
