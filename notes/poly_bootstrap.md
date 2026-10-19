# Polynomial approximation and bootstrapping route audit

Date: 2026-10-19. Independent route investigation. This note contains proved algebraic statements and proposed experiments; it contains **no reproduced bootstrap performance result** and establishes no scientific novelty claim.

## 1. Consequence for the primary quadratic class

For a public square-activation network

`f(x) = sum_i v_i (w_i^T x + b_i)^2 + c`

with a scalar or fixed small-vector output, the original circuit and any exact quadratic-form rewrite have ciphertext–ciphertext multiplicative depth one. The number of parallel products changes width, not depth. Arbitrary fractional public affine transforms can also consume CKKS rescale levels, so the complete level budget must count those transforms. Nevertheless, a strong shallow leveled CKKS implementation can give **zero bootstrap calls to both circuits** with the same sufficiently long parameter chain. Claiming that the 64-to-5 scalar product transformation saves bootstrapping is unjustified for this class.

There is no activation approximation error when the original model itself uses a square activation. If the original model instead uses ReLU, sigmoid, GELU, etc., replacing it by a quadratic is a model change; it requires the same fixed quality contract and cannot be silently attributed to a compiler improvement.

The relevant primary performance comparison for the quadratic class is therefore width/data movement/key memory versus linear transforms, not bootstrap count. A one-ciphertext packed hidden layer can square all 64 neurons in one ciphertext multiplication, so a scalar count of 64 squares is not a ciphertext count of 64 squares.

## 2. Prior art that closes obvious alternative routes

Verified primary sources, accessed on 2026-10-19:

