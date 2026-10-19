# PDF build and visual verification

The actual paper was compiled on 2026-10-19 with bundled Tectonic 0.17.0, including its BibTeX 0.99d pass. The successful final build ran TeX, BibTeX, a cross-reference rerun, and xdvipdfmx. The PDF is nine A4 pages with all fonts embedded.

The final log has no unresolved references/citations, font substitutions, overfull boxes, or engine warnings. The bibliography database is `references.bib`, style `plainnat.bst`.

All nine pages were rendered through Poppler `pdftoppm` at 85 dpi and visually inspected as a contact sheet. The mathematical proof page (page 2), primary table/figure page (page 6), and bibliography page (page 9) were additionally rendered at 130/130/115 dpi and inspected individually. No clipped equations, table overflow, missing figures, broken glyphs, unresolved references, or bibliography placeholders were observed. Page numbering, theorem references and the generated plot/table layout were checked. Text extraction also found no `??`, build-status placeholder, or empty DOI identifier.

During preparation three overfull boxes were repaired by rounding the displayed modulus bit count while retaining exact primes in the paper, referencing the locked mathlib revision instead of an inline unbreakable hash, and making the verification-document path breakable. AMS font definition files were supplied before the final build; the released PDF contains the actual mathematical fonts rather than substitution glyphs.

## Build environment caveat

The installed executable could not fetch its default TeX resource bundle through its own network client. The preparation used a local directory containing required resources from the official Tectonic bundle `tlextras-2022.0r0.tar`, obtained by bounded HTTP range requests. The full 2.88 GB bundle was not downloaded, and the temporary resource subset is not part of the research artifact.

The equivalent final build command is:

```sh
SOURCE_DATE_EPOCH=$(cat paper.build_epoch) tectonic --bundle "$TECTONIC_BUNDLE" --keep-logs --keep-intermediates paper.tex
```

`TECTONIC_BUNDLE` is an optional local resource directory in that workaround. On a clean machine with working Tectonic bundle access, `tectonic paper.tex` is sufficient. A normal TeX Live build can instead run LaTeX, BibTeX, and two LaTeX passes. This environment workaround does not change manuscript source or create a substitute PDF.

The actual reserved DOI `10.5281/zenodo.23474352` was inserted after creation of the manual Zenodo draft. The PDF was recompiled successfully; its DOI-bearing page 8 was rendered at 130 dpi and visually checked. The identifier is readable, the hyperlink targets `https://doi.org/10.5281/zenodo.23474352`, and the paragraph explicitly states that reservation alone is not publication. This validation does not assert a published deposit. The document remains nine A4 pages, with no diagnostics, unresolved references or clipping.

Hashes of this DOI-bearing manuscript state:

- `paper.tex`: `155d4f08521623ba02bf76ba2d7018fd73fc0842e8dcfd038ccb1579c246a723`
- `paper.pdf`: `ac1a63ee887431e3891044d9766d83a59d92c58a4236f340ae97949d62a33f02`

If other paper content changes, recompile and check the new PDF and update these hashes before archiving.

## Deterministic PDF timestamp

`paper.build_epoch` pins `SOURCE_DATE_EPOCH=1792368000`, exactly 2026-10-19 00:00:00 UTC. The paper phase of `scripts/check_release.py` loads this file and sets the build environment. Two consecutive actual Tectonic builds produced byte-identical PDFs, confirmed with SHA256 and `cmp`; both gave the PDF hash recorded above. This establishes repeatability for the pinned local Tectonic/resource environment, not byte identity across different LaTeX engines or package distributions. The final DOI-bearing page 8 was rendered again at 130 dpi and inspected after this change.
