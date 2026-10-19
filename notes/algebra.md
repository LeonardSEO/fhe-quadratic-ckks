# Independent algebra audit and cyclic orientation candidate

Checked 2026-10-19. Scope: `THEORY.md` section 3, `core.py` quadratic compiler, and archived `quadratic_model.npz`. This note independently reconstructs the proofs. It does not certify encrypted runtime, parameter security, or scientific priority.

## 1. Precise theorem and independent proof

Let Q be an n by n real symmetric matrix, with positive, negative and zero inertia p,m,z. Restrict to exact identities over the reals

    x^T Q x = sum_{j=1}^k (u_j^T x)(v_j^T x), for all x in R^n.

Each factor is a homogeneous real linear form. Public constants and additions are free, and only products of two variable-dependent scalars are counted. Then k_min=max(p,m), including singular Q and Q=0.

Lower bound: K=intersection_j ker(u_j^T) has dimension >= n-k, and the quadratic form is zero on K. Let P be any p-dimensional positive-definite subspace, obtained for example from positive eigenvectors. P intersects K only in {0}; otherwise a nonzero vector in the intersection has both positive and zero quadratic value. Therefore p+n-k <= n, so k>=p. The same argument with a negative-definite subspace gives k>=m. This does not require the u_j to be independent.

Upper bound: spectral decomposition gives sum_{i=1}^p a_i(x)^2 - sum_{i=1}^m b_i(x)^2 with square roots of absolute eigenvalues absorbed in the forms. Pair min(p,m) opposite-sign squares as (a_i+b_i)(a_i-b_i); each unpaired square costs one product. The total is max(p,m).

Verdict: proof established in this scope. The workspace argument is correct. This is an elementary consequence of real quadratic-form decomposition, not established here as a new mathematical theorem. It is not a minimum number of ciphertext multiplications, rotations, or arithmetic instructions. It does not transfer unchanged to finite fields, complex factors, or noncommutative block products. Over C, for example, two same-sign squares can pair as (a+ib)(a-ib), invalidating the real-inertia lower bound.

For uniform homogeneous-quadratic approximation on ||x||_2<=R, tolerance epsilon, the workspace eigenvalue-threshold statement is also correct: discard eigenvalues of magnitude <=epsilon/R^2. The remaining minimum is max(number of eigenvalues >epsilon/R^2, number <-epsilon/R^2). The lower bound follows because any approximant within spectral norm epsilon/R^2 stays definite on the corresponding strictly separated eigenspaces. This statement is about balls. For the box [-1,1]^d, radius sqrt(d) gives a sufficient threshold epsilon/d, not a box-optimal theorem.

## 2. Exact check of the reported 64 to 5 model

Loaded archived W,v,b,U,V from `quadratic_model.npz`. W is 64 by 8, and W,v,b are integers. Reconstructed Q_ij=sum_h W_hi v_h W_hj with Python integers. Independently implemented LDL^T using `fractions.Fraction` and checked every entry of L D L^T equals Q. The nonzero diagonal entries are

    -15
    -28/5
    1034/3
    -404970/3619
    9283/161988
    -245975847/46415
    -4282614277/81991949
    19655301980863/12847842831

There are 3 positive and 5 negative entries. Thus the 5-product optimum is exact for this 8-variable polynomial in the stated real cost model. This only shows that a representation of width 64 can be reduced, not that the 64 original squares were necessary for the polynomial.

The exact affine part of the original model is

    l = [-24,-46,72,22,20,86,48,-12]
    c = -1.

The archived floating-point U,V factors are not exact real factors of Q. Treating their binary64 coefficients as exact rational numbers gives the rigorous residual certificate

    sum_ij |Q_ij - sym(U^T V)_ij|
      = 118887319893811504123 / 81129638414606681695789005144064
      < 1.465400e-12.

Because |x_i x_j|<=1 on [-1,1]^8, this rational number bounds the polynomial error uniformly over the whole box before encrypted arithmetic. The archived l,c have zero coefficient error. This bound is stronger evidence than finite random tests but does not include CKKS error. A separate binary64 test on all 256 corners gave maximum difference 6.821210263296962e-13. This corner experiment is numerical and by itself would not prove a uniform quadratic error bound.

