package fluent

import (
	"strings"
	"testing"
	"time"

	"golang.org/x/text/language"
)

func dateBundle(t *testing.T, source string) *Bundle {
	t.Helper()
	resource, errs := NewResource(source)
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs[0])
	}
	bundle := NewBundle(language.English)
	if errs := bundle.AddResource(resource); len(errs) > 0 {
		t.Fatalf("add resource: %v", errs[0])
	}
	return bundle
}

var instant = time.Date(2026, time.September, 1, 12, 30, 45, 0, time.UTC)

func TestDateTimeDefaultsToTheWholeInstantInUTC(t *testing.T) {
	bundle := dateBundle(t, `when = { DATETIME($when) }`)

	formatted, errs, err := bundle.FormatMessage("when", WithVariable("when", instant))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) > 0 {
		t.Fatalf("unexpected resolver errors: %v", errs)
	}
	if formatted != "2026-09-01 12:30 UTC" {
		t.Fatalf("got %q", formatted)
	}
}

// An unformatted time renders as the same thing, so forgetting the function
// costs precision rather than meaning.
func TestDateTimeValueStringsItself(t *testing.T) {
	bundle := dateBundle(t, `when = { $when }`)

	formatted, _, err := bundle.FormatMessage("when", WithVariable("when", instant))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if formatted != "2026-09-01 12:30 UTC" {
		t.Fatalf("got %q", formatted)
	}
}

// A timestamp that has been through JSON arrives as a string; it does not have
// to be parsed back before it can be formatted.
func TestDateTimeReadsRFC3339Strings(t *testing.T) {
	bundle := dateBundle(t, `when = { DATETIME($when, timeStyle: "medium") }`)

	formatted, _, err := bundle.FormatMessage("when", WithVariable("when", "2026-09-01T12:30:45Z"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if formatted != "12:30:45" {
		t.Fatalf("got %q", formatted)
	}
}

func TestDateTimeOptions(t *testing.T) {
	for _, c := range []struct {
		options string
		want    string
	}{
		{`dateStyle: "short"`, "2026-09-01"},
		{`dateStyle: "full"`, "2026-09-01"},
		{`timeStyle: "short"`, "12:30"},
		{`timeStyle: "long"`, "12:30:45 UTC"},
		{`dateStyle: "medium", timeStyle: "short"`, "2026-09-01 12:30"},
		{`year: "numeric", month: "2-digit", day: "2-digit"`, "2026-09-01"},
		{`hour: "numeric", minute: "2-digit"`, "12:30"},
		{`dateStyle: "short", timeZone: "America/Sao_Paulo"`, "2026-09-01"},
		{`timeStyle: "short", timeZone: "America/Sao_Paulo"`, "09:30"},
		{`timeStyle: "short", timeZone: "America/Sao_Paulo", timeZoneName: "short"`, "09:30 -03"},
	} {
		bundle := dateBundle(t, `when = { DATETIME($when, `+c.options+`) }`)
		formatted, errs, err := bundle.FormatMessage("when", WithVariable("when", instant))
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.options, err)
		}
		if len(errs) > 0 {
			t.Fatalf("%s: unexpected resolver errors: %v", c.options, errs)
		}
		if formatted != c.want {
			t.Errorf("{ DATETIME($when, %s) } = %q, want %q", c.options, formatted, c.want)
		}
	}
}

// Anything that can only mean "write a name" is reported and falls back to
// digits, rather than inventing English for every locale.
func TestDateTimeRefusesToInventNames(t *testing.T) {
	bundle := dateBundle(t, `when = { DATETIME($when, month: "long", day: "numeric") }`)

	formatted, errs, err := bundle.FormatMessage("when", WithVariable("when", instant))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "locale data") {
		t.Fatalf("expected one explanatory error, got %v", errs)
	}
	if formatted != "2026-09-01" {
		t.Fatalf("got %q", formatted)
	}
}

func TestDateTimeReportsBadInput(t *testing.T) {
	for _, c := range []struct{ source, variable string }{
		{`when = { DATETIME($when) }`, "not a date"},
		{`when = { DATETIME($when, timeZone: "Mars/Olympus") }`, "2026-09-01T12:30:45Z"},
		{`when = { DATETIME($when, dateStyle: "enormous") }`, "2026-09-01T12:30:45Z"},
		{`when = { DATETIME($when, colour: "blue") }`, "2026-09-01T12:30:45Z"},
	} {
		bundle := dateBundle(t, c.source)
		_, errs, err := bundle.FormatMessage("when", WithVariable("when", c.variable))
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.source, err)
		}
		if len(errs) == 0 {
			t.Errorf("%s: expected a resolver error", c.source)
		}
	}
}

// A DATETIME passed in through WithFunction wins, so a catalog needing real
// locale-aware dates is not stuck with this one.
func TestDateTimeCanBeOverridden(t *testing.T) {
	bundle := dateBundle(t, `when = { DATETIME($when) }`)

	formatted, _, err := bundle.FormatMessage("when",
		WithVariable("when", instant),
		WithFunction("DATETIME", func(positional []Value, named map[string]Value) Value {
			return String("1 de setembro de 2026")
		}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if formatted != "1 de setembro de 2026" {
		t.Fatalf("got %q", formatted)
	}
}
