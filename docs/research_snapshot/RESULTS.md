# Measured results

All times below come from actual encrypted Lattigo v6.2.0 execution on Apple M4, one Go execution thread per role (one total in the combined-role arm). No intermediate decryption. Numerical threshold: 1e-3 absolute.

## Separate-process confirmation (primary protocol validation)

The client owns the secret key in a different address space. A fresh exec-child server starts before all client initialization, receives only public configuration, evaluation keys and encrypted requests, and returns final encrypted scores over TCP HTTP. Distinct PIDs and sanitized configuration were independently checked. Five counterbalanced fresh process/key rounds,31 timed encryptions each, B=128; one Go execution thread per role. No decrypted information reaches the server.

| d | Plan | Server ms | Full HTTP chain ms | Server-only peak RSS MB | Keys MB | Lifecycle setup ms |
|---:|---|---:|---:|---:|---:|---:|
| 8 | triangular_direct | 6.188 | 8.715 | 86.98 | 12.586 | 68.99 |
| 8 | oriented_direct | 5.541 | 8.310 | 65.77 | 7.866 | 53.19 |
| 8 | oriented_bsgs3 | 6.854 | 9.427 | 59.79 | 7.866 | 52.11 |
| 16 | triangular_direct | 9.429 | 12.139 | 127.22 | 25.172 | 110.61 |
| 16 | oriented_direct | 7.413 | 10.139 | 90.36 | 14.159 | 72.27 |
| 16 | oriented_bsgs3 | 8.713 | 11.547 | 82.49 | 14.159 | 74.51 |
| 32 | symmetric_bsgs3 | 16.174 | 19.983 | 105.43 | 17.306 | 171.58 |
| 32 | oriented_direct | 11.937 | 15.597 | 135.09 | 26.745 | 159.95 |
| 32 | oriented_bsgs3 | 13.502 | 17.181 | 85.98 | 15.732 | 118.79 |

Displayed times/RSS are medians of five process values. Below, percentages use these displayed marginal medians. Paired-round ratios are also retained in cost_profiles_separated.json; they differ when thermal/frequency drift changes absolute times.

- d8, oriented_direct: full chain 4.6% lower; server RSS 24.4% lower; keys 37.5% smaller. Paired speedup median 1.086x, range 1.029–1.154x, 5/5 rounds faster.
- d16, oriented_direct: full chain 16.5% lower; server RSS 29.0% lower; keys 43.8% smaller. Paired speedup median 1.207x, range 1.180–1.228x, 5/5 rounds faster.
- d32, oriented_bsgs3: full chain 14.0% lower; server RSS 18.4% lower; keys 9.1% smaller. Paired speedup median 1.148x, range 1.006–1.175x, 5/5 rounds faster.

d32 direct is faster than BSGS3 but increases server RSS/key size. d8 BSGS3 is slower than baseline. Keep these negatives when choosing a plan. The separate-process final worst encrypted error was 7.345e-8; the independent auditor also recomputed original neural outputs exactly for every saved fixture.

Lifecycle setup includes parent public-model encoding, child startup/config pipe transfer and child public encoding, key generation/serialization, HTTP key installation, warmup, and child shutdown. Child startup subcomponents and key-deserialization time are not added twice. Public neural-to-quadratic folding is a separate once-per-public-model cost, shared with folded baselines. This implementation redundantly encodes the public model in parent and child; that extra preparation is charged rather than hidden. Server RSS is explicit child exit rusage; no sum of non-simultaneous client/server peaks is claimed. These separated runs were executed after the combined-role sweeps; do not compare their absolute time across arms.

At d32, direct versus baseline measured lifecycle amortization is H=1 loopback inferences; the constrained BSGS3 plan has lower lifecycle setup and recurring cost. This is a measured local result, not WAN latency.

## Broad matched-batch sweep

B=128. Seven models fixed before evaluation; all raw settings retained. Baseline is the fastest complete codec-chain result among symmetric/triangular Q, direct and BSGS0–3, spectral/LDL affine squares, original packed network, feature layout and real CI. One warmup, nine fresh encryptions per setting. These are controlled same-machine comparisons, not executions of entire Orion/HEIR compiler suites.

| Model | Fastest tested baseline | Baseline chain ms | Candidate chain ms | Speedup |
|---|---|---:|---:|---:|
| archive8h64 | packed/triangular_direct | 6.370 | 5.836 | 1.091x |
| synthetic_d8h64seed20261020 | packed/triangular_direct | 6.310 | 5.704 | 1.106x |
| synthetic_d16h128seed20261021 | packed/triangular_direct | 7.844 | 6.813 | 1.151x |
| synthetic_d32h128seed20261022 | packed/symmetric_bsgs3 | 10.621 | 8.460 | 1.255x |
| heldout_d8h96seed20261023 | packed/symmetric_direct | 6.130 | 5.652 | 1.085x |
| heldout_d16h96seed20261024 | packed/symmetric_bsgs3 | 9.303 | 8.084 | 1.151x |
| spd_d16h64seed20261025 | packed/symmetric_bsgs3 | 8.823 | 8.311 | 1.062x |

