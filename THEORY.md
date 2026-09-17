# THEORY.md

What a maintainer needs to hold in mind to change `dusk` without damaging it. Not a
tour of the files — `WALKTHROUGH.md` does that, following the call chain. This document
answers the other question: which of the decisions here are load-bearing, which are
conventional, and which are still contested.

## What this models

`dusk` models one observer looking at one sky on one day.

The domain has three entities and they are not symmetric. There is a **place** — a
latitude, a longitude, and a civil timezone, which the code calls an `Observer`. There
is a **day**, which is a civil calendar date in that place's zone, not an interval and
not an instant. And there are **events**: moments when the Sun or the Moon crosses a
particular altitude, as seen from that place, on that day.

The asymmetry that shapes everything: the Sun's crossings are computable in closed
form, and the Moon's are not. The Sun's position is a short polynomial chain — mean
anomaly, equation of centre, ecliptic longitude, declination — and the two boundaries
of the day fall out symmetrically around transit by one arccosine. The Moon needs
Meeus's Chapter 47 series, sixty periodic terms for longitude and sixty more for
latitude, and there is no algebraic way to ask when it will cross the horizon. So the
code walks the day one minute at a time and watches for the altitude to change sign.

That single fact — closed form on one side, search on the other — is why the two halves
of this library look as different as they do, and it is the first thing to check any
proposed unification against.

A third entity sits apart from both. Lunar **phase** is Sun–Earth–Moon geometry, so it
has no observer and no day: it is a property of an instant, the same for everyone alive.
`LunarPhase` is correspondingly the only entry point that takes neither an `Observer`
nor a calendar day, and it is not an oversight.

## The organizing ideas

### Degrees all the way down

Meeus writes in degrees; Go's `math` writes in radians. Rather than convert at each of
several hundred call sites, `trig.go` wraps every trig function the package uses, and
**there is no radian anywhere outside that file**. A maintainer who adds a bare
`math.Sin` to `solar.go` has not introduced a unit bug that a test will catch cleanly —
they have introduced a number that is wrong by a factor of 57 and will look like an
astronomy error.

The second half of the discipline is normalisation: an angle that has been added to or
subtracted from passes through `mod360` (or `mod24` for hours) at the point it becomes
a return value, not at the point it is consumed. This is **a convention, not an
enforced invariant**. Nothing checks it; it holds because every function that returns an
angle currently calls it.

There is exactly one exception, and it is instructive. `equatorialToHorizontal` returns
an azimuth that skips normalisation, and the pole guard inside it can produce 360°. The
reason nobody has been bitten is that nothing reads the field — which is the real
finding, and the reason the repository's answer is to delete the azimuth rather than
normalise it.

`clamp` belongs to this layer too, and its own doc comment argues against it. It pins
values to `[-1, 1]` before `asin` and `acos` so that floating-point drift in a long
degree-mode chain cannot produce NaN — at the price of silently absorbing a genuinely
wrong 1.3. The package accepts that trade explicitly and pays for it with reference-data
tests rather than runtime detection. **Weakening those tests is therefore more dangerous
here than a coverage number suggests**, because they are the only thing standing where
an assertion would otherwise stand.

### Validate once at construction, range-check at every entry

Two invariants, two different enforcement points, and confusing them is how a
maintainer adds redundant checks or removes necessary ones.

Coordinates are validated **once**, in `NewObserver`. Latitude in range, longitude in
range, neither NaN nor infinite, location non-nil. The fields are unexported, so a
validated `Observer` is the only kind that can be built — with one hole: `dusk.Observer{}`
is still spellable by a caller. `validObserver` exists solely to close it, testing
`loc == nil` at every public entry point. That is the one interior re-check in the
package, and it is not defensive programming; it is the cost of Go having no way to
forbid the zero value.

Dates are validated **every time**, and for a different reason. `julianDate` goes
through `UnixNano`, which is undefined outside roughly 1677–2262 — and undefined here
means _an arbitrary wrong number_, not zero and not a sentinel, so nothing downstream
could detect it. Hence `validJulianDateRange` at every public entry, and three times
inside `MoonriseMoonset`: the caller's instant plus both derived local midnights,
because converting to local time can push a boundary date over the edge. The sentinel
for this lives in `epoch.go` beside the check rather than with the other sentinels in
`dusk.go`, deliberately.

