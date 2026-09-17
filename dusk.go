// Package dusk provides astronomical calculations: twilight times,
// sunrise/sunset, moonrise/moonset, and lunar phase.
//
// All angles are in degrees. The day-based entry points take a [Date] and an
// [Observer]; [LunarPhase] takes a [time.Time], because phase is Sun-Earth-Moon
// geometry and uses the whole instant.
//
// A zero [time.Time] in a result means the event did not occur. For the Sun,
// [Horizon] on the result says why: StaysAbove and StaysBelow mean the geometry
// forbade the crossing at that latitude and altitude. For the Moon, a zero time
// means the crossing simply fell outside this calendar day, which is routine --
// a lunar day runs about 24h50m -- and [MoonEvent.AboveHorizon] says which side
// of the horizon it started on.
//
// An error means the call could not be made: a nil location, coordinates that
// are not finite or out of range, or a date outside the supported span.
//
// # References
//
//   - Meeus, Jean. Astronomical Algorithms. 2nd ed. Willmann-Bell, 1998.
package dusk

import (
	"fmt"
	"math"
	"time"
)

// stringError is an immutable error type used for sentinel errors.
// Unlike errors.New, these can be declared as constants.
type stringError string

func (e stringError) Error() string { return string(e) }

// ErrNilLocation is returned when a nil *time.Location is passed to
// [NewObserver].
const ErrNilLocation = stringError("dusk: location must not be nil")

// ErrNonFiniteCoord is returned when NaN or Inf coordinates are passed
// to [NewObserver].
const ErrNonFiniteCoord = stringError("dusk: coordinates must be finite (NaN and Inf are not allowed)")

// ErrInvalidCoord is returned when latitude or longitude are outside
// the valid range in [NewObserver].
const ErrInvalidCoord = stringError("dusk: latitude must be in [-90, 90] and longitude in [-180, 180]")

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

// Lat returns the observer's latitude in degrees.
func (o Observer) Lat() float64 { return o.lat }

// Lon returns the observer's longitude in degrees (east positive, west negative).
func (o Observer) Lon() float64 { return o.lon }

// Location returns the observer's timezone.
func (o Observer) Location() *time.Location { return o.loc }

// String returns a human-readable representation of the observer.
func (o Observer) String() string {
	locName := "nil"
	if o.loc != nil {
		locName = o.loc.String()
	}

	return fmt.Sprintf("%.4f°, %.4f° (%s)", o.lat, o.lon, locName)
}

// Date is a calendar day. It has no zone and no time of day, because the
// day-based entry points use neither.
//
// Fields are normalized the way [time.Date] normalizes them, so Month 13 is
// January of the next year. A Date outside roughly 1677-2262 is rejected by the
// entry points with [ErrDateOutOfRange].
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// DateIn returns the calendar date on which t falls in loc. It is the call the
// day-based entry points used to make invisibly, on an instant the caller had
// no reason to think mattered; making it the caller's puts the zone where it
// can be seen. A nil loc is read as UTC.
func DateIn(t time.Time, loc *time.Location) Date {
	if loc != nil {
		t = t.In(loc)
	}

	return Date{Year: t.Year(), Month: t.Month(), Day: t.Day()}
}

// at returns midnight on d in loc.
func (d Date) at(loc *time.Location) time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, loc)
}

// Horizon says whether the Sun reached the altitude a call asked about. It is
// a value on the result rather than an error, because "the Sun did not set
// today" is an answer to the question, not a failure to answer it.
//
// The names describe the geometry, not the sunrise case. At a depression angle
// StaysAbove means the night never got that dark and StaysBelow means the day
// never got that light -- the opposite of what "circumpolar" and "never rises"
// suggested, which is why those names are gone.
type Horizon int

// The three outcomes of asking whether the Sun reached an altitude.
const (
	Crosses    Horizon = iota // the Sun reached the altitude; the times are real
	StaysAbove                // the Sun never descended to it
	StaysBelow                // the Sun never ascended to it
)

// SunEvent holds the times of sunrise, solar noon, sunset, and the duration
// of daylight for a single day.
//
// Rise and Set are zero unless Horizon is [Crosses]. Noon is always set: solar
// transit happens on every day at every latitude, including through the polar
// night. Duration is 24h under the midnight sun and 0 through the polar night.
type SunEvent struct {
	Rise     time.Time // zero unless Horizon is Crosses
	Noon     time.Time // always set
	Set      time.Time // zero unless Horizon is Crosses
	Duration time.Duration
	Horizon  Horizon
}

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

// TwilightEvent holds the two boundaries of a twilight band on one calendar
// day in the observer's timezone: Dawn where the Sun rises through the
// depression angle, Dusk where it sets through it.
//
// Both are on the same day. They are symmetric about solar transit, so a
// TwilightEvent never carries one boundary without the other -- when the
// geometry forbids the crossing, both are zero and Horizon says which way.
//
// There is deliberately no night duration. Dusk-to-dawn spans two days, so it
// is not this type's to hold; a caller wanting it subtracts today's Dusk from
// tomorrow's Dawn.
type TwilightEvent struct {
	Dawn    time.Time // zero unless Horizon is Crosses
	Dusk    time.Time // zero unless Horizon is Crosses
	Horizon Horizon
}

// LunarPhaseInfo describes the Moon's current phase.
//
// Two fields published until v5.0.0 were restatements of Elongation and are
// recovered in one expression each: the Moon is waxing when Elongation < 180,
// and a linear estimate of days into the lunation is Elongation / 360 * 29.53059.
// That estimate is why DaysApprox went: elongation does not advance linearly in
// time, so the number was not the lunation age its name promised, and a caller
// who writes the expression at least chooses the approximation knowingly.
type LunarPhaseInfo struct {
	Illumination float64 // percentage 0-100
	Elongation   float64 // degrees 0-360
	Name         string  // "New Moon", "Waxing Crescent", etc.
}

// equatorial represents right ascension and declination in degrees.
// Used internally for coordinate conversions.
type equatorial struct {
	ra  float64
	dec float64
}

// ecliptic represents ecliptic coordinates: longitude and latitude in degrees,
// and distance in kilometers.
// Used internally for lunar position calculations.
type ecliptic struct {
	lon  float64
	lat  float64
	dist float64
}
