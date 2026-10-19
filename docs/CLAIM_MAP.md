# Claim-to-evidence map

This map distinguishes exact mathematics, fixture correspondence, historical encrypted measurements, and checks of the release. Historical reports apply to the snapshots they inspected. They are not external peer review.

| Claim in paper | Exact evidence / release location | Boundary |
|---|---|---|
| Scalar preservation when A+Aᵀ=2Q, all finite real dimensions | paper.tex Theorem 1; checked `FHEQuadratic.quadratic_preservation`, `affine_score_preservation` in lean/ | Actual integrated Lake build and eleven principal axiom checks passed; commands and trust boundary in docs/FORMAL_VERIFICATION.md |
| Cyclic orientation, odd/even and self-inverse cases | scalar_compiler.py `cyclic_representative`; lean/FHEQuadratic/Orientation.lean; paper Theorem 2 | Python is general positive dimension; Go measured backend is power-of-two dimensions |
| General support upper bound floor(n/2)+1 | paper Theorem 2; `canonicalOffsets_card`, `orientation_support_upper_bound` | Matrix cyclic support, not encrypted runtime |
| Matching support lower bound | paper Theorem 3; lean/FHEQuadratic/LowerBound.lean `activeOffsets_card_lower_bound`, `orientation_support_optimal` | Every offset of Q must have at least one nonzero entry; admissible matrix representatives only |
| Exact coefficient correspondence | scripts/check_correspondence.py; lean/evidence/; artifacts/models.json; docs/EVIDENCE_MANIFEST.json | 41 exported matrices and 7 frozen network folds; not a universal correctness proof of Go/Python execution |
| Primary 9-row separate-process table | artifacts/separated_confirmation_final/records.json, each run's run.json/resource log/fixtures; artifacts/cost_profiles_separated.json; scripts/generate_paper_data.py -> generated/results_tables.tex and paper_data.json | Five fresh process/key rounds, 31 timed iterations, median of process medians; server RSS from child's exit_peak_rss_bytes |
| 4.6%,16.5%,14.0% chain reductions | same primary records; generated/paper_data.json | Ratios of marginal medians; not paired-round medians |
| Paired speedup medians/ranges/5 wins | artifacts/cost_profiles_separated.json; generated/paper_data.json | Round-paired baseline/candidate ratios, not ratio of marginal medians |
| Negative d8 BSGS / faster d32 direct costs | all nine primary plans retained; generated/tradeoffs_figure.tex | Candidate support optimum is not global runtime or memory optimum |
| Seven-model sweep / 336 records | artifacts/final_v2/; generated/sweep_table.tex | Combined-role serial mode sweeps; no cross-architecture absolute-time comparison |
| Combined-role setup amortization H=7 for d32 direct | artifacts/cost_profiles.json; LIFECYCLE.md | Combined client/server address space, different experiment from primary separated runs |
| Separate-process d32 loopback H=1 | artifacts/cost_profiles_separated.json and lifecycle fields of primary records | Measured loopback, not WAN prediction |
| Full lifecycle accounting | LIFECYCLE.md; bench/packed.go, bench/server_process.go; run setup and transport fields | Folding once per public model, other setup per client key, chain per inference; evidence I/O/plaintext validation excluded |
| Actual process separation/no intermediate decryption | bench/server_process.go; public configuration and HTTP request types; distinct PIDs and run metadata; artifacts/independent_isolation_audit.json and artifacts/independent_separated_confirmation_audit.json | Published benchmark fixtures deliberately public; server gets no secret key or decoded feedback |
| 560 reserved encrypted batches / empirical maxima | artifacts/heldout_inputs/summary.json and artifacts/heldout_inputs/; scripts/verify_evidence.py | Inputs/keys correlated within and across batches; no IID or domain-wide CKKS theorem |
| Parameter/table assessment | notes/security.md, PROTOCOL.md, prime values in final run records, scripts/verify_evidence.py | N8192, Q[58,40,40], P60, scale2^40, uniform ternary, sigma3.2, logQP198<214 published guidance; no new security proof/estimator run |
| Numerical target 1e-3 vs original network | PROTOCOL.md; model and run fixtures; input validator | Same threshold and unchanged model for all methods; empirical CKKS accuracy |
| Coefficient residual box bound | scalar_compiler.py; scripts/check_correspondence.py; affine certificates in artifacts/ | Exact rational coefficient mathematics, independent of CKKS noise |
| Prior-art distinctions and priority caveat | docs/RELATED_WORK.md; references.bib | Primary sources checked; no other compiler suite reproduced, no historical-priority proof |
| Release evidence integrity counts | scripts/verify_evidence.py --strict-ciphertexts output in release validation log | 685 runs/986 methods/2263 ciphertexts/249088 outputs are overlapping accounting totals, not independent trials |
| Lean versions/build/axioms | lean/lean-toolchain, lean/lake-manifest.json, docs/FORMAL_VERIFICATION.md | Actual successful Lake result and standard-only axiom dependency sets are reported; no source-only verification claim |
| Current reproducibility, PDF and hosted CI | REPRODUCING.md, RELEASE_CHECKLIST.md, release validation records and GitHub checks | New checks separate from historical timing campaign |

The paper's artifact DOI macro must contain only a real reserved DOI; the final publication claim requires the published public record and download/hash verification. A local manuscript or a reserved DOI alone is not a publication.