### A day is resolved in the observer's zone — and then two different things happen to it

This is the subtlety that costs the most when it is missed, and the repository has paid
for it more than once.

Every day-based entry point begins by resolving the caller's `time.Time` to a calendar
date **in the observer's zone** and discarding the time of day. So passing a
`time.UTC` midnight with a Detroit observer selects the previous day, silently, and
returns entirely plausible times for it. Three of the library's own README examples were
wrong in exactly this way before v4.0.0. The rule "callers must build dates in the
observer's timezone" is written in four places; it is a convention the type system does
not express, which is the standing complaint against it.

What happens **after** the date is extracted is where the halves part company, and both
answers are correct:

- **The solar path** rebuilds the day as UTC midnight. It must, because `meanSolarTime`
  applies the observer's longitude itself, after `julianDay` has rounded to an integer
  day number. Hand it a zone-adjusted instant and the longitude is applied twice.
- **The lunar path** rebuilds the day as true local midnight in `obs.loc` and converts
  to UTC, and computes the next local midnight the same way. It must, because it walks
  the day minute by minute and needs the real _length_ of the day: 1380 minutes on a
  spring-forward day, 1500 on a fall-back day, not a hard-coded 1440.

**This is the single most dangerous-looking duplication in the codebase.** The same
extraction opens all three day-based entry points — verbatim at the two solar sites, and
as the local-midnight variant at the lunar one — and a maintainer consolidating them into
one helper will pick one convention and silently break the other half. The reference tests
will catch it — and the failure will present as an algorithm bug, not a day-boundary
bug, which is the expensive way to find out. Since v4.1.0 each of the three sites names
the other convention and says why it differs, which is the cheapest available guard: the
duplication is still there, but it no longer reads as a contradiction.

### The two bodies have opposite horizon thresholds, and the Moon's is positive

The Sun is _seen_ to rise while geometrically below the horizon: refraction lifts it by
about 0.567° and its own semidiameter adds another 0.267°, so the crossing altitude is
**−0.833°**. That is `solarHourAngle`'s `h0`, and it is the number most people carry in
their heads.

The Moon inverts it, and this is the least intuitive fact in the library. It is the one
body close enough to Earth that **horizontal parallax dominates**: an observer on the
surface sees it from a different place than the geocentric coordinates assume, by about
0.951° at mean distance. Meeus's lunar threshold is

    h0 = 0.7275·π − 0.5667

which comes out to roughly **+0.125°**. The Moon's centre is _above_ the geometric
horizon when it is seen to rise.

Until v4.1.0 the code used 0.833 for both bodies, under a comment whose own arithmetic
did not reach it, and compared against `−0.833` for the Moon. The sign was backwards
relative to Meeus and the magnitude was off by about 0.96°, which biased every moonrise
early and every moonset late by 5–12 minutes. A maintainer who "simplifies" the lunar
threshold back to a shared constant will reintroduce exactly that, and the tests will
catch it only because they are now pinned to USNO rather than to the library's own
output.

The threshold also is not constant: it is recomputed per sample from the Moon's true
distance, which `lunarEclipticPosition` already returns. That is worth ±0.05° between
perigee and apogee and costs nothing.

### A reported crossing is interpolated, and that is a correctness property

The minute scan finds a _bracket_ — two samples straddling the threshold — not an
instant. Reporting the later sample was wrong in two ways, and the second was a real
defect: the scan's upper bound is closed, so a crossing detected on the final sample
carried a timestamp belonging to the **next** calendar day, in a struct documented as
holding events for a given day. It happened on about 0.17% of day-scans, and the event
was lost from the day it belonged to as well as misfiled.

The fix was not to open the bound — that loses the event from both days — but to report
the interpolated crossing. `crossingInstant` returns a time in `[cur−1m, cur)`, which is
**strictly inside the scanned day on every iteration including the last**. The closed
bound stops mattering.

