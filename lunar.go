package dusk

import "time"

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

// MoonEvent holds the rise and set times for the Moon on a given day, along
// with the duration between rise and set.
type MoonEvent struct {
	Rise     time.Time     // zero value if the Moon does not rise
	Set      time.Time     // zero value if the Moon does not set
	Duration time.Duration // zero if Rise or Set is missing
}

const lunarMonthDays = 29.53059

// lunarHorizonDepression accounts for atmospheric refraction (~0.566°) and
// the Moon's mean semidiameter (~0.25°) when detecting moonrise/moonset.
const lunarHorizonDepression = 0.833

// ---------------------------------------------------------------------------
// Exported functions
// ---------------------------------------------------------------------------

// LunarEclipticPosition returns the geocentric ecliptic longitude, latitude
// (degrees), and distance (km) of the Moon for the given instant.
//
// The algorithm is from Meeus, Astronomical Algorithms, Chapter 47.
func LunarEclipticPosition(t time.Time) Ecliptic {
	T := julianCentury(t)

	D := lunarMeanElongation(T)
	Lp := lunarMeanLongitude(T)
	M := solarMeanAnomalyFromCentury(T)
	Mp := lunarMeanAnomaly(T)
	F := lunarArgumentOfLatitude(T)

	A1 := mod360(119.75 + 131.849*T)
	A2 := mod360(53.09 + 479264.29*T)
	A3 := mod360(313.45 + 481266.484*T)

	E := 1 - 0.002516*T - 0.0000074*T*T
	E2 := E * E

	// Additive corrections in units of 0.000001 degrees (Meeus p. 338).
	Sl := 3958*sinx(A1) + 1962*sinx(Lp-F) + 318*sinx(A2)
	Sr := 0.0
	Sb := -2235*sinx(Lp) + 382*sinx(A3) + 175*sinx(A1-F) + 175*sinx(A1+F) + 127*sinx(Lp-Mp) - 115*sinx(Lp+Mp)

	for i := range tableLongDist {
		r := &tableLongDist[i]
		arg := D*r.D + M*r.M + Mp*r.Mʹ + F*r.F
		sa, ca := sincosx(arg)

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
		}
	}

	for i := range tableLat {
		r := &tableLat[i]
		arg := D*r.D + M*r.M + Mp*r.Mʹ + F*r.F
		sb := sinx(arg)

		switch r.M {
		case 0:
			Sb += r.Σb * sb
		case 1, -1:
			Sb += r.Σb * sb * E
		case 2, -2:
			Sb += r.Σb * sb * E2
		}
	}

	return Ecliptic{
		Lon:  mod360(Lp + Sl/1e6),
		Lat:  Sb / 1e6,
		Dist: 385000.56 + Sr/1000,
	}
}

// LunarPhase returns the lunar phase for the given instant.
//
// The phase angle uses the Meeus approach: solar ecliptic longitude from the
// mean-anomaly method, lunar ecliptic position from Chapter 47 tables.
func LunarPhase(t time.Time) LunarPhaseInfo {
	ec := LunarEclipticPosition(t)

	J := JulianDate(t) - j2000
	Msol := solarMeanAnomaly(J)
	C := solarEquationOfCenter(Msol)
	sunLon := solarEclipticLongitude(Msol, C)

	T := julianCentury(t)
	Mp := lunarMeanAnomaly(T)

	// elongation (0-360°, waxing = 0-180, waning = 180-360)
	d := acosx(cosx(ec.Lon-sunLon) * cosx(ec.Lat))
	if mod360(ec.Lon-sunLon) > 180 {
		d = 360 - d
	}

	// phase angle (Meeus p. 346)
	PA := 180 - d - 0.1468*((1-0.0549*sinx(Mp))/(1-0.0167*sinx(Msol)))*sinx(d)

	K := 100 * (1 + cosx(PA)) / 2

	days := d / 360 * lunarMonthDays

	return LunarPhaseInfo{
		Illumination: K,
		Elongation:   d,
		Angle:        PA,
		DaysApprox:   days,
		Waxing:       d < 180,
		Name:         lunarPhaseName(d),
	}
}

// LunarPosition returns the equatorial coordinates (RA, Dec) of the Moon for
// a given instant, using the Meeus ecliptic position converted to equatorial.
func LunarPosition(t time.Time) Equatorial {
	ec := LunarEclipticPosition(t)
	return EclipticToEquatorial(t, ec.Lon, ec.Lat)
}

