// Package decimal is an exact, arbitrary-precision fixed-point decimal number.
//
// A value is coefficient × 10^-scale, where the coefficient is a big.Int and the scale
// is a non-negative number of fractional digits. Because the coefficient is a big.Int
// there is NO precision ceiling — unlike int64/uint64 money types (which cap around
// 19 digits, ~$9.20 at 18 decimals), a Decimal holds a fiat cent and an 18-decimal
// on-chain wei balance with equal exactness. No float64 ever touches a value, so
// arithmetic is exact and reproducible; 0.1 + 0.2 is exactly 0.3.
//
// A Decimal is IMMUTABLE: every operation returns a new value and the wrapped big.Int is
// never mutated after construction, so a Decimal is a safe value object to copy, compare
// and share. The zero value is a valid 0 (scale 0).
//
// This is the numeric core beneath github.com/hanzoai/money — the ONE exact-number
// definition for the Hanzo stack. It is a leaf: stdlib only, no money/currency concern.
package decimal

import (
	"fmt"
	"math/big"
	"strings"
)

// Decimal is an exact fixed-point number: Coef × 10^-Scale.
type Decimal struct {
	coef  *big.Int // signed coefficient; nil == 0. Never mutated after construction.
	scale int32    // number of fractional digits; always >= 0.
}

var (
	bigTen = big.NewInt(10)
	bigOne = big.NewInt(1)
	bigTwo = big.NewInt(2)
)

func pow10(n int32) *big.Int {
	if n <= 0 {
		return new(big.Int).Set(bigOne)
	}
	return new(big.Int).Exp(bigTen, big.NewInt(int64(n)), nil)
}

// Zero is the additive identity (0 at scale 0).
func Zero() Decimal { return Decimal{} }

// bi returns a non-nil, non-aliased big.Int of the coefficient (safe to mutate).
func (d Decimal) bi() *big.Int {
	if d.coef == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(d.coef)
}

// New returns coef × 10^-scale. A negative scale is clamped to 0 (scale is fractional
// digits; a "negative scale" is folded into the coefficient).
func New(coef int64, scale int32) Decimal {
	return NewFromBig(big.NewInt(coef), scale)
}

// NewFromBig returns coef × 10^-scale, copying coef. Negative scale multiplies the
// coefficient up so the stored scale stays >= 0.
func NewFromBig(coef *big.Int, scale int32) Decimal {
	c := new(big.Int)
	if coef != nil {
		c.Set(coef)
	}
	if scale < 0 {
		c.Mul(c, pow10(-scale))
		scale = 0
	}
	return Decimal{coef: c, scale: scale}
}

// Parse reads a decimal string ("123", "-0.00132", "+6.60") EXACTLY — no float. The
// scale is the number of fractional digits written (so "6.60" has scale 2). An empty
// string is 0.
//
// The grammar is exactly [+|-] digits [. digits] with at least one digit, and it is
// enforced by checking the digits rather than by asking big.Int to. Delegating the
// check let malformed input through in the worst possible way: this function strips
// ONE leading sign, and big.Int.SetString then accepted a SECOND one, so "--5"
// parsed as +5 — a sign inversion on a value that is usually money, where a credit
// silently becomes a charge. A bare "-", "+" or "." fell through the other side of
// the same gap, taking the intPart=="" → "0" fallback and parsing as a confident
// zero. Both now return an error.
func Parse(s string) (Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Decimal{}, nil
	}
	body, neg := s, false
	switch body[0] {
	case '+':
		body = body[1:]
	case '-':
		neg, body = true, body[1:]
	}
	intPart, fracPart := body, ""
	if i := strings.IndexByte(body, '.'); i >= 0 {
		intPart, fracPart = body[:i], body[i+1:]
	}
	digits := intPart + fracPart
	if !allDigits(digits) {
		return Decimal{}, fmt.Errorf("decimal: invalid %q", s)
	}
	c, _ := new(big.Int).SetString(digits, 10)
	if neg {
		c.Neg(c)
	}
	return Decimal{coef: c, scale: int32(len(fracPart))}, nil
}

// allDigits reports whether s is one or more ASCII digits and nothing else — no
// sign, no separator, no exponent, no unicode digit that strconv would widen.
func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// MustParse is Parse that panics on error — for constants/tests.
func MustParse(s string) Decimal {
	d, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}

// Scale reports the number of fractional digits.
func (d Decimal) Scale() int32 { return d.scale }

// Coef returns a fresh big.Int of the coefficient (the value is Coef × 10^-Scale).
func (d Decimal) Coef() *big.Int { return d.bi() }

// rescaled returns the coefficient scaled to target (target >= d.scale, so exact — no
// rounding). Used to align two operands before add/compare.
func (d Decimal) coefAt(target int32) *big.Int {
	c := d.bi()
	if target > d.scale {
		c.Mul(c, pow10(target-d.scale))
	}
	return c
}

