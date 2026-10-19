# Reproducing the release

Work from a clean checkout of the tagged release. Paths below are relative to
its root. Do not write new measurements into frozen arms. The historical JSON
records preserve their original commands as provenance; those absolute paths
are not reproduction commands.

## Dependencies

- Python 3.11 or later. For model regeneration/plots: create a local virtual
  environment and `python3 -m pip install -r requirements-research.txt`.
- Go 1.25.3; `bench/go.mod` and `bench/go.sum` pin Lattigo 6.2.0 and dependencies.
- Elan with `lean/lean-toolchain` (Lean 4.24.0). `lean/lake-manifest.json` pins
  transitive dependencies and mathlib's exact Git commit. `lake exe cache get`
  downloads matching compiled mathlib artifacts; it is a setup cost.
- Tectonic 0.17.0 or a full TeX Live installation with BibTeX and the packages
  listed in `paper.tex`. Tectonic fetches its TeX bundle on first use.

Never commit virtual environments, compiler caches or generated secret keys.

## Archived evidence and exact checks

```sh
python3 scripts/check_correspondence.py --check
python3 scripts/verify_evidence.py
python3 scripts/generate_paper_data.py --check
```

For full ciphertext validation, extract the published `ciphertext-evidence`
supplement into the checkout root, preserving its `artifacts/` tree, then run:

```sh
python3 scripts/verify_evidence.py --strict-ciphertexts
```

Use the published file hashes before extraction. The JSON-only check explicitly
reports omitted binaries; it does not claim to have verified absent files.
Exact rational checks bridge Python's construction and the formal formula for
41 exported matrices and seven network folds. This finite bridge is separate
from the general Lean theorem and does not verify the whole Go backend.

## Lean

```sh
cd lean
lake exe cache get
lake build
lake env lean FHEQuadratic/Axioms.lean
```

See `docs/FORMAL_VERIFICATION.md` for checked declarations, actual logs and the
axiom policy. Build success and absence of `sorryAx` are required before using
the phrase formally verified. No native decision shortcut is used for the
central general proofs.

## Encrypted execution

```sh
cd bench
go test ./...
go vet ./...
go build -o fhebench .
cd ..
sh scripts/run_smoke.sh
```

The smoke uses a new output directory, fresh keys, serialized public evaluation
keys, HTTP loopback, a separate server executable process, encrypted input and
client-only final decryption. It is a functional reproduction. Its timing is a
new measurement, not a substitute for the frozen M4 study.

On macOS, the full fixed-protocol confirmation can be rerun with:

```sh
python3 tools/confirm.py --separated --rounds 5 --repeats 31 --out reruns/confirmation
python3 tools/validate_inputs.py --seeds 40 --out reruns/heldout_inputs
```

The original five-round order, model families, batch 128, one thread, quality
gate and parameter set are retained by the scripts. `confirm.py` uses macOS
`/usr/bin/time -l`; other systems need a separately documented resource wrapper.
Setup and recurring costs must remain separate. WAN values in historical
analysis are models, not measured network runs. See `docs/NUMERICAL_EVIDENCE.md`.

## Paper

Regenerate committed tables/figure, then compile:

```sh
python3 scripts/generate_paper_data.py
tectonic --keep-logs --keep-intermediates paper.tex
```

A TeX Live alternative is `latexmk -pdf -interaction=nonstopmode -halt-on-error
paper.tex`. Check the log for missing citations/references, then inspect rendered
pages. The frozen PDF records the compiler actually used in validation evidence.
