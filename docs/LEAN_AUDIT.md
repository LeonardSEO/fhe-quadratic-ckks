# Independent mathematical and Lean audit

This audit was performed separately from the proof construction on 2026-10-19. It is an internal artifact review, not external peer review or publication acceptance.

## Verdict and checked scope

The exact real algebra and cyclic-support theorems are established in the pinned Lean environment. The scalar-sum and support declarations quantify over every natural dimension `n` with `[NeZero n]`, arbitrary real matrices indexed by `ZMod n`, and arbitrary real inputs. The entrywise `orientation_admissible` theorem is stronger still: its type does not require `[NeZero n]`, since it does not take a finite sum. The result is not a finite test for dimensions 8, 16, or 32.

The following definitions were inspected directly in `lean/FHEQuadratic/Definitions.lean`:

- `quadratic A x` is the finite double sum of `x i * A i j * x j`.
- `Admissible Q A` is the entrywise equation `A i j + A j i = 2 * Q i j`.
- `activeOffsets A` contains `r` exactly when some `A i (i+r)` is nonzero.
- `CyclicDense Q` requires a nonzero entry on each cyclic diagonal. It does not assume a support bound on `A`, and it does not require every entry of `Q` to be nonzero.
- `orientation Q` compares the canonical values of `j-i` and `i-j`, doubling the smaller-offset entries, retaining equal-offset entries, and zeroing larger-offset entries.

The inspected declarations are `FHEQuadratic.quadratic_transpose`, `quadratic_preservation`, `orientation_admissible`, `orientation_preservation`, `affine_score_preservation`, `canonical_iff_val_le_half`, `canonicalOffsets_card`, `orientation_support_upper_bound`, `activeOffsets_card_lower_bound`, `orientation_support_exact`, and `orientation_support_optimal` (all names after the first share namespace `FHEQuadratic`).

## Independent mathematical reconstruction

Transposition preserves a scalar real quadratic form by exchanging its finite summation indices. Summing the admissibility equation therefore gives `2 quadratic A x = 2 quadratic Q x`. This establishes preservation for all real inputs; the same affine term and constant can then be added.

For orientation, an offset and its negative receive complementary doubling/zeroing unless they coincide. Equality includes zero and, in even dimensions, the half-turn; these coefficients remain undoubled. Thus the two transposed entries sum to twice the original symmetric coefficient. Active offsets lie in the canonical set `r.val <= (-r).val`, which equals the integer representatives from zero through `n/2`. Its cardinality is `n/2+1` for every positive dimension, including one and two.

For the lower bound, choose a nonzero `Q i (i+r)` using cyclic density. Admissibility forces at least one of `A i (i+r)` and `A (i+r) i` to be nonzero. Hence at least one offset in each negation orbit is active. The Lean proof injects the canonical orbit representatives into active offsets by choosing `r` when active and `-r` otherwise; two canonical representatives cannot be distinct negatives. This is a genuine derivation of the support lower bound, not a circular density definition. Paper Theorem 3 uses the equivalent union/intersection count and yields the same integer bound. Its argument, and the Lean injection, both handle fixed offsets correctly.

The optimum is among admissible matrix representatives under cyclic density. It is not an optimum among all scalar arithmetic circuits, CKKS circuits, rotation plans, or schedules. Density is necessary for the stated bound: a diagonal nonzero `Q` in dimension three can use one active offset, below the dense-case bound of two. No such sparse case is claimed by the lower-bound theorem.

## Executed verification

The audit used Lean 4.24.0, compiler commit `797c613eb9b6d4ec95db23e3e00af9ac6657f24b`, on `arm64-apple-darwin23.6.0`. mathlib's actual checkout was clean at `f897ebcf72cd16f89ab4577d0c826cd14afaafc7`. Transitive dependency revisions are locked by `lean/lake-manifest.json`.

First, an ordinary independent `lake build` and `python3 check_axioms.py` both exited zero. Then only the release project's `lean/.lake/build` directory was deleted, forcing a fresh compilation of every project proof module while retaining the pinned dependency cache. The replay commands, from `lean/`, were:

```sh
lake build
python3 check_axioms.py
lake env lean evidence/InspectTypes.lean
```

`lean/evidence/independent_build.log` records the successful rebuild, including fresh builds of `Definitions`, `Preservation`, `Orientation`, and `LowerBound`. `lean/evidence/independent_axioms.log` contains actual `#print axioms` output for all eleven principal declarations. Every reported dependency set is exactly `[propext, Classical.choice, Quot.sound]`. There is no `sorryAx` or extra mathematical axiom. `lean/evidence/independent_types.log` exposes the elaborated theorem types with implicit dimension arguments made explicit.

`lean/evidence/independent_audit.json` records SHA-256 hashes of all proof modules, the root import, toolchain, manifest, Lake configuration, and axiom-check script. The inputs were hashed before and after replay and were unchanged. The source inspection found no admitted goals, custom axiom declarations, native-evaluation proof escape, `unsafe` declaration, external implementation, trust-level override, or kernel-bypass mechanism in the project proof source. Dependency-closure inspection, rather than this textual inspection alone, supports the axiom verdict.

The separate command `python3 scripts/check_correspondence.py --check` exited zero: 41 exported matrices and all seven exact network folds agreed under rational arithmetic. This finite implementation check is distinct from the general Lean theorem. It does not make future Python executions or the Go backend formally verified.

## Verification boundary

The trust boundary is the ordinary Lean kernel and compiler/distribution plus the pinned imported library artifacts, using the standard logical axioms named above. The audit rebuilt project proofs against cached pinned dependencies; it did not rebuild all of mathlib from source or run an independent proof kernel. Imported dependencies are covered by principal-theorem transitive axiom inspection, not by a claim that every dependency source file was manually audited.

Formal verification covers these real algebraic definitions and theorems. It does not verify CKKS security, cryptographic parameters, noise bounds, floating-point arithmetic, the complete Python/Go compiler and evaluator, encryption/decryption implementation, operating-system behavior, or measured performance. The paper's general algebraic statements match the checked targets in this scope; empirical/security claims need their separate evidence.