// Rescale returns the value at exactly `scale` fractional digits. Increasing the scale is
// exact; decreasing rounds half-away-from-zero (banker's rounding is not used — money
// convention is half-up on magnitude).
func (d Decimal) Rescale(scale int32) Decimal {
	if scale < 0 {
		scale = 0
	}
	if scale >= d.scale {
		return Decimal{coef: d.coefAt(scale), scale: scale}
	}
	// reduce: divide by 10^(d.scale-scale), round half-away-from-zero.
	div := pow10(d.scale - scale)
	c := d.bi()
	neg := c.Sign() < 0
	if neg {
		c.Neg(c)
	}
	q, r := new(big.Int).QuoRem(c, div, new(big.Int))
	if new(big.Int).Mul(r, bigTwo).Cmp(div) >= 0 {
		q.Add(q, bigOne)
	}
	if neg {
		q.Neg(q)
	}
	return Decimal{coef: q, scale: scale}
}

// Round rounds to `scale` fractional digits (alias of Rescale for reducing scale; pads
// when increasing).
func (d Decimal) Round(scale int32) Decimal { return d.Rescale(scale) }

// Add returns d + b, at the finer of the two scales (exact).
func (d Decimal) Add(b Decimal) Decimal {
	s := max32(d.scale, b.scale)
	return Decimal{coef: new(big.Int).Add(d.coefAt(s), b.coefAt(s)), scale: s}
}

// Sub returns d − b, at the finer of the two scales (exact).
func (d Decimal) Sub(b Decimal) Decimal {
	s := max32(d.scale, b.scale)
	return Decimal{coef: new(big.Int).Sub(d.coefAt(s), b.coefAt(s)), scale: s}
}

// Mul returns d × b EXACTLY; the result scale is the sum of the operand scales.
func (d Decimal) Mul(b Decimal) Decimal {
	return Decimal{coef: new(big.Int).Mul(d.bi(), b.bi()), scale: d.scale + b.scale}
}

// Quo returns d ÷ b rounded to `scale` fractional digits (half-away-from-zero). Panics on
// divide-by-zero (a programming error, like integer division).
func (d Decimal) Quo(b Decimal, scale int32) Decimal {
	if b.IsZero() {
		panic("decimal: divide by zero")
	}
	if scale < 0 {
		scale = 0
	}
	// (d.coef × 10^(scale + b.scale)) / (b.coef × 10^d.scale), rounded half-up.
	num := d.bi()
	num.Mul(num, pow10(scale+b.scale))
	den := b.bi()
	den.Mul(den, pow10(d.scale))
	neg := num.Sign() < 0 != (den.Sign() < 0)
	num.Abs(num)
	den.Abs(den)
	q, r := new(big.Int).QuoRem(num, den, new(big.Int))
	if new(big.Int).Mul(r, bigTwo).Cmp(den) >= 0 {
		q.Add(q, bigOne)
	}
	if neg {
		q.Neg(q)
	}
	return Decimal{coef: q, scale: scale}
}

// Neg returns −d.
func (d Decimal) Neg() Decimal { return Decimal{coef: new(big.Int).Neg(d.bi()), scale: d.scale} }

// Abs returns |d|.
func (d Decimal) Abs() Decimal { return Decimal{coef: new(big.Int).Abs(d.bi()), scale: d.scale} }

// Cmp reports −1, 0, +1 as d <, ==, > b (scale-aligned).
func (d Decimal) Cmp(b Decimal) int {
	s := max32(d.scale, b.scale)
	return d.coefAt(s).Cmp(b.coefAt(s))
}

// Sign reports −1, 0, +1 as d <, ==, > 0.
func (d Decimal) Sign() int {
	if d.coef == nil {
		return 0
	}
	return d.coef.Sign()
}

// IsZero reports whether d == 0.
func (d Decimal) IsZero() bool { return d.Sign() == 0 }

// Equal reports whether d and b are numerically equal (scale-independent).
func (d Decimal) Equal(b Decimal) bool { return d.Cmp(b) == 0 }

// String renders the exact decimal ("6.6", "0.00132", "-0.5", "0"). Trailing fractional
// zeros implied by the scale are kept only up to the scale, then trimmed.
func (d Decimal) String() string {
	c := d.bi()
	neg := c.Sign() < 0
	if neg {
		c.Neg(c)
	}
	if d.scale <= 0 {
		out := c.String()
		if neg {
			out = "-" + out
		}
		return out
	}
	div := pow10(d.scale)
	q, r := new(big.Int).QuoRem(c, div, new(big.Int))
	out := q.String()
	if r.Sign() != 0 {
		frac := fmt.Sprintf("%0*s", d.scale, r.String())
		frac = strings.TrimRight(frac, "0")
		if frac != "" {
			out += "." + frac
		}
	}
	if neg {
		out = "-" + out
	}
	return out
}

// MarshalJSON emits the exact decimal string — a STRING, never a JSON number.
func (d Decimal) MarshalJSON() ([]byte, error) { return []byte(`"` + d.String() + `"`), nil }

// UnmarshalJSON accepts a quoted decimal string or a bare number, parsed exactly.
func (d *Decimal) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if s == "null" || s == "" {
		*d = Decimal{}
		return nil
	}
	p, err := Parse(s)
	if err != nil {
		return err
	}
	*d = p
	return nil
}

func max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
