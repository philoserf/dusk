# Changelog

## v5.1.0 — 2026-09-17

Two accuracy fixes and a hygiene fix. **Every solar time in this release moves** — sunrise,
sunset and all three twilight bands. No lunar value moves, and no exported signature changes.

### Sunrise and sunset are solved independently, not mirrored

`SunriseSunset` placed rise and set symmetrically about solar transit, which assumes the
Sun's declination is the same in the morning as in the evening. Near an equinox it moves
about 0.4° a day, so the afternoon half-day genuinely is shorter — and the mirrored
construction reported the two as equal **to the second, by definition**.

Each boundary is now re-solved against the declination at its own instant. The refraction
constant is corrected from `-0.83` to `-0.8333`, the value Meeus and USNO use.

Measured against published USNO values across twelve place/date pairs, `library − USNO` on
sunset, in seconds:

| place, date                        | before | after    |
| ---------------------------------- | ------ | -------- |
| Grand Rapids 2026-09-17            | +68    | +47      |
| Seattle 2026-09-17                 | +86    | +60      |
| Oslo 2026-09-17                    | +148   | +107     |
| 65°N 2026-09-17                    | +167   | +115     |
| 68°N 2026-09-17                    | +176   | **+115** |
| Oslo 2026-12-21 (solstice control) | −8     | −6       |
| Oslo 2026-03-20                    | −86    | −43      |

Worst case falls from 2m56s to 1m55s, which brings the accuracy `README.md` documents back
inside its own claim. The solstice control does not move: the fix leaves alone the case that
was already right.

**Sunrise gets worse, and you should know that before upgrading.** The mirrored construction
put the entire error on sunset, which left sunrise accidentally accurate; correcting the
geometry distributes it. Oslo's sunrise goes from −3s to −47s, 68°N's from −29s to −94s. The
model is more correct and one published quantity is less accurate — the same trade v4.1.0
made when the lunar parallax fix moved every moonrise later.

