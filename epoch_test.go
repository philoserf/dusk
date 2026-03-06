package dusk

import (
	"math"
	"testing"
	"time"
)

func TestJulianDate(t *testing.T) {
	tests := []struct {
		name    string
		time    time.Time
		want    float64
		epsilon float64
	}{
		{
			name:    "Meeus p.62: 1992-04-12 00:00 UTC",
			time:    time.Date(1992, 4, 12, 0, 0, 0, 0, time.UTC),
			want:    2448724.5,
			epsilon: 0.001,
		},
		{
			name:    "J2000 epoch: 2000-01-01 12:00 UTC",
			time:    time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC),
			want:    2451545.0,
			epsilon: 0.001,
		},
		{
			name:    "Unix epoch: 1970-01-01 00:00 UTC",
			time:    time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			want:    2440587.5,
			epsilon: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JulianDate(tt.time)
			if math.Abs(got-tt.want) > tt.epsilon {
				t.Errorf("JulianDate() = %f, want %f (±%f)", got, tt.want, tt.epsilon)
			}
		})
	}
}

func TestGMST(t *testing.T) {
	tests := []struct {
		name    string
		time    time.Time
		want    float64
		epsilon float64
	}{
		{
			// Meeus p.88 example: 1987-04-10 00:00 UTC → θ₀ = 197.693195°
			name:    "Meeus p.88: 1987-04-10 00:00 UTC",
			time:    time.Date(1987, 4, 10, 0, 0, 0, 0, time.UTC),
			want:    197.693195,
			epsilon: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := greenwichMeanSiderealTime(tt.time)
			if math.Abs(got-tt.want) > tt.epsilon {
				t.Errorf("greenwichMeanSiderealTime() = %f, want %f (±%f)", got, tt.want, tt.epsilon)
			}
		})
	}
}

func TestLocalSiderealTime(t *testing.T) {
	// Meeus p.88: 1987-04-10 00:00 UTC → GMST = 197.693195° = 13.1795h.
	// At longitude 0°, LST = GMST.
	tests := []struct {
		name      string
		time      time.Time
		longitude float64
		want      float64
		epsilon   float64
	}{
		{
			name:      "Meeus p.88: Greenwich (lon=0)",
			time:      time.Date(1987, 4, 10, 0, 0, 0, 0, time.UTC),
			longitude: 0.0,
			want:      197.693195 / 15.0, // 13.1795h
			epsilon:   0.01,
		},
		{
			name:      "Meeus p.88: 15°E (lon=15)",
			time:      time.Date(1987, 4, 10, 0, 0, 0, 0, time.UTC),
			longitude: 15.0,
			want:      mod24(197.693195/15.0 + 1.0), // +1 hour
			epsilon:   0.01,
		},
		{
			name:      "Meeus p.88: 75°W (lon=-75)",
			time:      time.Date(1987, 4, 10, 0, 0, 0, 0, time.UTC),
			longitude: -75.0,
			want:      mod24(197.693195/15.0 - 5.0), // -5 hours
			epsilon:   0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LocalSiderealTime(tt.time, tt.longitude)
			if math.Abs(got-tt.want) > tt.epsilon {
				t.Errorf("LocalSiderealTime() = %f, want %f (±%f)", got, tt.want, tt.epsilon)
			}
		})
	}

	// Verify that longitude shifts LST by the expected amount.
	t.Run("longitude offset shifts LST", func(t *testing.T) {
		dt := time.Date(1987, 4, 10, 0, 0, 0, 0, time.UTC)
		lst0 := LocalSiderealTime(dt, 0.0)
		lst15 := LocalSiderealTime(dt, 15.0)
		diff := lst15 - lst0
		if diff < 0 {
			diff += 24
		}
		if math.Abs(diff-1.0) > 0.001 {
			t.Errorf("15° longitude should shift LST by 1 hour, got shift = %f", diff)
		}
	})
}

func TestMeanObliquity(t *testing.T) {
	tests := []struct {
		name    string
		T       float64
		want    float64
		epsilon float64
	}{
		{
			name:    "J2000 (T=0) obliquity ~23.4393°",
			T:       0.0,
			want:    23.4393,
			epsilon: 0.001,
		},
		{
			name:    "J1900 (T=-1) obliquity ~23.4523°",
			T:       -1.0,
			want:    23.4523,
			epsilon: 0.001,
		},
		{
			name:    "J2100 (T=1) obliquity ~23.4263°",
			T:       1.0,
			want:    23.4263,
			epsilon: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := meanObliquity(tt.T)
			if math.Abs(got-tt.want) > tt.epsilon {
				t.Errorf("meanObliquity() = %f, want %f (±%f)", got, tt.want, tt.epsilon)
			}
		})
	}
}

func TestJulianCentury(t *testing.T) {
	tests := []struct {
		name    string
		time    time.Time
		want    float64
		epsilon float64
	}{
		{
			name:    "J2000 epoch → T = 0.0",
			time:    time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC),
			want:    0.0,
			epsilon: 1e-6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := julianCentury(tt.time)
			if math.Abs(got-tt.want) > tt.epsilon {
				t.Errorf("julianCentury() = %f, want %f (±%f)", got, tt.want, tt.epsilon)
			}
		})
	}
}
