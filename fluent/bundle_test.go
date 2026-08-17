package fluent

import (
	"strings"
	"testing"

	"golang.org/x/text/language"
)

const source = `
-brand = Fluent

greeting = Hello, { $name }!

login-code =
    .subject = Your { -brand } code
    .body =
        Your code is { $code }.

        It expires in { $minutes ->
            [one] one minute
           *[other] { $minutes } minutes
        }.

reference = { login-code.subject }
`

func testBundle(t *testing.T) *Bundle {
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

func TestFormatMessage(t *testing.T) {
	bundle := testBundle(t)

	formatted, errs, err := bundle.FormatMessage("greeting", WithVariable("name", "Anna"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) > 0 {
		t.Fatalf("unexpected resolver errors: %v", errs)
	}
	if formatted != "Hello, Anna!" {
		t.Fatalf("got %q", formatted)
	}
}

// A message declaring only attributes has no value of its own. Resolving it used
// to dereference a nil pattern; it reports the absence instead.
func TestFormatMessageWithoutValue(t *testing.T) {
	bundle := testBundle(t)

	if _, _, err := bundle.FormatMessage("login-code"); err == nil {
		t.Fatal("expected an error for a message with no value")
	}
}

func TestFormatMessageAttribute(t *testing.T) {
	bundle := testBundle(t)

	subject, errs, err := bundle.FormatMessageAttribute("login-code", "subject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) > 0 {
		t.Fatalf("unexpected resolver errors: %v", errs)
	}
	if subject != "Your Fluent code" {
		t.Fatalf("got %q", subject)
	}
}

// An attribute is an ordinary pattern: multiline, with placeables and selects.
func TestFormatMessageAttributeResolvesPatterns(t *testing.T) {
	bundle := testBundle(t)

	body, _, err := bundle.FormatMessageAttribute("login-code", "body",
		WithVariables(map[string]interface{}{"code": "123456", "minutes": 1}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(body, "Your code is 123456.") {
		t.Fatalf("got %q", body)
	}
	if !strings.Contains(body, "It expires in one minute.") {
		t.Fatalf("expected the 'one' variant, got %q", body)
	}

	body, _, err = bundle.FormatMessageAttribute("login-code", "body",
		WithVariables(map[string]interface{}{"code": "123456", "minutes": 5}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(body, "It expires in 5 minutes.") {
		t.Fatalf("expected the default variant, got %q", body)
	}
}

func TestFormatMissingMessageAttribute(t *testing.T) {
	bundle := testBundle(t)

	if _, _, err := bundle.FormatMessageAttribute("login-code", "footer"); err == nil {
		t.Fatal("expected an error for an unknown attribute")
	}
	if _, _, err := bundle.FormatMessageAttribute("nope", "subject"); err == nil {
		t.Fatal("expected an error for an unknown message")
	}
}

// Referencing an attribute from within another message goes through the same
// lookup, so the two cannot disagree about which attribute is which.
func TestMessageAttributeReference(t *testing.T) {
	bundle := testBundle(t)

	formatted, _, err := bundle.FormatMessage("reference")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if formatted != "Your Fluent code" {
		t.Fatalf("got %q", formatted)
	}
}

func TestHasMessageAttribute(t *testing.T) {
	bundle := testBundle(t)

	for _, c := range []struct {
		key, attribute string
		want           bool
	}{
		{"login-code", "subject", true},
		{"login-code", "body", true},
		{"login-code", "footer", false},
		{"greeting", "subject", false},
		{"nope", "subject", false},
	} {
		if got := bundle.HasMessageAttribute(c.key, c.attribute); got != c.want {
			t.Errorf("HasMessageAttribute(%q, %q) = %v, want %v", c.key, c.attribute, got, c.want)
		}
	}
}
