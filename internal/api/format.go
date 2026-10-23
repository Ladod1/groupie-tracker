package api

import (
	"sort"
	"strings"
	"time"
	"unicode"
)

const apiDateLayout = "02-01-2006"

func FormatLocation(raw string) string {
	parts := strings.SplitN(raw, "-", 2)
	city := titleWords(strings.ReplaceAll(parts[0], "_", " "))
	if len(parts) == 1 {
		return city
	}

	country := strings.ReplaceAll(parts[1], "_", " ")
	if len(country) <= 3 {
		country = strings.ToUpper(country)
	} else {
		country = titleWords(country)
	}
	return city + ", " + country
}

func FormatDate(raw string) string {
	clean := strings.TrimPrefix(raw, "*")
	t, err := time.Parse(apiDateLayout, clean)
	if err != nil {
		return clean
	}
	return t.Format("02 Jan 2006")
}

func sortDates(dates []string) {
	sort.SliceStable(dates, func(i, j int) bool {
		a, errA := time.Parse(apiDateLayout, strings.TrimPrefix(dates[i], "*"))
		b, errB := time.Parse(apiDateLayout, strings.TrimPrefix(dates[j], "*"))
		if errA != nil || errB != nil {
			return errA == nil && errB != nil
		}
		return a.Before(b)
	})
}

func buildConcerts(datesLocations map[string][]string) []Concert {
	concerts := make([]Concert, 0, len(datesLocations))
	for location, rawDates := range datesLocations {
		dates := append([]string(nil), rawDates...)
		sortDates(dates)
		for i, d := range dates {
			dates[i] = FormatDate(d)
		}
		concerts = append(concerts, Concert{Location: FormatLocation(location), Dates: dates})
	}
	sort.Slice(concerts, func(i, j int) bool {
		return concerts[i].Location < concerts[j].Location
	})
	return concerts
}

func titleWords(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}
