package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/philoserf/dusk/v5"
)

// testObserver returns a mid-latitude observer for deterministic tests.
func testObserver(t *testing.T) (dusk.Observer, *time.Location) {
	t.Helper()

	loc, err := time.LoadLocation("America/Detroit")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}

	obs, err := dusk.NewObserver(42.9634, -85.6681, loc)
	if err != nil {
		t.Fatalf("NewObserver: %v", err)
	}

	return obs, loc
}

// TestTwilightDawnComesFromYesterday pins down the library's most easily missed
// detail. dusk.TwilightEvent.Dawn is always *tomorrow* morning's, so a day
// report must take this morning's dawn from yesterday's event. Reporting
// today's Dawn field would silently show tomorrow's time.
func TestTwilightDawnAndDuskComeFromOneDay(t *testing.T) {
	t.Parallel()

	obs, _ := testObserver(t)
	date := dusk.Date{Year: 2025, Month: 6, Day: 21}

	got, err := twilightReports(date, obs)
	if err != nil {
		t.Fatalf("twilightReports: %v", err)
	}

	if len(got) != len(twilightBands) {
		t.Fatalf("got %d bands, want %d", len(got), len(twilightBands))
	}

	today, err := dusk.Twilight(date, obs, 6)
	if err != nil {
		t.Fatalf("Twilight(today, 6): %v", err)
	}

	civil := got[0]

	// One call supplies both boundaries. Before v5 the report took Dawn from
	// yesterday's event and Dusk from today's, because a TwilightEvent spanned
	// two days; this asserts that it no longer has to.
	if !civil.Dawn.Equal(toSecond(today.Dawn)) {
		t.Errorf("Dawn = %v, want today's Dawn %v", civil.Dawn, toSecond(today.Dawn))
	}

	if !civil.Dusk.Equal(toSecond(today.Dusk)) {
		t.Errorf("Dusk = %v, want today's Dusk %v", civil.Dusk, toSecond(today.Dusk))
	}

	if civil.Dawn.Day() != date.Day || civil.Dusk.Day() != date.Day {
		t.Errorf("Dawn on day %d and Dusk on day %d, want both on the reported day %d",
			civil.Dawn.Day(), civil.Dusk.Day(), date.Day)
	}

	if !civil.Dawn.Before(civil.Dusk) {
		t.Errorf("Dawn %v should precede Dusk %v on the same day", civil.Dawn, civil.Dusk)
	}
}

// TestNotesDistinguishStates checks the JSON-facing prose. The two states read
// the same way at any depression now that they are named for the geometry:
// above the angle is a night that never darkens, below it a day that never
// lightens.
func TestNotesDistinguishStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		got     string
		wantSub string
	}{
		{"solar stays above", solarNotes[dusk.StaysAbove], "midnight sun"},
		{"solar stays below", solarNotes[dusk.StaysBelow], "polar night"},
		{"solar crosses", solarNotes[dusk.Crosses], ""},
		{"twilight stays above", twilightNote(dusk.StaysAbove, 6), "never gets this dark"},
		{"twilight stays below", twilightNote(dusk.StaysBelow, 6), "this dark all day"},
		{"twilight crosses", twilightNote(dusk.Crosses, 6), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.wantSub == "" {
				if tt.got != "" {
					t.Errorf("note = %q, want empty", tt.got)
				}

				return
			}

			if !strings.Contains(tt.got, tt.wantSub) {
				t.Errorf("note = %q, want it to mention %q", tt.got, tt.wantSub)
			}
		})
	}
}

// TestBuildReportPropagatesRealErrors checks that a genuine failure is not
// swallowed as a polar note.
func TestBuildReportPropagatesRealErrors(t *testing.T) {
	t.Parallel()

	obs, _ := testObserver(t)
	date := dusk.Date{Year: 1600, Month: 1, Day: 1}

	_, err := buildReport(obs, date)
	if !errors.Is(err, dusk.ErrDateOutOfRange) {
		t.Errorf("err = %v, want ErrDateOutOfRange", err)
	}
}

// TestBuildReportPolar checks that polar geometry produces a complete report
// with notes instead of an error.
func TestBuildReportPolar(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}

	obs, err := dusk.NewObserver(69.6492, 18.9553, loc)
	if err != nil {
		t.Fatalf("NewObserver: %v", err)
	}

	date := dusk.Date{Year: 2025, Month: 12, Day: 21}

	report, err := buildReport(obs, date)
	if err != nil {
		t.Fatalf("buildReport: %v", err)
	}

	if report.Sun.Note == "" {
		t.Error("expected a polar note on the sun")
	}

	if !report.Sun.Rise.IsZero() || !report.Sun.Set.IsZero() {
		t.Error("expected zero rise and set times during polar night")
	}

	if report.Phase.Name == "" {
		t.Error("lunar phase should still be reported during polar night")
	}
}

