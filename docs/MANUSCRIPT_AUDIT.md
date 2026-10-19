# Independent manuscript and primary-literature audit

Audited 2026-10-19. This is an internal independent-agent review of the release sources, not external peer review or publication acceptance. Scope: `paper.tex`, `references.bib`, `docs/RELATED_WORK.md`, `docs/CLAIM_MAP.md`, generated tables, the primary separate-process records, relevant evaluator and compiler source, and the linked primary literature. No manuscript edits were made by this reviewer.

## Findings and gates

1. **Documentation correction required:** at inspection, `notes/security.md` labelled itself a final audit but its old in-process observations said no network exchange was timed and that RSS could not be isolated. Those observations apply to an earlier inspected experiment. The final selected HTTP/exec-child arm genuinely has loopback transport and child exit RSS. Retain the older observations with their exact historical scope and add current evidence pointers, rather than silently treating them as an audit of later code. Its references to unselected `final_ci/` and `final_standard/` directories also need explicit historical qualification. Sent to the evidence maintainer and release lead.
2. **Lean gate remains explicit:** the inspected paper intentionally contains a build-status placeholder. It must be replaced with actual build and axiom results before publication. Source declarations are not proof-check evidence. This audit does not claim that the Lean targets have passed.
3. **PDF gate remains separate:** this source-level audit cannot certify pagination, clipped tables, references, or rendered equations. The release lead must inspect the compiled PDF after the final DOI and formal-status edits.

No mathematical or numerical discrepancy was found in the inspected central manuscript claims. This conclusion applies to the inspected source and records, and does not independently establish cryptographic security or benchmark repeatability on another machine.

## Mathematical audit

- The scalar-preservation argument follows by finite index exchange and division by two over the reals. Admissibility implies symmetry of Q, so the preservation theorem need not separately assume it.
- The orientation treats self-inverse offsets without doubling. Its canonical set is exactly offsets 0 through floor(n/2); this works at n=1 and n=2 as well as odd/even larger dimensions.
- Under the stated density (each cyclic Q diagonal has a nonzero), admissibility forces at least one of r and -r into active support. Zero lies in both S and -S. Thus n = |S union -S| <= 2|S|-1 and integrality gives floor(n/2)+1. In even dimensions the forced half-turn only strengthens the intersection estimate. The proof does not assume its conclusion or merely verify measured dimensions.
- The support optimum is correctly restricted to matrices A with A+Aᵀ=2Q. It is not a lower bound for arbitrary FHE circuits, rotations, memory, or runtime. Sparse-Q limitations and the sole-consumer precondition are explicit.
- The coefficient residual box bound and input perturbation bound are valid triangle-inequality bounds. The paper appropriately does not turn them into an unproved CKKS noise theorem.
- The max(p,m) real-product argument is valid for homogeneous real linear factors and a symmetric real form. The paper explicitly separates this cost model from packed ciphertext cost.

## Independent numerical cross-check

A short independent Python read of `artifacts/separated_confirmation_final/records.json` grouped the baseline and selected candidates by dimension/round and recomputed medians without importing the paper generator:

| Dimension | Marginal-median chain reduction | Paired median speedup | Paired range | Wins |
|---|---:|---:|---:|---:|
| 8 | 4.6449841605% | 1.0860044935 | 1.0292275667–1.1538261233 | 5/5 |
| 16 | 16.4751312944% | 1.2065206348 | 1.1798132835–1.2279038009 | 5/5 |
| 32 | 14.0220940561% | 1.1477001055 | 1.0060239238–1.1748905965 | 5/5 |

These agree with the paper's rounded values. `python3 scripts/generate_paper_data.py --check` passed (nine separated profiles, nine combined profiles, seven broad-sweep rows). `python3 scripts/check_correspondence.py --check` passed (41 exact matrix bridges and all seven exact network folds). These are newly executed source/evidence checks, not new performance measurements.

The evaluator source records complete recurring time through final decode, with plaintext validation/evidence writes outside timing as disclosed. The paper clearly separates older combined-role RSS from actual child-process maximum RSS; marginal-median ratios from paired ratios; measured loopback from modeled WAN; setup from recurring inference; and empirical zero-failure evidence from uniform numerical guarantees. The slower d8 BSGS and higher-memory/key d32 direct case remain visible.

## Primary-source related-work checks

- [Fhelipe author manuscript](https://people.csail.mit.edu/devadas/pubs/pldi24_fhelipe.pdf), first page: author list, PACMPL 8 PLDI Article 152, June 2024, 25 pages, DOI 10.1145/3656382 match. Its automatic packing, layout conversion, and bootstrap planning claims support the manuscript's distinction.
- [HECO publisher page](https://www.usenix.org/conference/usenixsecurity23/presentation/viand): authors Viand/Jattke/Haller/Hithnawi, USENIX Security 2023, pages 4715–4732 match. Its imperative-to-FHE restructuring is relevant prior compiler work.
- [Coyote author publication page](https://raghav198.github.io/): title, Malik/Sheth/Kulkarni, ASPLOS 2023, joint schedule/lane assignment and rotation-cost objective match. The primary ACM page was unavailable to this audit (403); no unsupported DOI change was made.
- [Q-gym final institutional record](https://dspace.mit.edu/entities/publication/6e43de1d-b36d-4b18-9999-53e41dbeeea6): complete author list, 2022 issue date and final DOI 10.1145/3559009.3569673 match. The [author manuscript](https://people.csail.mit.edu/baghdadi/public/papers/qgym.pdf), Section 5.2, explicitly discusses encrypted DNN expression rewrites and BGV/BFV validation, so this is relevant encrypted-equivalence prior work. That earlier PDF has placeholder dates, correctly not used as final bibliographic metadata.
- [KeyMemRT v1](https://arxiv.org/abs/2601.18445v1): title, Ünay/Franke/Woodruff and 2026-01-26 date match. Its rotation-key lifetime-management abstraction differs from eliminating keys through a changed computation. Its performance is author-reported, not locally reproduced.
- [FHE-Agent v1](https://arxiv.org/abs/2511.18653v1): authors, 2025-11-23 date and Orion-based LLM configuration scope match. The manuscript does not adopt its claims as reproduced evidence.
- [Security Guidelines publisher record](https://cic.iacr.org/p/1/4/26): title and author list match; a parameter-table assessment is correctly distinguished from a security proof and a new estimator run.

No primary-source claim of historical priority for the precise terminal-scalar cyclic-orientation pass was established by this review. The paper's restraint on priority, elementary identities, compiler integration, and unreproduced whole-system comparisons is appropriate.