**About half the asymmetry remains.** Oslo's afternoon is 180 seconds shorter than its
morning; this release reports 86, up from zero. The hour angle is still measured about a
transit computed once for the day, so the Sun's motion in right ascension is unmodelled.
Tracked in [#114](https://github.com/philoserf/dusk/issues/114), with measurements.

**Twilight moves too.** v5.0.0 gave `Twilight` the same shared geometry, so all three bands
get the same correction and the same trade. Oslo's civil dusk improves from 149s to 100s;
its civil dawn degrades from 4s to 53s.

### Rendered times round instead of truncating

`"15:04"` drops the seconds, so every displayed time ran up to 59 seconds early — one-sided,
always. `cmd/dusk`'s **text** report now rounds. Its **JSON does not**: that is the
machine-readable answer and keeps its seconds, so the two renderings of one event may differ
by up to half a minute, by design.

Three of the four published examples now print exactly what USNO publishes, where they
previously printed a minute early:

| example                      | before        | after         | USNO          |
| ---------------------------- | ------------- | ------------- | ------------- |
| `ExampleSunriseSunset`       | 06:03         | 06:04         | 06:04         |
| `ExampleTwilight` (dawn)     | 04:30         | 04:31         | 04:31         |
| `ExampleSunriseSunset_polar` | 12:45         | 12:46         | 12:46         |
| `ExampleMoonriseMoonset`     | 10:10 / 22:06 | 10:11 / 22:07 | 10:11 / 22:07 |

`ExampleSunriseSunset` was the one nobody had noticed — its siblings documented the
truncation and it did not, so it had been advertising a sunrise a minute earlier than the
reference it was derived from, from a value 14 seconds off.

Callers formatting the library's own return values should round too; the examples now
demonstrate that rather than the truncation.

### Also

- `.prettierignore` now covers `.planning/` as well as `.issues/`. Both are hidden by a
  global ignore that prettier does not read, so the gate failed on files the repository does
  not carry.

### Upgrading

No signature changes and no import-path change: `go get -u` is the whole migration. If you
have pinned sunrise, sunset or twilight times in your own tests, expect them to move by up
to about a minute and re-derive them from a reference rather than from this library.

## v5.0.0 — 2026-09-17

**The import path is now `github.com/philoserf/dusk/v5`.** Every exported change below is
a break; none of them changes a computed value. Sunrise, sunset, twilight, moonrise,
moonset and illumination are bit-identical to v4.1.0 for every input that produced an
answer in v4.1.0 — and two inputs that produced a wrong answer, or no answer, now produce
the right one.

Four reshapes, each removing a rule the documentation had to state instead of the type
system.

### The calendar day is a type

`SunriseSunset`, `Twilight` and `MoonriseMoonset` take a `Date{Year, Month, Day}` instead
of a `time.Time`. `DateIn(t, loc)` converts an instant.

They always used only the calendar day, resolved in the observer's zone, and discarded the
rest — so a `time.UTC` midnight with a Detroit observer silently selected the previous day
and returned plausible times for it. Three of this library's own README examples were
wrong that way before v4.0.0, and the rule "build dates in the observer's timezone" ended
up written in five separate places. All five are gone.

`LunarPhase` keeps its `time.Time`: it uses the whole instant and takes no `Observer`.
Two entry-point shapes for two kinds of question.

```go
// before
dusk.SunriseSunset(time.Date(2025, 6, 21, 0, 0, 0, 0, loc), obs)

// after
dusk.SunriseSunset(dusk.Date{Year: 2025, Month: time.June, Day: 21}, obs)
dusk.SunriseSunset(dusk.DateIn(someInstant, loc), obs)
```

### Polar geometry is a value, not an error

`ErrCircumpolar` and `ErrNeverRises` are removed. `SunEvent` and `TwilightEvent` carry a
`Horizon` — `Crosses`, `StaysAbove` or `StaysBelow` — with `Rise`, `Set`, `Dawn` and
`Dusk` zero unless the body crossed. `error` now means a nil location, bad coordinates, or
a date outside the Julian range.

**This recovers information v4 destroyed.** Solar transit is defined on every day at every
latitude, and the error return discarded it. `SunEvent.Noon` is now always set, and
`Duration` is 24h under the midnight sun and 0 through the polar night. Tromsø on 21
December gains a row it could not previously print:

```
  09:31   Civil dawn
  11:42   Solar noon
  13:53   Civil dusk
```

The names changed with the carrier because the old ones were false at a depression angle:
`ErrCircumpolar` said "always above the horizon" and at 18° meant the Sun never got 18°
_below_ it. `StaysAbove` and `StaysBelow` read the same way at any angle.

`MoonEvent` is unchanged. `AboveHorizon` answers a different question — which side of the
horizon the Moon was on when the day began — and is not the same fact as whether a
crossing was possible.

### Twilight is one calendar day

`CivilTwilight`, `NauticalTwilight` and `AstronomicalTwilight` are replaced by
`Twilight(date, obs, depression)`. `TwilightEvent.Dawn` and `.Dusk` are now **both on the
day you asked for**; `NightDuration` is removed.

`Twilight(D).Dawn` used to be the morning of D+1, which meant callers wanting this
morning's dawn passed yesterday's date — a rule stated in five places and reversed by the
only consumer, which called each band twice and discarded half of each result.

**This fixes a real defect, not just a shape.** The two-day form failed if _either_ day's
geometry was impossible, and the failure took a valid result with it. At 75°N on
2025-11-26 the civil row reported a dawn of `11:28:33Z` beside the note "this dark all day
— the sun stays below 6 degrees". Both could not be true: that day does cross −6°, and its
real dusk at `12:06:27Z` was discarded because the call failed on **Nov 27's** dawn. A
same-day event is symmetric about transit, so both boundaries exist or neither does, and
the transition lands between days.

Overnight darkness spans midnight and so is no longer a field. Subtract:

```go
tonight, _ := dusk.Twilight(d, obs, 18)
tomorrow, _ := dusk.Twilight(dusk.Date{d.Year, d.Month, d.Day + 1}, obs, 18)
night := tomorrow.Dawn.Sub(tonight.Dusk)
```

Passing `depression = 0` gives sunrise and sunset, refraction included.

### `LunarPhaseInfo` loses two derived fields

`DaysApprox` and `Waxing` are removed. Both were restatements of `Elongation`, recoverable
in one expression each — `Elongation < 180` and `Elongation / 360 * 29.53059` — and both
recoveries are in the doc comment.

`DaysApprox` is worth a note: elongation does not advance linearly in time, so a linear
rescaling was never the lunation age the name promised. Its own comment said "rough". The
expression survives where writing it is a choice to accept the approximation, rather than
a number handed over under a misleading name.

This is the third pass of the same shape — v4.0.0 removed `LunarPhaseInfo.Angle` and four
`String()` methods on identical grounds. The README's `Angle` recovery note was signed by
`Waxing`, so it is re-signed by `Elongation`; all three removals are now one table.

### Also

- `cmd/dusk --json` drops `daysApprox` and `waxing` from the phase object, and reports
  `night` only on the deepest band that has one — the only band the summary ever printed.
  Polar reports gain `noon`, and midsummer gains `daylight`.
- The reference CLI is about 100 lines smaller. `horizonState`, `stateOf`, `callTwilight`,
  `twilightFunc` and `parseDate`'s midday anchor are all gone, each of them machinery that
  existed to undo a library decision.
- `FuzzSunriseSunset` asserted its invariants only on days that returned no error, so
  above the Arctic circle it asserted nothing. It now checks every day.

### Upgrading

`pkg.go.dev` reports no importers of `/v4`. If you have an unpublished one, the four
mechanical edits are: the import path, `Date` at three call sites, `Horizon` in place of
`errors.Is` on the two removed sentinels, and `Twilight(d, obs, 6|12|18)` in place of the
three named wrappers.

## v4.1.0 — 2026-09-17

The astronomy **changed**. v4.0.0 shipped with the note that every calculation returned
exactly what v3.0.0 returned; this release does not. Moonrise and moonset move by 5–12
minutes, because the threshold they were measured against was the Sun's.

Everything else is the standing documents catching up with the code, and the tests
catching up with what the documents promise.

### Bug fixes

- **The moonrise/moonset threshold omitted the Moon's horizontal parallax.** The constant
  was 0.833° — Meeus's `h0` for the **Sun** — carrying a comment whose own arithmetic
  (refraction 0.566 + semidiameter 0.25 = 0.816) did not reach it. The Moon is the one body
  close enough that parallax dominates, and Meeus gives it `h0 = 0.7275·π − 0.5667`, about
  **+0.125°**: its centre is _above_ the geometric horizon when it is seen to rise. The sign
  was backwards and the magnitude was off by about 0.96°, biasing every rise early and every
  set late.

  Measured against published USNO values over 28 rise/set events from 55°S to 64°N across
  all four seasons, the largest disagreement is now **32 seconds**, where before it ran to
  5–12 minutes in a consistent direction:

  | date / place      | before        | after         | USNO          |
  | ----------------- | ------------- | ------------- | ------------- |
  | 2024-01-15 NYC    | 10:06 / 22:13 | 10:11 / 22:07 | 10:11 / 22:07 |
  | 2024-01-20 NYC    | 12:19 / 03:03 | 12:25 / 02:56 | 12:25 / 02:56 |
  | 2024-01-25 NYC    | 16:50 / 07:40 | 16:56 / 07:34 | 16:56 / 07:34 |
  | 2024-01-15 Sydney | 09:50 / 23:00 | 09:51 / 23:00 | 09:51 / 23:00 |

  `h0` is recomputed per sample from the Moon's true distance, which the position series
  already returned and discarded, so perigee/apogee variation (±0.05°) comes along free.

- **A `MoonEvent` for one day could carry a time on the next.** The minute scan ran
  `i <= scanMinutes`, sampling the next local midnight itself, so a crossing detected there
  was filed under the wrong calendar day — and lost from the day it belonged to. Measured at
  33 out of 19,200 day-scans, about 0.17%.

  Fixed by reporting the **interpolated crossing** rather than the sample that detected it.
  The reported instant now lands in `[cur−1m, cur)`, strictly inside the scanned day on
  every iteration including the last, which also removes the one-minute quantization the
  doc comment used to concede. Opening the loop bound instead would have lost the event
  from both days.

- **`wrapAt`'s width budget excluded the caller's own indent**, so a wrapped condition's
  first line ran two columns past its continuations and the paragraph's right edge was
  ragged. The indent is now the function's business alone, on every line, so `width` means
  the same thing throughout.

### Removed

Unexported only — no exported symbol changed, added or disappeared in this release.

- **`solarPosition`** — residue of the v3 API shrink, with no caller but its own two tests
  and a doc comment arguing for a rounded-versus-continuous design the package does not
  have. Every function in the chain it composed is pinned harder elsewhere.
- **The azimuth half of `equatorialToHorizontal`**, which nothing read and which ran on all
  ~1441 iterations of every moonrise scan. The function is now `altitudeOf`, returning a
  bare `float64`, and the two-field `horizontal` struct is gone with it. This also resolves
  a filed defect in the deleted code: the pole guard could return an azimuth of 360°.
  Measured on its own, this removal improves `BenchmarkMoonriseMoonset` about 8%. The
  parallax fix above then spends part of that back — an `asinx` and a division per sample —
  so the **net for the release is about 6%**, from ~1.35 ms at v4.0.0 to ~1.27 ms
  (5×30 iterations, steady state, Apple M-series).
- **`solarMeanAnomalyFromCentury`** — the same formula as `solarMeanAnomaly` with the time
  argument scaled, contradicting the convention `CLAUDE.md` states outright. The two
  coefficients differ by 0.53 arcseconds across the whole valid date range, four orders of
  magnitude inside the nearest reference tolerance.
- **`lunarPosition`**, left without a production caller by the parallax fix, which needs the
  distance that `lunarPosition` discards. Its Meeus p. 342 reference value is kept and
  re-pointed at the composition the scan actually performs.

### Tests

- **The sun and moon fuzz targets now assert.** Both took `_ *testing.T` and dropped their
  results, so the entire verdict was "did not panic". The Sun's target now checks
  `Rise < Noon < Set`, non-zero times and positive duration; the Moon's checks that any
  non-zero time falls inside the scanned local day — the invariant the boundary bug above
  violated, which is why that fix had to land first.
- **The solar reference tests are pinned to real USNO values at tolerances the data
  supports.** One pinned value was attributed to USNO under a comment that contradicted both
  it and another comment three hundred lines down; checked against USNO's published data,
  the comment was wrong in _both_ digits. Sunrise/sunset and civil twilight now carry true
  USNO values at **2 minutes** (was 3 and 5), with measured margins of 16–63 seconds
  recorded in the comments.
- **Nautical twilight is labelled a regression pin**, because USNO's one-day service
  publishes civil twilight and no deeper band, so its values cannot be corroborated. Its
  tolerance tightens from 10 minutes to **4**, with the measured margin stated — twilight's
  own figure rather than a loose reading of sunrise's.
- The lunar pins become genuine USNO references and tighten from ±5m and ±20m to **±1m**.

### Documentation

- `README.md` required Go 1.24+ for a module whose `go` directive is 1.27 — the one drift
  here that cost a reader something before they ran anything.
- `README.md`'s API list still advertised the phase angle removed in v4.0.0, contradicting
  its own **Result types** section thirteen lines below. The recovery route moves to where a
  caller would look for it.
- `MoonEvent`'s doc comment still promised the duration field removed in v3.0.0 — and that
  field was removed precisely because a naive `Set.Sub(Rise)` is negative when the Moon is
  up at midnight, so the comment was inviting the bug it was deleted for.
- The three day-construction sites now name each other and say why they differ. Both
  conventions are correct for opposite reasons, and read in call order the second looked
  like a contradiction of the first.
- `THEORY.md`, `WALKTHROUGH.md`, `README.md` and `CLAUDE.md` rebuilt against the current
  source. Two of `THEORY.md`'s five open uncertainties are now settled — the v3/v4 API
  shrink, and the lunar error budget, which turned out to be almost entirely the missing
  parallax rather than the method. A sixth question, `epoch.go`'s two layers, moved down
  from the Seams section and was settled by splitting the file.

### Internal

- **`epoch.go` split into `epoch.go` and `coord.go`.** The file held a time layer that
  everything sits on and a coordinate layer reached by one call chain; no reading order
  could keep them together. The three surviving "moved from …" banner comments, each naming
  a file that no longer exists, are rewritten to say what their section _is_.
- `golangci/golangci-lint-action` is pinned by commit digest, which `ci.yml`'s own comment
  had stated as the policy while not keeping it. All three `uses:` lines are now digests.

### Known issues

- Rendered times **truncate** their seconds rather than rounding. With the algorithm now
  accurate to seconds, this is the largest remaining error in anything displayed with a
  minute layout — up to 59 seconds, one-sided, always early. `ExampleMoonriseMoonset` shows
  it: the library computes 10:10:53 where USNO publishes 10:11, and `Format("15:04")` prints
  `10:10`. Tracked as [#96](https://github.com/philoserf/dusk/issues/96); rounding is a
  change to the rendering contract and was deliberately not made here.

## v4.0.0 — 2026-09-06

Adds `cmd/dusk`, and completes the API shrink that v3 began. The astronomy is
unchanged: every calculation returns exactly what v3.0.0 returned.

### Breaking changes

- **Module path** changed from `github.com/philoserf/dusk/v3` to `github.com/philoserf/dusk/v4`
- **Result types no longer implement `fmt.Stringer`** — `SunEvent`, `MoonEvent`, `TwilightEvent` and `LunarPhaseInfo` lose their `String()` methods. Nothing in the library or the reference CLI consumed them; `cmd/dusk` formats every field itself. Callers who printed a result value directly should format the fields they want. `Observer.String()` is unaffected.
- **`LunarPhaseInfo.Angle` removed** — the Meeus phase angle was published but unused, including by the reference CLI. `Illumination` is derived from it and is unchanged; callers needing the angle can recover it as `acos(2*Illumination/100 - 1)`, signed by `Waxing`.

### Added

- **`cmd/dusk`, a reference CLI.** Reports a full day — sun, three twilight bands, moon, phase — for one place and date, as text or JSON. It calls every exported function, and every documented edge case is reachable with a single flag.

  ```bash
  go install github.com/philoserf/dusk/v4/cmd/dusk@latest
  dusk --lat 42.9634 --lon -85.6681 --tz America/Detroit --date 2025-06-21
  ```

  Events are rendered in **clock order** rather than grouped by the call that produced them, so a twilight table's dawn column no longer runs backwards and a moonset belonging to the previous night's rise no longer appears above the moonrise it precedes. Polar geometry is a result, not a failure: the report renders and exits 0. Only a misused command line and an out-of-range date fail.

### Bug fixes

- **Twilight bands no longer report yesterday's geometry.** A band's state was seeded from yesterday's call and overwritten only when tonight also failed, so on a polar transition day the report could claim "civil, nautical and astronomical twilight never arrives" directly above a real civil dusk (65°N, 2025-07-28, civil dusk 01:05, civil night 48m).
- **`--date` no longer reports the previous day where clocks move at midnight.** `ParseInLocation` anchors at 00:00, which does not exist in `America/Santiago` in September or `America/Havana` in March, and Go resolves it backwards. Dates are now anchored at midday in the observer's zone. This also samples the lunar phase at local noon rather than local midnight.
- **A Moon above the horizon all day is no longer reported as absent.** `MoonriseMoonset` signals that with zero rise and set plus `AboveHorizon`, not with a sentinel; the text renderer consulted only the times and disagreed with the JSON beside it.
- Whole-day twilight claims are scoped to tonight, since this morning's dawn comes from a separate call and can be real when tonight's is not.

### Documentation

- `julianDate`'s doc comment claimed `UnixNano` returns 0 outside its range. It does not — the result is undefined and wraps to an arbitrary value. The old wording could invite a maintainer to skip a range check on a new call path (#54).
- `walkthrough.md` rebuilt against the current source and extended to cover `cmd/dusk`; every snippet is verified executable.
- `THEORY.md` extended with the reference implementation's theory.
- Fixed three README examples that built dates with `time.UTC` while passing a non-UTC observer — because the library derives the day via `date.In(obs.loc)`, the sunrise example computed June 20 while claiming the solstice.

### Internal

- `MoonriseMoonset` now calls `lunarPosition` instead of reimplementing it inline twice (#53).
- Coverage is held by a ratchet — uncovered statements per package, checked in, diffed both ways — replacing an 80% threshold that had permitted 46 uncovered statements to appear silently. CI runs exactly `task`.
- Go directive raised to 1.27; golangci-lint moved to `default: all`.
- Removed ~158 lines of tests that exercised the standard library rather than the astronomy, and inlined three single-use helpers (#58).
- `.golangci.yml`'s depguard allow-list and `coverage.ratchet`'s keys both name the module path; both were updated with the bump. Uncovered-statement counts are unchanged at 10 and 19.

## v3.0.0 — 2026-03-30

### Breaking changes

- **Module path** changed from `github.com/philoserf/dusk/v2` to `github.com/philoserf/dusk/v3`
- **`NewObserver` constructor** replaces direct struct construction — validates at creation, rejects NaN/Inf/nil
- **`Observer` fields unexported** — `Lat`/`Lon`/`Loc` → `lat`/`lon`/`loc`; use `NewObserver` to construct
- **Elevation removed** — `Observer.Elev` field deleted; elevation correction (~0.5' for typical altitudes) dropped from `solarHourAngle` for simplicity
- **`LunarPhase` returns `(LunarPhaseInfo, error)`** — now validates date range; callers must handle the error
- **`MoonEvent.Duration` removed** — was incorrect when Moon set before rise; callers should compute from Rise/Set as needed
- **Public API surface reduced** — removed `ObjectTransit`, `Transit`, `SolarPosition`, `LunarPosition`, `LunarEclipticPosition`, `EclipticToEquatorial`, `EquatorialToHorizontal`, `HourAngle`, `AngularSeparation`, `JulianDate`, `ValidJulianDateRange`, `LocalSiderealTime`
- **Coordinate types unexported** — `Equatorial`, `Horizontal`, `Ecliptic` → `equatorial`, `horizontal`, `ecliptic`
- **`TwilightEvent.Duration` renamed to `NightDuration`** — clarifies this is the overnight darkness period, not daylight
- **Sentinel errors are now constants** — `ErrCircumpolar`, `ErrNeverRises` use an unexported `errString` type; immutable, no longer reassignable
- **Newly exported errors** — `ErrNilLocation`, `ErrInvalidCoord` (were unexported in v2), `ErrNonFiniteCoord` (new, for NaN/Inf inputs); `ErrDateOutOfRange` remains exported but is now a constant

### Improvements

- `eclipticToEquatorial` now applies full nutation (both Δψ and Δε), improving RA accuracy by up to ~17"
- Observer validation (NaN/Inf rejection) happens once at construction, not repeated in every function call
- Date range validation (`validJulianDateRange`) at all public entry points
- `SunriseSunset` and `twilight` normalize input to UTC midnight — safe for any time-of-day
- `MoonEvent.AboveHorizon` field indicates whether the Moon was above the horizon at start of day
- `Observer` has `Lat()`, `Lon()`, `Location()` accessors and `String()` method
- Zero panics in library code

### File consolidation

10 source files → 5: `dusk.go`, `solar.go`, `lunar.go`, `epoch.go`, `trig.go`. Twilight merged into solar, lunar tables merged into lunar, coordinate conversions merged into epoch, types/stringers/errors consolidated into dusk.

## v2.2.0 — 2026-03-30

### Bug fixes

- `DaysApprox` now uses a linear elongation-to-days formula matching the "days into lunation" documentation (#36)
- `solarHourAngle` no longer applies elevation correction to twilight calculations, per USNO convention (#39)
- Example tests handle errors from `time.LoadLocation` and computation functions instead of discarding (#43)

### Testing

- Fuzz tests assert error returns for out-of-range coordinates instead of silently skipping (#37)
- `TestLunarPhase` now asserts on `Angle` and `Elongation` ranges for all 8 phase test cases (#41)

### CI

- Enforce `gofumpt` formatting via `golangci-lint` formatters config (#38)

### Documentation

- `HourAngle` doc comment clarifies mixed-unit parameters (RA in degrees, LST in hours) (#40)
- `MoonriseMoonset` doc comment notes ~1-2 ms wall-clock cost per call (#42)

## v2.1.0 — 2026-03-10

### New features

- `fmt.Stringer` interface on all 8 exported result types (`SunEvent`, `MoonEvent`, `LunarPhaseInfo`, `Transit`, `TwilightEvent`, `Equatorial`, `Ecliptic`, `Horizontal`)
- `ValidJulianDateRange` helper and `ErrDateOutOfRange` sentinel for guarding `JulianDate` range (~1677–2262)

### Bug fixes

- `asinx`/`acosx` now clamp inputs to [-1, 1] via `clamp()`, preventing NaN from floating-point rounding
- `validateEquatorial` normalizes RA via `mod360` instead of rejecting values at 360.0; rejects NaN/Inf inputs

### Performance

- `ObjectTransit` transit maximum: replaced O(n) minute-by-minute scan with O(1) analytical solution (hour angle = 0)

### Refactoring

- Extracted `computeSolarParams` helper eliminating 3× duplication of the 6-step solar parameter sequence; `twilight()` reduced from 44 to 29 lines
- Renamed shadowed variable `F` → `frac` in `LunarPhase`

### Testing

- Fuzz tests for `SunriseSunset`, `LunarPhase`, `ObjectTransit`, `MoonriseMoonset`
- Benchmarks for `MoonriseMoonset`, `LunarEclipticPosition`, `SunriseSunset`, `ObjectTransit` (0 allocations)
- Polar twilight transition test (75°N, Nov 25→26)
- Negative elevation clamping, summer solstice solar position, GMST/julianCentury helper tests
- Southern hemisphere moonrise/moonset regression references
- Consistent `t.Run` subtests for `TestMod360`/`TestMod24`; table-driven `TestEclipticToEquatorial`

### Infrastructure

- `.golangci.yml` with explicit linter list (gocritic, revive, misspell, etc.); `captLocal` disabled for Meeus conventions
- 80% coverage threshold in CI (currently 99.7%); removed duplicate `go vet` step

### Documentation

- `Observer.Elev` doc comment: negative value clamping, scope (sunrise/sunset/twilight only)
- `gstToUT` precision note about J1900-epoch model accuracy

## v2.0.0 — 2026-03-06

Complete rewrite of the library. Zero external dependencies.

### Breaking changes

- **Module path** changed from `github.com/philoserf/dusk` to `github.com/philoserf/dusk/v2`
- **Removed `timezonemapper` dependency** — callers pass `*time.Location` explicitly via the new `Observer` struct
- **Renamed all exported functions** — `Get` prefix removed (e.g., `GetJulianDate` → `JulianDate`, `GetLocalSiderealTime` → `LocalSiderealTime`)
- **Replaced coordinate types** — `Coordinate`, `EquatorialCoordinate`, `EclipticCoordinate`, `HorizontalCoordinate` replaced by `Equatorial`, `Ecliptic`, `Horizontal` with named fields (`RA`/`Dec`, `Lon`/`Lat`, `Alt`/`Az`)
- **New `Observer` struct** — replaces separate `latitude`, `longitude` parameters; includes `Loc *time.Location` and `Elev float64`
- **New event structs** — `SunEvent`, `MoonEvent`, `TwilightEvent`, `Transit` replace raw time returns and ad-hoc structs
- **Removed exports** — `JulianPeriod`, `TemporalHorizontalCoordinate`, `TransitHorizontalCoordinate`, `GetEarthObliquity`, `GetUniversalTime`, `GetCurrentJulianPeriod`, `GetMeanSolarTime`, `GetDatetimeZeroHour`, `ConvertLocalSiderealTimeToGreenwichSiderealTime`, `ConvertGreenwichSiderealTimeToUniversalTime`, `GetAtmosphericRefraction`, `GetRelativeAirMass`, `GetApparentAltitude`, `GetArgumentOfLocalSiderealTimeForTransit`, `SunriseStatus`, `AboveHorizon`, `AtHorizon`, `BelowHorizon`, and others
- **Removed files** — `astrometry.go`, `coordinates.go`, `trigonometry.go`, `utils.go`, `lawrence.go` consolidated into domain-focused files

### New features

- `LunarPhase` — illumination, age in days, waxing/waning, and phase name
- `LunarEclipticPosition` — Meeus Chapter 47 ecliptic position with full periodic terms
- `MoonriseMoonset` — moonrise and moonset times via minute-by-minute altitude scan
- `ObjectTransit` — rise, set, and transit maximum for arbitrary equatorial coordinates
- `CivilTwilight`, `NauticalTwilight`, `AstronomicalTwilight` — twilight dusk/dawn pairs
- `ErrCircumpolar` and `ErrNeverRises` sentinel errors for polar edge cases
- `AngularSeparation` — robust atan2-based formula

### Improvements

- Meeus algorithms throughout (solar mean anomaly, lunar ecliptic, nutation, obliquity)
- Degree-based trig helpers (`sinx`, `cosx`, etc.) eliminate manual conversion
- `mod360`/`mod24` normalization helpers
- Zero-value `time.Time` convention for events that don't occur (circumpolar, never rises)
- High test coverage with table-driven tests and values from USNO, Stellarium, and Meeus
- Edge-case tests for equatorial, polar, and southern hemisphere observers

## v1.0.0

Initial release. Forked from [observerly/dusk](https://github.com/observerly/dusk).
