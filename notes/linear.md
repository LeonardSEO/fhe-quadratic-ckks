# Linear-transform route: packing-aware quadratic contraction

Status: independently derived cost/error analysis and primary-source check, 2026-10-19. These are candidate compilation rules, not measured encrypted speedups or a novelty certificate. No encrypted execution was performed by this subtask.

Plain NumPy sanity check against the existing quadratic_model.npz (B=3, H=64, seed77, x uniform[-1,1]) verified periodic diagonal matvec versus the stated BSGS identity (maximum difference 8.88e-16), direct contraction versus original network (5.68e-14), and the complex-square-root packed baseline (2.84e-14). This is algebra/layout validation only, not an encrypted benchmark.


## Important baseline correction

The local example has d=8 inputs, h=64 square units, and scalar output. Sixty-four scalar squares are not necessarily sixty-four CKKS multiplications. In a packed layout, all hidden outputs can be squared with ONE SIMD ciphertext square. A five-product decomposition can likewise need one SIMD product, but generally needs TWO linear transforms Ux,Vx, versus ONE for the original Wx+b. Scalar product count alone can therefore select a worse encrypted circuit.

Compare these candidates for f(x)=sum_j v_j(w_j x+b_j)^2+c:

1. Original affine -> square -> weighted reduction, with diagonal/BSGS/double-hoisted matvec.
2. Signed spectral square decomposition (at most d scalar squares), with one matvec plus signed reduction.
3. Inertia-paired products Ux * Vx (max(p,m) scalar products), with two matvecs sharing input rotations.
4. Direct contraction: Q=W^T diag(v)W, ell=2 W^T diag(v)b, c'=sum_j v_j b_j^2+c; evaluate x * (Qx+ell), then reduce, then add c'. This needs ONE matvec and ONE SIMD product and avoids irrational eigensystem constants.
5. Sparse triangular contraction and partial/groupwise contraction (below).

Do not discard 1,2,4 just because 3 is optimal in a scalar multiplication model.

## Explicit periodic layout and fair low-batch example

Let S be usable CKKS slots, B a fixed batch size, H a power of two with H*B=S, d a power of two dividing H. Slot i*B+t contains input feature x[(i mod d),t]. Thus every sample is repeated H/d times, with samples interleaved rather than placed in consecutive independent blocks. A ciphertext rotation by r*B shifts only feature coordinates and preserves sample t even at wraparound.

For arbitrary W in R^(H x d), define diagonal D_r[i*B+t]=W[i,(i+r) mod d], for r=0,...,d-1. Then sum_r D_r * Rot_(rB)(x) gives all H outputs of Wx in one ciphertext. This uses d plaintext products, not H. Biases are plaintext addition. Zero-pad hidden units when required. Only slots t=0,...,B-1 need be correct after final reduction; there is no need to mask every unused slot after every stage.

With d=8,h=H=64, fixed B=S/64:

| Candidate | Dense plaintext products | Ciphertext products | Transform rotations with ordinary BSGS | Final reduction rotations |
|---|---:|---:|---:|---:|
| Original, output weights absorbed using complex square roots | 8 | 1 square | 4 | 6 |
| Direct Q contraction | 8 | 1 product | 4 | 3 |
| Paired inertia, k=5 padded to8, with shared baby rotations | 16 | 1 product | 5 to7 depending BSGS split | 3 |

The paired transform count is 5 for b=4 (3 shared baby rotations and 1 giant rotation for each output), 7 for b=2 (1 shared baby and 3 giant rotations per output). The original and direct-Q transform each have four rotations for either split. These operation counts exclude a possible separate ell path in the paired circuit; the direct circuit includes ell as a plaintext addition before multiplication.

The complex-square-root baseline uses sqrt(v_j) W_j and sqrt(v_j)b_j. For v_j<0 take sqrt(v_j)=i sqrt(-v_j); then the square is still the required real signed value. This is legal only in a standard complex CKKS environment. It must not silently be used in a real-only/conjugate-invariant environment. For real-only CKKS, absorb sqrt(abs(v_j)), reorder positive and negative units into equally padded dyadic halves, reduce each half in parallel, then subtract the second half with one final rotation. This avoids a post-square plaintext multiplication but can require H=2 max(ceilpow2(n_positive),ceilpow2(n_negative)) slots. Count the resulting batch-capacity loss fairly.