Different mode sweeps run serially; use the counterbalanced confirmation for the stronger timing conclusion. The two held-out model seeds were not used for selecting the mechanism or cryptographic parameters.

## Fresh-process HTTP confirmation

Actual loopback TCP HTTP: evaluation keys posted once; ciphertext request and final ciphertext response on every inference. Five counterbalanced fresh processes per plan,31 timed iterations after one separately charged warmup. Entries are medians of five process medians; RSS is median process maximum, client and server together, not server-only memory.

| d | Plan | Server ms | Full HTTP chain ms | Peak RSS MB | Evaluation keys MB | Setup ms |
|---:|---|---:|---:|---:|---:|---:|
| 8 | triangular_direct | 4.572 | 6.409 | 124.24 | 12.586 | 37.27 |
| 8 | oriented_direct | 4.009 | 5.815 | 95.01 | 7.866 | 26.56 |
| 8 | oriented_bsgs3 | 4.915 | 6.848 | 84.79 | 7.866 | 27.45 |
| 16 | triangular_direct | 6.363 | 8.277 | 190.50 | 25.172 | 68.78 |
| 16 | oriented_direct | 5.020 | 6.842 | 131.02 | 14.159 | 42.55 |
| 16 | oriented_bsgs3 | 6.033 | 7.873 | 118.77 | 14.159 | 42.66 |
| 32 | symmetric_bsgs3 | 8.951 | 10.929 | 151.26 | 17.306 | 64.98 |
| 32 | oriented_direct | 6.834 | 8.776 | 204.64 | 26.745 | 79.29 |
| 32 | oriented_bsgs3 | 7.641 | 9.566 | 135.92 | 15.732 | 51.89 |

Setup includes parameter/key generation, public model encoding, key serialization, actual HTTP key distribution/installation, HTTP endpoint setup and explicit warmup. It does not double-count key decoding inside key RTT. Public exact network folding is a separate once-per-public-model cost: d8=0.77ms, d16=4.30ms, d32=17.08ms. Module import and frozen-file load are in frontend_profile.json. Publishing a precompiled model amortizes public folding across clients.

## Total cost and tradeoffs

For a reused client key and public model, measured loopback total is S + H*T plus once-per-public-model front-end folding. For a different network, separately add K/B + H*(input+output)/B and latency as an explicit model; those are estimates, not measured WAN results. Input/output bytes are the same across matched packed plans. The planner uses actual measured profiles and never asks an LLM to invent settings.

- d8: direct candidate reduces full HTTP-chain time by 9.3%; RSS change -23.5%; loopback setup versus baseline amortizes by H=1.
- d16: direct candidate reduces full HTTP-chain time by 17.3%; RSS change -31.2%; loopback setup versus baseline amortizes by H=1.
- d32: direct candidate reduces full HTTP-chain time by 19.7%; RSS change +35.3%; loopback setup versus baseline amortizes by H=7.
- d32 BSGS3 candidate reduces time by 12.5%, RSS by 10.1% and evaluation-key bytes by 9.1%; setup is also lower, so it wins from the first measured inference.

The d8 BSGS3 candidate is slower than its baseline in all five HTTP confirmation processes; it is not the default fast plan. d32 direct increases keys and peak RSS; never describe it as a memory win. CI initial-sweep small regressions remain in earlier artifacts. No universal runtime improvement is claimed.

## Quality and independent checks

- Final v2:336 model/method records, all pass the fixed absolute threshold; worst error over the complete portfolio1.062e-5 (CI archived model).
- Reserved-input validation: 560 encrypted batches,40 fresh key/input seeds per model/method,49,000 uniform dyadic input-model pairs plus corners and signed basis probes. d8 includes every one of256 box corners. Maximum observed absolute error against original neural computation 4.24394575e-06.
- Same-dimensional models reuse vectors, and outputs within a batch share keys. This is empirical zero failures; no pooled IID confidence claim or uniform-box CKKS worst-case theorem.
- Exact rational network expansion, inertia and affine-square coefficient residual certificates are separate from encrypted numerical accuracy.
- Independent audit recomputed hashes, timings,45 resource logs, all validation batches and14 affine certificates with zero mismatches. See notes/independent_audit.md and artifacts/independent_integrity_v2.json.

See [lifecycle accounting](LIFECYCLE.md) for the public preparation timings and the cold portfolio, including the original network baseline.

## Limits

This is a small exact scalar quadratic neural class, not a general Transformer or image-classification speedup. Historical publication priority of the scalar-consumer rewrite remains unestablished. Security is an explicit Category128 table assessment under RLWE assumptions, not a new security proof or executed lattice-estimator run. CI uses its appropriate conjugate-invariant RLWE assumption. Local HTTP and isolated server child RSS are measured in the separate-process arm; WAN performance and isolated client RSS are not. No test-set accuracy is substituted for an application-quality result on a trained classifier.
