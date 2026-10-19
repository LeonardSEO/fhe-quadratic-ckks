# Third-party attribution

Dependencies retain their original licenses. They are fetched through pinned
manifests and are not relicensed by the author's MIT or CC BY grant. Copies of
Go dependency license/notice files found in the verified module cache are in
`LICENSES/third-party/`.

| Go module | Version | Upstream license files |
|---|---|---|
| `github.com/ALTree/bigfloat` | `v0.2.0` | LICENSE.md |
| `github.com/davecgh/go-spew` | `v1.1.1` | LICENSE |
| `github.com/google/go-cmp` | `v0.7.0` | LICENSE |
| `github.com/kr/text` | `v0.2.0` | License |
| `github.com/pmezard/go-difflib` | `v1.0.0` | LICENSE |
| `github.com/stretchr/testify` | `v1.10.0` | LICENSE |
| `golang.org/x/crypto` | `v0.45.0` | LICENSE |
| `golang.org/x/exp` | `v0.0.0-20250506013437-ce4c2cf36ca6` | LICENSE |
| `golang.org/x/sys` | `v0.38.0` | LICENSE |
| `gopkg.in/yaml.v3` | `v3.0.1` | LICENSE, NOTICE |

Lean 4 is Apache-2.0; mathlib is Apache-2.0. The mathlib Git commit and transitive
Lean dependency commits are locked in `lean/lake-manifest.json`; each fetched
repository retains its own license file. No compiled Lean runtime or mathlib
cache is included in the source release. NumPy is BSD-3-Clause; Matplotlib uses
its own PSF-based license. Their wheels and source are not redistributed here.
Tectonic is MIT licensed and is a separate build tool. TeX packages retain their
own upstream licenses and are not part of this artifact's code license.

Primary publications are cited in `references.bib`; no full third-party article
is redistributed. GitHub Actions are referenced at pinned commits in CI and
retain their own licenses. Generated public research ciphertexts and evaluation
keys are evidence produced by the implementation, not secret keys.

CC BY 4.0 legal text was obtained from Creative Commons’ official repository:
https://github.com/creativecommons/creativecommons.org/blob/main/docroot/legalcode/by_4.0.txt
