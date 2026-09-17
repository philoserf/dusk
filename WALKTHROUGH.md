# dusk Walkthrough

A linear tour of `github.com/philoserf/dusk/v5` — a zero-dependency Go library for
sunrise and sunset, the three twilight bands, moonrise and moonset, and lunar phase,
plus the reference CLI that consumes all of it.

Read it top to bottom. It follows the dependency order rather than the file order: the
unit discipline first, then time, then coordinates, then the two bodies, then the
program that assembles a day out of them.

Snippets are quoted from the source and labelled with their file and symbol. An elided
middle is marked `...`.

## Overview

The whole problem is one sentence. An observer stands at a fixed point on Earth and
wants to know, for one calendar day, when the Sun and Moon cross particular altitudes —
the horizon, and three fixed depths below it — and separately, how much of the Moon is
lit.

There is no persistent state, no I/O outside the CLI's writers, no network, no
concurrency, and no external dependency. The Meeus coefficient tables are transcribed
into the source rather than fetched.

Five exported entry points:

| Question                         | Call                              |
| -------------------------------- | --------------------------------- |
| When does the sun rise and set?  | `SunriseSunset(date, obs)`        |
| When does the sky get dark?      | `Twilight(date, obs, depression)` |
| When does the moon rise and set? | `MoonriseMoonset(date, obs)`      |
| How lit is the moon?             | `LunarPhase(instant)`             |
| Where am I?                      | `NewObserver(lat, lon, loc)`      |

Four of the five take a `Date`. `LunarPhase` takes a `time.Time`, and that difference
is deliberate — it is the first real idea in the package, and we reach it below.

## Architecture

One Go package at the repository root, and a reference CLI under `cmd/dusk`.

```
trig.go     angles — the only place radians exist
epoch.go    time — Julian dates, sidereal time, nutation, obliquity
coord.go    coordinates — ecliptic to equatorial, equatorial to altitude
dusk.go     the vocabulary — Observer, Date, Horizon, the four result types
solar.go    the Sun — NOAA closed form
lunar.go    the Moon — Meeus ch. 47 plus a minute-by-minute scan
cmd/dusk/   main.go (flags), report.go (assembly), render.go (text and JSON)
```

The layering is real and points one way. `trig.go` depends on nothing; `epoch.go` uses
`trig.go`; `coord.go` sits on `epoch.go` and is reached only by the Moon's scan;
`solar.go` and `lunar.go` are the two algorithms; `dusk.go` holds the boundary types.

`cmd/dusk` is an executable specification rather than a product. It calls every
exported function, and every documented edge case is reachable with a single flag.

## Layer one: angles

Every angle in this package is in degrees. That is not a style choice — it is what
makes the Meeus tables transcribable from the printed page and checkable against it.
The entire conversion surface is one file.

`trig.go` — the wrappers

```go
func sinx(deg float64) float64    { return math.Sin(deg * degToRad) }
func cosx(deg float64) float64    { return math.Cos(deg * degToRad) }
func tanx(deg float64) float64    { return math.Tan(deg * degToRad) }
func asinx(x float64) float64     { return radToDeg * math.Asin(clamp(x)) }
func acosx(x float64) float64     { return radToDeg * math.Acos(clamp(x)) }
func atan2x(y, x float64) float64 { return radToDeg * math.Atan2(y, x) }
```

A raw `math.Sin` anywhere else in the package would be a silent catastrophe — it would
read a degree value as radians and return a plausible wrong number. That this hazard is
confined to fourteen lines is the single most load-bearing decision in the codebase.

`clamp` guards the two inverse functions, and its own comment argues against it:

`trig.go` — `clamp`

```go
// clamp restricts x to [-1, 1] before passing it to asin/acos. This prevents
// NaN from floating-point rounding in trig chains. Note: it also silently
// clamps genuinely wrong values (e.g., a miscalculated 1.3 → 1), which could
// mask upstream bugs. Correctness is validated by test coverage against Meeus
// and USNO reference data rather than runtime detection.
func clamp(x float64) float64 { return math.Max(-1, math.Min(1, x)) }
```

That trade is named and paid for. Without it, float noise at `1.0000000000000002`
becomes a NaN that propagates through the whole result. With it, a real bug can hide.
The repository chose reference-data tests over runtime detection, and says so.

**`clamp` must never be applied to the hour-angle cosine.** That value legitimately
exceeds ±1 — it is how the code learns the Sun never reaches the queried altitude — and
clamping it would turn an impossible event into a plausible time. `solarHourAngle`
tests the bound itself, which we reach shortly.