With the complex baseline, direct contraction saves only three of ten rotations, no ciphertext products, and can have larger coefficient norms. A claim of 64/5 encrypted speedup would be invalid here. At high batch B approaching S, structure-of-arrays (one ciphertext per input feature) has different counts: original h ciphertext squares versus paired k products, and substantially fewer affine combinations after contraction. Report the batch range; do not generalize a saturated throughput gain to single-request latency.

## Exact BSGS cost model

For a transform A with occupied cyclic diagonal set D, choose baby width b. Define B_b={r mod b:r in D}, G_b={b floor(r/b):r in D}. Write

  A x = sum_(g in G_b) Rot_g( sum_(a:g+a in D) Rot_-g(D_(g+a)) * Rot_a(x)).

Rotation units can be multiplied by B for the interleaved layout above. Ignoring rotations by zero, this has |D| plaintext products, |B_b minus {0}| baby rotations and |G_b minus {0}| giant rotations. Pre-rotate plaintext diagonals before encoding. For several transforms of the SAME input, baby rotations can be shared over the union of B_b sets; giant rotations usually cannot because their partial ciphertexts differ.

A measured backend model should separate decomposition, automorphism/external-product and modulus-down costs:

  T(A,b,l) = T_decomp(l) + n_b T_hoisted_rot(l)
             + |D| T_plainmul(l) + n_g T_giant_rot(l)
             + T_adds(l) + T_rescale(l).

Double-hoisting can move modulus-downs out of inner sums. Do not multiply a naive full-rotation latency by the number of baby steps when using a hoisted backend. Choose b using measured operation costs at the actual levels, not sqrt(|D|) alone. A split that minimizes rotation count may have a larger live baby-step cache or different memory traffic. Bound the cache explicitly (e.g. one to all baby ciphertexts) and benchmark the schedule.

An honest selector compares complete schedules:

  choose argmin_P [ T_client(P)+T_serialization(P)+T_server(P)+T_decrypt(P)
                   + (T_compile(P)+T_keygen(P)+T_key_distribution(P))/M ],

subject to the same input-domain error limit, security parameter, fixed B, result interface, and peak-memory cap; M is the declared reuse count. Keep server-only and total-latency metrics separate. Keys are charged by the UNION of rotation offsets used in matvecs and final reductions, including any composed-key plan and its extra operations. Identical logical rotation counts do not imply identical key bytes when offsets or key-switch decomposition parameters differ.

## Concrete candidate mechanism beyond a scalar rank rewrite

The most defensible new candidate is an **error-constrained, packing-aware quadratic contraction selector** with a finite, auditable search space:

1. Recognize a public affine -> coordinatewise degree-two polynomial -> public scalar linear reduction subgraph. Do not cross a nonlinear boundary outside this polynomial class.
2. Construct exact/symbolically checkable Q,ell,c' where coefficients are rational or finite binary values; otherwise record a certified coefficient-rounding bound.
3. Generate unchanged, direct Q, triangular Q, signed-square, paired-product, and predetermined groupwise variants.
4. For each variant enumerate layout width, BSGS split, shared-rotation schedule, available-key basis, and rescale location. Preserve a canonical input/output layout or charge encrypted conversion.
5. Propagate coefficient/numerical error, choose parameters from the same validated security envelope, estimate the complete-chain objective above, and reject any inadmissible plan.
6. Emit the minimum candidate plus a manifest containing exact coefficient identities, occupied diagonal sets, key offsets, levels/scales, expected operation counts, and the runner settings. Verify actual counters against the manifest during encrypted execution.

For triangular Q define A_ii=Q_ii, A_ij=2Q_ij for i<j, A_ij=0 for i>j; x^T A x=x^T Q x. Symmetric dense Q has d occupied cyclic diagonals; triangular A may still have all d, so this is NOT automatically cheaper. Sparse/structured Q may improve diagonal support, but doubling off-diagonal coefficients may worsen numerical error. Compare it rather than assuming.