Treat that half-open interval as an invariant rather than an implementation detail. It is
what `FuzzMoonriseMoonset` asserts, and it is the reason the boundary fix had to land
before the fuzz target did — written the other way round, the target would have arrived
failing on its own seed corpus and the temptation would have been to weaken the
invariant.

### Two carriers for "this did not happen", and the choice is contested

The package distinguishes an event that is _geometrically impossible_ from one that
merely _did not fall inside this calendar day_. That distinction is real and worth
keeping: at 70°N in December the Sun does not rise at all, while the Moon routinely
rises on Tuesday and sets on Wednesday because a lunar day runs about 24h50m.

Where the theory is under challenge is that the two meanings are carried by two
different **mechanisms**. The Sun uses an error — `ErrCircumpolar` or `ErrNeverRises`
returned in place of the whole result. The Moon uses a value — a zero `time.Time` for
the crossing that did not happen, plus `AboveHorizon` to say which side of the horizon
it was on.

The evidence against the split is that the only consumer reverses it. `cmd/dusk`
declares a `horizonState` type, a translator from the sentinels, a state field threaded
through two report structs, a wrapper whose whole job is separating expected polar
geometry from genuine failure, and two prose tables keyed on the state — roughly sixty
lines undoing a decision the library made about how to carry three outcomes. A second
consumer would write them again. And the error path destroys a valid result on the way
out: `computeSolarParams` has already produced the Julian date of solar transit, which
is well defined on every day at every latitude, and `SunriseSunset` discards it with
everything else. Solar noon happens during the polar night; the library knows when, and
throws it away.

Hold both readings. The distinction is principled; the carrier is not settled.

### `TwilightEvent` is asymmetric, and every consumer pays

`Twilight(D).Dusk` is the evening of day D. `Twilight(D).Dawn` is the morning of day
**D+1**. The doc comment says so, and tells callers wanting this morning's dawn to pass
yesterday's date.

Two things follow that a maintainer should know before touching the twilight path.
First, the implementation computes the whole solar parameter chain twice — once for
today's dusk and once for tomorrow's dawn — where `SunriseSunset` takes both boundaries
from one day's transit. Second, the error is all-or-nothing across two days of geometry:
near 65–70°N there are transition dates where tonight's dusk is real and tomorrow's dawn
is not, and the entire call fails. The doc comment's advice to compute each boundary
separately is an admission that the type is wrong for that case.

The reference CLI pays both costs visibly: it calls each of the three bands twice, once
with yesterday's date for the dawn and once with today's for the dusk, and discards half
of each result. Three rendered bands cost six library calls and twelve hour angles where
three would do.

### The gate is where the theory is enforced — and where it is not

`task` is the whole of quality control, and CI runs exactly it. Never a check in CI the
local gate does not run; never a tool in the gate the workflow does not install. The
toolchain is deliberately unpinned, so a red gate on untouched code is the signal
working rather than a failure to manage.

What the gate actually holds:

- **Formatting, in two halves.** gofumpt and goimports run inside golangci-lint for Go;
  prettier runs as `task docs` for Markdown and JSON. Both are declared in the
  repository rather than left to an editor, which is what makes them reproducible.
- **`default: all`, disabling only what fights this design** — and each disable carries
  a measured finding count and a reason, which is the file's own stated bar for adding
  another. Three of them (`mnd`, `exhaustruct_v5`, `gochecknoglobals`) exist because
  transcribed Meeus coefficients, staged result structs and immutable tables are what
  this package is made of.
- **Coverage as a ratchet, not a percentage.** `coverage.ratchet` records uncovered
  statements per package and the gate diffs against it, so it fails in both directions
  and on a package appearing or vanishing. An integer because a percentage holds still
  while a guarded branch adds one covered statement and one uncovered, and grows more
  forgiving as the repository grows. A moved count is often a reflow rather than lost
  coverage — blank lines split coverage blocks — so read the diff before believing it.
- **The published examples compile and their output is asserted.** `example_test.go`
  pins real times for Grand Rapids, Tromsø, Seattle and New York, which is why there is
  no `paths-ignore` on the workflow: a documentation-only push can break an example.