Two normalizers round out the file, `mod360` and `mod24`, both returning a
non-negative result where Go's `math.Mod` would keep the sign of the dividend.

## Layer two: time

`epoch.go` converts instants to the two clocks astronomy actually uses: the Julian day
count, and sidereal time.

`epoch.go` — `julianDate`

```go
func julianDate(t time.Time) float64 {
	ms := t.UTC().UnixNano() / 1e6

	return float64(ms)/86400000.0 + j1970
}
```

Three lines, and a constraint that reaches every public entry point. `UnixNano` is
defined only for roughly 1677-09-21 to 2262-04-11; outside that it wraps to an
arbitrary value rather than any recognizable sentinel. A date out of range therefore
produces a confidently wrong answer, not an obvious one — which is why the package
range-checks before computing rather than validating after.

`epoch.go` — `validJulianDateRange`

```go
func validJulianDateRange(t time.Time) error {
	if t.Before(julianDateMin) || t.After(julianDateMax) {
		return ErrDateOutOfRange
	}

	return nil
}
```

### Three ways of counting from J2000

The package needs the epoch offset in three different units, and mixing them is the
easiest arithmetic mistake available here.

| Helper          | Unit                  | Used by                           |
| --------------- | --------------------- | --------------------------------- |
| `julianCentury` | centuries, continuous | the Meeus polynomial series       |
| `julianDay`     | days, **rounded**     | `meanSolarTime` only              |
| `julianDate`    | days, continuous      | everything needing a real instant |

`julianDay` rounds, and that rounding is what makes the next function work:

`epoch.go` — `meanSolarTime`

```go
func meanSolarTime(t time.Time, longitude float64) float64 {
	return float64(julianDay(t)) - longitude/360.0
}
```

**This function applies the observer's longitude itself.** Hand it an instant that has
already been shifted into the observer's zone and the longitude is applied twice. That
single fact explains a structural decision we meet in `solar.go`, and its opposite in
`lunar.go`.

### Sidereal time

`epoch.go` — `greenwichMeanSiderealTime`

```go
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
```

Two different arguments, deliberately: `T` from midnight, `JD` from the instant. The
comment exists because the asymmetry looks like a bug and is not.

`localSiderealTime` adds the longitude and hands back **hours**, not degrees — a unit
change mid-file that the one consumer has to know about. `hourAngle` in `coord.go`
documents the matching expectation from the other side.

### Nutation and obliquity

Four small functions implement Meeus chapters 22 and 25: `meanObliquity`,
`nutationInObliquity` (Δε), `nutationInLongitude` (Δψ), and the three mean-longitude
arguments they need. Each carries a page citation, which is what makes the package
auditable against the printed book.

**Only the Moon's path uses them.** The Sun's declination is computed with mean
obliquity alone. That asymmetry is intentional and we come back to it.

## Layer three: coordinates

`coord.go` opens by naming its own position in the stack:

`coord.go` — file header

```go
// Coordinate conversions: ecliptic to equatorial, and equatorial to the
// observer's horizon. These sit one layer above everything in epoch.go, which
// they consume (julianCentury, the nutation and obliquity helpers, and
// localSiderealTime) and nothing in this file is consumed by. The only caller
// chain that reaches here is MoonriseMoonset's minute scan.
```

`coord.go` — `eclipticToEquatorial`

```go
	dpsi := nutationInLongitude(L, l, omega)
	lon += dpsi

	eps := meanObliquity(T) + nutationInObliquity(L, l, omega)

	ra := atan2x(sinx(lon)*cosx(eps)-tanx(lat)*sinx(eps), cosx(lon))
	dec := asinx(sinx(lat)*cosx(eps) + cosx(lat)*sinx(eps)*sinx(lon))
```

Full nutation: Δψ added to the longitude, Δε to the obliquity. Compare
`solarDeclination`, which uses `meanObliquity` and stops. The Sun's accuracy budget is
1–2 minutes and does not earn the nutation terms back; the Moon's minute-by-minute scan
does. **This is a deliberate asymmetry, not an oversight** — the most likely wrong
"fix" a maintainer could make here is to unify them.

`altitudeOf` is where a position becomes a number the scan can compare:

`coord.go` — `altitudeOf`

```go
func altitudeOf(t time.Time, obs Observer, eq equatorial) float64 {
	lst := localSiderealTime(t, obs.lon)
	ha := hourAngle(eq.ra, lst)

	return asinx(sinx(eq.dec)*sinx(obs.lat) + cosx(eq.dec)*cosx(obs.lat)*cosx(ha))
}
```

