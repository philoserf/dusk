package dusk

import "time"

// TwilightEvent holds the dusk and dawn times of a twilight period.
// Dusk is tonight's boundary (sun passes below the depression angle).
// Dawn is tomorrow morning's boundary (sun passes above the depression angle).
// To get this morning's dawn, call with yesterday's date.
type TwilightEvent struct {
	Dusk     time.Time     // evening boundary (today)
	Dawn     time.Time     // morning boundary (tomorrow)
	Duration time.Duration // time from Dusk to Dawn (overnight period below the depression angle)
}

// CivilTwilight computes the evening civil twilight period (Sun 6° below the
// horizon) for the given date and observer position. Dusk is tonight's civil
// dusk; Dawn is tomorrow morning's civil dawn.
func CivilTwilight(date time.Time, obs Observer) (TwilightEvent, error) {
	return twilight(date, obs, 6)
}

// NauticalTwilight computes the evening nautical twilight period (Sun 12°
// below the horizon) for the given date and observer position.
func NauticalTwilight(date time.Time, obs Observer) (TwilightEvent, error) {
	return twilight(date, obs, 12)
}

// AstronomicalTwilight computes the evening astronomical twilight period (Sun
// 18° below the horizon) for the given date and observer position.
func AstronomicalTwilight(date time.Time, obs Observer) (TwilightEvent, error) {
	return twilight(date, obs, 18)
}

// twilight computes the twilight period for a given depression angle (positive
// degrees below the geometric horizon). Only the calendar date is used; the
// time-of-day is ignored. The returned Dusk is today's "set" at the depression
// angle (evening boundary) and Dawn is tomorrow's "rise" at the depression
// angle (morning boundary).
func twilight(date time.Time, obs Observer, depression float64) (TwilightEvent, error) {
	if err := validateObserver(obs); err != nil {
		return TwilightEvent{}, err
	}

	// Evening twilight: sunset at the given depression angle for today.
	sp := computeSolarParams(date, obs.Lon)
	omega, err := solarHourAngle(sp.delta, depression, obs.Lat, obs.Elev)
	if err != nil {
		return TwilightEvent{}, err
	}
	dusk := universalTimeFromJD(sp.jTransit + omega/360).In(obs.Loc)

	// Tomorrow's "rise" at this depression = twilight dawn.
	tomorrow := date.AddDate(0, 0, 1)
	sp2 := computeSolarParams(tomorrow, obs.Lon)
	omega2, err2 := solarHourAngle(sp2.delta, depression, obs.Lat, obs.Elev)
	if err2 != nil {
		return TwilightEvent{}, err2
	}
	dawn := universalTimeFromJD(sp2.jTransit - omega2/360).In(obs.Loc)

	return TwilightEvent{
		Dusk:     dusk,
		Dawn:     dawn,
		Duration: dawn.Sub(dusk),
	}, nil
}
