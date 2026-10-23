package api

import (
	"reflect"
	"testing"
)

func TestFormatLocation(t *testing.T) {
	tests := map[string]string{
		"north_carolina-usa":      "North Carolina, USA",
		"london-uk":               "London, UK",
		"playa_del_carmen-mexico": "Playa Del Carmen, Mexico",
		"auckland-new_zealand":    "Auckland, New Zealand",
		"berlin":                  "Berlin",
	}
	for in, want := range tests {
		if got := FormatLocation(in); got != want {
			t.Errorf("FormatLocation(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatDate(t *testing.T) {
	tests := map[string]string{
		"23-08-2019":  "23 Aug 2019",
		"*05-12-1997": "05 Dec 1997",
		"not a date":  "not a date",
	}
	for in, want := range tests {
		if got := FormatDate(in); got != want {
			t.Errorf("FormatDate(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildConcertsSortsLocationsAndDates(t *testing.T) {
	got := buildConcerts(map[string][]string{
		"tokyo-japan":    {"10-02-2020", "08-02-2020"},
		"berlin-germany": {"01-01-2019"},
	})
	want := []Concert{
		{Location: "Berlin, Germany", Dates: []string{"01 Jan 2019"}},
		{Location: "Tokyo, Japan", Dates: []string{"08 Feb 2020", "10 Feb 2020"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildConcerts = %+v, want %+v", got, want)
	}
}