Its doc comment records a deletion: azimuth was computed here and read by nothing, so
v4.1.0 removed it along with the `horizontal` type it returned. The note tells the next
reader to reinstate it against a live consumer rather than on spec.

## Layer four: the vocabulary

`dusk.go` holds the boundary: the types a caller touches, and nothing that computes.

### Sentinel errors as constants

`dusk.go` — `stringError`

```go
// stringError is an immutable error type used for sentinel errors.
// Unlike errors.New, these can be declared as constants.
type stringError string

func (e stringError) Error() string { return string(e) }
```

`errors.New` returns a `*errorString` that a caller could reassign. A `const` of a
string type cannot be reassigned, so the sentinels are immutable by construction.
Four remain — `ErrNilLocation`, `ErrNonFiniteCoord`, `ErrInvalidCoord`, and
`ErrDateOutOfRange`, which lives in `epoch.go` beside the range check that returns it.

v5.0.0 removed two more, `ErrCircumpolar` and `ErrNeverRises`. What replaced them is
below.

### The Observer

`dusk.go` — `NewObserver`

```go
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

Fields are unexported and there is no other constructor, so **validation happens once**.
Every entry point then checks exactly one thing — that `loc` is non-nil, which is the
signature of a zero-value `Observer` that skipped the constructor:

`dusk.go` — `validObserver`

```go
func validObserver(obs Observer) error {
	if obs.loc == nil {
		return ErrNilLocation
	}

	return nil
}
```

That is why no entry point re-checks NaN. The invariant is carried by the type.

### The calendar day as a type

`dusk.go` — `Date`

```go
// Date is a calendar day. It has no zone and no time of day, because the
// day-based entry points use neither.
...
type Date struct {
	Year  int
	Month time.Month
	Day   int
}
```

Through v4 these entry points took a `time.Time` and kept only the calendar day, as
resolved in the observer's zone. Passing a `time.UTC` midnight with a Detroit observer
selected the previous day, silently, and returned entirely plausible times for it —
three of the library's own README examples were wrong that way. The rule "build dates
in the observer's timezone" ended up written in five separate places, which is the
price of a convention the type system cannot express.

`DateIn` is the conversion that used to happen invisibly:

`dusk.go` — `DateIn`

```go
func DateIn(t time.Time, loc *time.Location) Date {
	if loc != nil {
		t = t.In(loc)
	}

	return Date{Year: t.Year(), Month: t.Month(), Day: t.Day()}
}
```

**`LunarPhase` deliberately kept its `time.Time`.** Phase is Sun–Earth–Moon geometry,
uses the whole instant, and takes no `Observer`. The package has two entry-point shapes
because it answers two kinds of question, and the signatures now say which is which.

### One carrier for "did not happen", two meanings

`dusk.go` — `Horizon`

```go
// Horizon says whether the Sun reached the altitude a call asked about. It is
// a value on the result rather than an error, because "the Sun did not set
// today" is an answer to the question, not a failure to answer it.
//
// The names describe the geometry, not the sunrise case. At a depression angle
// StaysAbove means the night never got that dark and StaysBelow means the day
// never got that light -- the opposite of what "circumpolar" and "never rises"
// suggested, which is why those names are gone.
type Horizon int
```

The old names were false at a depression angle: `ErrCircumpolar` said "always above the
horizon" and at 18° meant the Sun never got 18° _below_ it. The reference CLI carried a
comment correcting the library's own vocabulary; both are gone.

The four result types are inert — no methods, no computed accessors:

`dusk.go` — `SunEvent`

```go
type SunEvent struct {
	Rise     time.Time // zero unless Horizon is Crosses
	Noon     time.Time // always set
	Set      time.Time // zero unless Horizon is Crosses
	Duration time.Duration
	Horizon  Horizon
}
```

`Noon` is always set. Solar transit is defined on every day at every latitude, including
through the polar night — and through v4 the error return threw it away.

`MoonEvent` was deliberately **not** given a `Horizon`:

`dusk.go` — `MoonEvent`

```go
type MoonEvent struct {
	Rise         time.Time // zero value if the Moon does not rise
	Set          time.Time // zero value if the Moon does not set
	AboveHorizon bool      // true if Moon was above the horizon at start of day
}
```

`AboveHorizon` says which side of the horizon the Moon was on when the day began. That
is not the same fact as whether a crossing was possible, and conflating them would lose
a distinction the CLI's prose depends on. The Sun's missing crossing is geometrically
impossible; the Moon's is routine, because a lunar day runs about 24h50m.

The struct also carries a note about a field that is _not_ there — `MoonEvent.Duration`
was removed in v3.0.0 because `Set` precedes `Rise` on a day the Moon is up at midnight,
making the subtraction negative.

## Layer five: the Sun

`solar.go` is NOAA's closed-form method: one hour-angle evaluation gives both crossings,
in microseconds. Two exported functions share one body.

### The shared geometry

`solar.go` — `solarCrossing`

```go
	// Build the day at UTC midnight, not in obs.loc: meanSolarTime applies the
	// observer's longitude itself, after julianDay has rounded, so handing it a
	// zone-adjusted instant would apply longitude twice. MoonriseMoonset does the
	// opposite for the opposite reason -- see lunar.go.
	day := date.at(time.UTC)
