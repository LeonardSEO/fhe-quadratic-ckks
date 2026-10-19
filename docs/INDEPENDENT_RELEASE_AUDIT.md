# Independent implementation and evidence audit

Date: 2026-10-19. This is an independent Codex-agent review within the same release task, not external peer review. It covers implementation, evidence aggregation, and the associated manuscript claims inspected during release preparation. `INDEPENDENT_RELEASE_AUDIT_SNAPSHOT.json` identifies inspected source and raw-evidence bytes by SHA-256. Later modifications are outside that snapshot until rechecked.

## Findings and repairs

1. **Evidence validation originally omitted explicit fixed-prime checks.** The manuscript said primes and congruences were validated, while the initial release validator only checked dimension, scale and thread count for the separated arm. The evidence builder added exact Standard/CI Q/P vectors, deterministic 64-bit primality tests, NTT congruences, logQP accounting, the pinned Lattigo module version, decoded-vector lengths and sample counts. This review separately checked all 45 primary records and independently found the same fixed Standard prime vector, all prime and congruent to 1 modulo 16384. This is an algebraic parameter check, not a new security estimator run.
2. **Unmeasured timing-cause attribution.** An early paper draft attributed differences between paired-ratio and marginal-median estimates to thermal/frequency conditions. No direct thermal observation supports that causal claim. The manuscript builder replaced it with the aggregation distinction and a qualified statement about round-level drift.
3. **Historical RSS limitation inconsistency.** The release snapshot now states that isolated child-server RSS is measured, while isolated client RSS and WAN performance are not. This agrees with the actual child `ProcessState.SysUsage` data. The original research directory was not edited.

No unresolved implementation/evidence defect was found that invalidates the scoped scalar-quadratic result. This is not a statement of global optimization, historical priority or uniform CKKS accuracy.

## Secret-key and execution boundary

`main` calls `MaybeRunServerProcess` before loading client fixtures or constructing client cryptographic state. The exec-child configuration has public parameters, matrix/affine coefficients, constant, layout dimensions and plan settings. The server process initializes only the public model and HTTP listener. Its key endpoint unmarshals an evaluation-key set; its inference endpoint unmarshals a ciphertext and calls `packedServerEvaluate`.

The evaluator contains only linear-transform evaluation, rescaling, public affine addition, one ciphertext product/relinearization, cyclic reductions and constant addition. It has no secret-key or decryptor argument. The only packed-path decryption inspected is after the final ciphertext returns to the client. Public plaintext fixtures and client quality statistics are deliberately archived research evidence; they are not messages in the server protocol. No intermediate decryption or model-dependent client feature extraction was found.

This is honest-but-curious protocol separation, not a hardened deployment or protection from host compromise. Older in-process measurements cannot establish memory isolation and are identified separately. Evaluation-key security remains conditional on the stated scheme assumptions.

## Costs, layout, baselines and numerical quality

The primary records use batch 128, one thread, N=8192 Standard CKKS, three Q primes, one P prime, scale 2^40, and 31 fresh encryptions per process after a separately charged warmup. All 45 primary records have exactly the same Q/P vector:

```
Q = [288230376151760897, 1099511480321, 1099511922689]
P = [1152921504606830593]
log2(QP) = 198.0000001934833
```

The pinned Lattigo source sets `DefaultXs = ring.Ternary{P: 2/3.0}`; the parameter constructor does not override this with a sparse secret. The 214-bit security-table comparison is an assessment under its stated assumptions, not a newly executed lattice estimate or security proof. This audit verified the actual integer parameters and implementation defaults; it does not independently prove the table or RLWE assumptions.

Packed variants share slot layout, inputs, public score, batch, ciphertext request/response count and quality threshold. The Go backend explicitly requires power-of-two dimension at least two and a layout period dividing the slot count. Its orientation branch therefore agrees with the general mathematical rule on its accepted dimensions; this does not extend the Go backend to arbitrary positive dimension. The Python/rational bridge separately covers odd, even and singleton fixtures. The constant/affine terms remain in the encrypted score.

