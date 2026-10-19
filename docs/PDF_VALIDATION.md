# PDF build and visual verification

The actual paper was compiled on 2026-10-19 with bundled Tectonic 0.17.0, including its BibTeX 0.99d pass. The successful final build ran TeX, BibTeX, a cross-reference rerun, and xdvipdfmx. The PDF is nine A4 pages with all fonts embedded.

The final log has no unresolved references/citations, font substitutions, overfull boxes, or engine warnings. The bibliography database is `references.bib`, style `plainnat.bst`.

All nine pages were rendered through Poppler `pdftoppm` at 85 dpi and visually inspected as a contact sheet. The mathematical proof page (page 2), primary table/figure page (page 6), and bibliography page (page 9) were additionally rendered at 130/130/115 dpi and inspected individually. No clipped equations, table overflow, missing figures, broken glyphs, unresolved references, or bibliography placeholders were observed. Page numbering, theorem references and the generated plot/table layout were checked. Text extraction also found no `??`, build-status placeholder, or empty DOI identifier.

During preparation three overfull boxes were repaired by rounding the displayed modulus bit count while retaining exact primes in the paper, referencing the locked mathlib revision instead of an inline unbreakable hash, and making the verification-document path breakable. AMS font definition files were supplied before the final build; the released PDF contains the actual mathematical fonts rather than substitution glyphs.

## Build environment caveat

The installed executable could not fetch its default TeX resource bundle through its own network client. The preparation used a local directory containing required resources from the official Tectonic bundle `tlextras-2022.0r0.tar`, obtained by bounded HTTP range requests. The full 2.88 GB bundle was not downloaded, and the temporary resource subset is not part of the research artifact.

The equivalent final build command is:

```sh
tectonic --bundle "$TECTONIC_BUNDLE" --keep-logs --keep-intermediates paper.tex
```

`TECTONIC_BUNDLE` is an optional local resource directory in that workaround. On a clean machine with working Tectonic bundle access, `tectonic paper.tex` is sufficient. A normal TeX Live build can instead run LaTeX, BibTeX, and two LaTeX passes. This environment workaround does not change manuscript source or create a substitute PDF.

The artifact DOI is omitted until an actual DOI is reserved. If DOI metadata or other paper content changes, recompile and check the new PDF before archiving; this record describes the inspected nine-page manuscript state and is not an assertion that a deposit is already published.