Additional exact norms: sum_ij|Q_ij|=1483; ||l||_1=330. The floating spectral norm of Q is approximately176.095558. Largest absolute hidden activation over the box is at most20.

## 3. Collapse a scalar square network, then optimize its execution representation

For public coefficients and a single hidden square activation layer,

    f(x)=v^T (W x+b)^{circ 2}+c0
        =x^T Q x+l^T x+c,
    Q=W^T diag(v) W,
    l=2 W^T diag(v) b,
    c=v^T b^{circ 2}+c0.

This exact collapse should be a baseline, since evaluating the width-H network is not the strongest scalar-polynomial implementation. No claim is made that this collapse is novel. For quantized/dyadic coefficients it can be performed exactly using integer/rational arithmetic before CKKS encoding.

Important freedom: scalar reduction observes only sym(A). Every matrix A with

    A+A^T=2Q

satisfies x^T A x=x^T Q x. Therefore one may optimize A=Q+K over skew-symmetric K, rather than insisting that the evaluated matrix is Q. A compiler can use this freedom to reduce packed cyclic diagonal support. The internal vector Ax need not equal Qx; only the final scalar is observable.

Also fold the affine part before the only ciphertext product:

    z=A x+l,
    f(x)=sum_i x_i z_i+c.

This uses an addition of public l rather than a separate encrypted linear dot product. It is correct only when all components are reduced to the scalar output; an application exposing Ax or intermediate features cannot use the same rewrite.

## 4. Concrete rotation-orbit orientation rule

Use cyclic indices modulo d. Define A by:

* A_ii=Q_ii.
* For r=1,...,floor((d-1)/2), set A_{i,i+r}=2 Q_{i,i+r}.
* If d is even, set A_{i,i+d/2}=Q_{i,i+d/2}.
* Set all remaining entries to zero.

For every pair i!=j except the half-offset case, exactly one orientation is kept with twice its original value. The half-offset cyclic diagonal already contains both orientations and keeps their original values. Consequently A+A^T=2Q exactly.

There are k=floor(d/2)+1 active cyclic diagonals including diagonal zero. The ordinary symmetric Q has d active cyclic diagonals when dense. Direct diagonal evaluation of A x thus uses k public-vector multiplications and k-1 rotations, versus d and d-1 for Q x. Both use one variable-variable ciphertext multiplication for x circ (A x+l), one relinearization if required by the backend, the same scalar reduction, and zero bootstraps when the chosen parameter chain covers the depth.

Minimality is precise: among matrices A satisfying sym(A)=Q, a generic dense Q requires at least one active cyclic diagonal from every pair {r,d-r}; the even half-offset requires itself. A nonzero diagonal requires offset zero. Hence at least floor(d/2)+1 nonzero cyclic diagonals are necessary, and the construction attains the bound. This is a cyclic-diagonal support optimum. It is NOT a lower bound on all homomorphic rotation counts: baby-step/giant-step, hoisting, alternative layouts, complex slots and richer circuit transformations remain possible.

### Exact interleaved batch layout

Let S be the CKKS slot count and d divide S. Use full batch B=S/d, with slot i*B+b containing feature i of sample b. Rotating globally by r*B then shifts only the feature index modulo d and preserves b. Use the same input ciphertext and packing for both baselines and candidate. A log2(d) butterfly sum with rotations B,2B,...,(d/2)B reduces the feature dimension when d is a power of two; every feature lane then carries the same answer for that sample. The client can select any one feature lane. No server decryption or client preprocessing with model-specific secret data is required.

If d*B<S with zero padding, the global cyclic shift need not implement a local feature cycle. One must account for masks/two-rotation repairs or replicate a periodic layout explicitly. Those costs cannot be silently excluded. Similarly, contiguous samples and interleaved features are different layouts with different rotation behavior.

### BSGS is a required stronger baseline

For consecutive active offsets 0,...,k-1, a baby-step/giant-step schedule with baby length b uses at most

    (b-1)+(ceil(k/b)-1)