```

That comment is the payoff from `meanSolarTime`. The day is built at **UTC** midnight —
not in the observer's zone — because the longitude correction is applied downstream.

What comes back is three instants rather than a hour angle:

`solar.go` — `solarDay`

```go
// solarDay holds one day's solar geometry, already solved into instants.
// rise and set are meaningless unless horizon is Crosses; transit always holds.
type solarDay struct {
	rise    float64
	transit float64
	set     float64
	horizon Horizon
}
```

This is the whole of what `SunriseSunset` and `Twilight` share. Each turns the same three
instants into its own result type.

### The six-step parameter chain

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

	return solarParams{J: J, T: T, delta: delta, jTransit: jTransit}
}
```

Read the units carefully. `J` is **days** since J2000, and `solarMeanAnomaly` takes days
— the repository fixed this unit deliberately, and `lunar.go` scales its own centuries
back to days rather than keeping a second function in the other unit. `T` is
**centuries**, and only `solarDeclination` wants it.

Two numbers come out — the declination and the Julian date of solar transit — plus the two
time bases they were computed from. Those are carried so the correction pass below can
re-solve on the same footing instead of inventing a second convention:

`solar.go` — `solarParams.declinationAt`

```go
func (sp solarParams) declinationAt(offsetDays float64) float64 {
	M := solarMeanAnomaly(sp.J + offsetDays)
	C := solarEquationOfCenter(M)
	lambda := solarEclipticLongitude(M, C)

	return solarDeclination(lambda, sp.T+offsetDays/36525.0)
}
```

### Turning declination into two times

`solar.go` — `solarHourAngle`

```go
func solarHourAngle(delta, depression, lat float64) (float64, Horizon) {
	var h0 float64
	if depression == 0 {
		h0 = -0.8333
	} else {
		h0 = -depression
	}

	num := sinx(h0) - sinx(lat)*sinx(delta)
	den := cosx(lat) * cosx(delta)

	cosHA := num / den
	if cosHA < -1 {
		return 0, StaysAbove
	}

	if cosHA > 1 {
		return 0, StaysBelow
	}

	return acosx(cosHA), Crosses
}
```

**This is the bound `clamp` must never touch.** `cosHA` outside ±1 is not float noise —
it is the arithmetic reporting that no hour angle solves the equation, because the Sun
never reaches that altitude on that day at that latitude. Clamping would return a
plausible time for an event that does not occur.

The `-0.8333` is refraction plus solar semidiameter, the value Meeus and USNO use. It read
`-0.83` until v5.1.0 — a transcription error worth about 1.6 seconds of half-day at the
equator, and, corrected on its own, very slightly the wrong way: a lower horizon means a
later sunset, and sunset was already late.

`solar.go` — `SunriseSunset`

```go
	noon := universalTimeFromJD(day.transit).In(obs.loc)

	// Transit is defined on every day at every latitude, so Noon is always set
	// -- including through the polar night, when the Sun reaches its highest
	// point below the horizon and there is no rise or set to report.
	if day.horizon != Crosses {
		return SunEvent{
			Noon:     noon,
			Duration: daylightOf(day.horizon),
			Horizon:  day.horizon,
		}, nil
	}

	rise := universalTimeFromJD(day.rise).In(obs.loc)
	set := universalTimeFromJD(day.set).In(obs.loc)
```

### Why the two boundaries are not mirrored

Through v5.0.0 rise and set were `jTransit ∓ omega/360` — one hour angle, applied both ways.
**That symmetry embeds an assumption the sky does not honour**: that declination is the same
at sunrise as at sunset. Near an equinox it moves about 0.4° a day, so the afternoon half-day
really is shorter than the morning, and a mirrored construction reports the two as equal _to
the second, by definition_.

v5.1.0 added a correction pass per boundary:

