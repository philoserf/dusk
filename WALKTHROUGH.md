# dusk Walkthrough

A linear tour of `github.com/philoserf/dusk/v4` — what each layer does, in the order
you have to understand it.

## Overview

`dusk` is a zero-dependency Go library that answers three questions about one place on
one day: when does the Sun cross a given depth below the horizon, when does the Moon
cross the horizon, and how much of the Moon is lit. The algorithms are Meeus's, from
_Astronomical Algorithms_ (2nd ed.), with the sunrise/sunset path following the NOAA
simplification of them.

Nothing outside the standard library is imported, and the module declares `go 1.27`.
The Meeus coefficient tables are transcribed into the source rather than fetched, so
the repository is the whole of the dependency graph.

The exported surface is five rows over seven functions:

| Function                                                                 | Answers                                            |
| ------------------------------------------------------------------------ | -------------------------------------------------- |
| `SunriseSunset(date, obs)`                                               | sunrise, solar noon, sunset, daylight duration     |
| `CivilTwilight` / `NauticalTwilight` / `AstronomicalTwilight(date, obs)` | tonight's dusk, tomorrow's dawn, dark duration     |
| `MoonriseMoonset(date, obs)`                                             | moonrise, moonset, whether the Moon was already up |
| `LunarPhase(date)`                                                       | illumination, elongation, phase name               |
| `NewObserver(lat, lon, loc)`                                             | the validated viewpoint all of the above need      |

Everything else in the package is unexported. `cmd/dusk` is the reference consumer: a
CLI that calls every one of them for a single place and date.

## Architecture

The library is a single package at the repository root. Five source files, and the
dependency direction between them is strictly one way.

```
trig.go      no dependencies
  └── epoch.go
        └── dusk.go
              ├── solar.go
              └── lunar.go
                    └── cmd/dusk/   (sees only the exported API)
```

| File        | What lives there                                                                |
| ----------- | ------------------------------------------------------------------------------- |
| `trig.go`   | Degree-mode trig wrappers, `clamp`, `mod360`/`mod24`                            |
| `epoch.go`  | Julian dates and their range check, sidereal time, nutation, coordinate changes |
| `dusk.go`   | Package doc, `Observer`, the result types, the sentinel errors                  |
| `solar.go`  | `SunriseSunset`, the three twilights, the solar helpers                         |
| `lunar.go`  | `LunarPhase`, `MoonriseMoonset`, and the transcribed Meeus tables               |
| `cmd/dusk/` | The reference CLI: `main.go` flags, `report.go` assembly, `render.go` output    |

Two conventions hold across all of it, and both are worth fixing in mind before
reading any formula:

- **Every angle is in degrees.** There is no radian anywhere outside `trig.go`.
- **Longitude is east-positive.** New York is `-74.006`, not `74.006`.

## Layer one: angles

Meeus's formulae are written in degrees; Go's `math` is written in radians. Rather than
convert at each of several hundred call sites, `trig.go` wraps every trig function the
package uses.

`trig.go` — the wrappers and `clamp`

```go
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
```

`clamp` is the piece worth pausing on, because its own comment argues against it.
Values that ought to lie in `[-1, 1]` arrive at `asin` and `acos` through long chains
of degree-mode trig, and floating-point rounding puts them at 1.0000000000000002 often
enough to matter. Clamping turns a NaN that would poison the whole result into a
correct answer — at the price of also absorbing a genuinely wrong 1.3. The package
accepts that trade and pays for it with reference-value tests against USNO and Meeus,
which is why weakening those tests is more dangerous here than a coverage percentage
suggests.

The other half of the discipline is normalisation. An angle that has been added to or
subtracted from is wrapped at the point it becomes a result, not left to its consumer.

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

`mod24` is the same function for hours. Between them they are called wherever an angle
or an hour becomes a return value, and the effect is that every angle handed between
functions in this package is already in `[0, 360)` — with one exception, noted when we
reach it.

