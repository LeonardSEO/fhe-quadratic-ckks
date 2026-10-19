# Archived research overview

This is historical documentation. Use the release root README.md and REPRODUCING.md for current commands. Some overview links and the compact research bundle were superseded by this release.

# Encrypted scalar quadratic research artifact

Working CKKS execution, proofs, fixed models, raw encrypted benchmarks, real HTTP
transport and independent checks. Existing files in the parent research folder
are preserved. The main method is **terminal scalar cyclic orientation**: choose
a scalar-equivalent matrix before existing hoisted/BSGS lowering. It reduces
diagonal plaintext work, rather than claiming64/5 ciphertext speedup.

- [Method, correctness, error and security](METHOD.md)
- [Measured results and limits](RESULTS.md)
- [One-time costs, amortization and cold comparisons](LIFECYCLE.md)
- [Compact source/reproduction bundle](source_bundle.tar.gz) and [checksum](source_bundle.sha256)
- [Frozen protocol](PROTOCOL.md), [final comparison extension](artifacts/final_v2_protocol.json)
- [Approach registry](ROUTES.md)
- [Independent audit](notes/independent_audit.md)
- [Raw integrity checks](artifacts/independent_integrity_v2.json)
- [Separate-process isolation audit](artifacts/independent_isolation_audit.json)
- [Figure](artifacts/figures/separated_time_memory.png)

Historical novelty and a uniform-box worst-case CKKS noise theorem are not
established. The result is a reproducible measured improvement for a restricted
exact neural computation class, with the same security/quality target as the
tested baseline portfolio. It is not a general encrypted AI acceleration claim.

## Reproduce a private two-process inference

Requirements: Go1.25.3 (module language version), Python3.11+ for analysis scripts;
macOS arm64 reproduces the reported timing/RSS environment. Go code itself is
portable; `/usr/bin/time -l` RSS parsing is Darwin-specific. Models and all
coefficients are already frozen; do not regenerate them to reproduce the saved
measurements.

```sh
cd <checkout>/bench
go mod download
go build -o fhebench .
FHE_PACKED_SERVER_PROCESS=1 FHE_PACKED_METHOD=oriented_direct ./fhebench \
  -model synthetic_d16h128seed20261021 -batch 128 -repeats 9 \
  -out ../artifacts/reproduction/separated_d16
```

The client owns the secret key. A fresh exec-child server receives only public
parameters/model configuration through stdin, evaluation keys through HTTP, and
encrypted inputs. Child argv/env contain no secret, and the child branches before
any client/key initialization. Only the client decrypts final outputs. Distinct
PIDs, public config checksum, HTTP wire bytes, server RSS and process lifecycle
times are recorded. No secret key is saved. The public synthetic fixtures and
final decoded values are saved solely as checkable research evidence.

Default packed mode uses binary codecs in one process; set
`FHE_PACKED_TRANSPORT=http` for same-process HTTP simulation, or
`FHE_PACKED_SERVER_PROCESS=1` for actual process isolation. The latter is the
protocol-isolation evidence. Earlier combined-process RSS is explicitly labeled;
separate server RSS and the OS process-tree resource report must not be mistaken
for isolated client memory or summed simultaneous total peak memory.

## Reproduce the full benchmark portfolio

```sh
cd <checkout>/bench
go test ./...
./fhebench -mode packed -batch 128 -repeats 9 -out ../artifacts/reproduction/packed
./fhebench -mode factors -batch 128 -repeats 9 -out ../artifacts/reproduction/factors
./fhebench -mode network -batch 128 -repeats 9 -out ../artifacts/reproduction/network
./fhebench -mode feature -batch 128 -repeats 9 -out ../artifacts/reproduction/feature
./fhebench -mode packed -ci -batch 128 -repeats 9 -out ../artifacts/reproduction/ci
```

Every frozen model is evaluated. All baseline and candidate settings remain in
the aggregate, including losses. `-model NAME` isolates one model. Environment
filters `FHE_PACKED_METHOD`, `FHE_FACTOR_PACKED_METHOD`, `FHE_NETWORK_METHOD` and
`FHE_FEATURE_METHOD` isolate one kernel for process RSS and counterbalanced runs.
The primary compared input batch is128 for every layout, not unequal ciphertext
capacity. `-mode feature -batch 4096` is a separate high-batch experiment, not a
comparable128-batch runtime.

```sh
cd <checkout>
python3.11 tools/confirm.py --rounds 5 --repeats 31 --out reproduction/confirmation
python3.11 tools/confirm.py --rounds 5 --repeats 31 --separated --out reproduction/separated
python3.11 tools/validate_inputs.py --seeds 40
python3.11 tools/profile_frontend.py
python3.11 tools/summarize.py
python3.11 tools/select_plan.py --dimension 32 --inferences 1 \
  --max-eval-key-mb 20 --max-rss-mb 160
```

Confirmation defaults use the fixed development sweep to choose the comparator;
held-out inputs are not used to tune plans. Validation/summarization commands
write their named analysis directories, so retain the shipped manifest/raw
records before intentionally regenerating those analyses. The ordinary benchmark
commands above write `reproduction/` and preserve the saved comparisons.

Use the0-dependency exact public API in `scalar_compiler.py` to fold another
integer/dyadic public square network and construct its matrix representative;
create a model JSON with `name,Q,linear,constant` for packed mode. The measured
profile planner applies only to the fixed M4/B128 regime. New shapes/hardware
need fresh kernel profiles under the same security/error constraints.

Optional analysis/plot dependencies are pinned in requirements-research.txt; install them in a local virtual environment if needed. Optional model regeneration uses NumPy2.3.5 and `generate_models.py`; it updates
timestamp/hash and is not the frozen reproduction path. The optional figure uses
Matplotlib3.10.7 (`.plotenv/bin/python tools/plot_results.py` in this workspace).
Lattigo and all Go dependencies are pinned by go.mod/go.sum. An earlier network
TLS recovery script `tools/bootstrap_go.py` and exact archive hashes are included;
it uses checksum-pinned proxy archives rather than changing versions. Normal
`go mod download` retains default checksum-database verification.

## Evidence map

- `artifacts/models.json`: seven fixed public models, original W/v/bias,
  expanded Q/linear/constant, exact LDL and spectral factors, coefficient bounds.
- `artifacts/model_validation.json`: exact expansion/certificates and7,768
  independent plaintext checks; explicitly not encrypted evidence itself.
- `artifacts/final_v2/`:336 encrypted model/method records; raw per-stage samples,
  evaluation-key/input/output byte sizes, ciphertext files/hashes and decoded scores.
- `artifacts/confirmation/`:45 counterbalanced fresh-process combined-role HTTP
  runs, five rounds,31 timed iterations each, OS RSS logs.
- `artifacts/separated_confirmation_final/`: actual exec-child server confirmation;
  server sees no client secret state. Process startup/config/public preparation,
  key transport, final inference, shutdown and server resource use are recorded.
- `artifacts/heldout_inputs/`:560 encrypted validation batches,40 reserved seeds,
  original-network crosschecks. Empirical zero failures; no pooled IID claim.
- `artifacts/frontend_profile.json`, `headroom.json`, `cost_profiles.json`: public
  folding cost, message magnitude checks and explicit setup/memory planner inputs.
- `notes/`: independent route work, primary-source verification and audit snapshots.

Raw artifacts distinguish measured local transport from modeled WAN effects,
once-per-model costs from once-per-client costs and recurring inference, algebra
proof from encrypted validation, and author-reported literature results from
locally reproduced evidence. No Orion/HEIR headline benchmark was reproduced.
