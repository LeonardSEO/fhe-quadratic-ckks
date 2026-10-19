#!/usr/bin/env python3
"""Run Lean's dependency inspection and reject nonstandard proof axioms."""
from pathlib import Path
import re
import subprocess

principals = [
    "quadratic_transpose", "quadratic_preservation", "orientation_admissible",
    "orientation_preservation", "affine_score_preservation", "canonical_iff_val_le_half",
    "canonicalOffsets_card", "orientation_support_upper_bound",
    "activeOffsets_card_lower_bound", "orientation_support_exact", "orientation_support_optimal",
]
allowed = {"propext", "Classical.choice", "Quot.sound"}
root = Path(__file__).resolve().parent
run = subprocess.run(["lake", "env", "lean", "FHEQuadratic/Axioms.lean"],
                     cwd=root, text=True, capture_output=True)
output = run.stdout + run.stderr
print(output, end="")
if run.returncode:
    raise SystemExit(run.returncode)
for name in principals:
    pattern = rf"['\"]?FHEQuadratic\.{name}['\"]? depends on axioms:\s*\[([^]]*)\]"
    match = re.search(pattern, output)
    if not match:
        if f"'FHEQuadratic.{name}' does not depend on any axioms" in output:
            continue
        raise SystemExit(f"Missing axiom dependency record for {name}")
    axioms = {s.strip() for s in match.group(1).split(",") if s.strip()}
    unexpected = axioms - allowed
    if unexpected:
        raise SystemExit(f"Unexpected axioms in {name}: {sorted(unexpected)}")
if "sorryAx" in output:
    raise SystemExit("sorryAx found")
print(f"PASS: {len(principals)} principal theorems use only standard logical axioms.")