// TestShortDuration covers the duration formatter.
func TestShortDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{"hours and minutes", 15*time.Hour + 21*time.Minute, "15h21m"},
		{"pads minutes", 4*time.Hour + 2*time.Minute, "4h02m"},
		{"rounds seconds away", 4*time.Hour + 2*time.Minute + 40*time.Second, "4h03m"},
		{"zero", 0, ""},
		{"negative", -time.Hour, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := shortDuration(tt.in); got != tt.want {
				t.Errorf("shortDuration(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestToSecond checks that sub-second noise is dropped and zero stays zero.
func TestToSecond(t *testing.T) {
	t.Parallel()

	t.Run("zero stays zero", func(t *testing.T) {
		t.Parallel()

		if got := toSecond(time.Time{}); !got.IsZero() {
			t.Errorf("toSecond(zero) = %v, want zero", got)
		}
	})

	t.Run("drops nanoseconds", func(t *testing.T) {
		t.Parallel()

		in := time.Date(2025, 6, 21, 5, 3, 12, 987654321, time.UTC)
		if got := toSecond(in); got.Nanosecond() != 0 {
			t.Errorf("toSecond kept %d nanoseconds", got.Nanosecond())
		}
	})
}

// observerAt returns an observer at the given place, failing the test on error.
func observerAt(t *testing.T, lat, lon float64, zone string) dusk.Observer {
	t.Helper()

	loc, err := time.LoadLocation(zone)
	if err != nil {
		t.Fatalf("LoadLocation(%s): %v", zone, err)
	}

	obs, err := dusk.NewObserver(lat, lon, loc)
	if err != nil {
		t.Fatalf("NewObserver: %v", err)
	}

	return obs
}

// wantNight is the night a band should report: tomorrow's Dawn minus today's
// Dusk, from the library's untruncated instants, or "" when the band does not
// cross on both days.
func wantNight(t *testing.T, obs dusk.Observer, date dusk.Date, degrees int) string {
	t.Helper()

	next := dusk.Date{Year: date.Year, Month: date.Month, Day: date.Day + 1}

	today, err := dusk.Twilight(date, obs, float64(degrees))
	if err != nil {
		t.Fatalf("Twilight(today, %d): %v", degrees, err)
	}

	tomorrow, err := dusk.Twilight(next, obs, float64(degrees))
	if err != nil {
		t.Fatalf("Twilight(tomorrow, %d): %v", degrees, err)
	}

	if today.Horizon != dusk.Crosses || tomorrow.Horizon != dusk.Crosses {
		return ""
	}

	return shortDuration(tomorrow.Dawn.Sub(today.Dusk))
}

// nightLength parses a band's Night back into a duration, for comparison.
func nightLength(t *testing.T, band TwilightReport) time.Duration {
	t.Helper()

	d, err := time.ParseDuration(band.Night)
	if err != nil {
		t.Fatalf("%s: Night %q: %v", band.Name, band.Night, err)
	}

	return d
}

// TestTwilightNightOnEveryBand is the regression for issue #136: from v5.0.0
// to v5.2.0 only the deepest band carried Night, so civil and nautical lost it
// from the JSON.
func TestTwilightNightOnEveryBand(t *testing.T) {
	t.Parallel()

	obs := observerAt(t, 43.1155, -85.5664, "America/Detroit")
	date := dusk.Date{Year: 2026, Month: 11, Day: 12}

	got, err := twilightReports(date, obs)
	if err != nil {
		t.Fatalf("twilightReports: %v", err)
	}

	for idx, band := range got {
		today, err := dusk.Twilight(date, obs, float64(band.degrees))
		if err != nil {
			t.Fatalf("Twilight(%d): %v", band.degrees, err)
		}

		if !band.Dawn.Equal(toSecond(today.Dawn)) || !band.Dusk.Equal(toSecond(today.Dusk)) {
			t.Errorf("%s: Dawn/Dusk = %v/%v, want today's %v/%v",
				band.Name, band.Dawn, band.Dusk, toSecond(today.Dawn), toSecond(today.Dusk))
		}

		want := wantNight(t, obs, date, band.degrees)
		if want == "" {
			t.Fatalf("%s: fixture should cross on both days", band.Name)
		}

		if band.Night != want {
			t.Errorf("%s: Night = %q, want %q", band.Name, band.Night, want)
		}

		// Deeper bands start later and end earlier, so their nights are shorter.
		if idx > 0 && nightLength(t, band) >= nightLength(t, got[idx-1]) {
			t.Errorf("%s night %q should be shorter than %s night %q",
				band.Name, band.Night, got[idx-1].Name, got[idx-1].Night)
		}
	}
}

// TestTwilightNightOmittedWhenABandDoesNotCross covers the bands with no night
// to measure: one that does not cross today, and one that crosses today but not
// tomorrow, as at a polar transition.
func TestTwilightNightOmittedWhenABandDoesNotCross(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		obs   dusk.Observer
		date  dusk.Date
		today []dusk.Horizon // civil, nautical, astronomical
		next  []dusk.Horizon
	}{
		{
			// Midsummer in Tromsø: the sun never sinks to any band.
			name:  "stays above",
			obs:   observerAt(t, 69.6492, 18.9553, "Europe/Oslo"),
			date:  dusk.Date{Year: 2026, Month: 6, Day: 21},
			today: []dusk.Horizon{dusk.StaysAbove, dusk.StaysAbove, dusk.StaysAbove},
			next:  []dusk.Horizon{dusk.StaysAbove, dusk.StaysAbove, dusk.StaysAbove},
		},
		{
			// Midwinter at McMurdo: civil depth all day, the deeper bands cross.
			name:  "stays below",
			obs:   observerAt(t, -77.8419, 166.6863, "Antarctica/McMurdo"),
			date:  dusk.Date{Year: 2026, Month: 6, Day: 21},
			today: []dusk.Horizon{dusk.StaysBelow, dusk.Crosses, dusk.Crosses},
			next:  []dusk.Horizon{dusk.StaysBelow, dusk.Crosses, dusk.Crosses},
		},
		{
			// Tromsø's last astronomical night of spring: astronomical crosses
			// tonight but not tomorrow, so its night has no end.
			name:  "crosses today but not tomorrow",
			obs:   observerAt(t, 69.6492, 18.9553, "Europe/Oslo"),
			date:  dusk.Date{Year: 2026, Month: 3, Day: 26},
			today: []dusk.Horizon{dusk.Crosses, dusk.Crosses, dusk.Crosses},
			next:  []dusk.Horizon{dusk.Crosses, dusk.Crosses, dusk.StaysAbove},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := twilightReports(tt.date, tt.obs)
			if err != nil {
				t.Fatalf("twilightReports: %v", err)
			}

			next := dusk.Date{Year: tt.date.Year, Month: tt.date.Month, Day: tt.date.Day + 1}

			for idx, band := range got {
				tomorrow, twErr := dusk.Twilight(next, tt.obs, float64(band.degrees))
				if twErr != nil {
					t.Fatalf("Twilight(%d): %v", band.degrees, twErr)
				}

				// The fixture's premise, stated so a library change that moves
				// it fails here rather than passing vacuously.
				if band.horizon != tt.today[idx] || tomorrow.Horizon != tt.next[idx] {
					t.Fatalf("%s: horizons %v/%v, fixture assumes %v/%v",
						band.Name, band.horizon, tomorrow.Horizon, tt.today[idx], tt.next[idx])
				}

				crossesBoth := tt.today[idx] == dusk.Crosses && tt.next[idx] == dusk.Crosses
				if crossesBoth != (band.Night != "") {
					t.Errorf("%s: Night = %q, crosses both days = %v", band.Name, band.Night, crossesBoth)
				}

				if want := wantNight(t, tt.obs, tt.date, band.degrees); band.Night != want {
					t.Errorf("%s: Night = %q, want %q", band.Name, band.Night, want)
				}
			}

			raw, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			var decoded []map[string]any

			err = json.Unmarshal(raw, &decoded)
			if err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}

			for idx, band := range decoded {
				if _, ok := band["night"]; ok != (got[idx].Night != "") {
					t.Errorf("%s: JSON night present = %v, want %v", got[idx].Name, ok, got[idx].Night != "")
				}
			}
		})
	}
}

// TestTwilightNightAcrossCalendarBoundaries checks that tomorrow is the next
// calendar day across a year end and both of Detroit's DST changes, and that
// Night is elapsed time rather than a clock-face difference.
func TestTwilightNightAcrossCalendarBoundaries(t *testing.T) {
	t.Parallel()

	obs := observerAt(t, 43.1155, -85.5664, "America/Detroit")

	for _, date := range []dusk.Date{
		{Year: 2026, Month: 12, Day: 31},
		{Year: 2026, Month: 3, Day: 7},
		{Year: 2026, Month: 10, Day: 31},
	} {
		got, err := twilightReports(date, obs)
		if err != nil {
			t.Fatalf("%v: twilightReports: %v", date, err)
		}

		for _, band := range got {
			want := wantNight(t, obs, date, band.degrees)
			if want == "" || band.Night != want {
				t.Errorf("%v %s: Night = %q, want %q", date, band.Name, band.Night, want)
			}
		}
	}
}
