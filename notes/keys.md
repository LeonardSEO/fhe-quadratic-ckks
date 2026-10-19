# Rotation-key memory and scheduling route audit

Status: independently investigated on 2026-10-19. This note contains verified reference identities, an exact candidate transformation, and experiment obligations. It contains no measured speedup claim and no encrypted implementation.

## Verified prior references

### KeyMemRT

[Ünay, Franke, Woodruff, *KeyMemRT Compiler and Runtime: Unlocking Memory-Scalable FHE*, arXiv:2601.18445v1, 26 January 2026](https://arxiv.org/html/2601.18445v1) is a real primary publication. It represents individual rotation keys in MLIR, analyses their lifetimes, and generates key loading, clearing, and prefetch hints. Its pipeline already includes BSGS, reuse across kernels, OpenFHE rotation hoisting, and bootstrap-key management. Reported improvements are author measurements, not reproduced here. The system is therefore direct prior art against claiming those techniques alone as new contributions.

Verified author implementation entrypoints:

- [Compiler](https://github.com/eymay/KeyMemRT-Compiler): HEIR/MLIR passes and `keymemrt-opt` / `keymemrt-translate` build commands.
- [Runtime and benchmark flows](https://github.com/eymay/KeyMemRT): `include/KeyMemRT.hpp`, Orion translator dependency, and model benchmark instructions. The README currently uses a custom OpenFHE fork and Ubuntu tooling; building this stack on the present macOS host may cost more than a direct CKKS kernel experiment.
- [Runtime source](https://github.com/eymay/KeyMemRT/blob/main/include/KeyMemRT.hpp): current code exposes ignore, imperative, prefetch, and speculative modes; deserializes individual automorphism keys into OpenFHE's evaluation-key map; and includes depth-specific key files/restoration support. A worker thread performs background deserialization. These are observations from public source, not successful local execution. The current implementation is mutable and must be pinned before reproduction.

An attempted GitHub API metadata fetch from the shell failed with a TLS connection reset, so no repository SHA has been recorded. Web access to the repositories and source succeeded. Do not present the current `main` URLs as immutable artifact pins.

### FHE-Agent

[Xu et al., *FHE-Agent: Automating CKKS Configuration for Practical Encrypted Inference via an LLM-Guided Agentic Framework*, arXiv:2511.18653v1, 23 November 2025](https://arxiv.org/html/2511.18653v1) is also real. It uses deterministic analyzers and encrypted evaluations under an LLM direction controller, targeting Orion/Lattigo. Its Table 1 compares an agent run with the best feasible result from ten one-shot LLM suggestions. Reported inference timings exclude key generation. Its stated security gate and measured improvements remain author claims until reproduced. This paper supports using calibrated deterministic tools; it does not make free-form LLM parameter search a new cryptographic execution method.

Targeted searches for this title, author, arXiv identifier, and GitHub code did not locate a verified official implementation. This is a bounded search result, not proof that code does not exist. Do not conflate the unrelated `mayank64ce/fhe-agentic-benchmarking` repository with FHE-Agent.

## Concrete candidate: terminal quadratic contraction folds rotation orbits

This is independently derived here and is mathematically exact. Originality has **not** been established. It is a candidate compiler rule, not an implemented result.

Assume the only required output of a subgraph is the scalar

`f(x) = x^T Q x`, with public real symmetric `Q`.

Let `S` denote the cyclic SIMD slot count. Embed `Q` into an `S x S` matrix by zero padding, or use an explicitly documented repeated-input layout. Let

`R_k(x)[i] = x[(i+k) mod S]`,

`d_k[i] = Q[i,(i+k) mod S]`,

`T_k(x) = sum_i d_k[i] x[i] x[(i+k) mod S]`.

Symmetry gives `d_{-k}[i] = d_k[i-k]`. Substituting `j=i-k` proves

`T_{-k}(x) = T_k(x)`.

Choose one representative from every nonzero orbit `{k,-k}` having a nonzero diagonal. Keep `k=0` once. If `S` is even, keep the self-inverse orbit `k=S/2` once. Define `c_k=2` for a two-element orbit and `c_k=1` for a self-inverse orbit. Then

`y = d_0 * x + sum_{k in representatives} c_k d_k * R_k(x)`

and

`f(x) = sum_i x[i] y[i]`.

Here `*` is slotwise multiplication. The encrypted circuit therefore has a linear-transform accumulation, **one ciphertext multiplication**, one relinearization, and a final slot reduction. It does not need one ciphertext multiplication per algebraic bilinear factor. Relative to ordinary `Qx` followed by the same contraction, it removes the duplicate half of diagonal terms, rotations, and direct rotation keys. This illustrates why the real scalar product-rank theorem is not itself an FHE multiplication lower bound.

For a full dense cyclic `S x S` matrix with even `S`, ordinary diagonal evaluation has `S` plaintext products and `S-1` matrix rotations. Orbit folding has `S/2+1` plaintext products and `S/2` matrix rotations, before output-reduction costs. Both have one ciphertext multiplication. With only the first `n` slots nonzero and `S >= 2n`, ordinary zero-padded diagonal support is at most `2n-1`, while folded support is at most `n`; avoid silently substituting these counts for the full cyclic case.

Crucial legality boundary: `y` need not equal `Qx`. This rewrite is valid only when every observable use is the final scalar contraction (or when the compiler separately proves that no other use observes `y`). It is invalid as a general replacement of matrix-vector multiplication or for exporting individual quadratic hidden units.

## Key-aware orientation and schedule

The representative of each orbit may be either sign. For a scalar contraction, reversing it changes the public diagonal coefficients by an index permutation but preserves the scalar. This creates a genuine compiler choice absent from an ordinary `Qx` kernel. Choose signs jointly with the existing reduction keys and a BSGS decomposition, rather than always picking positive rotations.

An implementable deterministic search for small/medium `n`:

1. Enumerate direct diagonal, folded diagonal, and folded BSGS plans for each legal gap `b` in a fixed range.
2. For each plan, canonicalize rotation indices modulo `S` and form the exact union of transform keys and output-reduction keys. Keep ciphertext level and key-switch decomposition parameters attached to every key identity.
3. For sparse orbit support, enumerate sign orientations if there are at most 16 orbits; otherwise use deterministic single-orbit flip descent from both all-positive and all-negative initializations, with fixed tie breaking. A legal sign flip must also shift the public coefficient diagonal.
4. Charge the complete candidate cost below and retain the Pareto set in time, peak memory, setup bytes, and error bound. Select using the preregistered deployment horizon and memory cap, not scalar multiplication count.
5. Emit the selected orientation, exact rotation sequence, exact key set, coefficient hashes, and cost inputs into a manifest. Confirm them through an operation trace during encrypted execution.

Suggested cost objective, for a fixed client key reused over `H` queries:

`C_H = T_compile + T_keygen(K) + B_keys(K)/B_setup`

`    + H * (T_encrypt + B_input/B_up + T_server(plan,K)`

`           + T_key_load(plan,K) + B_output/B_down + T_decrypt)`.

Require `max_t (M_ciphertext(t) + M_plaintext(t) + M_live_keys(t) + M_runtime(t)) <= M_cap`.

`T_server` should be calibrated by the exact library, modulus level, ring degree, and hoisted/unhoisted operation variant. A rotation counter is insufficient because same-input rotations can share decomposition while chained rotations generally cannot. Disk prefetch does not reduce generated/transmitted key bytes. Direct keys may be faster online but lose on setup horizon or memory; chaining may reduce key count but accumulate more key-switch noise. Do not claim that ordinary BSGS or this optimization objective is new; the research candidate is contraction equivalence plus its joint orientation/key/layout selection. Prior-art review must still establish whether that combination is original.

For a small quadratic kernel with only a few dozen keys, full streaming machinery is unlikely to be the first useful route: loading/deserializing keys per query can cost more than retaining them. Test the working set before building a memory runtime.

## Error argument specific to this rule

There is no model approximation in the exact-real rewrite. Let `||x||_inf <= X`, let initial decoded input error be at most `epsilon_x`, and let encoded diagonal error be at most `epsilon_d`. For a fixed folded operator `B` represented by the `c_k d_k`, define `L = max_i sum_j |B_ij|`; then `||Bx||_inf <= L X`.

Let the accumulated encrypted linear-transform error satisfy `||e_y||_inf <= E_y`, including input error, plaintext encoding, rotation/key-switch, and addition effects. Before the final reduction, the componentwise product error is bounded by

`X E_y + L X epsilon_x + epsilon_x E_y + E_mul`.

For `S` slots, the final absolute scalar error is at most `S` times that bound plus the reduction and rescaling error. In a zero-padded layout, inactive-slot numerical error still exists and must be included unless a mask/decoding proof removes it. Quantization/encoding error in `B` adds at most `S X^2 ||Delta B||_infinity` in the conservative row-sum formulation. Repeated layouts need the explicit repeat-count normalization in the correctness and error bounds.

This deterministic bound is conservative. The experiment must compare both a certified whole-domain coefficient bound and observed encrypted errors on fixed stress inputs (sign extrema, coordinate axes, extrema aligned with large rows) plus held-out random inputs. An empirical maximum is not a domain-wide proof.

## Decisive experiment and rejection conditions

- Use identical input/output packing, parameters, plaintext precision, modulus chain, and ciphertext count for ordinary diagonal and folded variants; include BSGS/hoisting in **both** when available.
- Compare against packed `Qx` contraction and packed low-rank/inertia alternatives, not an unpacked baseline with one ciphertext per scalar unit.
- Predetermine dense indefinite, dense PSD, low-rank indefinite, sparse-band, and diagonal matrices at multiple dimensions. Keep independent generated matrices/inputs for generalization.
- Measure direct versus BSGS versus chained keys. Record key bytes actually serialized, key generation time, total setup transfer, key deserialization, and peak process RSS in separate processes. Heap snapshots alone do not count total memory.
- Trace all rotations, ciphertext/plaintext products, rescalings, relinearizations, and bootstraps. Static estimates must match measured traces.
- Reject a claimed improvement if BSGS removes the apparent diagonal advantage, stronger plaintext encoding is required to meet the same error target, orientation saves keys but increases query cost beyond setup amortization, or setup/transfer dominates at the declared `H`.
- Compute the setup break-even explicitly: if a candidate saves online time `delta_t > 0` but costs extra setup `delta_s > 0`, it needs more than `delta_s/delta_t` queries before it wins; include transfer under the declared bandwidth.

## Prior-art limits and remaining obligations

Searches combining CKKS/FHE, quadratic forms, symmetric matrices, diagonal methods, and rotations did not identify an exact primary-source match for orbit folding. That negative search is weak evidence and **does not establish novelty**. It is elementary symmetric-contraction algebra and may already be implicit in domain-specific encrypted statistics or compiler work. A defensible final claim currently is only: “a concrete candidate compiler transformation has been derived and can be tested.”

The immediate remaining experiment is an ordinary-versus-folded packed diagonal CKKS benchmark with the same exact parameters and terminal slot reduction, followed by BSGS/hoisting ablations. No encrypted experiment, validated parameter set, or supported original-contribution claim has been delivered by this note.
