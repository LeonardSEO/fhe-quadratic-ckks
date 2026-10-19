# Independent CKKS packing and support analysis

Status: 19 October 2026. The operation-count lemmas and exact rational probe below
are established locally. This note contains no encrypted timing claim. The root
benchmark is responsible for measuring the candidates with equal security,
accuracy, hardware and batching. Existing complex packing, LDL factorization,
BSGS and delayed relinearization are prior art; scientific novelty of the
specific support-aware compilation rule remains to be assessed.

## Computation and layouts

Let `n` be the input dimension, `h` the original quadratic hidden width, `B` the
number of samples, and `S` the number of complex slots in standard CKKS. Public
weights give

```
f(x) = sum_i v_i (w_i^T x + b_i)^2 + c
     = x^T Q x + ell^T x + c0,
Q = W^T diag(v) W,
ell = 2 W^T diag(v) b.
```

The original input owner encrypts the raw features. All transformations discussed
here use public coefficients on the server; none asks the client to compute a
model-dependent intermediate value or decrypt an intermediate result. The client
decrypts only the final ciphertext. For complex execution it returns its real
part, which is a final output decoding convention.

Two different layouts must not be conflated:

* **Samples in slots, one feature per ciphertext:** `X_j[b] = x_b[j]`, with
  `B <= S`. There are `n` input ciphertexts. An arbitrary linear form has one
  ciphertext result, no rotations, and one public scalar multiplication for
  each nonzero coefficient. This is the layout in which scalar product savings
  can translate directly into fewer ciphertext nonlinear operations.
* **Feature blocks interleaved with samples:** slot `j*B+b` stores feature `j` of
  sample `b`. If the feature dimension is padded to a power of two `d`, capacity
  is `B <= S/d`. Rotation by `r*B` is a cyclic feature rotation without mixing
  sample lanes. Repeat input features across blocks only when the declared
  input encoding explicitly includes that duplication, or charge the server
  replication operations. There may be one input ciphertext, and many scalar
  nonlinear products can run in one ciphertext multiplication.

For arbitrary batch sizes and multiciphertext tiling, count all chunks, padding,
and layout conversions. A layout is part of the application contract, not free
compiler metadata.

## Strong baselines for full sample SIMD

The following counts concern only the quadratic term, with nonzero generic
coefficients, before implementation-specific simplifications of `0`, `1`, `-1`
and integer coefficients. Add the common linear and constant terms explicitly.
Alternatively use a numerically justified completion of the square when
`ell` is in the range of `Q`; do not silently ignore its conditioning or cost.

| Method | Public multiplications | Ciphertext nonlinear operations | Rotations |
|---|---:|---:|---:|
| Original neurons | `nnz(W)` | `h` squares | 0 |
| Dense folded `Qx`, then `x dot Qx` | `nnz(Q)` | `n` general multiplies | 0 |
| Symmetry-aware folded Q | at most `n(n+1)/2` | `n` general multiplies | 0 |
| Real spectral signed squares | `nnz(F_spectral)` | `rank(Q)` squares | 0 |
| Real signed LDL squares | `sum_i |support(f_i)|` | `rank(Q)` squares | 0 |
| Real opposite-sign product pairing | sum of supports of both paired `u+v,u-v` forms | `max(p,m)` general multiplies | 0 |
| Complex opposite-sign square pairing | sum of union supports | `max(p,m)` squares | 0 |

The symmetry-aware direct baseline evaluates

```
g_i = Q_ii X_i + 2 sum_{j>i} Q_ij X_j,
quadratic = sum_i X_i * g_i.
```

This uses half the dense public matrix coefficients and must be included. A
comparison only against original neurons or dense `Qx` overstates the new
method's advantage. Real signed LDL is another important baseline: triangular
factorization already provides at most `n(n+1)/2` coefficients without complex
pairing.