rotations for the linear map, while keeping k public-vector multiplications. Plaintext diagonals are pre-rotated to compensate giant steps. Optimize b over positive integers. For symmetric Q replace k with d. These counts do not include the common output reduction, key synthesis, or key generation.

| d | symmetric diagonals | oriented diagonals | direct matvec rotations symmetric/oriented | BSGS matvec rotations symmetric/oriented |
|---|---:|---:|---:|---:|
|8|8|5|7 / 4|4 / 3|
|16|16|9|15 / 8|6 / 4|
|32|32|17|31 / 16|10 / 7|
|64|64|33|63 / 32|14 / 10|

The number of distinct rotation evaluation keys can be smaller than the sum of matvec and reduction counts because offsets overlap. Composed keys can save memory while increasing runtime. Actual serialized key size and peak resident memory must be measured. Rotations of a common input can be hoisted by a suitable implementation; raw operation counts alone do not establish runtime improvement.

### Candidate compiler optimization formulation

This generalizes the fixed positive-offset construction to a workload-aware search space:

    minimize measured_or_calibrated_cost(schedule, key_set, A)
    subject to A+A^T=2Q, box_error<=epsilon,
               parameter/security constraints, unchanged layout/batch.

At minimum enumerate either orientation r or -r for each cyclic distance pair, then choose BSGS baby length, direct versus hoisted schedule, and a measured key-generation policy. The fixed orientation above is an executable starting rule and exact support certificate; the joint formulation is only a research method once its search and cost calibration are implemented and compared. A new mechanism/advantage must be isolated against collapse-only, symmetricQ+BSGS, and orientation-only ablations. Choosing a known component with a language model is not itself a contribution.

## 5. Error requirements that can be checked over the whole box

Let E_Q=Q-Qtilde, E_l=l-ltilde, E_c=c-ctilde be exact coefficient-conversion or approximation residuals. On [-1,1]^d,

    |f(x)-ftilde(x)| <= sum_ij |E_Qij|+||E_l||_1+|E_c|.

The right side can be evaluated exactly for dyadic/binary64 stored constants. The spectral bound d||E_Q||_2 is another sufficient bound but may be less convenient to certify. For integer/dyadic Q and the orientation rewrite, A+A^T=2Q can be exact before CKKS encoding: no eigendecomposition or irrational coefficients are needed.

An input representation perturbation ||e||_inf<=eta gives

    |f(x+e)-f(x)|
       <= (2 sum_ij|Q_ij|+||l||_1) eta
          +sum_ij|Q_ij| eta^2.

For the archived model this is 3296 eta+1483 eta^2. This follows by expanding 2x^TQe+e^TQe+l^Te and applying the box bound. The coefficient sum of the oriented A equals that of symmetric Q, because replacing q,q by 2q preserves their total absolute value. Its largest row norm can nevertheless increase, so intermediate scale/dynamic range must be checked.

For compositional encrypted arithmetic, suppose decoded input X approximates x with per-slot error eta, decoded computed Z approximates z=Ax+l with per-slot bounds zeta_i, ciphertext multiplication introduces additional decoded error nu_i, and final reduction/constant addition introduces at most rho. Then the deterministic conditional bound is

    |yhat-f(x)| <= sum_i [zeta_i + M_i eta + eta*zeta_i+nu_i]+rho,
    M_i=sum_j|A_ij|+|l_i|,

plus the independently bounded coefficient residual. Include input errors propagated through A when constructing zeta_i; do not count them twice. This formula is conditional on justified bounds for CKKS primitives. Empirically observed primitive errors or random-key trials do not establish universal bounds over every key and encryption randomness. Without a rigorous scheme/noise bound the proper claim is global compiler coefficient correctness plus empirical encrypted accuracy on predetermined inputs, not a proved uniform CKKS guarantee.

No rounding to integer output should be introduced unless the original application explicitly permits it. A scalar threshold classifier can preserve decisions when its reference margin exceeds the complete absolute error bound, but that margin needs independent dataset/domain evidence.

## 6. Independent verification commands

The bundled Python runtime used for the check was

    python3

