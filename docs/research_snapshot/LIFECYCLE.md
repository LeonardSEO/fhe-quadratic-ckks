# Lifecycle and setup accounting

All components are measured unless explicitly labeled as a network model. Costs arise at different reuse horizons.

| Cost | Scope | Measurement |
|---|---|---|
| Public neural-to-Q folding | Once per public model | Exact integer accumulation with rational storage/orientation certificate; frontend_profile.json |
| Public eigen/LDL construction | Once per factorized public model | factor_setup_profile.json; original network/cyclic-Q do not need it |
| Plaintext kernel encodings / affine completion | Once per public model and kernel plan | model_prepare_seconds and component timers in each run |
| Client secret/public keypair | Once per client key lifetime | client_keypair_generation_seconds |
| Evaluation keys | Once per client/model-required key set | Key generation and actual serialized sizes |
| Distribution | Once for evaluation keys; every encrypted request/response | Actual TCP HTTP in confirmation arms; codec/bytes in full sweep |
| Startup / redundant public encoding | Once per measured separate server | http_setup_seconds includes config serialization/pipe/exec/child encoding/readiness |
| Warmup | Once per benchmark service | Client encode/encrypt plus complete warmup request charged in S |
| Inference | Every batch | Packing, encode, fresh encryption, codecs, transport, compute, result/decrypt |
| Shutdown | Once per separate service | Explicit child shutdown timer included in lifecycle S |

Do not add nested HTTP key deserialize to key RTT, or child public-preparation/startup subcomponents to http_setup_seconds: that would double-count. Public model encodings are deliberately done in both the parent reference compiler and actual child server here; both costs are charged. No secret key is distributed. Public key stays with the encrypting client and is not required by the server.

## Public compiler preparation

The original all-Fraction reference folding profile is preserved in frontend_profile_fraction_reference.json. The shipped exact integer path performs native arbitrary-precision integer accumulation and then checks rational coefficient storage; it produces bit-identical frozen Q/linear/constant and zero residual. This avoids an artificial implementation overhead without changing computation or security. No NumPy integer overflow assumption is used in this path.

| Model | Exact fold + orientation ms | Spectral construction ms | Complete exact LDL helper ms |
|---|---:|---:|---:|
| archive8h64 | 0.761 | 0.044 | 5.314 |
| synthetic_d8h64seed20261020 | 0.768 | 0.042 | 5.233 |
| synthetic_d16h128seed20261021 | 4.296 | 0.111 | 44.443 |
| synthetic_d32h128seed20261022 | 17.084 | 0.206 | 333.350 |
| heldout_d8h96seed20261023 | 1.030 | 0.028 | 5.142 |
| heldout_d16h96seed20261024 | 3.743 | 0.058 | 37.647 |
| spd_d16h64seed20261025 | 3.114 | 0.061 | 37.733 |

The LDL helper includes residual certification and support matching. Affine-completion coefficient certificates in the Go factor baseline are separately timed. These once-only model construction phases were not smuggled into a per-inference speedup. The public JSON loading/module import timers are retained separately; they are a harness-wide load of all seven models, not seven repeated per-model loads.

## Full cold-codec portfolio

This table analytically combines individually measured phases from the matched one-process codec sweep: F + S + H*T. Baseline is re-selected for each horizon among all tested nonoriented methods, including the original network (which pays no neural-to-Q folding). Factor construction is charged where required. Same-security/quality model and B128. These are lifecycle estimates from local measured components, not wall-clock WAN measurements. Actual distribution/lifecycle is independently measured for the separate-process packed plans in RESULTS.md.

| Model | H=1 baseline total / candidate ms | H=10 baseline / candidate ms | H=1000 baseline / candidate ms |
|---|---:|---:|---:|
| archive8h64 | 30.16 / 27.53 | 90.61 / 80.06 | 6399.04 / 5857.82 |
| synthetic_d8h64seed20261020 | 30.14 / 27.80 | 89.77 / 79.13 | 6339.21 / 5726.01 |
| synthetic_d16h128seed20261021 | 46.82 / 40.32 | 120.97 / 105.76 | 7899.82 / 6850.83 |
| synthetic_d32h128seed20261022 | 80.14 / 67.67 | 175.73 / 152.91 | 10690.68 / 8534.33 |
| heldout_d8h96seed20261023 | 31.07 / 26.90 | 91.14 / 77.77 | 6159.72 / 5673.50 |
| heldout_d16h96seed20261024 | 52.64 / 44.91 | 136.48 / 125.36 | 9346.04 / 8128.81 |
| spd_d16h64seed20261025 | 47.11 / 42.17 | 131.14 / 123.10 | 8865.62 / 8350.58 |

Candidate wins 28/28 tested model/horizon combinations (H=1,10,100,1000). Raw selected plans, components and negative cases are in cold_cost_portfolio.json. This finite observed result does not imply a universal cold-start win.

For a WAN scenario add declared key payload/BW once and input+output payload/BW per inference, plus explicitly chosen latency. The main measured packed payload sizes are393,566 input bytes and131,390 output bytes; evaluation-key payload depends on schedule. Optional planner bandwidth is a conservative additive model on top of measured local time, not a claim that local HTTP equals a real network. At1Gbit/s the d32 direct extra9,439,440 evaluation-key bytes alone add about75.5ms; this can erase its cold advantage and change the chosen plan.

Default deterministic planner now uses the actual separate-process development profiles. Its RSS budget is server-only and uses the largest of five observed server peaks. It does not claim client/combined simultaneous peak memory. Only three measured plans per dimension are in that deployment profile; the full baseline tuning sweep remains accessible separately. New hardware/shapes require profiling.

## Optimistic baseline sensitivity

Give every baseline free public folding, factor construction, affine algebra and coefficient certification, while the candidate still pays all of its measured costs. This intentionally favors the baseline and checks that an expensive exact certificate is not the cause of the claimed amortized benefit. The candidate wins by H=1 on six models and by H=6 on d32; all seven win by H=10. This is an analytical sensitivity bound, not a new measured process run. Both parties may of course cache public preparation in a real deployment. Details are in artifacts/setup_sensitivity.json.
