package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var offsetRegex = regexp.MustCompile(`^(?:\+|\bin\s+)?(\d+)\s*(d|day|days|w|week|weeks|m|month|months|y|year|years)$`)

func ParseDate(input string) (time.Time, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return time.Time{}, nil
	}

	lower := strings.ToLower(input)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 1. Relative keywords
	switch lower {
	case "today":
		return today, nil
	case "tomorrow", "tmrw":
		return today.AddDate(0, 0, 1), nil
	case "yesterday":
		return today.AddDate(0, 0, -1), nil
	case "sunday", "sun":
		return nextWeekday(today, time.Sunday), nil
	case "monday", "mon":
		return nextWeekday(today, time.Monday), nil
	case "tuesday", "tue":
		return nextWeekday(today, time.Tuesday), nil
	case "wednesday", "wed":
		return nextWeekday(today, time.Wednesday), nil
	case "thursday", "thu":
		return nextWeekday(today, time.Thursday), nil
	case "friday", "fri":
		return nextWeekday(today, time.Friday), nil
	case "saturday", "sat":
		return nextWeekday(today, time.Saturday), nil
	}

	// 2. Relative offsets like "+3d", "+2w", "in 5 days", "1week"
	if matches := offsetRegex.FindStringSubmatch(lower); len(matches) == 3 {
		amount, _ := strconv.Atoi(matches[1])
		unit := matches[2]
		switch {
		case strings.HasPrefix(unit, "d"):
			return today.AddDate(0, 0, amount), nil
		case strings.HasPrefix(unit, "w"):
			return today.AddDate(0, 0, amount*7), nil
		case strings.HasPrefix(unit, "m"):
			return today.AddDate(0, amount, 0), nil
		case strings.HasPrefix(unit, "y"):
			return today.AddDate(amount, 0, 0), nil
		}
	}

	// 3. User configured format
	if cfgFormat := viper.GetString("date_format"); cfgFormat != "" {
		if t, err := time.ParseInLocation(cfgFormat, input, now.Location()); err == nil {
			return t, nil
		}
	}

	// 4. Standard fallback formats
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006/01/02",
		"01/02/2006 15:04:05",
		"01/02/2006 15:04",
		"01/02/2006",
		"02-01-2006 15:04:05",
		"02-01-2006 15:04",
		"02-01-2006",
		"02/01/2006",
		time.RFC3339,
	}

	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, input, now.Location()); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse date '%s'", input)
}

func nextWeekday(from time.Time, target time.Weekday) time.Time {
	daysAhead := int(target) - int(from.Weekday())
	if daysAhead <= 0 {
		daysAhead += 7
	}
	return from.AddDate(0, 0, daysAhead)
}