`solar.go` — `solarCrossing`

```go
	return solarDay{
		rise:    sp.jTransit - refineOmega(sp, -omega/360.0, depression, obs.lat, omega)/360.0,
		transit: sp.jTransit,
		set:     sp.jTransit + refineOmega(sp, +omega/360.0, depression, obs.lat, omega)/360.0,
		horizon: Crosses,
	}, nil
```

`solar.go` — `refineOmega`

```go
func refineOmega(sp solarParams, offsetDays, depression, lat, fallback float64) float64 {
	omega, horizon := solarHourAngle(sp.declinationAt(offsetDays), depression, lat)
	if horizon != Crosses {
		return fallback
	}

	return omega
}
```

The fallback is the one deliberate approximation here. A refined declination that puts a
boundary out of reach is the polar limit arriving mid-correction, on a day whose first pass
said the Sun crosses; keeping the first estimate avoids a boundary that vanishes and
reappears across a degree of latitude. `TestTwilight_RefinementAtThePolarLimit` pins a real
case — 65.5°N, civil, 2025-05-13 — found by search rather than guessed.

**It is a partial fix, and the code says so.** Measured against USNO, worst-case sunset falls
from 176s to 115s, which is what makes the README's two-minute claim true. Sunrise gets
_worse_ — the mirrored construction put the whole error on sunset, leaving sunrise
accidentally accurate, and correcting the geometry distributes it. About half the true skew
remains, because the hour angle is still measured about a transit computed once for the day:
the Sun's own motion in right ascension between transit and the boundary is unmodelled. That
is issue #114.

### Twilight, which is the same computation

`solar.go` — `Twilight`

```go
func Twilight(date Date, obs Observer, depression float64) (TwilightEvent, error) {
	day, err := solarCrossing(date, obs, depression)
	if err != nil {
		return TwilightEvent{}, err
	}

	if day.horizon != Crosses {
		return TwilightEvent{Horizon: day.horizon}, nil
	}

	return TwilightEvent{
		Dawn:    universalTimeFromJD(day.rise).In(obs.loc),
		Dusk:    universalTimeFromJD(day.set).In(obs.loc),
		Horizon: Crosses,
	}, nil
}
```

Identical arithmetic to `SunriseSunset`, with the depression threaded through — **including
the boundary correction**, which is why v5.1.0 moved every twilight time and not just
sunrise and sunset. Passing `depression = 0` gives sunrise and sunset, refraction included.

Through v4 this was shaped differently: `Twilight(D).Dusk` was the evening of D and
`Twilight(D).Dawn` was the morning of **D+1**, computed by running the entire parameter
chain a second time for tomorrow. It failed if either day's geometry was impossible —
and that took real results with it. At 75°N on 2025-11-26 the CLI reported a civil dawn
of `11:28:33Z` beside a note saying the Sun stayed below 6° all day; the day's real dusk
at `12:06:27Z` had been discarded because the call failed on Nov 27's dawn.

A same-day event cannot do that. One omega means **both boundaries exist or neither
does**, so the transition lands between days, where the geometry puts it.

## Layer six: the Moon

The Moon moves too fast for a closed form. `lunar.go` evaluates Meeus chapter 47 and
then walks the day a minute at a time.

### The position series

`lunarEclipticPosition` sums the periodic terms from Meeus tables 47.A and 47.B —
roughly a third of the file is those coefficients, transcribed rather than fetched. One
detail is worth stopping on:

`lunar.go` — `lunarEclipticPosition`

```go
	// Meeus gives the Sun's mean anomaly per century here; solarMeanAnomaly takes
	// days, which is the unit CLAUDE.md fixes for this quantity. julianCentury is
	// days/36525, so scaling T back recovers the same argument.
	M := solarMeanAnomaly(T * 36525)
```

The repository keeps one solar mean anomaly, in days, and multiplies back rather than
maintaining a second function in centuries. The comment exists so that `T * 36525` does
not read as a mistake.

The function returns longitude, latitude **and distance** — and the distance is not
decoration.

### Phase

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

	return LunarPhaseInfo{
		Illumination: K,
		Elongation:   d,
		Name:         lunarPhaseName(d),
	}, nil
