# Dusk v3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Shrink dusk to a clean, minimal astronomical library — 6 public functions, 5 source files, zero dead code.

**Architecture:** Consolidate 10 source files into 5. Unexport coordinate/epoch internals. Remove `ObjectTransit`, `AngularSeparation`, elevation, and all standalone coordinate functions from public API. Add `NewObserver` constructor that validates once at creation.

**Tech Stack:** Go 1.24, zero dependencies, gofumpt, golangci-lint, go test -race

**Spec:** `docs/superpowers/specs/2026-03-30-v3-design.md`

---

### Task 1: Module path and go.mod

**Files:**

- Modify: `go.mod`

- [ ] **Step 1: Update module path to v3**

```go
module github.com/philoserf/dusk/v3

go 1.24
```

- [ ] **Step 2: Run go mod tidy**

Run: `go mod tidy`
Expected: clean exit, no changes beyond go.sum

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: bump module path to v3"
```

---

### Task 2: Create `dusk.go` — types, constructor, stringers, errors

This is the new home for all public types, the `Observer` constructor, sentinel errors, `String()` methods, and the package doc comment. Build it test-first.

**Files:**

- Create: `dusk.go`
- Test: `dusk_test.go`

- [ ] **Step 1: Write tests for `NewObserver`**

Create `dusk_test.go`:

```go
package dusk

import (
	"math"
	"testing"
	"time"
)

