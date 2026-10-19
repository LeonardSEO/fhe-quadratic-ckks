import Mathlib.Data.ZMod.Basic
import Mathlib.Data.Real.Basic
import Mathlib.Algebra.BigOperators.Ring.Finset
import Mathlib.Tactic.Ring
import Mathlib.Tactic.Linarith
import Mathlib.Tactic.Abel
import Lean.Elab.Tactic.Omega

noncomputable section
open scoped BigOperators

namespace FHEQuadratic

variable {n : ℕ} [NeZero n]
abbrev Index (n : ℕ) := ZMod n
abbrev Mat (n : ℕ) := Index n → Index n → ℝ

def quadratic (A : Mat n) (x : Index n → ℝ) : ℝ :=
  ∑ i, ∑ j, x i * A i j * x j

def Symmetric (Q : Mat n) : Prop := ∀ i j, Q i j = Q j i

def Admissible (Q A : Mat n) : Prop := ∀ i j, A i j + A j i = 2 * Q i j

def activeOffsets (A : Mat n) : Finset (Index n) := by
  classical
  exact Finset.univ.filter fun r => ∃ i, A i (i + r) ≠ 0

def CyclicDense (Q : Mat n) : Prop := ∀ r : Index n, ∃ i, Q i (i + r) ≠ 0

def orientation (Q : Mat n) : Mat n := fun i j =>
  if (j - i).val < (i - j).val then 2 * Q i j
  else if (j - i).val = (i - j).val then Q i j
  else 0

def canonicalOffsets (n : ℕ) [NeZero n] : Finset (Index n) :=
  Finset.univ.filter fun r => r.val ≤ (-r).val

@[simp] theorem mem_activeOffsets (A : Mat n) (r : Index n) :
    r ∈ activeOffsets A ↔ ∃ i, A i (i + r) ≠ 0 := by
  simp [activeOffsets]

@[simp] theorem mem_canonicalOffsets (r : Index n) :
    r ∈ canonicalOffsets n ↔ r.val ≤ (-r).val := by
  simp [canonicalOffsets]

end FHEQuadratic
