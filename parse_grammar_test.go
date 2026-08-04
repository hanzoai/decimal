// Copyright (c) 2026, Hanzo AI, Inc. MIT OR Apache-2.0.

package decimal

import (
	"strconv"
	"testing"
)

// TestParseRejectsMalformed pins the grammar. Every case here USED TO PARSE, and the
// two families it covers are the two ways an exact parser can be worse than a float
// one: a wrong SIGN and a confident ZERO.
func TestParseRejectsMalformed(t *testing.T) {
	for _, s := range []string{
		// Double sign. "--5" parsed as +5: this function stripped one '-' and
		// big.Int.SetString consumed the other, so a credit read as a charge.
		"--5", "+-5", "-+5", "++5", "--5.00", "--0.01",
		// A sign or a point with no digits at all. These took the intPart==""
		// fallback and returned a zero that nothing distinguished from "0".
		"-", "+", ".", "-.", "+.", "-+", ".-",
		// Sign in the wrong place.
		"5-", "5+", "1.-2", "1.+2",
		// Not a decimal literal.
		"abc", "1.2.3", "1,234", "1_000", "0x10", "1e3", "1.2e3",
		"NaN", "Inf", "-Inf", "٥", "５",
	} {
		t.Run(strconv.Quote(s), func(t *testing.T) {
			d, err := Parse(s)
			if err == nil {
				t.Fatalf("Parse(%q) = %s with no error — malformed input became a value", s, d.String())
			}
			if !d.IsZero() {
				t.Fatalf("Parse(%q) returned %s alongside an error", s, d.String())
			}
		})
	}
}

// TestParseAcceptsWellFormed is the converse guard: the grammar check must not have
// narrowed what Parse legitimately reads. Every case is a literal a real caller
// writes, including the sparse forms around the decimal point.
func TestParseAcceptsWellFormed(t *testing.T) {
	cases := []struct{ in, want string }{
		{"0", "0"},
		{"5", "5"},
		{"-5", "-5"},
		{"+5", "5"},
		{"123", "123"},
		{"19.99", "19.99"},
		{"-19.99", "-19.99"},
		{"+6.60", "6.6"}, // String trims the trailing zero; the SCALE keeps it
		{"-0.00132", "-0.00132"},
		{".5", "0.5"}, // no integer part
		{"-.5", "-0.5"},
		{"+.5", "0.5"},
		{"1.", "1"}, // no fractional part
		{"-1.", "-1"},
		{"0.00", "0"},
		{"-0.00", "0"},         // negative zero is zero
		{"  42.42  ", "42.42"}, // surrounding space is trimmed
		{"000123", "123"},
		{"1234567890123456789012345678901234567890", "1234567890123456789012345678901234567890"},
	}
	for _, tc := range cases {
		t.Run(strconv.Quote(tc.in), func(t *testing.T) {
			d, err := Parse(tc.in)
			if err != nil {
				t.Fatalf("Parse(%q) errored: %v", tc.in, err)
			}
			if got := d.String(); got != tc.want {
				t.Fatalf("Parse(%q) = %s, want %s", tc.in, got, tc.want)
			}
		})
	}
}

// TestParseKeepsWrittenScale pins what the doc actually promises about trailing
// zeros: the SCALE is the number of fractional digits written, even where String
// trims them for display. Rescale to minor units reads the scale, not the rendering.
func TestParseKeepsWrittenScale(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want int32
	}{
		{"6.60", 2}, {"6.6", 1}, {"6", 0}, {"6.", 0},
		{"0.00", 2}, {"-0.00132", 5}, {".5", 1},
	} {
		d, err := Parse(tc.in)
		if err != nil {
			t.Fatalf("Parse(%q) errored: %v", tc.in, err)
		}
		if got := d.Scale(); got != tc.want {
			t.Fatalf("Parse(%q).Scale() = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// TestParseSignIsNeverInverted states the property behind the first family directly:
// no input may produce a value whose sign disagrees with the sign written on it.
func TestParseSignIsNeverInverted(t *testing.T) {
	for _, s := range []string{"-5", "-0.01", "-1234.56", "--5", "-+5", "+-5"} {
		d, err := Parse(s)
		if err != nil {
			continue // refusing is always an acceptable answer
		}
		if d.Sign() > 0 {
			t.Fatalf("Parse(%q) = %s — a negative literal parsed POSITIVE", s, d.String())
		}
	}
}