func TestNewObserver(t *testing.T) {
	tests := []struct {
		name    string
		lat     float64
		lon     float64
		loc     *time.Location
		wantErr bool
	}{
		{"valid NYC", 40.7128, -74.006, time.UTC, false},
		{"valid south pole", -90, 0, time.UTC, false},
		{"valid date line", 0, 180, time.UTC, false},
		{"valid negative lon", 0, -180, time.UTC, false},
		{"nil location", 40.7128, -74.006, nil, true},
		{"lat too high", 91, 0, time.UTC, true},
		{"lat too low", -91, 0, time.UTC, true},
		{"lon too high", 0, 181, time.UTC, true},
		{"lon too low", 0, -181, time.UTC, true},
		{"NaN lat", math.NaN(), 0, time.UTC, true},
		{"NaN lon", 0, math.NaN(), time.UTC, true},
		{"Inf lat", math.Inf(1), 0, time.UTC, true},
		{"Inf lon", 0, math.Inf(-1), time.UTC, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs, err := NewObserver(tt.lat, tt.lon, tt.loc)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			_ = obs
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestNewObserver ./...`
Expected: FAIL — `NewObserver` not defined

- [ ] **Step 3: Create `dusk.go` with package doc, types, constructor, errors, and stringers**

Create `dusk.go`:

```go
// Package dusk provides astronomical calculations: twilight times,
// sunrise/sunset, moonrise/moonset, and lunar phase.
//
// All angles are in degrees. Time parameters use [time.Time].
// Functions that produce local times accept an [Observer] with a timezone.
//
// Two sentinel errors distinguish polar edge cases:
// [ErrCircumpolar] (object always above the horizon) and
// [ErrNeverRises] (object never rises).
//
// Zero-value [time.Time] in result structs signals "event did not occur"
// for a specific day (e.g., the Moon rises but does not set before midnight).
// Check with [time.Time.IsZero]. This is distinct from sentinel errors,
// which indicate the geometry makes the event impossible at the given latitude.
//
// # References
//
//   - Meeus, Jean. Astronomical Algorithms. 2nd ed. Willmann-Bell, 1998.
package dusk

import (
	"errors"
	"fmt"
	"math"
	"time"
)

// ErrCircumpolar is returned when a celestial object is circumpolar
// (always above the horizon) at the given latitude.
var ErrCircumpolar = errors.New("dusk: object is circumpolar (always above the horizon)")

// ErrNeverRises is returned when a celestial object never rises above
// the horizon at the given latitude.
var ErrNeverRises = errors.New("dusk: object never rises at this latitude")

var (
	errNilLocation = errors.New("dusk: location must not be nil")
	errInvalidCoord = errors.New("dusk: latitude must be in [-90, 90] and longitude in [-180, 180]")
)

// Observer represents a geographic position on Earth.
type Observer struct {
	lat float64
	lon float64
	loc *time.Location
}

// NewObserver creates a validated Observer. Rejects nil loc, lat outside
// [-90, 90], lon outside [-180, 180], and NaN/Inf values.
func NewObserver(lat, lon float64, loc *time.Location) (Observer, error) {
	if loc == nil {
		return Observer{}, errNilLocation
	}
	if math.IsNaN(lat) || math.IsInf(lat, 0) || math.IsNaN(lon) || math.IsInf(lon, 0) {
		return Observer{}, errInvalidCoord
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return Observer{}, errInvalidCoord
	}
	return Observer{lat: lat, lon: lon, loc: loc}, nil
}

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
	Rise     time.Time     // zero value if the Moon does not rise
	Set      time.Time     // zero value if the Moon does not set
	Duration time.Duration // zero if Rise or Set is missing
}

// TwilightEvent holds the dusk and dawn times of a twilight period.
// Dusk is tonight's boundary (sun passes below the depression angle).
// Dawn is tomorrow morning's boundary (sun passes above the depression angle).
// To get this morning's dawn, call with yesterday's date.
type TwilightEvent struct {
	Dusk     time.Time
	Dawn     time.Time
	Duration time.Duration
}

// LunarPhaseInfo holds lunar phase information: illumination percentage,
// elongation in degrees, approximate age in days, phase angle, and the
// common phase name.
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
// Used internally for moonrise/moonset detection.
type horizontal struct {
	alt float64
	az  float64
}

// ecliptic represents ecliptic coordinates: longitude and latitude in degrees,
// and distance in kilometers.
type ecliptic struct {
	lon  float64
	lat  float64
	dist float64
}

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
	return fmt.Sprintf("Rise=%s Set=%s Duration=%s",
		formatTime(m.Rise),
		formatTime(m.Set),
		m.Duration)
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
	return fmt.Sprintf("Dusk=%s Dawn=%s Duration=%s",
		formatTime(tw.Dusk),
		formatTime(tw.Dawn),
		tw.Duration)
}
```

- [ ] **Step 4: Add stringer tests to `dusk_test.go`**

Append to `dusk_test.go` — keep `SunEvent`, `MoonEvent`, `LunarPhaseInfo`, `TwilightEvent` stringer tests from `stringer_test.go`. Drop `Transit`, `Equatorial`, `Horizontal`, `Ecliptic` stringer tests (types are unexported now).

```go
func TestSunEventString(t *testing.T) {
	loc := time.FixedZone("TEST", -5*3600)
	s := SunEvent{
		Rise:     time.Date(2024, 1, 15, 12, 1, 0, 0, time.UTC).In(loc),
		Noon:     time.Date(2024, 1, 15, 17, 30, 0, 0, time.UTC).In(loc),
		Set:      time.Date(2024, 1, 15, 22, 59, 0, 0, time.UTC).In(loc),
		Duration: 10*time.Hour + 58*time.Minute,
	}
	want := "Rise=07:01 Noon=12:30 Set=17:59 Duration=10h58m0s"
	if got := s.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSunEventString_Zero(t *testing.T) {
	s := SunEvent{}
	want := "Rise=--:-- Noon=--:-- Set=--:-- Duration=0s"
	if got := s.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMoonEventString(t *testing.T) {
	loc := time.FixedZone("TEST", 0)
	m := MoonEvent{
		Rise:     time.Date(2024, 1, 15, 8, 15, 0, 0, loc),
		Set:      time.Date(2024, 1, 15, 20, 30, 0, 0, loc),
		Duration: 12*time.Hour + 15*time.Minute,
	}
	want := "Rise=08:15 Set=20:30 Duration=12h15m0s"
	if got := m.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMoonEventString_Zero(t *testing.T) {
	m := MoonEvent{}
	want := "Rise=--:-- Set=--:-- Duration=0s"
	if got := m.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLunarPhaseInfoString(t *testing.T) {
	l := LunarPhaseInfo{
		Illumination: 75.3,
		DaysApprox:   11.2,
		Name:         "Waxing Gibbous",
	}
	want := "Waxing Gibbous 75.3% (day 11.2)"
	if got := l.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLunarPhaseInfoString_Zero(t *testing.T) {
	l := LunarPhaseInfo{}
	want := "0.0% (day 0.0)"
	if got := l.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTwilightEventString(t *testing.T) {
	loc := time.FixedZone("TEST", 0)
	tw := TwilightEvent{
		Dusk:     time.Date(2024, 1, 15, 18, 30, 0, 0, loc),
		Dawn:     time.Date(2024, 1, 16, 6, 15, 0, 0, loc),
		Duration: 11*time.Hour + 45*time.Minute,
	}
	want := "Dusk=18:30 Dawn=06:15 Duration=11h45m0s"
	if got := tw.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTwilightEventString_Zero(t *testing.T) {
	tw := TwilightEvent{}
	want := "Dusk=--:-- Dawn=--:-- Duration=0s"
	if got := tw.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

- [ ] **Step 5: Run all tests**

Run: `go test -run "TestNewObserver|TestSunEvent|TestMoonEvent|TestLunarPhaseInfo|TestTwilightEvent" ./...`
Expected: PASS (will fail on other tests due to type conflicts — that's expected until old files are removed)

- [ ] **Step 6: Commit**

```bash
git add dusk.go dusk_test.go
git commit -m "feat: add dusk.go with types, NewObserver constructor, stringers"
```

---

### Task 3: Update `epoch.go` — unexport and absorb coordinate conversions

Move coordinate conversion functions from `coord.go` into `epoch.go`. Unexport everything. Add nutation in longitude (Δψ) to `eclipticToEquatorial`.

**Files:**

- Modify: `epoch.go`
- Modify: `epoch_test.go`

- [ ] **Step 1: Write test for nutation in longitude**

Add to `epoch_test.go`:

```go
func TestEclipticToEquatorial(t *testing.T) {
	tests := []struct {
		name    string
		dt      time.Time
		lon     float64
		lat     float64
		wantRA  float64
		wantDec float64
		eps     float64
	}{
		{
			name:    "Meeus p.342: Moon 1992-04-12",
			dt:      time.Date(1992, 4, 12, 0, 0, 0, 0, time.UTC),
			lon:     133.162655,
			lat:     -3.229126,
			wantRA:  134.7,
			wantDec: 13.8,
			eps:     0.15,
		},
		{
			name:    "ecliptic lon=90 lat=0 (solstice geometry)",
			dt:      time.Date(2024, 6, 20, 12, 0, 0, 0, time.UTC),
			lon:     90.0,
			lat:     0.0,
			wantRA:  90.0,
			wantDec: 23.44,
			eps:     0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eq := eclipticToEquatorial(tt.dt, tt.lon, tt.lat)
			if math.Abs(eq.ra-tt.wantRA) > tt.eps {
				t.Errorf("RA = %.4f, want ~%.1f (within %.2f°)", eq.ra, tt.wantRA, tt.eps)
			}
			if math.Abs(eq.dec-tt.wantDec) > tt.eps {
				t.Errorf("Dec = %.4f, want ~%.2f (within %.2f°)", eq.dec, tt.wantDec, tt.eps)
			}
		})
	}
}
```

- [ ] **Step 2: Write tests for `equatorialToHorizontal` and `hourAngle`**

Add to `epoch_test.go`:

```go
func TestEquatorialToHorizontal(t *testing.T) {
	dt := time.Date(2024, 1, 16, 2, 0, 0, 0, time.UTC)
	obs := Observer{lat: 40.7128, lon: -74.006, loc: time.UTC}
	sirius := equatorial{ra: 101.287, dec: -16.716}

	h := equatorialToHorizontal(dt, obs, sirius)

	const eps = 1.0
	if math.Abs(h.alt-26.06) > eps {
		t.Errorf("Alt = %.4f, want ~26.06° (within %.0f°)", h.alt, eps)
	}
	if math.Abs(h.az-147.49) > eps {
		t.Errorf("Az = %.4f, want ~147.49° (within %.0f°)", h.az, eps)
	}
}

func TestEquatorialToHorizontal_Pole(t *testing.T) {
	dt := time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC)
	obs := Observer{lat: 90.0, lon: 0, loc: time.UTC}
	star := equatorial{ra: 0, dec: 45}

	h := equatorialToHorizontal(dt, obs, star)
	if h.az != 0 {
		t.Errorf("Az = %.4f, want 0 at pole (guard branch)", h.az)
	}
	if math.Abs(h.alt-45.0) > 0.5 {
		t.Errorf("Alt = %.4f, want ~45° at North Pole for Dec=45°", h.alt)
	}
}

func TestHourAngle(t *testing.T) {
	tests := []struct {
		name   string
		ra     float64
		lst    float64
		wantHA float64
	}{
		{"object on meridian", 90, 6, 0},
		{"RA=90 LST=0h", 90, 0, 270},
		{"RA=0 LST=6h", 0, 6, 90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ha := hourAngle(tt.ra, tt.lst)
			if math.Abs(ha-tt.wantHA) > 1e-9 {
				t.Errorf("hourAngle(%v, %v) = %v, want %v", tt.ra, tt.lst, ha, tt.wantHA)
			}
		})
	}
}
```

- [ ] **Step 3: Add nutation in longitude and absorb coordinate functions into `epoch.go`**

Append to `epoch.go` after the existing nutation helpers:

```go
// nutationInLongitude returns Δψ (nutation in longitude) in degrees.
//
// See Meeus p. 144.
func nutationInLongitude(L, l, omega float64) float64 {
	return (-17.20*sinx(omega) - 1.32*sinx(2*L) - 0.23*sinx(2*l) + 0.21*sinx(2*omega)) / 3600.0
}

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
	deps := nutationInObliquity(L, l, omega)
	eps := meanObliquity(T) + deps

	// Apply nutation in longitude to the ecliptic longitude.
	lonCorr := lon + dpsi

	ra := atan2x(sinx(lonCorr)*cosx(eps)-tanx(lat)*sinx(eps), cosx(lonCorr))
	dec := asinx(sinx(lat)*cosx(eps) + cosx(lat)*sinx(eps)*sinx(lonCorr))

	return equatorial{
		ra:  mod360(ra),
		dec: dec,
	}
}

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
	if math.Abs(cosAltCosLat) < 1e-10 {
		az = 0
	} else {
		az = acosx((sinx(eq.dec) - sinx(alt)*sinx(obs.lat)) / cosAltCosLat)
	}

	if sinx(ha) > 0 {
		az = 360 - az
	}

	return horizontal{alt: alt, az: az}
}

// hourAngle computes the hour angle in degrees.
func hourAngle(ra, lst float64) float64 {
	return mod360(lst*15 - ra)
}
```

Also unexport all public symbols in `epoch.go`:

- `JulianDate` → `julianDate`
- `ValidJulianDateRange` → `validJulianDateRange`
- `ErrDateOutOfRange` → `errDateOutOfRange`
- `LocalSiderealTime` → `localSiderealTime`

Update `epoch_test.go` to reference the unexported names throughout (`julianDate`, `localSiderealTime`, `validJulianDateRange`, `errDateOutOfRange`). The test file uses `package dusk` (internal), so unexported symbols are accessible.

- [ ] **Step 4: Run tests**

Run: `go test -run "TestEclipticToEquatorial|TestEquatorialToHorizontal|TestHourAngle|TestLocalSiderealTime|TestJulianDate|TestGreenwichMean" ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add epoch.go epoch_test.go
git commit -m "refactor: absorb coordinate conversions into epoch.go, add nutation in longitude"
```

---

### Task 4: Update `solar.go` — absorb twilight, remove elevation, use unexported types

Merge `twilight.go` into `solar.go`. Remove elevation from `solarHourAngle`. Update all functions to use the unexported `Observer` fields and skip `validateObserver` (construction guarantees validity). Add date range validation at entry points.

**Files:**

- Modify: `solar.go`
- Create: `solar_test.go` (rewrite to use `NewObserver` and merged twilight tests)

- [ ] **Step 1: Update `solar.go`**

Key changes:

- `SunriseSunset` and twilight functions access `obs.lat`, `obs.lon`, `obs.loc` (unexported fields)
- Remove `validateObserver(obs)` calls — replaced by `validJulianDateRange(date)` check
- `solarHourAngle` signature changes from `(delta, depression, lat, elev float64)` to `(delta, depression, lat float64)`
- Elevation correction deleted from `solarHourAngle`
- `SolarPosition` becomes unexported `solarPosition`, returns `equatorial`
- Append the three twilight functions and `twilight` helper from `twilight.go`
- All internal calls use `eclipticToEquatorial` (lowercase)

Updated `solarHourAngle`:

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

Updated `SunriseSunset`:

```go
func SunriseSunset(date time.Time, obs Observer) (SunEvent, error) {
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

Updated `twilight` (appended to solar.go):

```go
func twilight(date time.Time, obs Observer, depression float64) (TwilightEvent, error) {
	if err := validJulianDateRange(date); err != nil {
		return TwilightEvent{}, err
	}

	sp := computeSolarParams(date, obs.lon)
	omega, err := solarHourAngle(sp.delta, depression, obs.lat)
	if err != nil {
		return TwilightEvent{}, err
	}
	dusk := universalTimeFromJD(sp.jTransit + omega/360).In(obs.loc)

	tomorrow := date.AddDate(0, 0, 1)
	sp2 := computeSolarParams(tomorrow, obs.lon)
	omega2, err2 := solarHourAngle(sp2.delta, depression, obs.lat)
	if err2 != nil {
		return TwilightEvent{}, err2
	}
	dawn := universalTimeFromJD(sp2.jTransit - omega2/360).In(obs.loc)

	return TwilightEvent{
		Dusk:     dusk,
		Dawn:     dawn,
		Duration: dawn.Sub(dusk),
	}, nil
}
```

- [ ] **Step 2: Update `solar_test.go`**

Rewrite all `Observer{}` literal construction to use `NewObserver`. Remove tests for elevation correction. Keep all existing sunrise/sunset and twilight test cases and reference values. Merge in all tests from `twilight_test.go`.

Example pattern for every test that constructs an Observer:

```go
obs, err := NewObserver(40.7128, -74.006, edt)
if err != nil {
	t.Fatal(err)
}
```

Remove any test case that sets `Elev`. Remove `TestSunriseSunset_NegativeElevation`.

- [ ] **Step 3: Run tests**

Run: `go test -run "TestSunrise|TestCivil|TestNautical|TestAstronomical|TestTwilight" ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add solar.go solar_test.go
git commit -m "refactor: absorb twilight into solar.go, remove elevation, add date validation"
```

---

### Task 5: Update `lunar.go` — absorb tables, use unexported types, add table safety

Merge `lunar_tables.go` into `lunar.go`. Update to use unexported types (`equatorial`, `ecliptic`). Add `default` panic to lunar table switch. Add date range validation. Remove `validateObserver` calls.

**Files:**

- Modify: `lunar.go`
- Modify: `lunar_test.go`

- [ ] **Step 1: Absorb `lunar_tables.go` into `lunar.go`**

Move the table type definitions and `tableLongDist`/`tableLat` arrays into `lunar.go`, placing them at the bottom of the file after all functions.

- [ ] **Step 2: Update type references**

- `Ecliptic` → `ecliptic` (fields: `lon`, `lat`, `dist`)
- `Equatorial` → `equatorial` (fields: `ra`, `dec`)
- `EclipticToEquatorial` → `eclipticToEquatorial`
- `EquatorialToHorizontal` → `equatorialToHorizontal`
- `LunarEclipticPosition` → `lunarEclipticPosition`
- `LunarPosition` → `lunarPosition`
- Remove `validateObserver(obs)`, add `validJulianDateRange(date)`
- Access observer fields as `obs.lat`, `obs.lon`, `obs.loc`

- [ ] **Step 3: Add default panic to lunar table loops**

In both table loops, add a default case:

```go
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
default:
	panic(fmt.Sprintf("dusk: unexpected M value %v in lunar table", r.M))
}
```

Same pattern for the latitude loop.

- [ ] **Step 4: Update `lunar_test.go`**

Rewrite all `Observer{}` literals to use `NewObserver`. Update references to unexported types/functions (`LunarEclipticPosition` → `lunarEclipticPosition`, etc.). Keep all existing test cases and reference values.

- [ ] **Step 5: Run tests**

Run: `go test -run "TestLunar|TestMoonrise" ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add lunar.go lunar_test.go
git commit -m "refactor: absorb lunar tables, unexport types, add table safety default"
```

---

### Task 6: Update example tests, fuzz tests, and benchmarks

Rewrite to use `NewObserver`, reference only exported API, remove transit/coordinate examples.

**Files:**

- Modify: `dusk_test.go` (add examples)
- Create: `fuzz_test.go` (rewrite)
- Create: `benchmark_test.go` (rewrite)

- [ ] **Step 1: Add example tests to `dusk_test.go`**

Append examples that use only the v3 public API. Use `dusk_test` package (external). Note: since `dusk_test.go` already has internal tests, create `example_test.go` in the `dusk_test` package instead.

Create `example_test.go`:

```go
package dusk_test

import (
	"fmt"
	"time"

	"github.com/philoserf/dusk/v3"
)

func ExampleSunriseSunset() {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	obs, err := dusk.NewObserver(42.9634, -85.6681, loc)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	date := time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC)

	sun, err := dusk.SunriseSunset(date, obs)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("Sunrise: %s\n", sun.Rise.Format("15:04"))
	fmt.Printf("Sunset:  %s\n", sun.Set.Format("15:04"))
	// Output:
	// Sunrise: 05:06
	// Sunset:  20:22
}

func ExampleLunarPhase() {
	t := time.Date(2024, 1, 25, 18, 0, 0, 0, time.UTC)
	phase := dusk.LunarPhase(t)
	fmt.Printf("Phase: %s\n", phase.Name)
	fmt.Printf("Illumination: %.0f%%\n", phase.Illumination)
	// Output:
	// Phase: Full Moon
	// Illumination: 100%
}

func ExampleCivilTwilight() {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	obs, err := dusk.NewObserver(47.6062, -122.3321, loc)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	date := time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC)

	tw, err := dusk.CivilTwilight(date, obs)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("Dusk: %s\n", tw.Dusk.Format("15:04"))
	fmt.Printf("Dawn: %s\n", tw.Dawn.Format("15:04"))
	// Output:
	// Dusk: 21:51
	// Dawn: 04:31
}

func ExampleMoonriseMoonset() {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	obs, err := dusk.NewObserver(40.7128, -74.0060, loc)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, loc)

	evt, err := dusk.MoonriseMoonset(date, obs)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	if !evt.Rise.IsZero() {
		fmt.Printf("Moonrise: %s\n", evt.Rise.Format("15:04"))
	}
	if !evt.Set.IsZero() {
		fmt.Printf("Moonset:  %s\n", evt.Set.Format("15:04"))
	}
	// Output:
	// Moonrise: 10:06
	// Moonset:  22:13
}
```

- [ ] **Step 2: Rewrite fuzz tests**

Create `fuzz_test.go`. Remove the NaN/Inf bail-outs — let non-finite values hit `NewObserver` and assert error:

```go
package dusk

import (
	"math"
	"testing"
	"time"
)

func FuzzSunriseSunset(f *testing.F) {
	f.Add(40.7128, -74.006, int64(1710892800))
	f.Add(-33.87, 151.21, int64(1718928000))
	f.Add(69.65, 18.96, int64(1718928000))

	f.Fuzz(func(t *testing.T, lat, lon float64, unix int64) {
		date := time.Unix(unix, 0).UTC()
		if date.Year() < 1800 || date.Year() > 2200 {
			return
		}

		obs, err := NewObserver(lat, lon, time.UTC)
		if err != nil {
			return // invalid coordinates, expected
		}

		_, _ = SunriseSunset(date, obs)
	})
}

func FuzzLunarPhase(f *testing.F) {
	f.Add(int64(1704931200))
	f.Add(int64(1706140800))
	f.Add(int64(1710892800))

	f.Fuzz(func(t *testing.T, unix int64) {
		date := time.Unix(unix, 0).UTC()
		if date.Year() < 1800 || date.Year() > 2200 {
			return
		}
		p := LunarPhase(date)
		if math.IsNaN(p.Illumination) {
			t.Error("NaN illumination")
		}
		if p.Illumination < 0 || p.Illumination > 100 {
			t.Errorf("illumination out of range: %f", p.Illumination)
		}
	})
}

func FuzzMoonriseMoonset(f *testing.F) {
	f.Add(40.7128, -74.006, int64(1705276800))
	f.Add(-33.87, 151.21, int64(1705276800))

	f.Fuzz(func(t *testing.T, lat, lon float64, unix int64) {
		date := time.Unix(unix, 0).UTC()
		if date.Year() < 1800 || date.Year() > 2200 {
			return
		}

		obs, err := NewObserver(lat, lon, time.UTC)
		if err != nil {
			return
		}

		_, _ = MoonriseMoonset(date, obs)
	})
}
```

- [ ] **Step 3: Rewrite benchmarks**

Create `benchmark_test.go`. Remove `ObjectTransit` and `LunarEclipticPosition` benchmarks. Use `NewObserver`:

```go
package dusk

import (
	"testing"
	"time"
)

var edt = time.FixedZone("EDT", -5*3600)

func BenchmarkSunriseSunset(b *testing.B) {
	date := time.Date(2024, 3, 20, 0, 0, 0, 0, edt)
	obs, err := NewObserver(40.7128, -74.006, edt)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		SunriseSunset(date, obs) //nolint:errcheck
	}
}

func BenchmarkMoonriseMoonset(b *testing.B) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, edt)
	obs, err := NewObserver(40.7128, -74.006, edt)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		MoonriseMoonset(date, obs) //nolint:errcheck
	}
}
```

- [ ] **Step 4: Run all tests**

Run: `go test ./...`
Expected: PASS (at this point, old files still exist but new files compile independently)

- [ ] **Step 5: Commit**

```bash
git add example_test.go fuzz_test.go benchmark_test.go
git commit -m "test: rewrite examples, fuzz tests, and benchmarks for v3 API"
```

---

### Task 7: Delete old files

Remove all files that have been absorbed or are no longer needed.

**Files:**

- Delete: `doc.go`
- Delete: `coord.go`
- Delete: `coord_test.go`
- Delete: `stringer.go`
- Delete: `stringer_test.go`
- Delete: `twilight.go`
- Delete: `twilight_test.go`
- Delete: `transit.go`
- Delete: `transit_test.go`
- Delete: `lunar_tables.go`

- [ ] **Step 1: Delete old source files**

```bash
git rm doc.go coord.go stringer.go twilight.go transit.go lunar_tables.go
```

- [ ] **Step 2: Delete old test files**

```bash
git rm coord_test.go stringer_test.go twilight_test.go transit_test.go
```

Note: `example_test.go` was replaced in Task 6. Do not delete it.

- [ ] **Step 4: Run full test suite**

Run: `go test -race ./...`
Expected: PASS with zero compilation errors

- [ ] **Step 5: Commit**

```bash
git commit -m "chore: remove files absorbed into v3 structure"
```

---

### Task 8: Formatting, linting, and final verification

**Files:**

- Modify: `CHANGELOG.md`
- Modify: `.claude/CLAUDE.md`

- [ ] **Step 1: Run gofumpt**

Run: `gofumpt -w dusk.go epoch.go solar.go lunar.go trig.go`

- [ ] **Step 2: Run golangci-lint**

Run: `golangci-lint run`
Expected: clean

- [ ] **Step 3: Run full test suite with race detector**

Run: `go test -race -count=1 ./...`
Expected: PASS

- [ ] **Step 4: Run go vet**

Run: `go vet ./...`
Expected: clean

- [ ] **Step 5: Verify exported API surface**

Run: `go doc -all . | grep -E "^(func|type|var) [A-Z]"`
Expected: only `NewObserver`, `SunriseSunset`, `CivilTwilight`, `NauticalTwilight`, `AstronomicalTwilight`, `MoonriseMoonset`, `LunarPhase`, `Observer`, `SunEvent`, `MoonEvent`, `TwilightEvent`, `LunarPhaseInfo`, `ErrCircumpolar`, `ErrNeverRises`

- [ ] **Step 6: Update CHANGELOG.md**

Add v3.0.0 entry at top documenting breaking changes, removals, and fixes.

- [ ] **Step 7: Update `.claude/CLAUDE.md`**

Update the architecture table to reflect the new 5-file structure. Remove references to `ObjectTransit`, `AngularSeparation`, `Elev`, and deleted files.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "chore: format, lint, update docs for v3"
```

- [ ] **Step 9: Run task (full check)**

Run: `task`
Expected: all checks pass (fmt, vet, lint, test)