The recurrent timer starts before packing and stops after final client decryption/decoding. It includes ciphertext codecs and actual loopback HTTP/TCP. Timing excludes archival evidence writes and plaintext quality checking, as disclosed. Setup includes redundant parent/child public encoding and child startup, keys, distribution/installation and warmup; key deserialization is not added again to the enclosing round trip. Startup subcomponents must not be double-counted. Evaluation-key byte sizes are serialized bytes, distinct from peak RSS. Logical operation counts are derived from the chosen lowering and its BSGS index sets; they are not hardware performance-counter measurements. WAN transport is modeled separately, not measured.

The tuned portfolio contains ordinary triangular folding, symmetric direct/BSGS plans, spectral/LDL affine squares, original network and feature-layout arms. The primary comparison fixes a baseline selected from the earlier packed sweep and tests direct and BSGS orientation in five counterbalanced rounds. This is a strong tested baseline set, not proof that no better quadratic implementation exists. The dim-8 BSGS regression and dim-32 direct memory/key increase remain visible.

All inspected encrypted results use the same absolute threshold 1e-3. Exact coefficient correspondence and seven network folds are checked separately from empirical CKKS noise. Repeated encryptions, replicated lanes and same-dimensional repeated fixtures are not independent input-distribution trials. No uniform numerical-error guarantee follows from the empirical zero-failure records. Historical ciphertext hashes authenticate bytes but cannot recover unpersisted secret keys; fresh execution is required for independently decryptable functionality checks.

## Independent re-execution and raw-record arithmetic

Commands run successfully from this release checkout:

```
python3 scripts/verify_evidence.py --strict-ciphertexts
python3 scripts/check_correspondence.py --check
python3 scripts/generate_paper_data.py --check
(cd bench && go test ./...)
bash scripts/run_smoke.sh
```

The strict evidence check inspected 685 run files, 986 method records, 2263 present ciphertext hashes and 249088 saved output values. No ciphertext was missing locally. Its maximum stored encrypted error was 1.0610017454837362e-5. The rational bridge passed 41 matrix instances and all seven exact network folds. Its strengthened version additionally recomputed all 14 stored LDL/spectral residual certificates exactly and checked the Go even-dimension construction formula against the formal rule on the seven frozen models; the strengthened check was rerun successfully. Generated paper data matched the committed deterministic outputs. Go tests passed.

The independent new smoke run is in `artifacts/reruns/smoke-OwtkqfJY/validation.json`; it is a functional rerun, not a reproduction of historical Apple M4 timing. Triangular direct used client/server PIDs 38586/38587 and maximum error 3.237870636885187e-8. Oriented direct used 38588/38589 and error 2.9004523532938364e-8. Both used actual encrypted two-process execution. These new artifacts do not overwrite the frozen measurements.

Separately written raw-record aggregation (not importing the paper-data generator) reproduced:

| Dimension/plan | Chain ms | Child RSS MB | Keys MB | Paired median speedup |
|---|---:|---:|---:|---:|
| 8 triangular direct | 8.714583 | 86.982656 | 12.585902 | 1 |
| 8 oriented direct | 8.309792 | 65.765376 | 7.866182 | 1.08600449 |
| 8 oriented BSGS3 | 9.426875 | 59.785216 | 7.866182 | 0.94919493 |
| 16 triangular direct | 12.138750 | 127.221760 | 25.171822 | 1 |
| 16 oriented direct | 10.138875 | 90.357760 | 14.159142 | 1.20652063 |
| 16 oriented BSGS3 | 11.546667 | 82.493440 | 14.159142 | 1.04385456 |
| 32 symmetric BSGS3 | 19.982750 | 105.431040 | 17.305622 | 1 |
| 32 oriented direct | 15.597125 | 135.086080 | 26.745062 | 1.22536322 |
| 32 oriented BSGS3 | 17.180750 | 85.983232 | 15.732382 | 1.14770011 |

The baseline denominator for each paired ratio is the baseline in the same round. Ratio-of-marginal-medians percentages are different estimands and must stay labeled. All three selected time/memory plans won all five paired rounds; dim-8 BSGS did not.

## Boundary of this audit

This audit does not certify Lean build success, axiom dependencies, rendered PDF layout, clean-checkout CI, published GitHub state or a published Zenodo DOI. Those require their own actual checks and release records. The implementation files, scalar compiler, frozen model JSON and primary records inspected here were byte-identical to the original research source. The result supported by this audit is a restricted representation pass with observed encrypted gains and explicit tradeoffs, not a general FHE inference solution or a historical originality proof.
