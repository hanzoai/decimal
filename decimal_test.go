package decimal

import "testing"

func TestParseString(t *testing.T) {
	for _, s := range []string{"0", "6.6", "0.00132", "-0.5", "100", "-123.456", "1000000"} {
		if got := MustParse(s).String(); got != s {
			t.Errorf("Parse/String(%q) = %q", s, got)
		}
	}
	if MustParse("6.60").String() != "6.6" { // trailing zero trimmed
		t.Errorf("6.60 should print 6.6")
	}
}

func TestNoFloat(t *testing.T) {
	// the canonical float bug: 0.1 + 0.2 != 0.3 in float64; exact here.
	if got := MustParse("0.1").Add(MustParse("0.2")).String(); got != "0.3" {
		t.Fatalf("0.1 + 0.2 = %q, want 0.3", got)
	}
}

func TestArithmetic(t *testing.T) {
	if got := MustParse("1").Add(MustParse("0.00132")).String(); got != "1.00132" {
		t.Errorf("add = %q", got)
	}
	if got := MustParse("6.60").Sub(MustParse("0.60")).String(); got != "6" {
		t.Errorf("sub = %q", got)
	}
	// 200 tokens × $6.60/1M = 200 × 6.60 / 1e6 = 0.00132 exactly (the anti-cents-flooring case)
	price := MustParse("6.60")
	perM := MustParse("1000000")
	if got := New(200, 0).Mul(price).Quo(perM, 18).String(); got != "0.00132" {
		t.Fatalf("token cost = %q, want 0.00132", got)
	}
	if MustParse("5").Cmp(MustParse("5.0")) != 0 || MustParse("5").Cmp(MustParse("5.01")) != -1 {
		t.Errorf("cmp")
	}
}

func TestRounding(t *testing.T) {
	if got := MustParse("1.455").Round(2).String(); got != "1.46" { // half-up
		t.Errorf("round = %q", got)
	}
	if got := MustParse("-1.455").Round(2).String(); got != "-1.46" { // away from zero
		t.Errorf("round neg = %q", got)
	}
	if got := MustParse("1.5").Round(0).String(); got != "2" {
		t.Errorf("round to int = %q", got)
	}
}

func TestBigPrecisionNoCeiling(t *testing.T) {
	// The whole point: an 18-decimal on-chain balance that OVERFLOWS int64/uint64 money
	// types. $100,003.01 in 18-dec wei = 100003010000000000000000 (24 digits).
	wei := MustParse("100003.01").Rescale(18)
	if got := wei.Coef().String(); got != "100003010000000000000000" {
		t.Fatalf("18-dec coef = %s", got)
	}
	if got := wei.String(); got != "100003.01" {
		t.Fatalf("round-trip = %q", got)
	}
	// arithmetic stays exact at that magnitude
	sum := wei.Add(MustParse("0.000000000000000001")) // + 1 wei
	if sum.String() != "100003.010000000000000001" {
		t.Fatalf("wei add = %q", sum.String())
	}
}