It contains NumPy but not SymPy. The exact LDL verification used Fraction and is reproducible with this small reference (run from the workspace root):

```python
import numpy as np
from fractions import Fraction as F
z=np.load('quadratic_model.npz')
W,v,b,U,V=(z[s] for s in ['W','v','bias','U','V'])
d=W.shape[1]
Q=[[sum(int(W[h,i])*int(v[h])*int(W[h,j]) for h in range(len(v)))
    for j in range(d)] for i in range(d)]
L=[[F(i==j) for j in range(d)] for i in range(d)]
D=[]
for j in range(d):
    D.append(F(Q[j][j])-sum(L[j][k]**2*D[k] for k in range(j)))
    assert D[-1] != 0 # this particular archive has no zero pivots
    for i in range(j+1,d):
        L[i][j]=(F(Q[i][j])-sum(L[i][k]*D[k]*L[j][k]
                              for k in range(j)))/D[j]
assert all(sum(L[i][k]*D[k]*L[j][k] for k in range(d))==Q[i][j]
           for i in range(d) for j in range(d))
print(sum(t>0 for t in D),sum(t<0 for t in D))
E=[[F(Q[i][j])-sum((F(float(u[i]))*F(float(w[j]))
                   +F(float(w[i]))*F(float(u[j])))/2
                   for u,w in zip(U,V))
    for j in range(d)] for i in range(d)]
print(sum(abs(a) for row in E for a in row))
A=[[0]*d for _ in range(d)]
for i in range(d):
    A[i][i]=Q[i][i]
    for r in range(1,(d-1)//2+1):
        A[i][(i+r)%d]=2*Q[i][(i+r)%d]
    if d%2==0:
        A[i][(i+d//2)%d]=Q[i][(i+d//2)%d]
assert all(A[i][j]+A[j][i]==2*Q[i][j]
           for i in range(d) for j in range(d))
```

A generic Fraction LDL routine needs symmetric pivoting or 2x2 blocks for zero pivots. The simple script is intentionally restricted to the inspected archive; it must not silently be advertised as general inertia code.

## 7. Prior-art and contribution boundary

Primary sources inspected:

