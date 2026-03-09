package dusk

import (
	"fmt"
	"time"
)

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "--:--"
	}
	return t.Format("15:04")
}

// String returns a human-readable representation of the equatorial coordinates.
func (e Equatorial) String() string {
	return fmt.Sprintf("RA=%.3f° Dec=%.3f°", e.RA, e.Dec)
}

// String returns a human-readable representation of the horizontal coordinates.
func (h Horizontal) String() string {
	return fmt.Sprintf("Alt=%.3f° Az=%.3f°", h.Alt, h.Az)
}

// String returns a human-readable representation of the ecliptic coordinates.
func (e Ecliptic) String() string {
	return fmt.Sprintf("Lon=%.3f° Lat=%.3f° Dist=%.1fkm", e.Lon, e.Lat, e.Dist)
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

// String returns a human-readable representation of the transit event.
func (t Transit) String() string {
	return fmt.Sprintf("Rise=%s Max=%s Set=%s Duration=%s",
		formatTime(t.Rise),
		formatTime(t.Maximum),
		formatTime(t.Set),
		t.Duration)
}

// String returns a human-readable representation of the twilight event.
func (tw TwilightEvent) String() string {
	return fmt.Sprintf("Dusk=%s Dawn=%s Duration=%s",
		formatTime(tw.Dusk),
		formatTime(tw.Dawn),
		tw.Duration)
}
