package dusk

import "time"

// Transit holds rise, set, and maximum transit times for a celestial object,
// along with the duration above the horizon.
// When using ObjectTransit, objects that never rise or set are reported via
// ErrCircumpolar or ErrNeverRises, and successful results always have
// non-zero Rise, Set, and Maximum times.
type Transit struct {
	Rise     time.Time
	Set      time.Time
	Maximum  time.Time
	Duration time.Duration
}

// ObjectTransit computes rise, set, and transit maximum for a celestial
// object at the given observer position on the specified date. Only the
// calendar date (year/month/day in UTC) is used; the time-of-day is ignored.
//
// Observer elevation (obs.Elev) is not used; it only affects sunrise/sunset
// and twilight calculations. Transit maximum precision is ±1 minute.
//
// Returns an error if obs.Loc is nil or coordinates are out of range.
// Returns ErrCircumpolar if the object never sets, or ErrNeverRises if
// it never rises at the given latitude.
func ObjectTransit(date time.Time, eq Equatorial, obs Observer) (Transit, error) {
	if err := validateObserver(obs); err != nil {
		return Transit{}, err
	}
	if err := validateEquatorial(eq); err != nil {
		return Transit{}, err
	}

	if err := objectTransitCheck(eq.Dec, obs.Lat); err != nil {
		return Transit{}, err
	}

	argLST := argOfLSTForTransit(obs.Lat, eq.Dec)

	// LST of rise and set in hours.
	riseLST := mod24(24 + eq.RA/15 - argLST/15)
	setLST := mod24(eq.RA/15 + argLST/15)

	// Convert LST → GST → UT.
	riseGST := lstToGST(riseLST, obs.Lon)
	setGST := lstToGST(setLST, obs.Lon)

	riseUT := gstToUT(date, riseGST)
	setUT := gstToUT(date, setGST)

	// Build time.Time values from UT hours on the given date (UTC midnight).
	midnight := datetimeZeroHour(date)

	rise := midnight.Add(time.Duration(riseUT * float64(time.Hour))).In(obs.Loc)
	set := midnight.Add(time.Duration(setUT * float64(time.Hour))).In(obs.Loc)

	// If set is before rise, push set to the next day.
	if set.Before(rise) {
		nextDay := midnight.AddDate(0, 0, 1)
		setUTNext := gstToUT(nextDay, setGST)
		set = nextDay.Add(time.Duration(setUTNext * float64(time.Hour))).In(obs.Loc)
	}

	// Find transit maximum by minute-by-minute scan from rise to set.
	maximum := findTransitMaximum(rise, set, obs, eq)

	return Transit{
		Rise:     rise,
		Set:      set,
		Maximum:  maximum,
		Duration: set.Sub(rise),
	}, nil
}

// objectTransitCheck returns nil if an object at the given declination rises
// and sets at the given latitude, ErrCircumpolar if it never sets, or
// ErrNeverRises if it never rises.
func objectTransitCheck(dec, lat float64) error {
	product := tanx(lat) * tanx(dec)
	if product >= 1 {
		return ErrCircumpolar
	}
	if product <= -1 {
		return ErrNeverRises
	}
	return nil
}

// argOfLSTForTransit returns the argument of LST for transit in degrees:
// acos(-tan(lat) * tan(dec)).
func argOfLSTForTransit(lat, dec float64) float64 {
	return acosx(-tanx(lat) * tanx(dec))
}

// lstToGST converts local sidereal time (hours) to Greenwich sidereal time
// (hours) by subtracting the observer longitude (degrees) converted to hours.
func lstToGST(lst, longitude float64) float64 {
	return mod24(lst - longitude/15)
}

// gstToUT converts Greenwich sidereal time (hours) to Universal Time (hours)
// for the given date.
//
// This uses the Duffett-Smith / J1900-epoch algorithm rather than the Meeus
// J2000-based formulas used elsewhere in the library. The two epoch systems
// produce equivalent results; this particular algorithm is retained because it
// maps GST→UT directly without iterative inversion.
func gstToUT(datetime time.Time, GST float64) float64 {
	d := datetimeZeroHour(datetime)
	JD := JulianDate(d)
	// January 0 is Go's representation of December 31 of the prior year.
	JD0 := JulianDate(time.Date(datetime.Year(), 1, 0, 0, 0, 0, 0, time.UTC))
	days := JD - JD0
	T := (JD0 - 2415020) / 36525 // Duffett-Smith epoch (J1900)
	R := 6.6460656 + 2400.051262*T + 0.00002581*T*T
	B := 24 - R + float64(24*(datetime.Year()-1900))
	T0 := (0.0657098 * days) - B
	T0 = mod24(T0)
	A := GST - T0
	if A < 0 {
		A += 24
	}
	return 0.997270 * A
}

// findTransitMaximum scans minute-by-minute between rise and set to find the
// time of highest altitude (transit maximum). Precision is ±1 minute.
func findTransitMaximum(rise, set time.Time, obs Observer, eq Equatorial) time.Time {
	maxAlt := -90.0
	var maxTime time.Time

	for t := rise; !t.After(set); t = t.Add(time.Minute) {
		h := EquatorialToHorizontal(t, obs, eq)
		if h.Alt > maxAlt {
			maxAlt = h.Alt
			maxTime = t
		}
	}

	return maxTime
}