* [Halevi and Shoup, Faster Homomorphic Linear Transformations in HElib, CRYPTO2018, ePrint2018/244](https://eprint.iacr.org/2018/244). Establishes that efficient packed linear maps and evaluation-key time/memory tradeoffs are mature prior art. Its abstract reports performance and key-memory results; those are author results, not independently reproduced here.
* [Dathathri et al., EVA, PLDI2020](https://arxiv.org/abs/1912.11951) and the [official EVA implementation](https://github.com/microsoft/EVA). Compiler insertion of relinearization/rescaling and parameter management are existing components.
* [Dathathri et al., CHET](https://arxiv.org/abs/1810.00845). Layout and encrypted tensor compilation are existing work; do not compare solely to the unoptimized original width-H expression.
* [J.H.Maddocks, Restricted quadratic forms, inertia theorems, and the Schur complement,1988](https://doi.org/10.1016/0024-3795(88)90177-2). General classical inertia context; it is not cited as proof that the exact max(p,m) product formula first appeared there.

The independent prior-art agent searched CKKS quadratic forms, symmetric diagonals, skew-symmetric freedom, cyclic diagonal encryption, and upper-triangular evaluation. It did not locate a primary source explicitly applying scalar-reduction equivalence A+A^T=2Q to the above cyclic-distance folding. Absence of such a search hit is not evidence of scientific priority. Treat the orientation rule plus support-optimality theorem and its FHE implementation as a candidate contribution until closer prior art is ruled out and runtime is demonstrated against a strong BSGS/hoisted baseline.

At this writing, this note establishes exact algebra and an execution candidate. It does not establish an end-to-end encrypted speedup, general neural-network inference benefit, globally optimal FHE circuit, or publication-level novelty.

## 8. Executable implementation added during this investigation

`fhe_research/bench/packed.go` now implements the rule with Lattigo v6.2.0. The fixed main sweep includes symmetric Q, the ordinary upper-triangular representative (A_ii=Q_ii, A_ij=2Q_ij for i<j), and cyclic orientation. Each is evaluated with direct diagonals and official BSGS log-ratios 0,1,2,3. The triangular comparator is essential: it already exploits scalar symmetry, but generically retains all d cyclic diagonals. The cyclic representative places the same polynomial's terms onto floor(d/2)+1 diagonals. This is the added execution mechanism beyond merely removing the lower triangle. For even d, the chosen half-offset keeps both directions at their original values; this costs more nonzero scalar entries than a one-direction half-offset, but does not add a ciphertext/public-vector multiplication.

The default parameters were repaired before final evaluation to N8192, Q target bits [58,40,40], P target bits [60], default scale2^40. A controlled smoke experiment with P50 had errors around1.5e-5 on a normalized 8-dimensional polynomial, while P60 gave errors around3e-8 for every setting. Both are within the eventual1e-3 requirement, but P60 provides considerably more numerical margin. The total target log2(QP)=198 remains below the independently cited214 table bound. Every compared strategy must use the same final parameter instance; this note does not substitute for the security agent's assessment.

Both standard and conjugate-invariant ring modes are implemented. They hold the declared batch fixed, including periodic replication in extra CI slots. Ring-dependent prime arrays and actual slot counts are reported. Their security assumptions need the independently reviewed ring-specific assessment; one must not silently use a standard-ring table as a proof for a different number field.

The packed evaluator has no secret-key or decryptor argument. The client decrypts only the returned final ciphertext. Every timed repetition uses a fresh cryptographic encryption of the same declared benchmark inputs. A hard gate rejects nonfinite outputs or real absolute error>1e-3 against the original expanded polynomial, and successful JSON records `quality_passed:true`. The compiler's coefficient error is separate from this measured randomized CKKS error.

Input validation supports independent seeds (`FHE_PACKED_INPUT_SEED`) and a separately selected validation arm (`FHE_PACKED_VALIDATION=1`). At full standard d8 batch512 it includes all256 corners plus signed coordinates, zero and independent random dyadic box inputs. Larger dimensions use predetermined seeded corner subsets, signed coordinates, zero and random dyadic box inputs; changing seeds creates separate validation invocations. No finite test set establishes a worst-case CKKS noise guarantee.

`fhe_research/bench/transport.go` adds an actual loopback HTTP/TCP protocol arm selected with `FHE_PACKED_TRANSPORT=http`. The server receives serialized evaluation keys through `/keys`, installs them, accepts an encrypted inference request through `/infer`, and returns only a final ciphertext. Its API and handler capture no secret key/decryptor. The client measures actual roundtrip time and HTTP application bytes including headers, while separate server headers report body-read, deserialize, encrypted compute and serialize times. These headers contain no decrypted values. Key-distribution roundtrip includes key deserialization and must not be added twice in amortization. Loopback measurements are not WAN latency, and the byte counter excludes TCP/IP headers.

Reproduction from the bench directory, after building the root-owned CLI:

```sh
go build -o fhebench .
./fhebench -model all -batch 128 -repeats 9 -out ../artifacts/reproduction_codec
FHE_PACKED_TRANSPORT=http ./fhebench -model synthetic_d8h64seed20261020 -batch 128 -repeats 9 -out ../artifacts/reproduction_http
FHE_PACKED_INPUT_SEED=20271019 FHE_PACKED_VALIDATION=1 ./fhebench -model heldout_d8h96seed20261023 -batch 0 -repeats 3 -out ../artifacts/reproduction_holdout
```

`FHE_PACKED_METHOD` selects one named method for separate process-level RSS measurement. Default output retains every predetermined setting so a stronger baseline cannot be omitted after observing times. Stage snapshots of Go `HeapAlloc` are labeled snapshots; they are not process peak RSS.

The curl-assisted dependency bootstrap is `fhe_research/tools/bootstrap_go.py`. It caches pinned source archives when Go TLS is unavailable, then resolves offline. Every resulting Go h1 checksum is checked against the authenticated Lattigo archive's go.sum or the saved official checksum-database record. Archive SHA256 values are preserved in `artifacts/dependency_archive_sha256.json`. A mirror is a transport fallback, not a source of replacement package versions.

### Preliminary 15-setting encrypted comparison inspected independently

The root-produced `artifacts/final_v2/packed/aggregate.json` contains all15 predetermined settings on seven models at batch128 and final P60 parameters. Inspecting the median complete local codec chain, `oriented_direct` beats the fastest nonoriented setting on all seven: ratios approximately1.0914,1.1062,1.1513,1.2554,1.0845,1.1507,1.0616 in model-generation order. These are measured artifacts from that run, not guarantees of future timing or scientific-priority claims. The best baseline is sometimes triangular direct and sometimes symmetric BSGS3, so comparing only to symmetric direct or BSGS0 would overstate the result.

The mechanism is not simply fewer rotations. For the d32 model, fastest oriented direct uses21 total rotations and16 rotation keys, while the best symmetric BSGS3 baseline uses15 rotations and10 rotation keys. Nevertheless the oriented representative halves public diagonal terms and runs faster. Its key footprint is larger:26,745,062 versus17,305,622 serialized bytes. Its additional local setup cost is approximately6.905ms against2.161ms recurring savings, giving a measured codec-only break-even of four inferences before considering network distribution.

A horizon-aware choice repairs that tradeoff: d32 oriented BSGS3 measures9.333ms complete local chain versus10.621ms for the best baseline,15.732MB keys versus17.306MB, and30.155ms setup versus38.837ms. It improves runtime, setup and serialized keys together in that run. Thus a key-reuse horizon and memory budget belong in the execution policy; the fastest recurrent schedule need not minimize one-shot total cost. Actual HTTP key-distribution measurements are needed before replacing this local amortization calculation with a network claim.

### Separate server process and server-only memory evidence

The final optional protocol arm uses `FHE_PACKED_SERVER_PROCESS=1`. `MaybeRunServerProcess()` is called before the root CLI creates any client key/fixture. The parent execs the same binary with `--fhe-server-process`, a sanitized environment containing only `GOMAXPROCS=1`, and a typed public JSON configuration on stdin. This schema contains serialized public parameters, the real/imaginary public coefficient matrix, public affine constants, layout, and schedule. Unknown fields are rejected. The child readiness record's PID and public-configuration SHA256 are checked by the client. The child installs evaluation keys received through HTTP and accepts only ciphertext inference requests. It never executes client key generation or final decryption.

`server_process.go` reuses the same public HTTP handler and encrypted evaluator. The child's address space is distinct from the client's: server-only peak RSS comes from that PID's `Getrusage` response header and exit resource usage, excluding the client's secret key and its allocations. This is separate process execution on one benchmark machine, not a claim of separate physical hosts or protection against OS compromise. Ordinary HTTP mode retains its explicit same-process/shared-memory label.

Process startup, configuration serialization/pipe transfer, public encoding in the child, listener readiness, key HTTP transfer and shutdown are recorded. `http_setup_seconds` includes child startup/preparation; nested startup/prepare durations are subcomponents, not additional costs to sum. Client-side public transform preparation remains counted, so this arm does not hide the child's repeated encoding. Key roundtrip includes its server decode. Shutdown is a separate end-of-service cost. Go heap snapshots in this arm describe the client process; they must not substitute for the child's RSS.

A coordinated functional smoke on three predetermined d8/d16/d32 models, batch128, fresh encryption on two repetitions, passed the finite/1e-3 gate with maximum real errors2.74e-8,3.78e-8,4.39e-8. The child/server PIDs differ from client PIDs and match every response header. Exit RSS values were45.20MB,56.72MB,105.43MB for oriented direct in this functional smoke. All children were terminated and reaped. These smoke timings/memory values are not the final counterbalanced comparison; root-owned confirmation artifacts supersede them.

For example, from the bench directory:

```sh
FHE_PACKED_SERVER_PROCESS=1 FHE_PACKED_METHOD=oriented_direct ./fhebench -model synthetic_d16h128seed20261021 -batch 128 -repeats 9 -out ../artifacts/reproduction_separated
```
