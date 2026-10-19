# Security and numerical assurance audit

Historical snapshot status: then-current `bench/packed.go` and `PROTOCOL.md` independently inspected; parameters checked against published guidance and completed CI artifacts. Last checked 2026-10-19. Numerical assurance is empirical CKKS accuracy, with separate global algebraic residual bounds.

## Frozen threat model

The client owns and retains the secret key. It encrypts its feature vectors locally and sends ciphertexts and the required evaluation key to an honest-but-curious server. Public model coefficients, compilation decisions, dimensions, batching, parameters, message lengths, evaluation-key inventory, operation trace and timing are observable. The server returns a final ciphertext. Only the client decrypts it. No plaintext result, numerical error, failure indication depending on decrypted data, or intermediate decryption is returned to the server.

The intended guarantee is input confidentiality against this passive server under the usual RLWE and evaluation-key assumptions. It does not hide the public model or metadata, establish circuit privacy, provide malicious-server correctness, or protect against host compromise. A benchmark may run the two roles in one process, but its evaluator must receive neither a secret key nor a decryptor. The process boundary in such a benchmark is a protocol simulation, not an isolation mechanism.

[Lattigo's version-pinned security policy](https://github.com/tuneinsight/lattigo/blob/v6.2.0/SECURITY.md) states its default IND-CPA guarantee and warns that CKKS plaintext decryption results must remain with secret-key holders. Accordingly, client-side benchmark correctness logs are test artifacts; they must not be fed back into a deployed server protocol. The documented dense-to-sparse bootstrapping-key problem affected v5.0.0–v6.1.0 and was fixed in v6.1.1. This experiment uses v6.2.0 without bootstrapping or sparse-key encapsulation.

## Frozen parameters for the two-rescale pipeline

```go
ckks.ParametersLiteral{
    LogN: 13,
    LogQ: []int{58, 40, 40},
    LogP: []int{60},
    LogDefaultScale: 40,
    // Omitted Xs/Xe use the version-pinned defaults below.
    // RingType selects a separately reported Standard/ConjugateInvariant arm.
}
```

This gives ring degree 8192, two available rescale levels, nominal log2(Q)=138, log2(P)=60 and log2(QP)=198. Standard has 4096 complex SIMD slots; ConjugateInvariant has 8192 real slots. Actual application batch is 128 in both arms. Every run logs its generated Q/P primes and logQP because `LogQ` and `LogP` describe prime-generation targets. Each arm's baseline and candidate use the identical `ckks.Parameters` instance. Their precision requirement is unchanged. The final P=60 replaces a development P=50 setting to improve key-switch precision; no development timing is final evidence.

[Lattigo v6.2.0 `core/rlwe/security.go`](https://github.com/tuneinsight/lattigo/blob/v6.2.0/core/rlwe/security.go) defines the default secret as `ring.Ternary{P: 2/3.0}` and error as `ring.DiscreteGaussian{Sigma: 3.2, Bound: 19.2}`. [The ternary sampler](https://github.com/tuneinsight/lattigo/blob/v6.2.0/ring/sampler_ternary.go) uses this as the dense probability-based distribution; no fixed low Hamming weight is requested. Do not replace it with a sparse `H` secret while keeping this assessment.

The published [Security Guidelines for Implementing Homomorphic Encryption](https://cic.iacr.org/p/1/4/26/pdf), Table 5.1 and Table 5.2, bounds the **combined modulus QP** for hybrid key switching. For N=8192, uniform ternary secrets and error standard deviation 3.19, its classical Category 128 limit is log2(QP)=214. Both actual moduli (Standard 198.00000019348326 bits; CI 198.00000210679104 bits) are below that limit; sigma 3.2 is slightly larger. The table uses Lattice Estimator commit `8f1ff7e` and the classical `RC.MATZOV` cost model. This is a table-based assessment against known attacks under those assumptions, not a proof of 128-bit security or a newly reproduced estimator run. Older draft tables gave different limits and must not be substituted silently.

The constructor checks algebraic validity (primes, NTT compatibility and nonzero distributions), not cryptographic security. Verify both constructor success and the external bound. The machine's Apple M4 architecture changes performance, not the parameter assessment. Evaluation keys are included through QP. Different auxiliary moduli, bootstrap parameters, secret distributions or a different ring require a new assessment.

## Conjugate-invariant degree and security interpretation

[Lattigo v6.2.0 `ring/ring.go`](https://github.com/tuneinsight/lattigo/blob/v6.2.0/ring/ring.go) defines Standard as Z[X]/(X^N+1) and ConjugateInvariant as Z[X+X^-1]/(X^(2N)+1), represented by **N independent coefficients**. Its `StandardRing()` conversion doubles the ambient polynomial degree. This does not turn an N=8192 CI secret into an independent dimension-16384 secret. The parameter generator requires CI primes congruent to 1 modulo 4N, versus 2N in Standard. Consequently the arms have different exact generated primes; they share bit-size settings and security targets, and each arm uses identical primes for all baseline/candidate settings.

[Kim and Song, *Approximate Homomorphic Encryption over the Conjugate-invariant Ring*](https://eprint.iacr.org/2018/952), Section 3.2, identifies a CI element inside ambient degree n with a vector of length n/2 and analyzes ordinary lattice attacks in that dimension. Here n=2N, hence the attack-estimation dimension is **8192 for either arm**. Their paper supplies a hardness foundation over the real subfield and argues that known attacks do not improve on this ordinary-LWE comparison. This investigation relies on that CI-RLWE assumption/known-attack correspondence and applies the 8192-dimensional QP=214 table screen, never the 16384-dimensional limit. It does not reproduce the paper's reduction or establish a new concrete CI security theorem. CI is evaluated as a separate established scheme arm; it is not a source of a doubled security claim.

## Scale, depth and headroom obligations

The implemented circuit evaluates z=A*x, rescales, adds the public affine vector, multiplies x*z with one relinearization, rescales, performs the cyclic reduction and adds the constant. Input scale is 2^40. The linear-transform plaintext scale is the actual last Q prime, so its first rescale returns the input scale; the second rescale follows the actual remaining Q prime. No unchecked scale reassignment appears. Public affine plaintexts have the corresponding level/default scale; the final scalar addition uses the evaluator's scale handling. The circuit consumes two rescale levels, one variable-variable multiplication depth, one relinearization and zero bootstraps.

The archived original model has the coefficient-based bound |f(x)|<=1814 on the input box, independently recomputed from the frozen JSON. Orientation preserves the entrywise absolute coefficient sum: each symmetric pair (q,q) becomes (2q,0), while the half-cycle pair is retained. Therefore the cleartext row terms and every subset accumulated during cyclic reduction are also bounded by 1814 on this box; synthetic model bounds are below 1. At the last level the nominal ratio Q0/scale is about 2^18 (centered representation needs a factor-two margin), comfortably above these bounds. This cleartext headroom check is not a worst-case CKKS noise theorem. The final method uses real model coefficients. Its Standard arm nevertheless records imaginary output error; its CI arm operates over real slots.

## Explicit numerical requirements and propagation

Application acceptance should be frozen as max absolute error <=1e-3 per real output, compared with the original public quadratic model in float64 (or higher precision for a cancellation stress case). Model algebra error and CKKS error must be reported separately. If the class consists of normalized inputs with ||x||_infinity<=1, preserve this domain in both generation and holdout tests. Do not infer a global guarantee from Gaussian or uniform random samples alone.

For the final oriented representation f(x)=x^T A x+b^T x+c0, with sym(A)=Q, let R=Q-sym(A). The original-model deviation on the whole input box obeys

    |x^T R x| <= sum_i,k |R_ik|.

Use the analogous sum of absolute residual coefficients for affine terms. Exact rational reconstruction of the stored binary64 coefficients establishes this global **algebraic transformation** bound independently of sampled tests. It does not bound randomized CKKS arithmetic noise. For input perturbation ||e||_infinity<=eta, the original-model change is bounded by (2*sum|Q|+||b||_1)*eta+sum|Q|*eta^2. Additional linear-transform encoding, rescale, product and key-switch errors require their own primitive bounds to obtain a domain-wide encrypted theorem; this report does not claim such a theorem.

The implemented benchmark validates every decoded lane, including replicated lanes, against the original public score. Its inputs include zero, all-ones, all-minus-ones, alternating corners, one coordinate vector and fixed-seed dyadic uniform-box samples. Separate held-out model seeds test compilation generalization. Input fixture seeds currently depend on dimension, so same-dimension model runs reuse their input set; independent held-out inputs must be reported separately. Lattigo key generation and public-key encryption use their own CSPRNG, not `math/rand`, which appears only in fixture generation. Across M independent random test inputs with zero failures, the one-sided 95% binomial bound on the failure probability for that tested input distribution is 1-0.05^(1/M). Replicated lanes and repeated encryptions do not create M independent input samples. This does not certify a worst-case input-box guarantee.

## Independent final harness checklist

- Verify the exact serialized Q/P, N, ring, distributions, scales, levels and Lattigo module version.
- Trace secret-key/decryptor creation and every call to decryption. Require all evaluator functions to be secret-free and no intermediate decryption.
- Trace random generators: cryptographic key/encryption randomness must use Lattigo's CSPRNG, not the deterministic model RNG.
- Include preparation, key generation, serialized evaluation-key bytes/distribution, encryption, all request/response bytes, server evaluation, and final client decryption. Key distribution can be a reported byte count plus separately stated network model if no network is measured.
- Separate one-time compilation/key setup from recurrent inference; calculate break-even counts using the complete extra setup cost and observed recurrent savings.
- Equalize batch 128, original inputs, original model, hardware, parameters and permitted error. Both packed variants send one input ciphertext and return one output ciphertext. CI's extra slots replicate the same examples; they do not increase the application batch.
- Check real error against the original model as well as compiled cleartext values. Check complex magnitudes for headroom. If only sampled encrypted error is established, label the assurance accordingly.
- Count ciphertext multiplications, public multiplications, rotations, relinearizations, rescales, depth and bootstraps from actual execution. Report serialized key size and process peak RSS separately from Go heap telemetry.

## Audit observations from the final executable/artifacts

`packedServerEvaluate` receives parameters, evaluator, public encoded model data, ciphertext and public dimensions; it receives neither `sk` nor a decryptor. Its body contains only homomorphic linear transformation, rescale, additions, level dropping, one `MulRelinNew`, rotations and return. The only decrypt call in `packed.go` is after final output deserialization at the client. No intermediate decryption or client assistance was found. Public evaluation keys are genuinely serialized/deserialized into a separate server key set. Key inventory includes both matrix and scalar-reduction rotations, and excludes the identity.

**Historical scope:** the following paragraphs describe the earlier `final_ci` and `final_standard` in-process snapshots, not the selected later HTTP or separate-process evidence. Those earlier directories are not included in the release subset. Current selected evidence is `artifacts/final_v2`, `artifacts/confirmation`, and `artifacts/separated_confirmation_final`; the latter records distinct child/server PIDs, measured HTTP exchange and isolated child server peak RSS. See `docs/NUMERICAL_EVIDENCE.md`.

The local chain genuinely encodes, freshly encrypts, serializes/deserializes request and response, evaluates, and finally decrypts/decodes. Disk evidence writes and plaintext checking occur outside inference timing and are disclosed. Key setup/warmup are reported separately. No real network exchange is timed, and no network-latency claim follows from this in-process execution. Stage heap snapshots are accurately labeled as something other than peak RSS. The complete-process RSS includes simulated client and server roles and must not be described as an isolated server measurement.

All seven completed `artifacts/final_ci/*/ci/run.json` files inspected have batch 128, ring degree 8192, identical Q=[288230376152137729,1099511922689,1099512938497], P=[1152921504606748673], and logQP=198.00000210679104. Their largest reported real error across methods, lanes and repetitions is 1.054975123793156e-5 (archived model), below 1e-3. Other CI model maxima are below 2.105e-7.

All seven `artifacts/final_standard/*/standard/run.json` files use batch 128, ring degree 8192, Q=[288230376151760897,1099511480321,1099511922689], P=[1152921504606830593], and logQP=198.00000019348326. Their largest reported real error is 3.176100221935485e-6 (archived model); other model maxima are below 5.888e-8. These observations establish tested accuracy only. Independent held-out-input evidence and RSS results are audited separately as they complete.

An independent Python check recomputed every final-repeat published first-block output against the original W/v/bias square network (not just Q), totaling 5376 comparisons per arm. Maximum discrepancy was 1.9368102925909625e-6 for Standard and 5.583302140621527e-6 for CI; the neural and polynomial clear references agreed exactly on these dyadic fixtures. These checks reuse the fixed inputs and do not count as additional independent input samples. A deterministic 64-bit Miller–Rabin check also verified primality of every logged Q/P prime, the required 2N/4N congruences, and the recomputed logQP totals. The library's constructor had already checked NTT compatibility.

Final assessment: the **final primary parameters pass the published Category 128 table check** with the ring assumptions stated above. The inspected implementation follows the frozen passive protocol. Algebraic correctness has a global coefficient certificate; CKKS numerical acceptance remains empirical over the reported fixtures and fresh encryption trials.
