# Scalar-consumer quadratic compilation

This implementation demonstrates a restricted encrypted improvement. The rule
changes the linear map *before* lowering it to CKKS: it preserves the terminal
scalar polynomial instead of preserving an unnecessary intermediate vector.
It is applicable to real scalar quadratic scores with public coefficients.
Historical publication priority remains unestablished.

## Target and assumptions

The public model is

\[
 f(x)=a\left(\sum_{k=1}^h v_k(w_k^T x+b_k)^2+c\right)
     =x^TQx+\ell^Tx+c_0.
\]

Inputs are real, within [-1,1]^d, d in {8,16,32}; the primary application batch
is128 independent examples. Public weights are integer/dyadic in the frozen
experiments, with no approximation, pruning or retraining. The reusable rule
accepts arbitrary finite symmetric public Q; the measured regime is dense small
quadratics with d a power of two and batch dividing4096/d. It supports indefinite
and positive definite Q. It does not preserve Qx for another consumer, and it is
not a general deep-network rewrite.

One client owns input, secret key and final outputs. The passive server receives
public model/parameters, evaluation keys and encrypted inputs. The model,
dimensions, batch, schedule, key sizes, ciphertext lengths, timing and memory
are public observables. No decrypted outputs/errors are returned to the server.
The evaluator and HTTP handler accept no secret key or decryptor. Actual
protocol confirmation uses an exec-child server in a distinct address space,
entered before any client key/fixture initialization. Its environment is
sanitized, and only typed public configuration enters through stdin; evaluation
keys and encrypted requests enter through HTTP. Older combined-role runs are
retained as controlled performance ablations, not address-space isolation proof. Synthetic
benchmark fixtures are published deliberately; this publication is not part of
a private deployment. Client input packing only replicates coordinates and adds
no model-dependent plaintext feature computation.

## Compiler construction and proof

For real x, x^T K x=0 for every skew-symmetric K. Consequently all
representatives A with (A+A^T)/2=Q compute the same scalar:

\[
 f(x)=\sum_i x_i ((Ax)_i+\ell_i)+c_0.
\]

Existing diagonal/BSGS kernels optimize a fixed linear map. This pass first
chooses a cheaper representative from this scalar-equivalence class.

Let r=(j-i) mod d. Keep Q's diagonal, keep the half-turn diagonal r=d/2 when
d is even, double Qij on offsets1 through floor((d-1)/2), and set the partner
offsets to zero. The two entries Aij,Aji then sum to2Qij, including the
self-inverse half-turn case, so sym(A)=Q entry by entry. Thus the transformation
is an identity on the entire real input domain, not a sampled algebraic fit.
`scalar_compiler.cyclic_representative` verifies it with exact rationals for
stored binary64 coefficients.

The resulting cyclic support has at most floor(d/2)+1 diagonals. For a generic
dense symmetric Q this is minimal among scalar-equivalent cyclic diagonal
matrices: every pair of offsets {r,-r} needs at least one member, or the
corresponding Qij would be zero. Diagonal0 and the even half-turn need their own
members. This proves a restricted support optimum; it is not a global lower
bound on rotations, ciphertext products or runtime. BSGS can use fewer rotations
than the number of nonzero diagonals.

For sparse Q, omit exact zero diagonals and retain the existing baseline if the
measured cost model prefers it. The minimum-support construction alone does not
determine the best BSGS split or evaluation-key set.

## Encrypted layout and execution

In Standard CKKS there are4096 complex slots. Pack the real coordinate i of
example b in slot i*B+b, repeated periodically every d*B slots. Rotating by r*B
therefore changes only the feature coordinate. All methods use the same input
layout, client encryption, output size and batch. The client decrypts only the
final ciphertext and reads the real score.

1. Public preprocessing folds the network to Q,ell,c0, chooses A, constructs
   diagonal plaintexts, and selects direct or official Lattigo BSGS lowering.
2. Client generates a fresh keypair and required evaluation keys, sends only
   evaluation keys, packs/encodes/encrypts input and sends ciphertext bytes.
3. Server computes Ax using the existing hoisted linear-transform kernel,
   rescales once and adds public ell. It drops one level of the unchanged x
   ciphertext without decryption.