```

Three fields, and that is the second reduction this struct has had. v4.0.0 removed
`Angle` — the Meeus phase angle `PA` above, published and read by nobody. v5.0.0 removed
`DaysApprox` and `Waxing`, both restatements of `Elongation`: the Moon is waxing when
`Elongation < 180`, and a linear estimate of lunation age is
`Elongation / 360 * 29.53059`.

`DaysApprox` is the interesting one. Elongation does not advance linearly in time, so
the rescaling was never the lunation age its name promised — its own doc comment said
"rough". The expression survives in the doc comment, where writing it is a choice to
accept the approximation rather than a number handed over under a misleading name.

### The horizon threshold, and why it is positive

`lunar.go` — `moonAltitudeAboveHorizon`

```go
func moonAltitudeAboveHorizon(t time.Time, obs Observer) float64 {
	ec := lunarEclipticPosition(t)
	eq := eclipticToEquatorial(t, ec.lon, ec.lat)
	h0 := 0.7275*asinx(earthRadiusKm/ec.dist) - 0.5667

	return altitudeOf(t, obs, eq) - h0
}
```

**The Sun's threshold is negative and the Moon's is positive.** Meeus gives the lunar
`h0 = 0.7275·π − 0.5667`, where π is the equatorial horizontal parallax — about +0.125°.
The Moon is the one body close enough that parallax outweighs refraction, so its centre
is _above_ the geometric horizon when it is seen to rise.

Until v4.1.0 this code used 0.833 for both bodies. The sign was backwards and the
magnitude off by about 0.96°, biasing every moonrise early and every moonset late by
5–12 minutes. `h0` is recomputed per sample from the Moon's true distance, which is why
it is a function and not a constant — and why `ecliptic.dist`, which looks like an
unread field, must not be deleted.

### The minute scan

`lunar.go` — `MoonriseMoonset`

```go
	// Construct in local time then convert to UTC so DST is handled:
	// spring-forward days are 23h, fall-back days are 25h. The scan walks the day
	// one minute at a time and takes its length from the gap between these two
	// midnights, so it needs real local instants. SunriseSunset and twilight build
	// the day at UTC midnight instead, for the opposite reason -- see solar.go.
	d := date.at(obs.loc).UTC()
	nextMidnight := Date{date.Year, date.Month, date.Day + 1}.at(obs.loc).UTC()
```

**This is the opposite construction to the solar path, and both are correct.** The scan
needs the day's real _length_ — 1380 minutes on a spring-forward day, 1500 on a
fall-back day, not a hard-coded 1440. Each site names the other and says why it differs;
consolidating them into one helper is the most likely damage a maintainer without the
theory would do.

The loop itself:

```go
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

A sign change between two samples is a crossing. The reported instant is not the sample:

`lunar.go` — `crossingInstant`

```go
// The result lies in [cur-1m, cur) -- strictly inside the scanned day even on
// the final iteration, which is what keeps a closed upper bound from attributing
// an event to the wrong calendar day.
...
func crossingInstant(cur time.Time, prevDiff, curDiff float64) time.Time {
	frac := prevDiff / (prevDiff - curDiff)

	return cur.Add(-time.Minute + time.Duration(frac*float64(time.Minute)))
}
```

That half-open interval is an invariant, not an implementation detail, and the fuzz test
asserts it. Note the interpolation is on the difference `altitude − h0`, never on
altitude against a fixed threshold — which is what lets `h0` vary per sample.

The scan's resolution _is_ its accuracy in one direction only: an event shorter than one
minute is invisible to it by construction, and no interpolation recovers a bracket that
was never sampled. Against published USNO values over 28 rise/set events from 55°S to
64°N, the largest disagreement is 32 seconds.

## The reference CLI

`cmd/dusk` exists to be an executable specification. Three files: flags, assembly,
rendering.

### The day arrives as a day

`cmd/dusk/main.go` — `parseDate`

```go
// Until v5.0.0 this had to manufacture an instant and anchor it at midday,
// because the library took a time.Time and resolved the day from it: a few
// zones move their clocks at midnight, so 00:00 does not exist there and
// ParseInLocation resolved it to 23:00 the day before, quietly reporting the
// wrong day. dusk.Date carries the day itself, so there is no instant to
// mangle and nothing to work around.
func parseDate(arg string, loc *time.Location) (dusk.Date, error) {
	if arg == "" {
		return dusk.DateIn(time.Now(), loc), nil
	}

	day, err := time.Parse(dateLayout, arg)
	if err != nil {
		return dusk.Date{}, fmt.Errorf("%w: --date %q is not YYYY-MM-DD: %w", errUsage, arg, err)
	}

	return dusk.Date{Year: day.Year(), Month: day.Month(), Day: day.Day()}, nil
}
```

America/Santiago in September and America/Havana in March are the two zones this used to
break in. The zone is still needed — to answer "what is today" when `--date` is omitted
— but it can no longer move a day the user typed.

