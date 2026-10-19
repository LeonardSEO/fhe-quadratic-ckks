# Release validation

New checks below ran on 2026-10-19. They are distinct from the frozen benchmark measurements. Hosted CI results and the final tag/commit belong to release evidence; this document does not assert that an unrun hosted check passed.

## Portable commands

From a clean checkout, install Python 3.11 with `numpy==2.3.5`, Go 1.25.3, and the Lean toolchain in `lean/lean-toolchain`. Install mathlib's selected cache as shown in CI, then run:

```sh
python3 scripts/check_release.py --phase evidence
python3 scripts/check_release.py --phase go --smoke
python3 scripts/check_release.py --phase lean
python3 scripts/check_release.py --phase paper
```

The paper phase uses `tectonic` on PATH (or `CODEX_TECTONIC_PATH`) when present, otherwise `latexmk`; it checks final-log unresolved citations/references and, with Poppler installed, extracted PDF reference markers. Visual PDF inspection remains a separate gate. `--phase all --smoke` performs all four phases. Extract the separate ciphertext supplement at the checkout root and add `--strict-ciphertexts` to require all frozen binary evidence. No original workspace path is required.

`run_smoke.sh` builds a real executable, runs baseline `triangular_direct` and candidate `oriented_direct` for the frozen synthetic d8 model, batch 128, three repetitions, standard CKKS, and a distinct exec-child HTTP server. It checks different client/server PIDs, key-installation PID, nonzero server RSS, all-lane recorded error at most 0.001, ciphertext hashes, no bootstrap, and fixed ring/scale/thread settings. It writes a unique directory under `artifacts/reruns/`. No secret key is serialized. This functional check does not estimate a performance speedup.

## Executed local checks

- `go mod verify`: all pinned module contents verified.
- `go test -count=1 ./...`: passed, including genuine encrypted feature-layout inference and its accuracy/output-level gates.
- `go vet ./...`: passed.
- Exact correspondence: 41 matrix cases and seven exact network folds passed. Strict integrity and all 2,263 ciphertext checks also passed under pinned Python 3.11.14.
- Stored evidence: 685 run files, 986 method records, 2,263 present ciphertext hashes and 249,088 stored decoded fixture outputs passed; maximum stored error 1.0610017454837362e-5. Generated paper data matched committed outputs.
- Fresh release-directory two-process inference passed; candidate maximum error 2.6519349993758823e-8, baseline 3.228845979474926e-8. Saved `artifacts/reruns/smoke-UUC8UqoE/`.
- A clean directory copy excluding dependency caches, Git state, new reruns and binary supplements passed module verification, Go test/vet and fresh two-process inference. Candidate error 2.870968078549918e-8 and baseline 2.735210570536939e-8; saved `artifacts/reruns/clean-copy-encrypted-smoke/`. These differing errors reflect randomized encryption, not different accuracy requirements.

A Python 3.11 versus 3.14 comparison additionally exposed a generated JSON floating-point serialization difference. Stable `math.fsum` setup aggregation repaired it; the isolated clean-copy evidence gate passed on both Python 3.11.14 and 3.14.5 after the fix. That check validated 3,217 present files while explicitly permitting the separately distributed 2,277 binary supplement files to be absent. The clean-copy check exposed a missing `artifacts/reruns/` parent in the smoke script; the script now creates it. It also exposed initial transient test output included in the evidence manifest; that output is relocated to reruns and excluded from frozen integrity records. The feature smoke Go test now writes to `t.TempDir()`.

General Lean build/axiom results are recorded in [FORMAL_VERIFICATION.md](FORMAL_VERIFICATION.md), not inferred from the Python finite checks. Final manuscript compilation and rendered inspection are recorded by the release checklist.

## CI and trust

`.github/workflows/verify.yml` has independent evidence, encrypted implementation, Lean, and paper jobs on Ubuntu 24.04. CI timings are never reported as a reproduction of Apple M4 timings. Linux Go RSS conversion is explicitly KiB-to-bytes; Darwin uses bytes. Actions are pinned to commit hashes resolved against GitHub's primary API. The official elan 4.1.2 Linux asset was downloaded and SHA256 checked locally; CI verifies that SHA256 before running it. Exact pins are in `CI_PINS.json`. Lean and mathlib are pinned independently by the project's toolchain/manifest. Distribution LaTeX is used in CI; local compiler identity is separately disclosed.