What the gate does **not** hold is the more useful list, because it is where drift
actually lives. Nothing compares the README's stated Go minimum against `go.mod`.
Nothing compares the twilight depression angles the CLI prints against the ones the
library uses — they are literals in two modules. Nothing checks a snippet in
`WALKTHROUGH.md` against the source it was quoted from. And the solar tests do not
enforce the 1–2 minute accuracy the README promises; their tolerances run 3, 5 and 10
minutes, and one of the two values labelled USNO is the library's own output.

**Every one of those gaps is a claim a document makes that no check can falsify.** That
is the shape of defect this repository produces, and the reason its standing documents
are re-read at release rather than trusted.

### A pin is a reference value or a regression pin, and the comment says which

The test suite is the only thing standing between a refactor of the solar chain and a
silently degraded library, and it can only do that job if a reader can tell which
assertions carry external authority.

Both kinds are legitimate. A **reference value** comes from USNO, Stellarium or Meeus and
says the library is _correct_; its tolerance should be what the reference actually
supports. A **regression pin** is the library's own output, recorded to detect drift; it
says nothing about correctness and its tolerance is arbitrary.

Conflating them is how a suite comes to look stronger than it is. Before v4.1.0 the solar
tests carried a comment attributing two values to USNO where one was wrong in both
digits, at tolerances 1.5–5× wider than the accuracy the README promised — so a change
pushing sunrise 2m30s off USNO would have broken the headline claim and left the gate
green.

The current state, and the shape to preserve: sunrise/sunset and civil twilight are real
USNO values at 2 minutes; moonrise/moonset are real USNO values at 1 minute; nautical
twilight is a **regression pin** at 4 minutes, labelled as such because USNO's public
one-day service publishes civil twilight and no deeper band, so it cannot be corroborated
the same way. Each records its measured margin, so the next reader can tell a tight test
from a lucky one.

The corollary is the standing rule: when a pinned value moves, re-derive it from the
reference, do not widen the tolerance to admit it.

### `cmd/dusk` is an executable specification

It is not a product, and reading it as one leads to the wrong changes. Its purpose is
that every exported function has a caller and every documented edge case is reachable
with a single flag — so the CLI is where the library's contract is demonstrated to be
usable, and where its awkwardness shows up first.

Its exit contract is part of the specification: **polar geometry is a result, so the
report renders and exits 0.** Only a misused command line and an out-of-range date exit
1, and they are distinguished by the message (`usage:` versus `unsupported date:`)
rather than by the status. `main` is four lines; everything testable lives in a `run`
that takes its streams as parameters, which is what keeps every branch reachable from a
test.

Read the other way, the CLI is the library's bug report. Three of the sharpest open
findings are things it had to work around: the seven-line comment in `parseDate`
explaining why the date is anchored at midday, the `horizonState` machinery, and the
double twilight call.

## The seams

**Solar/lunar** is principled. Closed form versus series-plus-search is a real
difference in the mathematics, and it justifies the separate files, the separate
day-construction, and the 1–2 ms per moonrise call. Do not try to unify them.

**`trig.go` → `epoch.go` → everything** is principled and strictly one-directional.
`trig.go` depends on nothing, `epoch.go` on `trig.go` alone, `solar.go` and `lunar.go`
on both. Nothing reaches back up.

**The time layer and the coordinate layer are now separate files**, and the split is
principled rather than cosmetic. `epoch.go` is consumed by everything; `coord.go` is
reached by exactly one call chain, the Moon's minute scan. They sat at different depths
inside one file until v4.1.0, and no reading order could keep them together — the
walkthrough had to introduce one before `dusk.go` and the other after `LunarPhase`.
Splitting them changed no statement. If they are ever merged back, that reading problem
returns with them.

**The library/CLI boundary is where the theory is thinnest.** It is the only place two
independently reasonable designs meet, and it is where the highest-severity structural
findings cluster — the error-versus-value carrier, the twilight asymmetry, the
depression angles duplicated across a module boundary. A maintainer looking for the
next significant change should look here first.