`sincosx` exists because the lunar table loop needs both the sine and the cosine of the
same argument sixty times per evaluation, and `math.Sincos` computes them together.

## Layer two: time and coordinates

`epoch.go` is the foundation everything else stands on, and it holds two groups that do
not depend on each other.

### Julian dates, and the range that bounds them

Astronomical formulae take time as a Julian date: a continuous count of days since
4713 BC. The conversion goes through `UnixNano`, and that choice is the source of the
package's most carefully documented constraint.

`epoch.go` — `julianDate`

```go
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

"Undefined" here means _an arbitrary wrong number_ — not zero, not a sentinel, nothing
a caller could detect. So the package range-checks before computing, at every public
entry point, and `MoonriseMoonset` checks three times: the caller's instant and both
derived local midnights, because the conversion to local time can push a boundary date
over the edge.

`epoch.go` — `validJulianDateRange`

```go
var (
	julianDateMin = time.Unix(0, math.MinInt64).UTC()
	julianDateMax = time.Unix(0, math.MaxInt64).UTC()
)

...

func validJulianDateRange(t time.Time) error {
	if t.Before(julianDateMin) || t.After(julianDateMax) {
		return ErrDateOutOfRange
	}

	return nil
}
```

`ErrDateOutOfRange` is declared here rather than with the other sentinels in `dusk.go`,
deliberately — it lives beside the check that returns it.

Three derived quantities sit on top of `julianDate`, and the differences between them
matter later:

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

`julianDay` **rounds to an integer**. That is the NOAA method's day number, and it is
why the solar path works in whole days while the lunar path works in continuous
instants. `meanSolarTime` then applies the observer's longitude itself — remember this
when we reach `SunriseSunset`, because it is the reason the solar path must _not_ be
handed a zone-adjusted time.

### Sidereal time

Converting a star's fixed coordinates into "where is it in my sky right now" needs
sidereal time, and the function that computes it carries a warning against an
optimisation that looks obvious.

`epoch.go` — `greenwichMeanSiderealTime`

```go
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

Two different time arguments in one formula, on purpose. `localSiderealTime` then adds
the observer's longitude and converts to hours.

### Coordinate conversions

The second group in `epoch.go` converts between the two coordinate systems the package
needs: ecliptic (where the Meeus tables give positions) → equatorial (right ascension
and declination) → horizontal (altitude and azimuth, as seen from one place).

`epoch.go` — `eclipticToEquatorial`

