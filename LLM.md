# decimal

Exact, arbitrary-precision fixed-point decimal for Go — a value is
`coefficient × 10⁻ˢᶜᵃˡᵉ` over a `big.Int` coefficient: immutable, no float, no
precision ceiling. One type holds a fiat cent and an 18-decimal on-chain wei
balance with equal exactness. Stdlib only — `go.mod` requires nothing. API in
`README.md`.

The numeric core beneath `hanzoai/money`. Written from scratch, not a fork.

## Licensing

`MIT OR Apache-2.0`, at your option — per HIP-0137 (`hanzoai/hips`, `HIPs/hip-0137-one-license.md`). Relicensed from BSD-3-Clause,
which HIP-0137 puts out of scope for `hanzoai`. A stdlib-only dependency graph
means there is no upstream licence to inherit; the per-file
`// Copyright (c) 2026, Hanzo AI, Inc.` headers moved with it.