Groupwise fusion partitions hidden units into G sets, constructs Q_g for each, and reduces sum_g x^T Q_g x. This can retain sparsity that full coefficient fusion destroys. It pays for additional transforms/products. Search only a fixed family (e.g. original order contiguous groups and support-identical groups) with dynamic programming over candidate intervals; do not claim global optimum over all set partitions. Each edge cost is a complete profiled schedule including shared rotations. A global all-pairs fusion that changes sparse W into dense Q can lose even with fewer squares.

This selector is a concrete compilation method, but novelty remains UNPROVED. The public-source checks below establish significant overlap with known layout search, BSGS, hoisting, sparsity/encoding co-optimization, and generic constant folding. A new contribution requires a precise mechanism (for example a proved finite-search optimality statement under a defined error/key-memory model) and measured advantage over all strong candidate baselines, especially direct Q. Merely packaging existing candidates is not a scientific novelty claim.

## Error argument usable by an implementation

Mathematical identity: expanding squares gives f=x^T Qx+ell^Tx+c'. The contraction changes no model function. For any real or complex x and symmetric Q this identity holds exactly. The final SIMD circuit computes each sample independently because every rotation offset is a multiple of B; the periodic input packing maintains its feature-coordinate identity modulo d.

Domain: fix ||x||_infinity <= R. Let Q_tilde=Q+E_Q, ell_tilde=ell+e_l, c_tilde=c'+e_c be encoded/rounded clear coefficients. A coefficient-only uniform error bound is

  E_coeff <= R^2 sum_ij |E_Qij| + R sum_i |e_li| + |e_c|.

This covers coefficient construction and quantization, but not CKKS execution. For the direct route let decoded input x_hat=x+e_x with ||e_x||_infinity<=eps_x; let y_hat=Q_tilde x_hat+ell_tilde+e_LT with ||e_LT||_infinity<=eps_LT, where e_LT includes plaintext encoding, key switching, summation and rescale. Then each pointwise product error relative to x_i (Q_tilde x+ell_tilde)_i is bounded by

  R eps_LT + eps_x ( R ||Q_tilde||_infinity + ||ell_tilde||_infinity )
    + R ||Q_tilde||_infinity eps_x + eps_x eps_LT
    + ||Q_tilde||_infinity eps_x^2 + eps_mul.

Here matrix ||Q||_infinity is maximum absolute row sum. Summing d coordinates multiplies this crude bound by d, then add the final reduction/key-switch error eps_reduce and E_coeff. This is conservative; a componentwise bound sum_i of the corresponding row-norm expressions is tighter. eps_mul must include relinearization and rescale, and eps_reduce must include all reduction rotations. A backend noise theorem or interval analysis must supply these eps values; an observed maximum over random samples cannot be called a uniform-domain theorem.

For paired factors exact identity uses sym(U^T V)=Q. Coefficient residual contributes R^2 sum_ij |sym(U_tilde^T V_tilde)-Q|_ij. If each factor value has error eps_u,eps_v and magnitudes bounded by R||u_i||_1,R||v_i||_1, product error is <= R||u_i||_1 eps_v+R||v_i||_1 eps_u+eps_u eps_v+eps_mul, summed over i, plus linear/constant/reduction errors. Spectral/eigen factors can cause cancellation or large row norms; scalar rank optimality does not minimize this bound. Scale each factor pair (u,v)->(alpha u,v/alpha) to balance R||u||_1 and R||v||_1 and the measured noise budget, preserving the polynomial. Do not truncate eigenvalues unless the same permitted approximation bound is charged explicitly.

## Scale, depth, memory obligations