For the raw neuron baseline, absorbing `sqrt(abs(v_i))` into each affine form
and its bias removes a final general public weighting operation; signs remain
cheap. The original fixture uses small integer `W,v`, so libraries may perform
these coefficients at scale one. Irrational factors generally need another
scale. Either allow and measure the optimized integer baseline, or explicitly
preencode all coefficients at the same scale for a controlled mechanism
experiment. A controlled experiment is not automatically the strongest
end-to-end baseline.

## Concrete rule: complex sign pairing with support reuse

Suppose a verified signed Gram factorization is

```
Q = sum_{i in P} f_i f_i^T - sum_{j in M} f_j f_j^T.
```

For a positive row `u` and a negative row `v`, construct one complex linear form
`z = (u + i v)^T x`. Then

```
Re(z^2) = (u^T x)^2 - (v^T x)^2.
```

An unmatched positive row uses `z=u^T x`; an unmatched negative row uses
`z=i v^T x`. Thus `Re(sum_j z_j^2) = x^T Qx` with `max(p,m)` ciphertext squares.
There is no homomorphic conjugation or real-part extraction. The imaginary
cross terms are discarded only after final decryption by the input owner.
No imaginary output is returned to the server.

The useful new compiler choice is to share the public coefficient multiplication
on overlapping supports. If both real rows use input ciphertext `X_t`, computing
`(u_t+i v_t)*X_t` once produces both components. For a pair `(i,j)` this costs

```
|support(f_i) union support(f_j)|
```

public multiplications instead of the sum of the two support sizes. Let
`w_ij=|support(f_i) intersect support(f_j)|`. Then among maximum-cardinality
opposite-sign pairings,

```
M_complex = M_real - max_matching sum_(i,j) w_ij,
M_real = sum_i |support(f_i)|.
```

This is a concrete maximum-weight bipartite matching problem, not a language
model parameter search. When public multiplications have unequal measured cost
(different scale, level, or encoded representation), replace each input's unit
overlap weight by its saved measured cost. Under the stated layout all pairs
save exactly one square, so maximum-cardinality pairing is appropriate unless
numerical gates require a different factorization.

The rule does not prove that complex pairing beats all other layouts. Its
advantage must be selected by the complete measured cost, including encoding,
keys, communication, and the real-only CKKS alternative described below.

### LDL construction, support optimality and limits

For a symmetric permutation `P`, factor `P Q P^T = L D L^T`. With diagonal D,
the signed factor rows are the rows of `sqrt(abs(D))*L^T`, with signs `sign(D)`.
Undo the input coordinate permutation in the compiler. In full sample SIMD,
this merely selects different input ciphertext handles; in feature packing a
slot permutation may require rotations and masks and must be charged.

For a generic nonsingular triangular factor the supports are nested:
`S_i={i,...,n-1}`. For lengths `a_i=|S_i|` and `b_j=|S_j|` of opposite signs,
the matching weight is `min(a_i,b_j)`. Sort each sign group by descending support
size and pair in that order, leaving the smallest excess rows unmatched.

Proof: if `a>=b` and `c>=d`,

```
min(a,c)+min(b,d) >= min(a,d)+min(b,c).
```

One way to see it is to write `min(a,c)` as the integral of
`1[t<=a]*1[t<=c]` over nonnegative `t`: pairing the longer intervals together
maximizes their total intersection. Any crossed matching can therefore be
uncrossed without loss. If a larger available excess-side support is unmatched,
replacing a smaller matched one cannot lower the objective. Repeated exchanges
give the stated matching.

**Ablation limitation:** native triangular row order is already descending by
support size. Pairing the two sign lists by their first row index therefore
already implements this optimum. There is no legitimate claim of a sorted vs
first-index improvement on dense triangular LDL. General sparse factors can
have nonnested supports; then maximum-overlap matching can differ from a naive
pairing. Reverse-order pairing is a useful deliberately adverse check but is
not a strong baseline.

