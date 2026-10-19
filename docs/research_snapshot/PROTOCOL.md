# Frozen evaluation protocol

Date: 19 October 2026. Models were frozen before final encrypted evaluation in
`artifacts/models.json`; SHA256
`985f69a2c3fe52bc6a565a6b21f11c7fdf2494850083ae8cecd477c63d0df7c3`.
Development smoke runs selected P=60 instead of P=50 to improve key-switch
precision. Smoke measurements are separate from final measurements.

## Computation and application

Public one-hidden-layer square-activation scalar scoring networks
`f(x) = normalization * (sum_i v_i (w_i^T x + b_i)^2 + c)`.
Equivalent `x^T Q x + linear^T x + constant` is compiled offline. No
activation approximation, rank truncation, retraining, or model changes.
Applications are batched private quadratic scores / degree-two neural scores.
This investigation does not claim task accuracy on a trained image classifier.
Dimensions 8, 16, 32; seven fixed models including archived integer model,
synthetic indefinite and positive definite cases, and separately seeded held-out
models. Primary fixed batch is 128 input examples. Real input box [-1,1]^d.

## Threat model and security

One client holds secret key and inputs. Server sees public model, dimensions,
batch, ciphertext and evaluation-key sizes and all schedule/timing metadata.
Server is passive/honest-but-curious. Only client decrypts the final score.
Client outputs, numerical errors and acceptance decisions are never returned to
the server as a decryption oracle. No intermediate decryption or assistance.
Benchmark input fixtures are intentionally published synthetic evidence.

Lattigo v6.2.0 CKKS; N=8192; Q bit sizes [58,40,40]; P bit sizes [60]; default
uniform ternary secret, default Gaussian sigma=3.2; scale 2^40; classical
Category 128 target, assessed against the published 2025 security guidelines'
uniform-ternary QP bound 214 bits at N=8192. Actual generated primes and logQP
are logged. logQP approximately 198, including P. Standard and conjugate
invariant rings are measured as separate arms at matched batch, Q, P, N,
quality and hardware. CI has more slots; fixed batch remains 128.

## Numerical requirements

Each final decrypted output must differ from the original floating-point neural
reference by at most 1e-3 absolute. Same requirement for every baseline.
Compilation residuals are bounded globally on the input box by exact rational
coefficient reconstruction. Encrypted error validation checks every output
lane on fixed dyadic inputs including zero, all-ones, all-minus-ones, alternating
corners and a unit coordinate, plus independent random samples. Additional
held-out input validation must be reported separately. This is empirical CKKS
accuracy assurance; a domain-wide CKKS worst-case noise theorem is not claimed.
Archived model |f| <= 1814; Q0/scale is approximately 2^18, so no final
plaintext wrap is permitted. Both methods have two rescale levels, one
variable-variable multiplication depth and zero bootstraps.

## Baselines and measurements

Packed symmetric-Q diagonal contraction, direct and official Lattigo BSGS
ratios 0 and 1, all with hoisting provided by the official implementation.
Candidate oriented-Q contraction uses exactly the same six-setting harness,
parameters, input packing, data, codec and client/server boundaries.
All baseline settings are retained and the fastest baseline is used; no
comparison against only an untuned baseline. A second real-valued CI arm tests
whether doubled real slot availability invalidates gains.

Apple M4 MacBook Air, 16 GiB RAM, Darwin arm64, Go 1.25.3, GOMAXPROCS=1.
Nine repetitions per setting after one explicit, separately charged warmup.
Report raw samples, medians and variability; subsequent confirmation runs keep
same frozen protocol. Full local chain includes packing/encoding, fresh public
key encryption, binary input serialization/deserialization, server evaluation,
binary output serialization/deserialization, final client decrypt/decode.
Key generation, model encoding, key codec/distribution bytes and warmup are
one-time costs. Real network latency is unmeasured: byte counts and explicit
bandwidth/latency models are used rather than invented network timings.
Evidence file I/O and plaintext verification are outside inference timing and
disclosed. Peak RSS requires one method per fresh process with /usr/bin/time -l;
Go heap stage snapshots are auxiliary, not a substitute for RSS.

No secret key is persisted. Ciphertext evidence is genuine but only the live
client can decrypt it; published references and decoded outputs permit checking
the numerical claims, and fresh reruns recreate the full experiment.

## Contribution boundary

The proposed rule exploits a terminal scalar consumer: select A with
sym(A)=Q before ordinary linear-transform lowering. Its cyclic construction
removes one of every +/- diagonal pair. The real quadratic product theorem,
eigendecomposition, LDL, complex packing, BSGS, hoisting and key lifetime
management are existing ingredients. Do not claim their invention, a universal
FHE speedup, a publication-priority result, or a 64/5 runtime improvement.