- Encode matvec diagonals at scale Delta, input at Delta: sum all plaintext products at scale Delta^2 and rescale ONCE after the linear transform when modulus/noise bounds permit. Bias must be encoded at the correct post-rescale level/scale. The direct product uses a modulus-switched copy of x at the resulting level; this is not a decrypt or refresh.
- Then one ciphertext multiply, relinearize, rescale, reduce. Usually two consumed scale levels (linear plaintext multiplication then nonlinear ciphertext multiplication), zero bootstraps for this shallow class. Actual level bookkeeping differs between fixed/manual/automatic scaling backends; record exact prime chain and scale after each stage.
- Retain x only until its final direct multiply. Paired transforms retain at least one linear-transform output while computing the other; a global hoisted baby cache can dominate peak memory. Share only while peak-live accounting permits.
- Relinearization is not intrinsically needed before final decryption; delaying it past rotations can be impossible/expensive depending backend APIs. Any lazy final multiplication or omission of relinearization must be made available to the baseline too.
- Smaller post-compilation modulus chains can lower N under the same security target, but compare both fixed-parameter and optimized-equal-security cases; charge all keygen/key bytes. Identical plaintext function does not imply identical required precision when coefficient norms change.

## Primary-source evidence and baseline requirements

1. [Orion paper, ASPLOS 2025, sections3.1-3.3](https://arxiv.org/html/2311.03470v3): its linear kernels use diagonal encoding, BSGS and double hoisting. Thus a naive diagonal-only comparator is weaker than this established baseline. Orion's author-reported network speedups were not reproduced in this subtask.
2. [Official Orion implementation](https://github.com/baahl-nyu/orion): implementation to consult/reuse rather than a similarly named unrelated runtime. Pin a commit before a reproduction.
3. [HEIR passes](https://heir.dev/docs/passes/): official documentation includes BSGS rotate-and-reduce lowering, diagonal matvec lowering, constant folding, layout-assignment conversion folding and split preparation/application of linear transforms. These operations themselves are prior art.
4. [HEIR layout design](https://heir.dev/docs/design/layout/): layouts are represented as mappings to ciphertexts/slots; encrypted layout conversions can carry substantial rotations. Integrating quadratic candidates before layout selection is a plausible extension, not demonstrated novelty.
5. [HEIR ciphertext management](https://heir.dev/docs/design/management/): management policies insert rescale/modulus-switch/relinearization; operation latency depends on ciphertext level. It supports the need to compare schedules at actual levels.
6. [Ebel, Garimella, Reagen, Network and Compiler Optimizations for Efficient Linear Algebra Kernels in Private Transformer Inference](https://arxiv.org/html/2512.11135v1): authors compare packed-row and BSGS kernels in Orion, reporting regime-dependent advantages. This further motivates including layout competitors; their measurements are external claims here.
7. [Ran et al., SpENCNN, ICML 2023](https://proceedings.mlr.press/v202/ran23b.html): primary prior art on joint encoding and sparsity optimization. A claim that jointly choosing coefficient sparsity and packing is new would be too broad.
8. [Bossuat et al., Efficient Bootstrapping for Approximate Homomorphic Encryption with Non-Sparse Keys](https://eprint.iacr.org/2020/1203): primary reference behind double hoisting. Search metadata verified the paper; the PDF fetch failed in this session, so detailed claims should use the Orion exposition or fetch the paper before relying on exact formulas.

No source reviewed here proves absence of prior art for quadratic contraction selection. No novelty conclusion is established.

## Decisive experiment recommendations

- At minimum fix d=8, h in{16,64,256}, dense/sparse/block-diagonal W, indefinite and positive-semidefinite outputs, and batches spanning B=1 to slot saturation. Reserve independent seeds. Include the existing h64 example as one case, not the entire class.
- Compile original packed BSGS+hoisting, direct Q, signed eigen square, paired inertia, and a structure-of-arrays throughput baseline. Choose best existing schedule under the same B and hardware.
- Log plaintext products, ciphertext products/squares, key switches, rescale, modulus switches, rotations by offset/level, input/output bytes, key bytes, peak live ciphertext count, RSS and timing for every full-chain stage.
- Ablations: (a) coefficient fusion alone, (b) layout choice alone, (c) shared babies alone, (d) key-cache constraint alone, (e) final combined selector. This isolates any new mechanism's value.
- If paired inertia loses at B=1 but wins near saturation, report a throughput-specific result and the crossover. Do not hide the negative result.