4. Server multiplies x elementwise by Ax+ell, relinearizes once and rescales.
   A binary reduction over d feature blocks gives one score per example; add c0.
5. Return final ciphertext; client alone decrypts/decodes it.

The comparison uses one ciphertext multiplication, one relinearization, two
rescale levels, log2(d) reduction rotations and zero bootstraps on both sides.
The pass reduces plaintext diagonal products from d to d/2+1, not the ciphertext
multiplication depth. It can reduce input rotations/keys, but BSGS interactions
mean that the fastest candidate can use more rotations/keys than a tuned
baseline. Actual d32 direct and BSGS3 tradeoffs are preserved in RESULTS.md.

Backend: pinned [Lattigo v6.2.0](https://github.com/tuneinsight/lattigo/tree/v6.2.0),
including its official hoisted direct and BSGS transforms. [Halevi–Shoup's
linear-transform work](https://eprint.iacr.org/2018/244) is prior art for these
kernels. Their performance figures were not reproduced as cross-library results.

## Correctness and numerical requirements

The original network expansion and all dyadic stored Q,ell,c0 are checked
exactly. The orientation introduces zero coefficient error. LDL/spectral
comparators have separate exact rational residual bounds on the entire box,
including affine completion shifts and constants. No small eigenvalues are
dropped. For a residual polynomial x^T R x+s^Tx+t, on max|xi|<=R0 the absolute
error is at most R0^2 sum|Rij|+R0 sum|si|+|t|.

All methods must give finite real final output with absolute error <=1e-3
against the original neural computation. This is the same numerical/model
quality requirement, not a relaxed threshold for the candidate. The archived
integer model remains unnormalized; synthetic model normalization is fixed
before benchmarking and applies identically to every method.

For input approximation ||e||infinity<=eta, the exact quadratic's perturbation
is bounded by2R0*sum|Qij|*eta +sum|Qij|*eta^2 +||ell||1*eta. Orientation preserves
sum|Aij|=sum|Qij|, so it does not increase this global coefficient-mass bound.
In the archived model the bound is3296eta+1483eta^2. A conditional encrypted
bound can add the actual linear-transform, product, rescale and rotation errors:
if |ei|<=eta, |linear-error_i|<=zeta and product/reduction errors are bounded,
apply |(xi+ei)(gi+di)-xigi|<=|xi||di|+|gi||ei|+|ei||di| and sum over i.
Those intermediate noise bounds have not been established as a uniform-box
worst-case CKKS theorem here. Coefficient correctness and empirical encrypted
accuracy are distinct evidence.

Encrypted validation uses frozen boundary patterns, signed basis points,
all256 corners for d8, uniform dyadic box samples, reserved model seeds and40
reserved input/key seeds. It checks all decoded slots and compares final scores
with the original network. It reports empirical maxima/zero failures, not an
unjustified IID confidence bound or proof for every real input in the box.

## Parameters and security assessment

Standard CKKS N8192; Q bit sizes58,40,40; P bit size60; default scale2^40;
uniform ternary secret; default truncated Gaussian noise sigma3.2. The exact
Standard primes are

```
Q = [288230376151760897,1099511480321,1099511922689]
P = [1152921504606830593]
log2(QP) = 198.00000019348326
```

The target is classical Category128, assessed against the published
[Security Guidelines](https://cic.iacr.org/p/1/4/26), uniform-ternary dimension8192
limit214 bits including P, with the distributions above. The audit independently
checked prime validity/congruences and logQP. No new security proof or fresh
lattice-estimator execution is claimed. The final50-bit earlier prototype
parameters are not the final benchmark parameters: P60 was chosen before final
evaluation for precision. Q0/scale is approximately2^18; the archived global
|f| bound1814 fits comfortably inside the centered final modulus range.

The CI comparator uses the same bit-size configuration and target, different
NTT primes,8192 independent coefficients and8192 real slots at matched batch.
Its larger ambient standard-ring conversion does not double its secret
dimension/security. See [Kim–Song](https://eprint.iacr.org/2018/952) and
notes/security.md for the conjugate-invariant RLWE assumption and table mapping.

CKKS privacy here is passive/IND-CPA under its assumptions. No client decryption
feedback, result-release oracle, malicious-server integrity or circuit privacy
claim is added. These limits follow the selected threat model and official
[Lattigo security guidance](https://github.com/tuneinsight/lattigo/blob/v6.2.0/SECURITY.md).

## Joint plan selection and cost model

The construction chooses the algebraic representative; the scheduler then
chooses direct/BSGS and evaluation keys for a deployment horizon H. Measured
objective: public preparation + key generation/encoding + key distribution +
warmup + H*(packing+encryption+communication+server+return+decryption), with
evaluation-key and RSS budgets. One-time and per-inference terms are separate.
`tools/select_plan.py` minimizes this explicit objective using development
profiles, never hidden client work or an LLM-generated parameter guess.

The profiles use actual HTTP key distribution and ciphertext transport on local
TCP. For WAN estimates, key/input/output bytes and a declared bandwidth/latency
model are needed; the planner's optional bandwidth term is explicitly a model.
In the combined-role HTTP profile, d32 direct amortizes its larger setup after
seven inferences versus the tuned baseline. The separate-process lifecycle is
reported separately in RESULTS.md; BSGS3 improves time, keys and server RSS from
the first measured inference in that architecture. Under a network key-bandwidth constraint that threshold
can grow. The d8 BSGS3 plan is slower and is not selected for fast inference.

## Earlier real product theorem and competing routes

For symmetric real Q with positive/negative inertia(p,m), the minimum number
of products of real linear forms in x^TQx=sum Lj(x)Rj(x) is max(p,m) when
additions/constants are free. The intersection of k left kernels has dimension
at least d-k; it cannot intersect a p-dimensional positive-definite subspace
or an m-dimensional negative-definite subspace except in zero, proving
k>=p,m. Pair signed eigen-squares as u^2-v^2=(u+v)(u-v) for achievability.
The archived model has exact inertia(3,5), verified by rational LDL.

That theorem does not optimize ciphertext operations: at B128 the64 original
hidden squares occupy two ciphertexts, and the fused scalar contraction has one
ciphertext product. It is not a64/5 speedup. The feature-layout alternative
uses sparse pivoted LDL and support-overlap matching to pair positive and
negative modes as Re((u+iv)^2); it reduces public linear coefficients and
nonlinear products at high batch. The client may discard the imaginary channel
only after final decryption. It was implemented and executed, but loses to the
packed oriented rule at the primary B128 workload. Its conditioning and
conjugate-invariant-capacity caveats are recorded rather than suppressed.

## Contribution and prior work

The measured contribution is a reusable terminal-scalar equivalence rewrite
with a proven generic cyclic-support optimum, implemented before existing
hoisted/BSGS lowering and combined with an explicit setup/key-memory plan
selector. The elementary identity, quadratic fusion, inertia, LDL/eigen,
complex packing, BSGS and hoisting are not new inventions here.

[Orion](https://arxiv.org/abs/2311.03470),
[HEIR](https://arxiv.org/abs/2508.11095), Fhelipe, HECO and Coyote already optimize
layouts, kernels and schedules. Our exact comparison is the matrix
representative changed under a terminal scalar consumer, rather than a claim to
invent general joint compilation. The benchmark uses strong implementations of
relevant kernels in one backend; it does not reproduce entire compiler suites.

[KeyMemRT](https://arxiv.org/abs/2601.18445) and its
[compiler](https://github.com/eymay/KeyMemRT-Compiler) /
[runtime](https://github.com/eymay/KeyMemRT) are real prior work on key liveness,
loads, clearing and prefetch. Eliminating keys through a computation rewrite is
distinct from managing lifetimes of an unchanged key set.
[FHE-Agent](https://arxiv.org/abs/2511.18653) is a real Orion-based configuration
preprint; its reported baseline is best-of-ten one-shot LLM configurations and
its FHE timing excludes key generation. No official FHE-Agent implementation was
identified in the accessed primary manuscript. Its performance was not locally
reproduced. An instruction for an LLM to search settings is not our method.

The independent contribution review in notes/prior_art.md documents exact
comparisons and searches. No located primary comparator stated this precise
cyclic scalar-consumer rule, but absence of a search hit is not proof of
historical novelty. Publication-level first/new claims remain open.