For zero or unstable pivots, use a symmetric Bunch-Kaufman factorization with
1x1 and 2x2 D blocks; diagonalize only each 2x2 block and absorb the two resulting
rows into the factors. The resulting supports are nested by pivot block before
structural zeros, so block-support pairing remains available. An initial
implementation may reject unsupported pivots and fall back to a spectral
factorization; this is valid only when the supported class and fallback are
reported. Adding diagonal jitter changes the model and is not a valid repair.
[LAPACK's DSYTRF documentation](https://netlib.org/lapack/explore-html/d8/d0e/group__hetrf_ga431b081d6c9c48af82ec003a7d3070ff.html)
specifies the symmetric block-pivot factorization;
[SciPy's official LDL documentation](https://docs.scipy.org/doc/scipy/reference/generated/scipy.linalg.ldl.html)
documents its permutation semantics.

### Exact local support and growth probe

Run from the repository root:

```
python3 fhe_research/packing_probe/support_probe.py
```

This standard-library-only probe uses exact rational LDL, verifies reconstruction
entry by entry, exhaustively checks the small matching optimum, and writes
`fhe_research/packing_probe/support_probe.json`. For the historical 8-input
fixture with inertia `(3,5)`, both tested orders use 36 real LDL coefficients
or 25 complex paired coefficients, and eight real squares or five complex
squares. The compiler avoids 11 public multiplications and three squares relative
to real triangular LDL. These are operation counts, not measured speedups.

The unpivoted diagonal includes a pivot near `0.0573`, followed by a pivot near
`-5299.49`. Its `||Q_abs||_infinity` bound is about `201534.65`. Greedy symmetric
largest-absolute-diagonal pivoting gives permutation `[3,1,4,0,6,5,7,2]`, the same
operation counts, and an infinity-norm bound about `780.20`. This greedy pivot
rule suffices for this fixture but is not a general replacement for 2x2 pivoting.

## Numerical gate

Let F be the real signed factor matrix, and define

```
Q_abs = F^T F.
```

For opposite-sign complex pairing, exactly

```
sum_j |z_j(x)|^2 = x^T Q_abs x,
```

independently of which opposite-sign rows are paired. Thus pairing does not
increase this total magnitude budget, but choosing LDL instead of spectral
factors may do so. Spectral factors have `Q_abs=|Q|`; an unfortunate LDL can
have much larger `Q_abs` even when Q itself is moderate.

If encrypted/encoded linear forms have total error vector `e` and the final
sum contributes absolute error at most `epsilon_sum`, then

```
|Re(sum_j (z_j+e_j)^2) - x^T Qx|
 <= 2 sqrt(x^T Q_abs x) ||e||_2 + ||e||_2^2 + epsilon_sum.
```

On `||x||_2<=R`, replace the first square root by
`R*sqrt(||Q_abs||_2)`. This is a uniform analytic requirement, not a random-input
test. Bound each linear-form error from input encryption error, coefficient
rounding, public multiplication, addition and rescale errors at the actual
levels. The error bound must cover both real and imaginary parts. If A is the
exact complex factor and its stored approximation is `A+E`, a separate model
coefficient error bound is

```
||Re((A+E)^T(A+E))-Q||_2 <= 2||A||_2||E||_2+||E||_2^2.
```

For a permitted maximum error `epsilon`, reject a factorization that cannot
meet the bound with the fixed cryptographic parameters. Record
`gamma=||Q_abs||_2/||Q||_2` as a diagnostic, not a substitute for the actual
error bound. The probe supplies a conservative exact rational upper bound on
`gamma^2` using `||Q_abs||_2<=||Q_abs||_infinity` and
`||Q||_2>=||Q||_F/sqrt(n)`.

Affine completion introduces offsets inside the forms; then replace
`|z_j(x)|` by `R*||a_j||_2+|beta_j|`, or use the homogeneous augmented vector
`(x,1)`. Solving `Q t=ell/2` and shifting `x+t` is exact for nonsingular Q but
can be ill conditioned. A separately evaluated common linear term is a safe
initial benchmark choice.

## Packed-feature baseline and BSGS

For interleaved feature blocks a public matrix product is

```
Y = sum_{s in D} d_s elementwise_mul Rot_{s*B}(X).
```

With `s=q*g+r`, pre-rotate the public diagonal as
`d'_s=Rot_{-q*g*B}(d_s)` and evaluate

```
Y = sum_q Rot_{q*g*B}(sum_r d'_(qg+r) * Rot_{r*B}(X)).
```

This uses `|D|` public multiplications, `|D|-1` additions, and the number of
distinct nonzero baby offsets plus distinct nonzero giant offsets in rotations.
For dense diagonals `0,...,d-1`, rotations are
`g+ceil(d/g)-2`. Use BSGS and available hoisting in the strong baseline; compare
actual runtime because rotation count alone omits decomposition and mod-down
cost. The foundations are described in
[Halevi and Shoup's HElib paper](https://www.shoup.net/papers/helib.pdf),
[Bossuat et al.'s double-hoisting work](https://eprint.iacr.org/2020/1203), and
the [Orion publication](https://arxiv.org/abs/2311.03470).

For `n=8,h=64` and `B<=S/64`, a client encoding with eight repeated copies of
the eight features permits all 64 hidden outputs in one ciphertext. This
baseline uses eight public diagonal multiplications, BSGS input rotations,
one ciphertext square, and six rotations for a 64-block reduction. Scaling
each affine by a complex square root of `v_i` can absorb arbitrary signed
output weights, making even the raw baseline stronger in standard complex
CKKS. The five-factor method padded to eight blocks also uses eight public
diagonal multiplications and one square, with only three reduction rotations.
A folded `Qx` baseline uses eight public multiplications, one general
ciphertext multiplication, and three reduction rotations. Consequently the
scalar 64-to-5 claim says very little about low-batch packed latency.

The algebra agent's scalar-output cyclic orientation is a further strong packed
competitor: choose a generally nonsymmetric A with `sym(A)=Q` and only cyclic
diagonals `0,...,floor(n/2)`. Use `A_ii=Q_ii`; for `1<=r<n/2`, use
`A_(i,i+r)=2Q_(i,i+r)`; for the even-dimension half-turn use coefficient Q on
both paired positions. Then evaluate `reduce(X*(A X+ell))+c0`. For generic dense
Q this uses `floor(n/2)+1` public diagonal multiplications, one ciphertext
multiplication, and a reduction. At `n=8`, BSGS with `g=2` needs three matvec
rotations for those five diagonals, versus four for an eight-diagonal transform.
The agent developing that route supplies its proof; it should be benchmarked
before claiming any packed-factor advantage.

## Relinearization, levels, memory and keys

For all sum-of-products methods, multiply without relinearizing, sum the
degree-two ciphertexts, and relinearize the sum once. Apply the same rule to
the original-neuron, direct-Q, real LDL and complex candidates. Count
relinearizations separately from ciphertext multiplications. This optimization
is established prior art, including current
[OpenFHE performance guidance](https://github.com/openfheorg/openfhe-development/blob/main/docs/static_docs/Best_Performance.md).

The full sample SIMD opposite-sign method needs no rotation or conjugation
keys, and the same relinearization key as its real competitors. Its key-size
advantage over another zero-rotation baseline is therefore zero. Streaming one
factor at a time can keep the input ciphertexts, a degree-two output accumulator,
one linear-form accumulator and one temporary product live. Streaming is also
available to the baselines. Report actual peak resident memory; do not count a
materialized baseline against a streaming candidate as a factorization gain.

At N ring coefficients and L active RNS limbs, a raw degree-one ciphertext
contains about `2*N*L*8` coefficient bytes and a degree-two ciphertext
`3*N*L*8`, excluding metadata and allocator overhead. A relinearization or
rotation key has multiple decomposition digits and Q/P limbs, so its actual
serialized size is authoritative. For packed BSGS, count the union of baby,
giant and reduction offsets, sharing identical keys across steps. Key count
times the measured serialized single-key size is a useful model, but report
the total serialized key bundle too.

Each method has one ciphertext multiplicative depth for the quadratic term;
public coefficient multiplication and rescale may consume another modulus
level. Complex coefficients, real fractional coefficients, and small integers
need not follow the same library path. Record actual input/output levels and
scales. No bootstrapping is required for this shallow class at suitable secure
parameters, so bootstrap count should be zero in these comparisons.

Use a level-aware measured cost model rather than a scalar product count:

```
T_eval = sum_level (Mpm*t_pm + Msq*t_sq + Mcc*t_cc
                   + R*t_rot + A*t_add + Relin*t_relin + Rescale*t_rescale),
```

plus allocated-byte and key-byte constraints and any conversion operations.
CPU calibration must use the same N, levels, precision and thread settings.
Plaintext preparation, public model compilation and key generation are one-time
costs only when the stated deployment actually reuses them; charge their first
use and state the amortization threshold.

## Alternatives that must remain active

1. **Same-sign complex pairing:** `z=u+i v` gives
   `z*conj(z)=u^2+v^2`. Allowing same-sign pairs can lower nonlinear count to
   `ceil(rank(Q)/2)`, but each such pair costs a conjugation and a conjugation
   evaluation key. For inertia `(3,5)`, three opposite-sign square pairs plus
   one negative same-sign pair use four nonlinear products and one
   conjugation. This is not a contradiction to the real `max(p,m)` theorem;
   it uses complex arithmetic plus an automorphism. Select only if the saved
   linear work and product outweigh conjugation, key distribution and memory.
2. **Real-only conjugate-invariant CKKS:** this is a serious strong baseline.
   [Kim and Song](https://eprint.iacr.org/2018/952) establish a real variant
   with twice as many slots under their stated security/storage/cost comparison.
   Current [Lattigo CKKS documentation](https://pkg.go.dev/github.com/tuneinsight/lattigo/v6/schemes/ckks)
   sets default log-slots to `LogN` for this variant and `LogN-1` for Standard.
   The complex rule requires Standard. At fixed B and security the minimum
   ring may still be the same, in which case unused real slots confer no
   benefit; at large B real LDL may achieve better throughput. Compare this
   explicitly, and do not generalize a Standard-only result to all CKKS.
3. **Support-aware coefficient order or sparse pivoting:** model-independent
   sparse LDL orderings can reduce fill. In full sample SIMD coordinate
   permutations are handle selection; in packed features they may cost actual
   encrypted movement. Optimize the complete representation, not nnz alone.
4. **Spectral fallback:** although dense, it gives controlled intermediate
   magnitudes and may win when triangular growth forces more precision.

Complex packing is not itself new: [nGraph-HE2](https://eprint.iacr.org/2019/947)
reports CKKS packing and lazy-rescaling optimizations. Its client-aided
activation execution must not be imported into this task's no-intermediate-
decryption protocol. OpenFHE's current
[complex-number example](https://github.com/openfheorg/openfhe-development/blob/main/src/pke/examples/simple-complex-numbers.cpp)
also confirms that current complex mode is available; older claims that OpenFHE
cannot handle complex slots are stale. No performance figure from any of these
external sources has been independently reproduced in this note.

## Decisive evaluation gates

Fix several model families before timing, including balanced and unbalanced
inertia, PSD, sparse factors, an LDL-growth stress case, and held-out models.
Use full sample SIMD and low-batch feature packing at the same B for every
comparison. Compare against the fastest applicable real LDL, symmetry-aware
direct Q, cyclic-orientation packed Q, original neurons, and the real-only ring
variant. Include all legal baseline optimizations rather than choosing a
single weak original graph.

Separate setup, encryption, upload, server execution, download and final
decryption. Check all complex result errors against the original real model;
do not validate only a transformed plaintext program. Report uniform coefficient
and arithmetic error bounds as well as independent random and boundary-input
tests. A claim is supported only in the rows of this evaluation matrix where
actual encrypted execution, fixed error requirements and complete costs show
an advantage.