**Zero external dependencies is a boundary too**, and `depguard` runs in strict mode
allowing only `$gostd` and this module. The Meeus tables are transcribed into the source
rather than fetched. This is not frugality: the coefficient tables _are_ the algorithm,
and a dependency that supplied them would make the astronomy someone else's to get
right.

## What this is shaped to accommodate

**A new twilight band costs one line.** The shared `twilight` is fully parameterised by
depression angle; only the three exported wrappers fix 6, 12 and 18. A blue hour at 4°
or an aviation band would slot in — plus an entry in the CLI's own table, which is
precisely the duplication that argues for exporting the parameter instead.

**A new output format costs nothing in the library.** `cmd/dusk` already renders text
and JSON from the same assembled report, and the assembly is separate from both.

**A new celestial body would not fit.** The package's vocabulary is Sun and Moon by
name, from `solarHourAngle` to `moonAltitudeAboveHorizon`, and the horizon-crossing
threshold is per-body — a constant for the Sun, a distance-dependent function for the
Moon — rather than a parameter. Adding planets means generalising the position source,
the threshold and the day-scan together: a rewrite of the interior, not an addition.

**Sub-minute precision is partly there and partly not.** Interpolation removed the
one-minute quantization from the _reported_ instant, and the result agrees with USNO
within 32 seconds across a 28-event sample. What the minute scan still cannot do is
_detect_ an event shorter than its step — the Moon grazing the horizon at high latitude
is invisible to it by construction, and no amount of interpolation recovers a bracket
that was never sampled.

**Dates outside 1677–2262 would require replacing `julianDate`.** The bound is
`UnixNano`'s, not astronomy's, and the whole range-check apparatus exists to keep
callers on the right side of it.

Where a maintainer who did not understand the theory would do damage, in order of
likelihood: consolidating the two day-constructions; adding `math.Sin` to a file outside
`trig.go`; applying `clamp` to the hour-angle cosine, which would turn an impossible
event into a plausible time; loosening a reference-data tolerance to make a change pass;
and pinning a tool to silence the gate.

## Uncertainties

Where I am reading intent from code and could be wrong. Five entries stood here before
v4.1.0 and **two of them are now settled**; a sixth question, about `epoch.go`'s two
layers, is carried down from the Seams section because splitting the file settled it too.
Settled entries are kept, struck through, with what settled them — a resolved uncertainty
is worth more than a deleted one, because it tells the next reader the question was asked
and answered rather than never noticed.

**Whether the solar `−0.83` was chosen or inherited.** _Still open._ It is the standard
Meeus `h0` for the Sun — refraction plus semidiameter — and it is right. The lunar
constant that sat beside it was demonstrably the solar value given a lunar-sounding
justification after the fact, so one of the two was copied; that one is now fixed, but I
still cannot tell from the code whether the solar figure was arrived at independently or
happens to be correct for the same borrowed reason.

**What `AboveHorizon` is measured against.** _Still open, and sharper now._ It compares
the Moon's altitude to the same threshold the scan uses, so it reports whether the Moon
was _visibly_ up at local midnight rather than geometrically above the horizon. That is
defensible and agrees with the rise and set times it sits beside. Since v4.1.0 the
threshold is parallax-corrected and **positive**, which makes the field slightly stricter
than it was — the Moon must be a little higher to count as up. No caller has complained
because the reference tests do not probe the boundary case, and nothing documents which
reading is intended.

**Whether the twilight asymmetry is a contract or an accident.** _Still open._ The doc
comment describes it confidently enough to read as a decision, and the CLI calls it "the
single most easily missed detail in the library's contract" — which is how you describe
something you have accepted. But the implementation's shape suggests it fell out of
computing tomorrow separately rather than being chosen, and nothing records the choice
being made. This is now the largest unresolved design question in the library.

**~~Whether the v3/v4 API shrink was finished or merely paused.~~** _Settled: it was
paused._ `solarPosition` and `solarMeanAnomalyFromCentury` were residue, and the doc
comment arguing for a rounded-versus-continuous solar position described a distinction
the package does not act on. Both are gone. The pattern recurred immediately —
`lunarPosition` was orphaned by the parallax fix in the same release — which suggests the
real lesson is not about v3 at all: **an unexported function with a test is invisible to
`unused`**, so the gate cannot tell you when the last production caller goes away. Check
by hand after removing a call site.