### Assembly

`cmd/dusk/report.go` — `buildReport`

```go
	// LunarPhase is the one entry point that takes an instant rather than a day,
	// because phase is Sun-Earth-Moon geometry and changes measurably within a
	// day. Midday in the observer's zone is the representative moment of their
	// day, and saying so is now the caller's job rather than an accident of how
	// --date happened to be parsed.
	phase, err := phaseReport(time.Date(date.Year, date.Month, date.Day, 12, 0, 0, 0, obs.Location()))
```

The two entry-point shapes meet here, and the CLI has to say which instant of the day it
means. Through v4 that instant arrived by accident.

The three twilight bands cost one call each, plus one more:

`cmd/dusk/report.go` — `twilightReports`

```go
	// Bands run shallow to deep, so the last one that crosses is the deepest
	// that crosses - which is the band the summary's Dark row names.
	deepest := -1

	var deepestDusk time.Time

	for i, band := range twilightBands {
		event, err := dusk.Twilight(date, obs, float64(band.degrees))
		...
	}
```

and then, for that one band only:

```go
	// A band that crosses today need not cross tomorrow: near a polar
	// transition the night has no end to measure to, and the row is omitted.
	if tomorrow.Horizon == dusk.Crosses {
		reports[deepest].Night = shortDuration(tomorrow.Dawn.Sub(deepestDusk))
	}
```

Overnight darkness spans midnight, so no single day's event can carry it. The summary
prints the night of exactly one band — the deepest that has one — so the CLI pays one
extra call rather than three. Through v4 it made six calls to render three bands.

### Rendering a day in the order it is lived

`cmd/dusk/render.go` — `renderText`

```go
// The library returns its results grouped by the call that produced them -
// sun, three twilight bands, moon - but a day is not lived in that order.
// Printed that way, the twilight table's dawn column runs backwards and a
// moonset that belongs to the previous night's rise appears above the
// moonrise it precedes. Sorting every event by clock time removes both.
```

This is the best-reasoned code in the repository, and its comments are the clearest
statement of the library's contract anywhere — frequently better than the doc comments
they describe.

Times are rounded, not truncated, and where that happens matters:

`cmd/dusk/render.go` — `timeline`

```go
		// Round here rather than at the Format call below, so the day check and
		// the reader see the same value. A 23:59:45 sunset rounds to 00:00 and
		// belongs to tomorrow; comparing the unrounded instant would print it as
		// today's, with no marker and no way to tell.
		when = when.Round(time.Minute)
```

`"15:04"` drops the seconds, so until v5.1.0 every displayed time ran up to 59 seconds
early — one-sided, always. That was noise while the moonrise threshold was minutes wrong;
once the library agreed with USNO to within half a minute, the layout became the larger
error. **The JSON report does not round**: it is the machine-readable answer and keeps its
seconds, so the two renderings of one event may differ by up to half a minute, by design.

`cmd/dusk/render.go` — `moonCondition`

```go
	case moon.Rise.IsZero() && moon.Set.IsZero():
		// A lunar day runs about 24h50m, so at high latitudes the Moon can be
		// up for the whole calendar day without crossing. Unlike the Sun, the
		// library reports this with AboveHorizon rather than a sentinel error
		// - MoonriseMoonset never returns one - and without consulting it the
		// report reads as though the Moon were absent.
		if moon.AboveHorizon {
			return "The moon stays above the horizon all day."
		}

		return "The moon neither rises nor sets today."
```

That is the distinction `MoonEvent` keeps and `SunEvent` does not need: two zero times
mean something different for the Moon than for the Sun, and only `AboveHorizon` says
which.

Notes are keyed by tables rather than switches, deliberately:

`cmd/dusk/report.go` — the note tables

```go
// The JSON-facing prose for each state. Tables rather than switches: a switch
// over an integer type needs a trailing return the compiler cannot prove is
// unreachable, and that line can never be covered or tested.
var (
	solarNotes = map[dusk.Horizon]string{
		dusk.StaysAbove: "midnight sun - the sun does not set today",
		dusk.StaysBelow: "polar night - the sun does not rise today",
	}
	...
)
```

That is the coverage ratchet influencing structure rather than just tests — worth being
conscious of, and not a defect.

## Running it

A transcript of `go run ./cmd/dusk --lat 69.6492 --lon 18.9553 --tz Europe/Oslo --date 2025-12-21`,
captured while writing this walkthrough. Nothing re-runs it.