- [Orion paper, arXiv:2311.03470](https://arxiv.org/abs/2311.03470): its authors already propose automatic bootstrap placement and scale management. The paper reports benchmark improvements; those are author claims until reproduced here. [Official implementation](https://github.com/baahl-nyu/orion).
- [HEIR official pass documentation](https://heir.dev/docs/passes/#-ilp-bootstrap-placement): `ilp-bootstrap-placement` jointly models levels and Orbit-style scales and places scale changes, rescales, and bootstraps. It is not new merely to put these decisions in an ILP. [Polynomial lowering](https://heir.dev/docs/passes/#-lower-polynomial-eval) already supports Horner and Paterson–Stockmeyer in monomial/Chebyshev representations. HEIR source builds may require a substantial LLVM build; using the already available FHE backend is preferable for the first experiment.
- [RBOOT, USENIX Security 2026](https://www.usenix.org/conference/usenixsecurity26/presentation/yang-zhaomin): explicitly incorporates ReLU evaluation into CKKS bootstrapping's nonlinear EvalMod step. Its abstract reports 2.77x faster end-to-end inference and 81% lower memory versus polynomial approximation systems on four CNNs. These numbers are not reproduced here. Generic activation/bootstrap fusion is therefore occupied prior art.
- [OpenFHE official CKKS bootstrap documentation](https://github.com/openfheorg/openfhe-development/blob/main/src/pke/examples/CKKS_BOOTSTRAPPING.md): bootstrap setup/precomputation and required linear-transform plaintexts are real costs. Excluding these costs or the associated evaluation keys invalidates a memory comparison.

These sources do not establish that the candidate below is new. A focused prior-art search for low-rank refresh cuts is still necessary before making such a claim.

## 3. Independent alternative: choose a basis at a refresh cut

This is a concrete, testable compiler construction rather than a request for an AI to search parameters.

### Precisely defined class

Let `h(x)` be any polynomial neural subcomputation. At a point where ciphertexts need refreshing before further nonlinear computation, assume its only subsequent use is through a public linear map `B in R^(r x m)`. The public matrix has exact rank `k < r`, and `h` has a certified coordinatewise range `|h_i| <= H_i` over the input domain. Use feature-major packing: each ciphertext holds one feature across `S` independent samples. Thus an `r`-feature intermediate occupies `r` ciphertexts at a full batch of `S` samples. The layout assumption is decisive and must be compared against packed-output alternatives.

Factor `B = D C`, with `C in R^(k x m)`, `D in R^(r x k)`. Replace

`h -> C -> D -> bootstrap(r coordinates) -> nonlinear consumer`

by

`h -> C -> bootstrap(k coordinates) -> D -> nonlinear consumer`.

Both sides must use the same exact linear factorization. This isolates the refresh-cut change from ordinary low-rank matrix multiplication. The original uncompressed dense-B implementation is an additional baseline, not the decisive one.

### Exact semantics and minimality

Ignoring CKKS error, bootstrap is the identity on messages. Both circuits therefore return `D C h = B h`. There is no intermediate decryption and no client help. The server sees only public weights, layouts, dimensions, keys, and ciphertext messages.

Among representations that transmit a `t`-dimensional **linear state** across the cut and linearly reconstruct `B h` for all `h in R^m`, the minimum state dimension is exactly `rank(B)`:

- Any factorization `B = D' C'` through `R^t` has `rank(B) <= t`.
- A rank factorization of width `k = rank(B)` attains the bound.

This does not prove a minimum number of ciphertexts across arbitrary SIMD layouts, nonlinear encodings, restricted reachable sets, or approximate compression. Those broader claims would be false or require a different theorem.

### Range normalization and an explicit error condition

For row `c_i` of `C`, set

`R_i = sum_j |C_ij| H_j`.

Rows with `R_i=0` are identically zero and need no bootstrap. For the others choose

`C'_ij = C_ij / R_i`, `D'_ji = D_ji R_i`.

Then `|C' h| <= 1` coordinatewise and `B = D' C'`. Suppose input cut ciphertexts have featurewise error `e_h`, the pre-bootstrap linear implementation contributes `eta_C`, bootstrap adds at most `beta_i` to basis coordinate i on its certified domain, and post-bootstrap reconstruction contributes `eta_D`. Then the output error is bounded componentwise by

`|e_out| <= |B| |e_h| + |D'| (|eta_C| + beta) + |eta_D|`,

provided the bootstrap domain includes the input errors as well as the exact range. Use a margin in normalization to cover those errors. The direct-output bootstrap baseline has its own bound; a comparison must enforce the same final error threshold for both. A basis with poorly conditioned reconstruction can lose all its performance advantage by requiring a more precise bootstrap.

An SVD yields a stable starting factorization but does not minimize the relevant componentwise error amplification `max_j sum_i |D'_ji| beta_i`, nor the packed transform cost. Exact rational rank factorizations can avoid an unproved floating rank decision, but may be poorly conditioned. Rank tolerances must not silently truncate the model.

### Clean family with no error-amplification loophole

Partition `m = k g` coordinates into k equal blocks and let each C row average one block: coefficients are `1/g` on its block and zero elsewhere. Let every D row be a signed average of the k basis coordinates, with coefficients `+/- 1/k`. With `H_i=1`, both `||C||_infinity` and `||D||_infinity` equal 1. Therefore `|C h| <= 1`, `|D C h| <= 1`, and bootstrap basis error bounded by beta reconstructs with error at most beta. Select D with rank k; duplicate output rows are permitted but must be disclosed. This supplies an exact family where changing the cut does not intrinsically weaken the error requirement.

This family may be too easy for an existing algebra-aware placement compiler; that is a falsification opportunity, not a reason to remove the baseline.

### Total-cost acceptance rule

For a fixed factorization/layout/configuration, candidate acceptance requires

`T_C + k T_boot + T_D + T_extra_movement < T_C + T_D + r T_boot`,

with operations evaluated at their actual modulus levels and scales. In practice `T_D` can differ before and after refresh because the modulus chain changes. The correct inequality uses those separately measured costs, plus any conversions, masks, key loading, and peak live ciphertexts. A `k/r` bootstrap ratio is not an end-to-end speedup.

Bootstrap keys may be identical across all coordinates, so fewer bootstrap calls do **not** automatically mean smaller serialized key material. Peak memory can fall if fewer live ciphertexts and temporary buffers are needed, while evaluation-key size remains unchanged. Conversely, supporting multiple bootstrap precision configurations can increase key memory.

### Predetermined experiment matrix

Use an official bootstrapping-enabled Lattigo or OpenFHE example as the primitive baseline and hold its security parameters fixed across methods. First measure a real bootstrap and linear transforms, then proceed only if the cost inequality predicts a useful region.

Freeze these cases before measuring results:

| Case | m | r | k | Purpose |
| --- | ---: | ---: | ---: | --- |
| cut-A | 64 | 32 | 4 | clear low-rank opportunity |
| cut-B | 128 | 64 | 8 | larger held-out width |
| cut-C | 64 | 32 | 16 | weaker rank advantage |
| cut-D | 64 | 32 | 32 | full-rank negative control |

For each case evaluate batch sizes `1`, `S/8`, and `S`, where S is the fixed slot count. Include both feature-major and an output-packed baseline. Use public fixed seeded matrices and disjoint held-out inputs. For a genuine neural computation, set `h_i = (a_i^T x + b_i)^2` with a bounded public affine layer and append a second square after the refresh cut. The full function is a predetermined degree-four polynomial network; the refresh is only justified when the fixed inherited level budget requires it. Also report the best zero-bootstrap leveled implementation: if that is faster at the same security/quality target, reject refresh-cut optimization as a final method for this shallow class.

Required ablations: bootstrap before D versus after D using the identical C,D; dense B versus factored C,D; normalized versus unnormalized basis only where both domains are valid; computation-only versus complete lifecycle; one-shot versus amortized setup. Do not force the baseline to bootstrap if a valid leveled chain is better.

### Status and decisive gap

The linear-state theorem and error bounds above are established algebraically. The useful cost region and novelty are unverified. No encrypted experiment was run for this alternative. It is a reserve route if quadratic packed linear transforms erase the main candidate's advantage. It should not displace the simpler no-bootstrap experiment without measured evidence.

## 4. Error contract for the primary exact quadratic experiment

For `f(x) = x^T Q x + l^T x + c` on `||x||_infinity <= R`, let compiled coefficients have residuals `Delta Q`, `Delta l`, `Delta c`. A uniform coefficient-conversion certificate is

`epsilon_compile <= R^2 sum_ij |Delta Q_ij| + R sum_i |Delta l_i| + |Delta c|`.

If decoded encrypted input has error at most delta in infinity norm, then original-polynomial input sensitivity is bounded by

`2 R sum_ij |Q_ij| delta + sum_ij |Q_ij| delta^2 + ||l||_1 delta`.

For an inertia-factor implementation `sum_i (u_i^T x)(v_i^T x)`, suppose the two encrypted linear forms have errors bounded by `delta_ui` and `delta_vi`. Its product perturbation is at most

`sum_i [R ||u_i||_1 delta_vi + R ||v_i||_1 delta_ui + delta_ui delta_vi + eta_muli]`,

plus final sum/linear/bias errors. This explicitly exposes cancellation and coefficient-norm amplification that a product count misses. FHE primitive errors need validated bounds or a clearly distinguished empirical estimate; sampled error is not a uniform proof over the continuous domain.

If a classifier is used, a uniform logit error <= epsilon preserves any prediction whose top-two original-logit margin exceeds `2 epsilon`. Low-margin cases require the predetermined quality metric rather than an unsupported claim of identical labels.

## 5. Next executable obligation

For the primary quadratic route: validate the same CKKS parameters and error contract for packed-square, direct-Q, and factor-based circuits; record actual ciphertext operations and total costs. Neither polynomial approximation nor bootstrapping is needed to make that first decisive experiment valid. For the reserve route: start from a real bootstrap primitive and test cut-A with the matched factored baseline before expanding the matrix.