**~~How much of the lunar error budget is method and how much is the missing parallax.~~**
_Settled: almost all of it was the parallax._ The README attributed a ~20 minute spread to
the simplified approach plus the one-minute scan. Measured against published USNO values
over 28 rise/set events from 55°S to 64°N across all four seasons, the corrected library's
largest disagreement is **32 seconds**, and USNO publishes only to the minute. The series
truncation and the scan were never the dominant terms. The general lesson is worth
keeping: this library's documented error bars were inherited assumptions, not
measurements, and the one that was checked turned out to be an order of magnitude
pessimistic while concealing a systematic bias.

**~~Whether `epoch.go`'s two layers were a deliberate consolidation.~~** _Carried down
from Seams, and settled by splitting them._ Nothing in the repository stated the file count was deliberate, and the
coordinate layer now lives in `coord.go` under a header naming its one consumer.

## Index

The issues this theory refers to. The v4.1.0 milestone closed sixteen findings; what
remains open is the v5.0.0 set, which is where the library/CLI boundary — the thinnest
part of the theory — is actually addressed.

**Open:**

| Issue                                              | What it is                                                                        |
| -------------------------------------------------- | --------------------------------------------------------------------------------- |
| [#63](https://github.com/philoserf/dusk/issues/63) | The day hazard left as a convention instead of a type                             |
| [#66](https://github.com/philoserf/dusk/issues/66) | `LunarPhaseInfo` publishing two one-line derivations of a third                   |
| [#70](https://github.com/philoserf/dusk/issues/70) | Error-as-carrier, reversed by its only consumer, destroying solar noon on the way |
| [#76](https://github.com/philoserf/dusk/issues/76) | The depression angle hidden behind three names                                    |
| [#77](https://github.com/philoserf/dusk/issues/77) | Twilight spanning two days, and the double call it forces                         |
| [#96](https://github.com/philoserf/dusk/issues/96) | Rendered times truncate their seconds, now the largest error in the output        |

**Closed in v4.1.0**, and referred to above where they changed the theory:

| Issue                                              | What it was                                                       |
| -------------------------------------------------- | ----------------------------------------------------------------- |
| [#69](https://github.com/philoserf/dusk/issues/69) | The lunar threshold's missing horizontal parallax                 |
| [#67](https://github.com/philoserf/dusk/issues/67) | The scan closed at both ends                                      |
| [#75](https://github.com/philoserf/dusk/issues/75) | Two fuzz targets that asserted nothing                            |
| [#72](https://github.com/philoserf/dusk/issues/72) | Solar tolerances wider than the documented accuracy               |
| [#62](https://github.com/philoserf/dusk/issues/62) | The azimuth nothing read                                          |
| [#65](https://github.com/philoserf/dusk/issues/65) | …and the pole guard that could return 360°, closed by deleting it |
| [#64](https://github.com/philoserf/dusk/issues/64) | `epoch.go`'s two layers                                           |
| [#73](https://github.com/philoserf/dusk/issues/73) | `solarPosition`, the v3 shrink's residue                          |
| [#74](https://github.com/philoserf/dusk/issues/74) | …and the question of whether it was residue at all                |
| [#78](https://github.com/philoserf/dusk/issues/78) | Two day-constructions, neither comment naming the other           |
| [#79](https://github.com/philoserf/dusk/issues/79) | One solar mean anomaly formula under two names                    |
| [#61](https://github.com/philoserf/dusk/issues/61) | README's Go minimum against `go.mod`                              |
| [#71](https://github.com/philoserf/dusk/issues/71) | README advertising a removed phase angle                          |
| [#68](https://github.com/philoserf/dusk/issues/68) | `MoonEvent` promising a removed duration                          |
| [#81](https://github.com/philoserf/dusk/issues/81) | `wrapAt`'s width budget excluding the caller's indent             |
| [#80](https://github.com/philoserf/dusk/issues/80) | A workflow action pinned by mutable tag                           |