```go
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

This applies the full nutation correction — both Δψ (in longitude) and Δε (in
obliquity). Note that the solar path does **not** use this function for declination;
`solarDeclination` uses mean obliquity only. That asymmetry is intentional: the
sunrise/sunset path is the NOAA simplification, and nutation is far below its
1–2 minute accuracy.

`epoch.go` — `equatorialToHorizontal`

```go
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
```

This is the one exception to the normalisation rule. `az` is the only angle in the
package that leaves a function without passing through `mod360`, and the two branches
above interact: when the pole guard fires and the object is west, `az` becomes
`360 - 0 = 360`, outside the half-open range everything else is kept in. It is filed as
[#65](https://github.com/philoserf/dusk/issues/65) — and only `alt` is ever read by a
caller, which is why [#62](https://github.com/philoserf/dusk/issues/62) proposes
deleting the azimuth half outright rather than fixing it.

A note on reading order: this file is two layers, not one. The time group is needed
before anything else in the package can be understood; the coordinate group is consumed
by exactly two call chains and cannot be introduced until the Moon. A banner comment
inside the file records the merge that produced the arrangement —
`// Coordinate conversions (moved from coord.go)` — and no linear reading order fixes
the split. That is [#64](https://github.com/philoserf/dusk/issues/64).

## Layer three: the Observer and the result types

`dusk.go` holds the package documentation, the viewpoint, and the shapes every
calculation returns.

The `Observer` is not a container of three numbers — it is a _validated_ container of
three numbers, and that distinction is the package's smallest consequential decision.

`dusk.go` — `Observer` and `NewObserver`

```go
type Observer struct {
	lat float64
	lon float64
	loc *time.Location
}

...

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

The fields are unexported, so the only way to obtain a usable `Observer` is through
this constructor — validation happens once, at the boundary, and no interior function
re-checks. A zero-value `Observer` is still constructible by a caller writing
`dusk.Observer{}`, and `validObserver` catches exactly that case by testing `loc` for
nil at every public entry point.

### Sentinel errors as constants

`dusk.go` — `stringError`

```go
// stringError is an immutable error type used for sentinel errors.
// Unlike errors.New, these can be declared as constants.
type stringError string

func (e stringError) Error() string { return string(e) }

...

const ErrCircumpolar = stringError("dusk: object is circumpolar (always above the horizon)")

...

const ErrNeverRises = stringError("dusk: object never rises at this latitude")
```

`errors.New` returns a pointer, which can only be a `var`, which a caller can reassign.
A string-backed type can be `const`, and cannot. Every sentinel in the package follows
this pattern.

### The result types, and two ways of saying "did not happen"

```go
type SunEvent struct {
	Rise     time.Time
	Noon     time.Time
	Set      time.Time
	Duration time.Duration
}

...

type MoonEvent struct {
	Rise         time.Time // zero value if the Moon does not rise
	Set          time.Time // zero value if the Moon does not set
	AboveHorizon bool      // true if Moon was above the horizon at start of day
}
```

The Moon uses a **value** to say an event did not occur — a zero `time.Time`, plus
`AboveHorizon` to say which side of the horizon it was on. The Sun uses an **error**:
`ErrCircumpolar` or `ErrNeverRises` in place of the whole result. The distinction the
package is drawing is real — the Sun's case is geometrically impossible, the Moon's
merely did not fall inside this calendar day — but it needs two carriers, and the only
consumer converts one back into the other. That is
[#70](https://github.com/philoserf/dusk/issues/70).

`TwilightEvent` carries the package's most easily missed contract, and says so:

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

## Layer four: the Sun

`solar.go` computes sunrise, noon and sunset by the NOAA method, and reuses the same
machinery for the three twilight bands.

Every solar calculation begins with the same six-step parameter chain, extracted so the
two entry points cannot drift apart:

`solar.go` — `computeSolarParams`

```go
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

Mean solar time → mean anomaly → equation of centre → ecliptic longitude → declination,
plus the Julian date of transit. Two numbers come out: where the Sun is on the sky's
north–south axis today (`delta`), and when it crosses the meridian (`jTransit`).

### Resolving "the day"

`solar.go` — `SunriseSunset`

```go
	localDate := date.In(obs.loc)

	date = time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, time.UTC)

	err = validJulianDateRange(date)
	if err != nil {
		return SunEvent{}, err
	}

	sp := computeSolarParams(date, obs.lon)
```

Three lines that repay attention. The caller's instant is resolved to a calendar day
**in the observer's zone**, and then rebuilt as UTC midnight. The time of day is
discarded entirely.

The consequence is the hazard this repository documents more than any other: passing
`time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC)` with a Detroit observer selects **June
20**, silently, and returns entirely plausible times for the wrong day. Three of the
library's own README examples were wrong in exactly this way before v4.0.0. The
parameter type promises an instant and the function honours a date, which is
[#63](https://github.com/philoserf/dusk/issues/63).

The rebuild target is UTC rather than `obs.loc` because `meanSolarTime` applies the
longitude itself, after `julianDay` has rounded — handing it a zone-adjusted instant
would apply longitude twice. `MoonriseMoonset` does the opposite for equally good
reasons, and neither site mentions the other, which is
[#78](https://github.com/philoserf/dusk/issues/78).

### Turning declination into two times

`solar.go` — `solarHourAngle`

```go
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

This is where polar geometry enters the type system. The cosine of the hour angle falls
outside `[-1, 1]` exactly when no such crossing exists: below `-1` the Sun never gets
low enough (midnight sun), above `1` it never gets high enough (polar night). Note that
the bounds are checked _before_ `acosx`, whose own `clamp`
would otherwise absorb the out-of-range cosine and hand back a plausible-looking time
for an event that cannot happen.

The `-0.83` for `depression == 0` is refraction plus the solar semidiameter: the Sun is
_seen_ to rise while its centre is still geometrically below the horizon.

With the hour angle in hand, sunrise and sunset are symmetric about transit:

```go
	Jrise := sp.jTransit - omega/360.0
	Jset := sp.jTransit + omega/360.0
```

One day, one parameter evaluation, two boundaries.

### Twilight, which does it differently

The three exported twilight functions are one-line wrappers over a shared `twilight`
with the depression angle as a parameter:

`solar.go` — the twilight wrappers

```go
func CivilTwilight(date time.Time, obs Observer) (TwilightEvent, error) {
	return twilight(date, obs, 6)
}
```

Nautical is 12 and astronomical is 18. Because the number is hidden behind a name, the
CLI has to reconstruct the mapping as a table of function pointers to get it back —
[#76](https://github.com/philoserf/dusk/issues/76).

The shared body is the same computation as `SunriseSunset`, up to a point:

`solar.go` — `twilight`

```go
	dusk := universalTimeFromJD(sp.jTransit + omega/360).In(obs.loc)

	// Tomorrow's "rise" at this depression = twilight dawn.
	tomorrow := date.AddDate(0, 0, 1)

	err = validJulianDateRange(tomorrow)
	if err != nil {
		return TwilightEvent{}, err
	}

	sp2 := computeSolarParams(tomorrow, obs.lon)

	omega2, err2 := solarHourAngle(sp2.delta, depression, obs.lat)
	if err2 != nil {
		return TwilightEvent{}, err2
	}

	dawn := universalTimeFromJD(sp2.jTransit - omega2/360).In(obs.loc)
```

Where `SunriseSunset` takes both boundaries from one day, `twilight` takes the evening
from today and recomputes the entire chain for tomorrow to get the morning. That has
three consequences, all paid today: the only consumer calls each band twice to undo it,
the error is all-or-nothing across two days of geometry, and a near-polar transition
date where tonight's dusk is real and tomorrow's dawn is not fails the whole call. The
doc comment tells callers to compute each boundary separately if they need partial
results — an admission that the type is wrong for the case. This is
[#77](https://github.com/philoserf/dusk/issues/77).

One function in this file has no caller at all. `solarPosition` computes the Sun's
equatorial coordinates through `eclipticToEquatorial`, and its doc comment argues for a
design distinction ("this function uses continuous Julian days for precise position at
any instant") that the package does not act on, because the exported `SolarPosition`
that consumed it was removed in v3. See
[#73](https://github.com/philoserf/dusk/issues/73) and
[#74](https://github.com/philoserf/dusk/issues/74).

## Layer five: the Moon

The Moon is harder than the Sun in the way that matters computationally: there is no
closed form. `lunar.go` evaluates Meeus's Chapter 47 series — sixty periodic terms for
longitude and distance, sixty more for latitude — and then, for rise and set, walks the
day one minute at a time.

### The position series

`lunar.go` — `lunarEclipticPosition`

```go
	D := lunarMeanElongation(T)
	Lp := lunarMeanLongitude(T)
	M := solarMeanAnomalyFromCentury(T)
	Mp := lunarMeanAnomaly(T)
	F := lunarArgumentOfLatitude(T)

	...

	E := 1 - 0.002516*T - 0.0000074*T*T
	E2 := E * E

	...

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

Each table row is four small integer multipliers and two amplitudes; the argument is a
linear combination of the five fundamental angles, and the amplitude is scaled by
powers of `E` — the eccentricity correction — according to how many times the Sun's
mean anomaly appears in that term. The identifiers are Meeus's own (`D`, `Lp`, `Mp`,
`F`, `Σl`), which is why the lint config permits them by name.

The `M` field of each row is a coefficient, not the angle: `switch r.M` is asking "how
many factors of `E` does this term take", and the three cases are the only values the
tables contain.

`solarMeanAnomalyFromCentury` here is the same formula as `solarMeanAnomaly` with the
time argument scaled, and it has this one caller —
[#79](https://github.com/philoserf/dusk/issues/79).

### Phase

`LunarPhase` is the exception to every convention in the package: it takes an
**instant** rather than a day, and no `Observer` at all, because the Sun–Earth–Moon
geometry is the same for everyone.

`lunar.go` — `LunarPhase`

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

The elongation `d` is the angular separation of Moon and Sun, and the `> 180` flip is
what distinguishes a waxing crescent from a waning one — `acos` alone cannot, since it
only returns `0..180`. `K` is the illuminated fraction as a percentage.

Two of the five published fields are `d` restated: `DaysApprox` is `d` in units of
days, and `Waxing` is `d < 180`. That is
[#66](https://github.com/philoserf/dusk/issues/66).

### The minute scan

`MoonriseMoonset` is the slowest thing in the library, by design.

`lunar.go` — `MoonriseMoonset`, building the day

```go
	localDate := date.In(obs.loc)
	// Construct in local time then convert to UTC so DST is handled:
	// spring-forward days are 23h, fall-back days are 25h.
	d := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, obs.loc).UTC()
	nextMidnight := time.Date(localDate.Year(), localDate.Month(), localDate.Day()+1, 0, 0, 0, 0, obs.loc).UTC()
```

This is the other half of the day-construction asymmetry. The lunar path needs _real_
local instants because it is about to walk the day minute by minute, and it needs the
real length of the day: `scanMinutes` comes out at 1380 on a spring-forward day and
1500 on a fall-back day, not a hard-coded 1440.

`lunar.go` — `MoonriseMoonset`, the scan

```go
	prevAlt := equatorialToHorizontal(d, obs, lunarPosition(d)).alt
	aboveAtStart := prevAlt > -lunarHorizonDepression

	for i := 1; i <= scanMinutes; i++ {
		cur := d.Add(time.Duration(i) * time.Minute)

		hz := equatorialToHorizontal(cur, obs, lunarPosition(cur))

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
```

A sign change in altitude across the threshold is a crossing, and the minute it is
detected is the answer. Each iteration runs the full sixty-term series twice over
(longitude/distance and latitude) plus two coordinate conversions, which is where the
documented 1–2 ms per call goes.

Three things about this loop are worth carrying away:

- **`i <= scanMinutes` makes the interval closed at both ends.** The final sample is
  next local midnight exactly, so a crossing detected there is recorded with a
  timestamp belonging to the following calendar day — and lost from the day it belongs
  to. Measured at about 0.17% of day-scans; [#67](https://github.com/philoserf/dusk/issues/67).
- **The threshold omits the Moon's horizontal parallax.** `lunarHorizonDepression` is
  0.833 — Meeus's `h0` for the _Sun_ — where the Moon's own is about `+0.125°`. The
  Moon is close enough that parallax dominates, and the sign is effectively backwards:
  every rise comes out 5–12 minutes early and every set that much late.
  [#69](https://github.com/philoserf/dusk/issues/69), and the workspace's current next
  step for this repository.
- **A zero result is a normal result.** A lunar day runs about 24h50m, so the Moon
  routinely rises without setting before midnight. `AboveHorizon` is what tells the two
  cases apart.

## The reference CLI

`cmd/dusk` is an executable specification rather than a product. It calls every
exported function, and every documented edge case is reachable with a single flag.

Its exit contract is part of the specification: **polar geometry is a result, so the
report renders and exits 0.** Only a misused command line and an out-of-range date
exit 1, and they are told apart by the message rather than the status.

`cmd/dusk/main.go` — `run`

```go
// run is the whole program. main only supplies the real streams and the exit
// status, which keeps every branch below reachable from a test.
//
// stdout is the data channel - the report, the version - and stderr is where
// the tool talks to whoever is driving it, so that `dusk ... --json | jq`
// pipes a record and not a complaint.
func run(args []string, stdout, stderr io.Writer) error {
```

`main` is four lines; everything testable lives in `run`, which takes its streams as
parameters.

### The date hazard, paid for at the call site

`cmd/dusk/main.go` — `parseDate`

```go
	// Parsed without a zone, then anchored at midday in the observer's. A few
	// zones move their clocks at midnight - America/Santiago in September,
	// America/Havana in March - and there 00:00 does not exist, so
	// ParseInLocation resolves it to 23:00 the day before and the whole report
	// silently comes out for the previous day. Midday exists everywhere; the
	// library ignores the time of day and keeps only the calendar date.
	day, err := time.Parse(dateLayout, arg)
```

Seven lines of comment to build a date. The CLI is the sophisticated caller, and this
is what the library's day-handling costs it.

### Turning errors back into values

`cmd/dusk/report.go` — `horizonState` and `stateOf`

```go
// horizonState says whether a crossing happened, or why the geometry forbade
// it. The library signals this with two sentinel errors; carrying it as a value
// lets the renderer compose its own prose instead of parsing ours.
type horizonState int

const (
	stateCrosses    horizonState = iota // the times are real
	stateStaysAbove                     // ErrCircumpolar at this angle
	stateStaysBelow                     // ErrNeverRises at this angle
)
```

The sentinels mean the opposite thing at a depression angle than they do at the
horizon, and the CLI is where that gets sorted out:

```go
	// At a depression angle the two sentinels mean the opposite of what they
	// mean at the horizon: staying above the angle means the night never gets
	// that dark, staying below it means the day never gets that light.
	twilightNotes = map[horizonState]string{
		stateStaysAbove: "never gets this dark tonight - the sun stays above %d degrees",
		stateStaysBelow: "this dark all day - the sun stays below %d degrees",
	}
```

These are tables rather than switches, and the comment says why: a switch over an
integer type needs a trailing return the compiler cannot prove unreachable, and that
line can never be covered.

### Calling each band twice

`cmd/dusk/report.go` — `twilightReports`

```go
	yesterday := date.AddDate(0, 0, -1)

	...

	for _, band := range twilightBands {
		report := TwilightReport{Name: band.name, degrees: band.degrees}

		// Yesterday's call supplies this morning's dawn and nothing else. Its
		// state is discarded: on a polar transition day it describes a night
		// the report is not about, and letting it stand printed "twilight
		// never arrives" above tonight's real dusk time.
		morning, _, err := callTwilight(band.fn, yesterday, obs, band.name)
```

Three rendered bands cost six library calls, twelve solar-parameter evaluations and
twelve hour angles, and half of each result is discarded. This is the consumer-side
half of [#77](https://github.com/philoserf/dusk/issues/77).

### Rendering a day in the order it is lived

`cmd/dusk/render.go` — `renderText`

```go
// The library returns its results grouped by the call that produced them -
// sun, three twilight bands, moon - but a day is not lived in that order.
// Printed that way, the twilight table's dawn column runs backwards and a
// moonset that belongs to the previous night's rise appears above the
// moonrise it precedes. Sorting every event by clock time removes both.
```

`timeline` collects every non-zero event, labels any that lands on a different calendar
day, and sorts:

```go
	add := func(at time.Time, label string) {
		if at.IsZero() {
			return
		}

		if at.Day() != day {
			label += " (" + at.Format("2 Jan") + ")"
		}

		events = append(events, dayEvent{at: at, label: label})
	}
```

That date suffix is the one thing keeping the day-boundary bug in
[#67](https://github.com/philoserf/dusk/issues/67) from being entirely silent.

`conditions` supplies what a list of times cannot — the geometry that stopped an event
from happening — and collapses the three twilight bands into one sentence, because if
the Sun never drops 6° below the horizon it never drops 12 or 18 either.

## Running it

Everything below is a transcript of a command run while writing this document, not a
live block.

An ordinary day at a mid latitude — the summer solstice in Michigan:

```
$ dusk --lat 42.9634 --lon -85.6681 --tz America/Detroit --date 2025-06-21
Saturday 21 June 2025
42.9634°N  85.6681°W  ·  America/Detroit

  02:37   Moonrise
  03:45   Astronomical dawn
  04:42   Nautical dawn
  05:28   Civil dawn
  06:03   Sunrise
  13:44   Solar noon
  17:35   Moonset
  21:25   Sunset
  22:00   Civil dusk
  22:46   Nautical dusk
  23:43   Astronomical dusk

  Daylight   15h21m
  Dark        4h02m  (astronomical, tonight)
  Moon       Waning Crescent, 19%
```

Every event in clock order, sun and moon and twilight interleaved. Now the polar night
at Tromsø, where `SunriseSunset` returns `ErrNeverRises` and the report still renders:

```
$ dusk --lat 69.6492 --lon 18.9553 --tz Europe/Oslo --date 2025-12-21
Sunday 21 December 2025
69.6492°N  18.9553°E  ·  Europe/Oslo

  The sun does not rise today (polar night). Twilight still reaches
  civil depth around midday.
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

Note what is missing: there is no "Solar noon" row. The Sun transits the meridian on
every day of the year at every latitude, and `computeSolarParams` has already worked
out when — but `SunriseSunset` discards `sp.jTransit` along with everything else when
the hour angle is impossible. That is the concrete cost in
[#70](https://github.com/philoserf/dusk/issues/70).

Six months later, the same place, the opposite geometry:

```
$ dusk --lat 69.6492 --lon 18.9553 --tz Europe/Oslo --date 2025-06-21
Saturday 21 June 2025
69.6492°N  18.9553°E  ·  Europe/Oslo

  The sun does not set today (midnight sun).
  The sun never drops 6° below the horizon tonight, so civil,
  nautical and astronomical twilight never arrives.
  The moon is already up at midnight, so today's moonset precedes
  its moonrise.

  19:41   Moonset
  22:18   Moonrise

  Moon       Waning Crescent, 21%
```

Moonset above moonrise, with a sentence explaining why — that is `AboveHorizon` being
read rather than inferred from the times.

The JSON is the same report with the prose kept as `note` fields:

```
$ dusk --lat 69.6492 --lon 18.9553 --tz Europe/Oslo --date 2025-12-21 --json
{
  "lat": 69.6492,
  "lon": 18.9553,
  "zone": "Europe/Oslo",
  "date": "2025-12-21",
  "sun": {
    "note": "polar night - the sun does not rise today"
  },
  "twilight": [
    {
      "name": "Civil",
      "dawn": "2025-12-21T09:31:12+01:00",
      "dusk": "2025-12-21T13:53:14+01:00",
      "night": "19h38m"
    },
...
```

The `sun` object has no `rise`, `noon` or `set` keys at all, rather than three
`0001-01-01T00:00:00Z` strings. That is `omitzero` rather than `omitempty` — a zero
`time.Time` is a struct, and `omitempty` does not drop struct zero values.

## How it is held together

There is one gate, `task`, and CI runs exactly it — never a check the local gate does
not run, never a tool in the gate that the workflow does not install.

```
$ task
task: [tidy] go mod tidy -diff
task: [vet] go vet ./...
task: [lint] golangci-lint config verify
task: [lint] golangci-lint run ./...
0 issues.
task: [docs] prettier --check .
Checking formatting...
All matched files use Prettier code style!
task: [nilaway] nilaway ./...
task: [test] go test -race -covermode=atomic -coverprofile=coverage.out ./...
ok  	github.com/philoserf/dusk/v4	coverage: 98.3% of statements
ok  	github.com/philoserf/dusk/v4/cmd/dusk	coverage: 94.5% of statements
task: [ratchet] ...
ratchet holds: 2 packages
```

Seven steps, ordered so that a failure is as cheap to read as possible. `golangci-lint`
runs `default: all` and disables only what fights this repository's deliberate design,
each disable carrying a measured finding count and a reason. `prettier` is the same
thing for Markdown and JSON.

Coverage is held by a **ratchet** rather than a percentage. `coverage.ratchet` records
the count of uncovered statements per package, and `task ratchet` diffs the current
counts against it, so the check fails in both directions: a number that rises is lost
coverage, one that falls is coverage to lock in. An integer rather than a percentage
because a percentage holds still while a guarded branch adds one covered statement and
one uncovered, and it grows more forgiving as the repository grows.

Tests are table-driven throughout, with expected values from USNO, Stellarium and
Meeus, at tolerances the reference data supports: 1–2 minutes for sunrise and sunset,
up to ~20 minutes for moonrise and moonset, 1–2% for illumination. Every test calls
`t.Parallel()` at both levels, enforced by `paralleltest`.

Three fuzz targets exist, and they are not equal. `FuzzLunarPhase` keeps its `*testing.T`
and checks illumination for NaN and for the documented range:

`fuzz_test.go` — `FuzzLunarPhase`

```go
		if math.IsNaN(p.Illumination) {
			t.Error("NaN illumination")
		}

		if p.Illumination < 0 || p.Illumination > 100 {
			t.Errorf("illumination out of range: %f", p.Illumination)
		}
```

The other two discard the handle they would need to fail:

`fuzz_test.go` — `FuzzSunriseSunset`

```go
	f.Fuzz(func(_ *testing.T, lat, lon float64, unix int64) {
		...
		_, err = SunriseSunset(date, obs)
		if err != nil {
			return // circumpolar or never-rises is valid
		}
	})
```

The `SunEvent` is dropped, the success path is empty, and the target's whole verdict is
"did not panic" — [#75](https://github.com/philoserf/dusk/issues/75). Fuzzing is not
part of the gate: `task test` replays each target's seed corpus in microseconds, while
`task fuzz` runs the engine for a chosen time limit, and a search with a time limit
belongs to whoever chose it.

## What this pass turned up

Everything this reading surfaced is already filed. Nothing new went to `.issues/`.

The three places the narrative above had to stop, back up, or hold two things in view
at once are each a structural finding rather than a reader's problem:

| Where reading broke down                                           | Filed as                                                                                               |
| ------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| `epoch.go` splits across two layers with no workable reading order | [#64](https://github.com/philoserf/dusk/issues/64)                                                     |
| The solar and lunar paths build "the day" two ways, silently       | [#78](https://github.com/philoserf/dusk/issues/78)                                                     |
| `solarPosition` is reachable from no entry point                   | [#73](https://github.com/philoserf/dusk/issues/73), [#74](https://github.com/philoserf/dusk/issues/74) |

The correctness findings referenced along the way, in the order they appear:

| Issue                                              | Where                                                       |
| -------------------------------------------------- | ----------------------------------------------------------- |
| [#65](https://github.com/philoserf/dusk/issues/65) | `equatorialToHorizontal` can return an azimuth of 360°      |
| [#62](https://github.com/philoserf/dusk/issues/62) | …and nothing reads that azimuth, so delete it               |
| [#70](https://github.com/philoserf/dusk/issues/70) | Polar geometry as an error destroys solar noon              |
| [#63](https://github.com/philoserf/dusk/issues/63) | Day-based entry points take an instant                      |
| [#77](https://github.com/philoserf/dusk/issues/77) | `twilight` spans two days; the CLI calls each band twice    |
| [#76](https://github.com/philoserf/dusk/issues/76) | Three wrappers hide the depression angle                    |
| [#79](https://github.com/philoserf/dusk/issues/79) | `solarMeanAnomaly` duplicated in two units                  |
| [#66](https://github.com/philoserf/dusk/issues/66) | `LunarPhaseInfo` publishes two derivations of a third field |
| [#67](https://github.com/philoserf/dusk/issues/67) | The minute scan is closed at both ends                      |
| [#69](https://github.com/philoserf/dusk/issues/69) | The moon threshold omits horizontal parallax                |
| [#75](https://github.com/philoserf/dusk/issues/75) | Two fuzz targets assert nothing                             |
