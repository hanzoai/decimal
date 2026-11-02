// Copyright (c) 2026, Hanzo AI, Inc. MIT OR Apache-2.0.

package decimal

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"
)

// TestZeroValueUsable pins the documented contract that the zero value is a valid 0
// (scale 0) that participates in arithmetic without construction.
func TestZeroValueUsable(t *testing.T) {
	var d Decimal
	if got := d.String(); got != "0" {
		t.Errorf("zero value String = %q, want 0", got)
	}
	if !d.IsZero() || d.Sign() != 0 {
		t.Errorf("zero value IsZero/Sign")
	}
	if got := d.Add(New(1, 0)).String(); got != "1" {
		t.Errorf("zero value Add = %q, want 1", got)
	}
	if d.Cmp(Zero()) != 0 || !d.Equal(Zero()) {
		t.Errorf("zero value should equal Zero()")
	}
}

// TestNewNegativeScaleFoldsIntoCoef pins New/NewFromBig's documented negative-scale
// handling: a "negative scale" is folded into the coefficient, scale stays >= 0.
func TestNewNegativeScaleFoldsIntoCoef(t *testing.T) {
	if got := New(5, -2).String(); got != "500" {
		t.Errorf("New(5, -2) = %q, want 500", got)
	}
	if New(5, -2).Scale() != 0 {
		t.Errorf("New(5, -2).Scale = %d, want 0", New(5, -2).Scale())
	}
	if got := NewFromBig(big.NewInt(-7), -1).String(); got != "-70" {
		t.Errorf("NewFromBig(-7, -1) = %q, want -70", got)
	}
}

// TestStringPadsLeadingFractionalZeros pins String's zero-padding between the point
// and the first significant fractional digit.
func TestStringPadsLeadingFractionalZeros(t *testing.T) {
	for in, want := range map[string]string{
		"0.005":    "0.005",
		"-0.005":   "-0.005",
		"0.001234": "0.001234",
		"10.01":    "10.01",
		"-10.01":   "-10.01",
	} {
		if got := MustParse(in).String(); got != want {
			t.Errorf("String(Parse(%q)) = %q, want %q", in, got, want)
		}
	}
}

// TestImmutabilityAcrossOps pins the documented immutability contract: no operation
// mutates its receiver or its operands, so a Decimal is a safe value object.
func TestImmutabilityAcrossOps(t *testing.T) {
	d := MustParse("1.23")
	b := MustParse("-4.5")
	ops := []func(){
		func() { d.Add(b) },
		func() { d.Sub(b) },
		func() { d.Mul(b) },
		func() { d.Quo(b, 6) },
		func() { d.Rescale(0) },
		func() { d.Neg() },
		func() { d.Abs() },
		func() { d.Cmp(b) },
		func() { _ = d.String() },
	}
	for i, op := range ops {
		op()
		if got := d.String(); got != "1.23" {
			t.Fatalf("op %d mutated receiver: %q, want 1.23", i, got)
		}
		if got := b.String(); got != "-4.5" {
			t.Fatalf("op %d mutated operand: %q, want -4.5", i, got)
		}
	}
	// Coef() must hand out a fresh big.Int: mutating it must not leak into d.
	c := d.Coef()
	c.SetInt64(999)
	if got := d.String(); got != "1.23" {
		t.Fatalf("Coef() alias leaked: %q, want 1.23", got)
	}
}

// TestQuoRoundingHalfAwayFromZero pins Quo's documented rounding on both signs,
// including exact ties and negative divisors.
func TestQuoRoundingHalfAwayFromZero(t *testing.T) {
	cases := []struct {
		num, den string
		scale    int32
		want     string
	}{
		{"1", "3", 2, "0.33"},
		{"1", "6", 1, "0.2"},
		{"2.5", "1", 0, "3"},
		{"-2.5", "1", 0, "-3"},
		{"5", "-2", 0, "-3"},
		{"-5", "-2", 0, "3"},
		{"-1", "2", 0, "-1"},
		{"1", "2", 0, "1"},
		{"0", "7", 4, "0"},
	}
	for _, c := range cases {
		if got := MustParse(c.num).Quo(MustParse(c.den), c.scale).String(); got != c.want {
			t.Errorf("%s / %s @%d = %q, want %q", c.num, c.den, c.scale, got, c.want)
		}
	}
}

