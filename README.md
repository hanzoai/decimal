# decimal

Exact, arbitrary-precision fixed-point decimal for Go. A value is `coefficient × 10⁻ˢᶜᵃˡᵉ`
with a **big.Int coefficient** — no precision ceiling, no float, immutable.

Unlike int64/uint64 decimal & money libraries (which cap around 19 digits — about `$9.20`
at 18 decimals), this holds a fiat cent and an 18-decimal on-chain wei balance with equal
exactness. `0.1 + 0.2` is exactly `0.3`.

```go
d := decimal.MustParse("6.60")
cost := decimal.New(200, 0).Mul(d).Quo(decimal.New(1_000_000, 0), 18) // 200 tokens @ $6.60/1M
cost.String() // "0.00132"  (never floored to 0)
```

The numeric core beneath [`hanzoai/money`](https://github.com/hanzoai/money). Stdlib only.
BSD-3-Clause.
