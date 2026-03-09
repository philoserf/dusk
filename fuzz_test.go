package dusk

import (
	"math"
	"testing"
	"time"
)

func FuzzSunriseSunset(f *testing.F) {
	f.Add(40.7128, -74.006, 10.0, int64(1710892800)) // NYC 2024-03-20
	f.Add(-33.87, 151.21, 58.0, int64(1718928000))   // Sydney 2024-06-21
	f.Add(69.65, 18.96, 0.0, int64(1718928000))      // Tromsø summer
	f.Add(-0.18, -78.47, 2800.0, int64(1710892800))  // Quito

	f.Fuzz(func(_ *testing.T, lat, lon, elev float64, unix int64) {
		if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			return
		}
		if math.IsNaN(lat) || math.IsNaN(lon) || math.IsNaN(elev) {
			return
		}
		if math.IsInf(lat, 0) || math.IsInf(lon, 0) || math.IsInf(elev, 0) {
			return
		}
		date := time.Unix(unix, 0).UTC()
		if date.Year() < 1800 || date.Year() > 2200 {
			return
		}
		evt, err := SunriseSunset(date, Observer{Lat: lat, Lon: lon, Elev: elev, Loc: time.UTC})
		if err != nil {
			return
		}
		_ = evt
	})
}

func FuzzLunarPhase(f *testing.F) {
	f.Add(int64(1704931200)) // 2024-01-11
	f.Add(int64(1706140800)) // 2024-01-25
	f.Add(int64(1710892800)) // 2024-03-20

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

func FuzzObjectTransit(f *testing.F) {
	f.Add(40.7128, -74.006, 100.0, 20.0, int64(1710892800))
	f.Add(-33.87, 151.21, 250.0, -60.0, int64(1710892800))

	f.Fuzz(func(_ *testing.T, lat, lon, ra, dec float64, unix int64) {
		if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			return
		}
		if dec < -90 || dec > 90 {
			return
		}
		if math.IsNaN(lat) || math.IsNaN(lon) || math.IsNaN(ra) || math.IsNaN(dec) {
			return
		}
		if math.IsInf(lat, 0) || math.IsInf(lon, 0) || math.IsInf(ra, 0) || math.IsInf(dec, 0) {
			return
		}
		date := time.Unix(unix, 0).UTC()
		if date.Year() < 1800 || date.Year() > 2200 {
			return
		}
		tr, err := ObjectTransit(date, Equatorial{RA: ra, Dec: dec}, Observer{Lat: lat, Lon: lon, Loc: time.UTC})
		if err != nil {
			return
		}
		_ = tr
	})
}

func FuzzMoonriseMoonset(f *testing.F) {
	f.Add(40.7128, -74.006, int64(1705276800)) // NYC 2024-01-15
	f.Add(-33.87, 151.21, int64(1705276800))   // Sydney

	f.Fuzz(func(_ *testing.T, lat, lon float64, unix int64) {
		if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			return
		}
		if math.IsNaN(lat) || math.IsNaN(lon) {
			return
		}
		if math.IsInf(lat, 0) || math.IsInf(lon, 0) {
			return
		}
		date := time.Unix(unix, 0).UTC()
		if date.Year() < 1800 || date.Year() > 2200 {
			return
		}
		evt, err := MoonriseMoonset(date, Observer{Lat: lat, Lon: lon, Loc: time.UTC})
		if err != nil {
			return
		}
		_ = evt
	})
}