```text
Sunday 21 December 2025
69.6492°N  18.9553°E  ·  Europe/Oslo

  The sun does not rise today (polar night). Twilight still
  reaches civil depth around midday.
  The moon neither rises nor sets today.

  06:28   Astronomical dawn
  07:46   Nautical dawn
  09:31   Civil dawn
  11:42   Solar noon
  13:53   Civil dusk
  15:37   Nautical dusk
  16:56   Astronomical dusk

  Dark       13h33m  (astronomical, tonight)
  Moon       New Moon, 2%
```

Four things in one screen. The Sun never rises, and the report says so and exits **0** —
polar geometry is a result, not a failure. `Solar noon` appears between the civil dawn
and the civil dusk: that row was unreachable before v5.0.0, because transit was computed
and then discarded with the error return. The timeline is in clock order rather than
call order. And the `Dark` row names the deepest band that actually has a night.

The exit contract is part of the specification: only a misused command line (`usage:`)
and an out-of-range date (`unsupported date:`) exit 1, and they are told apart by the
message rather than the status.

## How it is held together

**CI runs exactly `task`** — the same gate that runs locally. Nothing is checked in CI
that a developer cannot run, and nothing is in the local gate that CI does not install.

Coverage is held by a **ratchet**, not a percentage. `coverage.ratchet` records the count
of uncovered statements per package and `task ratchet` diffs against it, so the gate
fails in **both** directions: a number that rises is lost coverage, one that falls is
coverage to lock in. An integer rather than a percentage because a percentage holds still
while a guarded branch adds one covered statement and one uncovered, and grows more
forgiving as the repository grows.

Tolerances are stated with the data that supports them: 2 minutes for sunrise, sunset
and civil twilight, 4 for nautical and astronomical because a 12–18° depression
amplifies declination error, 1 minute for moonrise and moonset, 1–2% for illumination.
Each pinned value says whether it is a **reference value** or a **regression pin** — USNO
publishes civil twilight and no deeper band, so the nautical pins are uncorroborated and
labelled as such.

`.golangci.yml` runs `default: all` and disables only what fights this repository's
deliberate design, each disable carrying a measured finding count and a reason.

## What this pass turned up

The previous regeneration recorded four doc comments left describing deleted symbols, and
the pattern behind them: **nothing in the gate reads prose.** That entry is now in
`CLAUDE.md`'s Gotchas, and the pattern repeated at smaller scale in v5.1.0 — one comment,
caught the same way, by reading the source beside the document.

| Location                     | What it still said                                                                                                                                                                                                        |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `solar.go` — `solarCrossing` | that it returns "the Julian date of solar transit, and the hour angle", and that callers build results "from jTransit and omega" — it had returned three instants in a `solarDay` since the commit that wrote the comment |

That one was introduced _by_ the change whose doc it describes, in the same commit, and
survived review and a green gate. It is the strongest argument for the Gotchas entry: the
comment most likely to go stale is the one attached to the code you are editing, because it
is the one you have stopped reading.

Five walkthrough snippets went stale in the same release, all in the solar section, and were
caught by the extract-and-compare check rather than by eye. That check is worth keeping in
the release procedure for exactly this reason — prose describing code cannot be verified by
reading the prose.

## Index

| Concept                             | File        | Symbol                         |
| ----------------------------------- | ----------- | ------------------------------ |
| Degrees-to-radians, the only place  | `trig.go`   | `sinx` … `atan2x`              |
| The clamp that must not reach cosHA | `trig.go`   | `clamp`                        |
| Julian date and its 1677–2262 bound | `epoch.go`  | `julianDate`                   |
| Longitude applied once, downstream  | `epoch.go`  | `meanSolarTime`                |
| Full nutation, for the Moon only    | `coord.go`  | `eclipticToEquatorial`         |
| Validate once                       | `dusk.go`   | `NewObserver`, `validObserver` |
| The calendar day as a type          | `dusk.go`   | `Date`, `DateIn`               |
| Polar geometry as a value           | `dusk.go`   | `Horizon`                      |
| Shared solar geometry               | `solar.go`  | `solarCrossing`                |
| Where an impossible event is found  | `solar.go`  | `solarHourAngle`               |
| Both twilight boundaries, one day   | `solar.go`  | `Twilight`                     |
| The positive lunar threshold        | `lunar.go`  | `moonAltitudeAboveHorizon`     |
| The half-open crossing interval     | `lunar.go`  | `crossingInstant`              |
| The opposite day construction       | `lunar.go`  | `MoonriseMoonset`              |
| A day in the order it is lived      | `render.go` | `renderText`, `timeline`       |
