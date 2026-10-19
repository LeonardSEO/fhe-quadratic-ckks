# Terminal-Scalar-Aware Compilation of Quadratic Scores for CKKS

Leonard van Hemert · research artifact, version 1.0.0

This release studies a compiler pass for public scalar quadratic scores
`xᵀQx + ℓᵀx + c`. It chooses an admissible representative `A + Aᵀ = 2Q`
whose cyclic-diagonal support is smaller, then uses existing CKKS linear-transform
kernels. The intermediate vector `Qx` must have no other consumer.

The general construction has at most `floor(n/2)+1` active cyclic diagonals. This
is a matching restricted optimum when every cyclic diagonal of `Q` contains a
nonzero entry. It is not an optimum over all encrypted circuits or a claim of
historical priority. The empirical backend and experiments cover power-of-two
packing with dimensions 8, 16 and 32.

The frozen separate-process measurements show selected plans lowering full
loopback inference medians by 4.6%, 16.5% and 14.0%, respectively, versus the
selected optimized baseline, with lower server peak RSS. A faster d32 direct
plan uses more memory and larger evaluation keys; d8 BSGS is slower. Ratios of
marginal medians differ from paired ratios. These are synthetic quadratic neural
scores, not a trained-task accuracy study. Read [paper.tex](paper.tex) and the
raw-derived generated tables for scope and limitations.

## Contents

- `paper.tex`, `references.bib`, `generated/`: manuscript and raw-derived tables/figure.
- `lean/`: pinned Lean 4/mathlib project for general algebraic statements.
- `scalar_compiler.py`, `compiler_ldl.py`, `bench/`: compiler and Lattigo execution.
- `artifacts/`: frozen public models, JSON records, fixtures and manifests.
- `scripts/`: exact coefficient checks, evidence validation and table generation.
- `docs/`: claim map, formal verification boundary, related work and audits.
- `docs/research_snapshot/` and `notes/`: preserved research context and negative routes.

See [REPRODUCING.md](REPRODUCING.md), [docs/FORMAL_VERIFICATION.md](docs/FORMAL_VERIFICATION.md),
[docs/CLAIM_MAP.md](docs/CLAIM_MAP.md), and [RELEASE_CHECKLIST.md](RELEASE_CHECKLIST.md).
The formal-verification document records actual build status; source existence
alone is not verification.

## Quick checks

Python 3.11+, Go 1.25.3, Lean 4.24.0 and the pinned dependencies are required for
the corresponding checks. From this repository's root:

```sh
python3 scripts/check_correspondence.py --check
python3 scripts/verify_evidence.py
python3 scripts/generate_paper_data.py --check
(cd bench && go test ./... && go vet ./...)
(cd lean && lake build && lake env lean FHEQuadratic/Axioms.lean)
```

Complete public ciphertext and evaluation-key binaries are distributed separately
as the release evidence supplement, with hashes and extraction instructions.
The Git checkout checks all present binaries and reports omitted ones explicitly;
`--strict-ciphertexts` requires the supplement. They are public research fixtures,
not client secret keys. New benchmark runs belong in `artifacts/reruns/` and never
replace archived measurements. Hosted CI proves functional checks on its own
hardware; it does not reproduce Apple M4 timings.

## Security and numerical boundary

The client alone owns the secret key and decrypts only the final result. The
honest-but-curious server receives public weights, evaluation keys and encrypted
inputs; model, dimensions, batching, schedule, timing, sizes and memory are
observable. The fixed classical-128 target is assessed against published
parameter guidance, not a new security proof or a new estimator run. CKKS error
is measured empirically under a fixed absolute `1e-3` gate; no uniform worst-case
noise theorem is claimed. Formal algebra does not verify the complete Go
implementation, cryptographic security, measured timing or CKKS noise.

## Licensing and citation

Original code and Lean: MIT. Paper, documentation and data: CC BY 4.0.
See [LICENSES/README.md](LICENSES/README.md), [THIRD_PARTY.md](THIRD_PARTY.md) and
[CITATION.cff](CITATION.cff). No affiliation, funding, ORCID or external peer-review
status is asserted. The DOI and public release identifiers are added only after
actual reservation/publication; consult release metadata for their state.
