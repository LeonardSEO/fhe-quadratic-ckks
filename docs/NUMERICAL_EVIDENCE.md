# Numerical evidence and reproducibility boundary

The archived measurements were collected on 2026-10-19 using Apple M4, Darwin
arm64, Go 1.25.3 and Lattigo 6.2.0. Numerical tolerances are absolute real-score
error at most 0.001. The script checks described below were newly executed during
release preparation; they are not new performance measurements.

## Frozen evidence inventory

- `artifacts/models.json`: seven predetermined public synthetic square networks,
  expanded exact dyadic quadratic and affine coefficients, frozen normalization,
  LDL/spectral coefficients and residual certificates. Two model seeds are held
  out from mechanism/parameter development.
- `artifacts/final_v2`: 336 encrypted model/method settings across packed,
  spectral/LDL affine factors, original-network, feature and conjugate-invariant
  kernels. The matched application batch is 128. One warmup and nine timed fresh
  encryptions per setting. These sweeps ran serially and are exploratory timing
  evidence, not the strongest counterbalanced comparisons.
- `artifacts/confirmation`: 45 counterbalanced fresh-process experiments,
  five rounds per development model and three plans; 31 timed encryptions after
  one separately charged warmup. HTTP client/server roles share one address
  space. OS process RSS includes both roles.
- `artifacts/separated_confirmation_final`: 45 counterbalanced experiments with
  fresh exec-child HTTP server, five rounds per development model and three
  plans; 31 timed encryptions per setting. Child exit rusage measures isolated
  server peak RSS; no isolated client or simultaneous sum-of-peaks claim.
- `artifacts/heldout_inputs`: 560 encrypted validation batches, 40 reserved seeds
  for seven models and two plans. Batch size fills the standard-ring slots
  (`4096/d`), so these quality runs are not timing comparators to B=128.

The checked inventory has 685 `run.json` files, 986 method records and 2,263
persisted input/output ciphertexts. It includes 249,088 saved first-block decoded
outputs. Other replicated lanes/repeats are represented by client-reported maxima,
not individually saved decoded outputs. No secret key is archived, so saved
ciphertexts cannot independently be decrypted: fresh encrypted reruns are the
operational reproduction path. Public fixtures and decoded synthetic scores are
research evidence, not private application inputs.

## New release checks

```
python3 scripts/integrity.py --strict
python3 scripts/check_correspondence.py --check
python3 scripts/verify_evidence.py --strict-ciphertexts
python3 scripts/generate_paper_data.py --check
```

`integrity.py` verifies byte counts and SHA-256 for the selected frozen artifacts.
Its manifest omits new `artifacts/reruns/` output and never rewrites old evidence.
Without `--strict`, absent supplementary `.ct`/`.bin` binaries are counted explicitly;
all committed records and present binaries remain checked. An ordinary Git clone
can regenerate tables without the supplement. Strict verification requires the
published supplement extracted at the checkout root.

`check_correspondence.py` independently folds every original network with exact
rational arithmetic, confirms the stored coefficients and zero storage residual,
and recomputes all fourteen LDL/spectral rational coefficient residual certificates.
It exports 41 matrix checks: dense and sparse examples of dimensions 1 through 17,
and the seven frozen matrices. Exact rational equality connects the Python
implementation's exported matrices to the mathematical cyclic rule; each frozen
power-of-two matrix is also checked against the Go branch's even-dimensional rule.
The arbitrary-dimension theorem is established by Lean, not these finite tests.
The Go backend supports the measured power-of-two layouts; its implementation is
not formally verified.

`verify_evidence.py` reads every archived run, verifies medians from raw stage
samples, sample counts and finite quality bounds, input/output ciphertext hashes,
fixture hashes, decoded lengths, and saved score errors against the exact-fold
quadratic reference. It recomputes every saved fixture reference. The original
network equivalence follows from the independent exact fold. It checks distinct
client/server PIDs and key-handler server PID for the isolated arm. The script
matches every exact Q/P prime vector, N, scale and thread count against the frozen
Standard/CI configuration, uses deterministic 64-bit Miller--Rabin primality tests,
checks required 2N/4N NTT congruences and recomputes log2(QP). The 214-bit published
parameter-table screen is an assessment under its stated assumptions, not a new
security proof or lattice-estimator run. Lattigo's pinned default distributions
are documented in the manuscript and upstream versioned sources; historical
records did not serialize their distribution objects separately.

`generate_paper_data.py` independently aggregates raw records, rather than adopting
the old summary prose. It produces `generated/paper_data.json`, `results_tables.tex`,
`combined_table.tex`, `sweep_table.tex`, and `tradeoffs_figure.tex`. Its arithmetic
uses `math.fsum` to give byte-identical output on tested Python 3.11 and 3.14.
The vector tradeoff figure is rendered by TikZ/pgfplots during paper compilation.

## Estimands and limitations

Each displayed confirmation time is the median of five per-process medians.
Memory is the median of five process peak RSS values. A percentage from these
marginal medians is `1 - median(candidate)/median(baseline)`. A paired speedup is
first computed within each counterbalanced round, then aggregated. These are
separate estimands and can differ. All nine plans remain visible, including d8
BSGS being slower than its baseline and d32 direct increasing keys and server RSS.

Setup includes parameter construction, client key generation, public-key
serialization, public model preparation, evaluation-key generation/serialization,
HTTP setup and measured key installation roundtrip, warmup, and (isolated arm)
child shutdown. Startup subcomponents and deserialization inside the key roundtrip
are not charged twice. Both parent and child public preparation are charged.
Exact public network folding is an additional once-per-model cost. Recurring
chain time includes client encoding/encryption, serialization, HTTP transport,
server encrypted computation, return transport, and final client decryption.
WAN formulas are explicit models; only loopback HTTP was measured.

The broad maximum stored encrypted error is 1.0610017454837362e-5. Held-out input
maximum is 4.243945753046319e-06 (read exact value from the generated data); all pass
0.001. These are empirical client-side observations, not uniform worst-case CKKS
noise guarantees. Vectors are reused by same-dimensional models and lanes share
keys, so no pooled IID confidence interval is asserted. Exact algebra, empirical
CKKS accuracy, reported cryptographic assumptions and hardware timing remain
separate claims.

Historical audit notes in `notes/` and documentation in `docs/research_snapshot/`
refer to the snapshots they inspected. Their old commands are not release commands.
The release corrects the former contradictory RSS limitation: isolated child
server RSS is measured in the separate-process arm; isolated client RSS and WAN
performance are not measured.
