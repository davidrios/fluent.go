package fluent

import (
	"fmt"
	"strings"
	"time"
)

// DateTimeValue wraps a point in time in order to comply with the Value API.
//
// Its own String is what an unformatted { $when } produces: an unambiguous
// rendering in UTC. That is deliberately the same thing DATETIME produces with
// no options, so forgetting the function costs correctness rather than meaning.
type DateTimeValue struct {
	Value time.Time
}

// String formats a DateTimeValue in UTC, naming the zone.
func (value *DateTimeValue) String() string {
	return formatDateTime(value.Value, dateTimeOptions{
		location:     time.UTC,
		date:         true,
		clock:        clockMinutes,
		timeZoneName: true,
	})
}

// DateTime returns a new DateTimeValue with the given value; used for variables.
func DateTime(val time.Time) *DateTimeValue {
	return &DateTimeValue{
		Value: val,
	}
}

// How much of the clock a rendering includes.
type clockPrecision int

const (
	clockNone clockPrecision = iota
	clockMinutes
	clockSeconds
)

type dateTimeOptions struct {
	location     *time.Location
	date         bool
	clock        clockPrecision
	timeZoneName bool
}

// dateTimeFunction implements Fluent's DATETIME.
//
// # What it does and does not do
//
// Fluent models DATETIME on ECMA-402's Intl.DateTimeFormat, whose whole subject
// is rendering a date the way a given locale writes one. Go has no CLDR data, so
// that cannot be honestly implemented here — and rendering "September" for every
// locale would be worse than not offering it, because it would look localized.
//
// So this formats in a locale-neutral way: ISO 8601 order, numeric throughout,
// and never a month or weekday name. The options are accepted as ECMA-402 spells
// them so a catalog written against a fuller implementation keeps working, and
// the ones that can only mean "use a name" (month: "long" and friends) record an
// error and fall back to numeric rather than inventing English.
//
// The default zone is UTC rather than the machine's, because a formatted date
// with no zone in it is a date that means something different to every reader,
// and a server rarely knows which zone its reader is in.
func dateTimeFunction(resolver *resolver, positional []Value, named map[string]Value) Value {
	if len(positional) != 1 {
		resolver.errors = append(resolver.errors, fmt.Errorf("DATETIME takes exactly one argument, got %d", len(positional)))
		return &NoValue{value: "DATETIME()"}
	}

	instant, ok := asTime(positional[0])
	if !ok {
		resolver.errors = append(resolver.errors, fmt.Errorf("DATETIME cannot read '%s' as a date", positional[0].String()))
		return &NoValue{value: "DATETIME()"}
	}

	options := dateTimeOptions{location: time.UTC}
	explicit := false

	for name, value := range named {
		text := value.String()
		switch name {
		case "timeZone":
			location, err := time.LoadLocation(text)
			if err != nil {
				resolver.errors = append(resolver.errors, fmt.Errorf("DATETIME: unknown timeZone '%s': %w", text, err))
				continue
			}
			options.location = location
		case "dateStyle":
			// All four styles render alike: without CLDR there is no long form
			// to render, and ISO is the one order every locale can read.
			if !isDateTimeStyle(text) {
				resolver.errors = append(resolver.errors, fmt.Errorf("DATETIME: unknown dateStyle '%s'", text))
				continue
			}
			options.date, explicit = true, true
		case "timeStyle":
			if !isDateTimeStyle(text) {
				resolver.errors = append(resolver.errors, fmt.Errorf("DATETIME: unknown timeStyle '%s'", text))
				continue
			}
			if text == "short" {
				options.clock = clockMinutes
			} else {
				options.clock = clockSeconds
			}
			if text == "full" || text == "long" {
				options.timeZoneName = true
			}
			explicit = true
		case "year", "month", "day":
			if !isNumericOption(resolver, name, text) {
				continue
			}
			options.date, explicit = true, true
		case "hour", "minute":
			if !isNumericOption(resolver, name, text) {
				continue
			}
			if options.clock < clockMinutes {
				options.clock = clockMinutes
			}
			explicit = true
		case "second":
			if !isNumericOption(resolver, name, text) {
				continue
			}
			options.clock = clockSeconds
			explicit = true
		case "timeZoneName":
			if text != "short" && text != "long" {
				resolver.errors = append(resolver.errors, fmt.Errorf("DATETIME: unknown timeZoneName '%s'", text))
				continue
			}
			options.timeZoneName, explicit = true, true
		default:
			resolver.errors = append(resolver.errors, fmt.Errorf("DATETIME: unknown option '%s'", name))
		}
	}

	// No options at all means the whole instant, which is the only default that
	// cannot lose information the catalog did not ask to lose.
	if !explicit {
		options.date = true
		options.clock = clockMinutes
		options.timeZoneName = true
	}

	return &StringValue{Value: formatDateTime(instant, options)}
}

// isNumericOption reports whether a component option asks for digits. Anything
// else -- "long", "short", "narrow" -- asks for a name this cannot write.
func isNumericOption(resolver *resolver, name, value string) bool {
	if value == "numeric" || value == "2-digit" {
		return true
	}
	resolver.errors = append(resolver.errors, fmt.Errorf(
		"DATETIME: %s: '%s' needs locale data this implementation does not have; using digits", name, value))
	return true
}

func isDateTimeStyle(value string) bool {
	switch value {
	case "full", "long", "medium", "short":
		return true
	}
	return false
}

func formatDateTime(instant time.Time, options dateTimeOptions) string {
	location := options.location
	if location == nil {
		location = time.UTC
	}
	instant = instant.In(location)

	parts := make([]string, 0, 3)
	if options.date {
		parts = append(parts, instant.Format("2006-01-02"))
	}
	switch options.clock {
	case clockMinutes:
		parts = append(parts, instant.Format("15:04"))
	case clockSeconds:
		parts = append(parts, instant.Format("15:04:05"))
	}
	if options.timeZoneName {
		parts = append(parts, instant.Format("MST"))
	}
	return strings.Join(parts, " ")
}

// asTime reads the argument DATETIME was handed.
//
// A DateTimeValue is the direct case. A string is accepted when it is RFC 3339,
// which is what a timestamp that has been through JSON looks like -- so a caller
// holding one does not have to parse it back before formatting it.
func asTime(value Value) (time.Time, bool) {
	if datetime, ok := value.(*DateTimeValue); ok {
		return datetime.Value, true
	}
	if text, ok := value.(*StringValue); ok {
		if parsed, err := time.Parse(time.RFC3339, text.Value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}
