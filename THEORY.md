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
bug, which is the expensive way to find out. Only the lunar site explains its own
choice; neither mentions the other.

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

**Inside `epoch.go` is a historical accident being lived with.** The file holds a time
layer that everything sits on and a coordinate layer consumed by exactly two call
chains, and the two sit at different depths. The file records the merge that produced
it in a banner comment naming a file that no longer exists. No reading order fixes this,
which is worth knowing before concluding your own reading is at fault.

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
name, from `solarHourAngle` to `lunarHorizonDepression`, and the horizon-crossing
threshold is a per-body constant rather than a parameter. Adding planets means
generalising the position source, the threshold and the day-scan together — a rewrite
of the interior, not an addition to it.

**Sub-minute precision would require rethinking the lunar method.** The minute scan's
resolution is its accuracy floor, and events shorter than a minute — the Moon grazing
the horizon at high latitude — are invisible to it by construction.

**Dates outside 1677–2262 would require replacing `julianDate`.** The bound is
`UnixNano`'s, not astronomy's, and the whole range-check apparatus exists to keep
callers on the right side of it.

Where a maintainer who did not understand the theory would do damage, in order of
likelihood: consolidating the two day-constructions; adding `math.Sin` to a file outside
`trig.go`; applying `clamp` to the hour-angle cosine, which would turn an impossible
event into a plausible time; loosening a reference-data tolerance to make a change pass;
and pinning a tool to silence the gate.

## Uncertainties

Where I am reading intent from code and could be wrong.

**Whether the solar `-0.83` was chosen or inherited.** It is the standard Meeus `h0` for
the Sun — refraction plus semidiameter — and it is right. But the lunar constant beside
it, 0.833, is demonstrably the solar value given a lunar-sounding justification after
the fact, with the Moon's horizontal parallax missing entirely. One of the two was
copied. I cannot tell from the code whether the solar one was arrived at independently.

**What `AboveHorizon` is measured against.** It compares the Moon's altitude to the same
refraction-corrected threshold the scan uses, so it reports whether the Moon was
_visibly_ up at local midnight rather than geometrically above the horizon. That is
defensible and probably intended — it agrees with the rise and set times it sits beside
— but nothing says so, and a caller could reasonably read the field either way.

**Whether the v3/v4 API shrink was finished or merely paused.** Several unexported
functions survive with no caller, and one of them carries a doc comment arguing for a
design distinction the package does not act on. The residue is documented; what I cannot
tell is whether what remains was kept deliberately or simply not reached.

**Whether the twilight asymmetry is a contract or an accident.** The doc comment
describes it confidently enough to read as a decision, and the CLI's comment calls it
"the single most easily missed detail in the library's contract" — which is how you
describe something you have accepted. But the implementation's shape suggests it fell
out of computing tomorrow separately rather than being chosen, and nothing records the
choice being made.

**How much of the lunar error budget is method and how much is the missing parallax.**
The README attributes the ~20 minute spread to the simplified approach plus the
one-minute scan. The parallax omission alone accounts for 5–12 minutes of systematic
bias. Whether the remainder is the series truncation, the scan, or something else is not
something I can settle without reference data the repository does not carry.

## Index

Everything this pass found that is actionable was already filed; no new findings went to
`.issues/`. The open issues this theory refers to, in the order they appear above:

| Issue                                              | What it is                                                        |
| -------------------------------------------------- | ----------------------------------------------------------------- |
| [#65](https://github.com/philoserf/dusk/issues/65) | The unnormalised azimuth, and the pole guard that can return 360° |
| [#62](https://github.com/philoserf/dusk/issues/62) | …which nothing reads, so it should be deleted rather than fixed   |
| [#63](https://github.com/philoserf/dusk/issues/63) | The day hazard left as a convention instead of a type             |
| [#78](https://github.com/philoserf/dusk/issues/78) | Two day-constructions, neither comment naming the other           |
| [#70](https://github.com/philoserf/dusk/issues/70) | Error-as-carrier, reversed by its only consumer                   |
| [#77](https://github.com/philoserf/dusk/issues/77) | Twilight spanning two days, and the double call it forces         |
| [#76](https://github.com/philoserf/dusk/issues/76) | The depression angle hidden behind three names                    |
| [#61](https://github.com/philoserf/dusk/issues/61) | README's Go minimum against `go.mod`                              |
| [#72](https://github.com/philoserf/dusk/issues/72) | Solar tolerances wider than the documented accuracy               |
| [#64](https://github.com/philoserf/dusk/issues/64) | `epoch.go`'s two layers                                           |
| [#69](https://github.com/philoserf/dusk/issues/69) | The lunar threshold's missing horizontal parallax                 |
| [#67](https://github.com/philoserf/dusk/issues/67) | The scan closed at both ends                                      |
| [#74](https://github.com/philoserf/dusk/issues/74) | `solarPosition`, the shrink's residue                             |
| [#75](https://github.com/philoserf/dusk/issues/75) | Two fuzz targets that assert nothing                              |
