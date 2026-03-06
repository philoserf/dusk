package dusk

import (
	"time"
)

type SunriseStatus int

const (
	AboveHorizon = SunriseStatus(1)
	AtHorizon    = SunriseStatus(0)
	BelowHorizon = SunriseStatus(-1)
)

type Twilight struct {
	From     time.Time
	Until    time.Time
	Duration time.Duration
}

// For all twilight funcs, please reference for information on timezones and their respective locations:
// @see https://en.wikipedia.org/wiki/list_of_tz_database_time_zones
// @see https://www.iana.org/time-zones
// @see https://pkg.go.dev/time#LoadLocation

/*
GetLocalTwilight()

@param datetime - the datetime of the observer (in UTC)
@param longitude - is the longitude (west is negative, east is positive) in degrees of some observer on Earth
@param latitude - is the latitude (south is negative, north is positive) in degrees of some observer on Earth
@param elevation - is the elevation (above sea level) in meters of some observer on Earth
@param degreesBelowHorizon - is the degrees below horizon for the designated "twilight period", with 0° being "night" e.g., as soon as the sun is below the horizon.
@param location - the timezone location for the observer (e.g., from time.LoadLocation)
@returns the start and end times of the twilight period for the given degreesBelowHorizon, in the observer's local time.
*/
func GetLocalTwilight(datetime time.Time, longitude, latitude, elevation, degreesBelowHorizon float64, location *time.Location) (Twilight, *time.Location) {
	if location == nil {
		location = time.UTC
	}

	s := GetSunriseSunsetTimesInUTC(datetime, degreesBelowHorizon, longitude, latitude, elevation)

	r := GetSunriseSunsetTimesInUTC(datetime.Add(time.Hour*24), degreesBelowHorizon, longitude, latitude, elevation)

	return Twilight{
		From:     s.Set.In(location),
		Until:    r.Rise.In(location),
		Duration: r.Rise.Sub(s.Set),
	}, location
}

/*
GetLocalCivilTwilight()

@param datetime - the datetime of the observer (in UTC)
@param longitude - is the longitude (west is negative, east is positive) in degrees of some observer on Earth
@param latitude - is the latitude (south is negative, north is positive) in degrees of some observer on Earth
@param elevation - is the elevation (above sea level) in meters of some observer on Earth
@param location - the timezone location for the observer (e.g., from time.LoadLocation)
@returns the start and end times of Civil Twilight, as designated by when the Sun is -6 degrees below the horizon.
*/
func GetLocalCivilTwilight(datetime time.Time, longitude, latitude, elevation float64, location *time.Location) (Twilight, *time.Location) {
	// civil twilight is designated as being 6 degrees below horizon:
	var degreesBelowHorizon float64 = -6

	return GetLocalTwilight(datetime, longitude, latitude, elevation, degreesBelowHorizon, location)
}

/*
GetLocalNauticalTwilight()

@param datetime - the datetime of the observer (in UTC)
@param longitude - is the longitude (west is negative, east is positive) in degrees of some observer on Earth
@param latitude - is the latitude (south is negative, north is positive) in degrees of some observer on Earth
@param elevation - is the elevation (above sea level) in meters of some observer on Earth
@param location - the timezone location for the observer (e.g., from time.LoadLocation)
@returns the start and end times of Nautical Twilight, as designated by when the Sun is -12 degrees below the horizon.
*/
func GetLocalNauticalTwilight(datetime time.Time, longitude, latitude, elevation float64, location *time.Location) (Twilight, *time.Location) {
	// nautical twilight is designated as being 12 degrees below horizon:
	var degreesBelowHorizon float64 = -12

	return GetLocalTwilight(datetime, longitude, latitude, elevation, degreesBelowHorizon, location)
}

/*
GetLocalAstronomicalTwilight()

@param datetime - the datetime of the observer (in UTC)
@param longitude - is the longitude (west is negative, east is positive) in degrees of some observer on Earth
@param latitude - is the latitude (south is negative, north is positive) in degrees of some observer on Earth
@param elevation - is the elevation (above sea level) in meters of some observer on Earth
@param location - the timezone location for the observer (e.g., from time.LoadLocation)
@returns the start and end times of Astronomical Twilight, as designated by when the Sun is -18 degrees below the horizon.
*/
func GetLocalAstronomicalTwilight(datetime time.Time, longitude, latitude, elevation float64, location *time.Location) (Twilight, *time.Location) {
	// astronomical twilight is designated as being 18 degrees below horizon:
	var degreesBelowHorizon float64 = -18

	return GetLocalTwilight(datetime, longitude, latitude, elevation, degreesBelowHorizon, location)
}
