package dusk

import (
	"math"
	"testing"
	"time"
)

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
			eq := EclipticToEquatorial(tt.dt, tt.lon, tt.lat)
			if math.Abs(eq.RA-tt.wantRA) > tt.eps {
				t.Errorf("RA = %.4f, want ~%.1f (within %.2f°)", eq.RA, tt.wantRA, tt.eps)
			}
			if math.Abs(eq.Dec-tt.wantDec) > tt.eps {
				t.Errorf("Dec = %.4f, want ~%.2f (within %.2f°)", eq.Dec, tt.wantDec, tt.eps)
			}
		})
	}
}

func TestEquatorialToHorizontal(t *testing.T) {
	// Sirius observed from NYC on 2024-01-16 02:00 UTC (~9pm EST).
	// Sirius transits ~00:40 local in mid-January; at 9pm it is well up in the SE.
	dt := time.Date(2024, 1, 16, 2, 0, 0, 0, time.UTC)
	obs := Observer{Lat: 40.7128, Lon: -74.006}
	sirius := Equatorial{RA: 101.287, Dec: -16.716}

	h := EquatorialToHorizontal(dt, obs, sirius)

	// Algorithm-computed reference: Sirius from NYC at 2024-01-16 02:00 UTC.
	// Alt ~26°, Az ~147° (SE sky, still rising toward transit).
	const eps = 1.0
	if math.Abs(h.Alt-26.06) > eps {
		t.Errorf("Alt = %.4f, want ~26.06° (within %.0f°)", h.Alt, eps)
	}
	if math.Abs(h.Az-147.49) > eps {
		t.Errorf("Az = %.4f, want ~147.49° (within %.0f°)", h.Az, eps)
	}
}

func TestEquatorialToHorizontal_Pole(t *testing.T) {
	// Observer at the North Pole — cosAltCosLat guard triggers, azimuth defaults to 0.
	dt := time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC)
	obs := Observer{Lat: 90.0, Lon: 0}
	star := Equatorial{RA: 0, Dec: 45}

	h := EquatorialToHorizontal(dt, obs, star)
	if h.Az != 0 {
		t.Errorf("Az = %.4f, want 0 at pole (guard branch)", h.Az)
	}
	// Altitude should equal declination at the pole.
	if math.Abs(h.Alt-45.0) > 0.5 {
		t.Errorf("Alt = %.4f, want ~45° at North Pole for Dec=45°", h.Alt)
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
			ha := HourAngle(tt.ra, tt.lst)
			if math.Abs(ha-tt.wantHA) > 1e-9 {
				t.Errorf("HourAngle(%v, %v) = %v, want %v", tt.ra, tt.lst, ha, tt.wantHA)
			}
		})
	}
}

func TestAngularSeparation(t *testing.T) {
	tests := []struct {
		name                 string
		ra1, dec1, ra2, dec2 float64
		want                 float64
		eps                  float64
	}{
		{"same point", 90, 45, 90, 45, 0, 1e-9},
		{"opposite poles", 0, 90, 0, -90, 180, 1e-9},
		{"90 degrees apart", 0, 0, 90, 0, 90, 1e-9},
		{"pole to equator", 0, 90, 0, 0, 90, 1e-9},
		{"small angle 1 degree apart", 0, 0, 0, 1, 1.0, 0.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AngularSeparation(tt.ra1, tt.dec1, tt.ra2, tt.dec2)
			if math.Abs(got-tt.want) > tt.eps {
				t.Errorf("AngularSeparation = %.10f, want %.10f", got, tt.want)
			}
		})
	}
}
