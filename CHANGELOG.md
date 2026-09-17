# Changelog

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
  `BenchmarkMoonriseMoonset` improves about 8%, from ~1.37 ms to ~1.26 ms.
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
  source. Three of `THEORY.md`'s five open uncertainties are now settled, including the
  lunar error budget, which turned out to be almost entirely the missing parallax rather
  than the method.

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