// MoonriseMoonset computes the moonrise and moonset times for the given date
// at the specified observer position and timezone.
//
// The algorithm scans minute-by-minute through the day to detect altitude
// sign changes. This is slow by design (~1440 ecliptic-position evaluations).
// A single call takes approximately 1-2 ms on modern hardware (Apple M-series
// or equivalent; see BenchmarkMoonriseMoonset). Callers computing
// moonrise/moonset for many dates (e.g., a 30-day calendar ≈ 30-60 ms)
// should expect proportional cost and may benefit from caching or
// parallelization.
//
// Observer elevation (obs.Elev) is not used; it only affects sunrise/sunset
// and twilight calculations.
//
// An error is returned if obs.Loc is nil.
func MoonriseMoonset(date time.Time, obs Observer) (MoonEvent, error) {
	if err := validateObserver(obs); err != nil {
		return MoonEvent{}, err
	}

	localDate := date.In(obs.Loc)
	d := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, obs.Loc).UTC()
	nextMidnight := time.Date(localDate.Year(), localDate.Month(), localDate.Day()+1, 0, 0, 0, 0, obs.Loc).UTC()
	scanMinutes := int(nextMidnight.Sub(d).Minutes())

	var rise, set time.Time

	ec0 := LunarEclipticPosition(d)
	eq0 := EclipticToEquatorial(d, ec0.Lon, ec0.Lat)
	prevAlt := EquatorialToHorizontal(d, obs, eq0).Alt

	for i := 1; i <= scanMinutes; i++ {
		cur := d.Add(time.Duration(i) * time.Minute)

		ec := LunarEclipticPosition(cur)
		eq := EclipticToEquatorial(cur, ec.Lon, ec.Lat)
		hz := EquatorialToHorizontal(cur, obs, eq)

		if rise.IsZero() && hz.Alt > -lunarHorizonDepression && prevAlt <= -lunarHorizonDepression {
			rise = cur.In(obs.Loc)
		}
		if set.IsZero() && hz.Alt < -lunarHorizonDepression && prevAlt >= -lunarHorizonDepression {
			set = cur.In(obs.Loc)
		}
		if !rise.IsZero() && !set.IsZero() {
			break
		}

		prevAlt = hz.Alt
	}

	var dur time.Duration
	if !rise.IsZero() && !set.IsZero() && set.After(rise) {
		dur = set.Sub(rise)
	}

	return MoonEvent{
		Rise:     rise,
		Set:      set,
		Duration: dur,
	}, nil
}

// ---------------------------------------------------------------------------
// Unexported helpers (Meeus Chapter 47)
// ---------------------------------------------------------------------------

// lunarMeanElongation returns the Moon's mean elongation in degrees.
//
// T is Julian centuries since J2000.0.
// See Meeus eq. 47.2 p. 338.
func lunarMeanElongation(T float64) float64 {
	return mod360(297.8501921 + 445267.1114034*T - 0.0018819*T*T + T*T*T/545868 - T*T*T*T/113065000)
}

// lunarMeanAnomaly returns the Moon's mean anomaly in degrees.
//
// T is Julian centuries since J2000.0.
// See Meeus eq. 47.4 p. 338.
func lunarMeanAnomaly(T float64) float64 {
	return mod360(134.9633964 + 477198.8675055*T + 0.0087414*T*T + T*T*T/69699 - T*T*T*T/14712000)
}

// lunarArgumentOfLatitude returns the Moon's argument of latitude in degrees.
//
// T is Julian centuries since J2000.0.
// See Meeus eq. 47.5 p. 338.
func lunarArgumentOfLatitude(T float64) float64 {
	return mod360(93.272095 + 483202.0175233*T - 0.0036539*T*T + T*T*T/3526000 - T*T*T*T/863310000)
}

// lunarPhaseName returns the common name for the lunar phase based on the
// elongation angle in degrees.
func lunarPhaseName(age float64) string {
	age = mod360(age)

	switch {
	case age < 22.5 || age >= 337.5:
		return "New Moon"
	case age < 67.5:
		return "Waxing Crescent"
	case age < 112.5:
		return "First Quarter"
	case age < 157.5:
		return "Waxing Gibbous"
	case age < 202.5:
		return "Full Moon"
	case age < 247.5:
		return "Waning Gibbous"
	case age < 292.5:
		return "Last Quarter"
	default:
		return "Waning Crescent"
	}
}