// TestQuoDivideByZeroPanics pins Quo's documented panic on a zero divisor.
func TestQuoDivideByZeroPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Quo by zero did not panic")
		}
	}()
	MustParse("1").Quo(Zero(), 2)
}

// TestMulAccumulatesScales pins Mul's exact result scale (sum of operand scales) and
// that the extra scale does not leak into the printed value.
func TestMulAccumulatesScales(t *testing.T) {
	p := MustParse("1.20").Mul(MustParse("3.0"))
	if p.Scale() != 3 {
		t.Errorf("Mul scale = %d, want 3", p.Scale())
	}
	if got := p.String(); got != "3.6" {
		t.Errorf("Mul = %q, want 3.6", got)
	}
}

// TestSubBelowZero pins subtraction crossing zero at mixed scales.
func TestSubBelowZero(t *testing.T) {
	if got := MustParse("0.5").Sub(MustParse("1")).String(); got != "-0.5" {
		t.Errorf("sub = %q, want -0.5", got)
	}
	if got := MustParse("0").Sub(MustParse("0.001")).String(); got != "-0.001" {
		t.Errorf("sub = %q, want -0.001", got)
	}
}

// TestRescaleIncreaseIsExact pins that increasing the scale never rounds.
func TestRescaleIncreaseIsExact(t *testing.T) {
	d := MustParse("1.5").Rescale(4)
	if d.Scale() != 4 || d.Coef().String() != "15000" {
		t.Errorf("Rescale(4) = coef %s scale %d, want 15000 @4", d.Coef(), d.Scale())
	}
	if got := d.String(); got != "1.5" {
		t.Errorf("Rescale(4) String = %q, want 1.5", got)
	}
}

// TestJSONRoundTrip pins the JSON contract: marshal emits a quoted exact string,
// unmarshal accepts both quoted strings and bare numbers, and null/empty decode to 0.
func TestJSONRoundTrip(t *testing.T) {
	d := MustParse("6.60")
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != `"6.6"` {
		t.Errorf("marshal = %s, want \"6.6\"", b)
	}
	for _, in := range []string{`"6.60"`, `6.60`, `"-0.005"`, `-0.005`} {
		var got Decimal
		if err := json.Unmarshal([]byte(in), &got); err != nil {
			t.Fatalf("unmarshal %s: %v", in, err)
		}
		if want := MustParse(strings.Trim(in, `"`)); !got.Equal(want) {
			t.Errorf("unmarshal %s = %q, want %q", in, got.String(), want.String())
		}
	}
	var n Decimal
	if err := json.Unmarshal([]byte("null"), &n); err != nil || !n.IsZero() {
		t.Errorf("unmarshal null = %q, err %v; want 0", n.String(), err)
	}
	var bad Decimal
	if err := json.Unmarshal([]byte(`"abc"`), &bad); err == nil {
		t.Error("unmarshal \"abc\": want error")
	}
}

// TestEqualIsScaleIndependent pins that equality is numeric, not representational.
func TestEqualIsScaleIndependent(t *testing.T) {
	if !MustParse("6.60").Equal(MustParse("6.6")) {
		t.Error("6.60 != 6.6")
	}
	if MustParse("6.60").Equal(MustParse("6.61")) {
		t.Error("6.60 == 6.61")
	}
	if !New(10, 1).Equal(New(1, 0)) {
		t.Error("10e-1 != 1")
	}
}

// TestParseEmptyIsZero pins the documented empty-string-is-0 behavior.
func TestParseEmptyIsZero(t *testing.T) {
	for _, s := range []string{"", "  ", "\t\n"} {
		d, err := Parse(s)
		if err != nil || !d.IsZero() {
			t.Errorf("Parse(%q) = %q, err %v; want 0", s, d.String(), err)
		}
	}
}
