import FHEQuadratic.Definitions

namespace FHEQuadratic

open scoped BigOperators

variable {n : ℕ} [NeZero n]

/-- Transposing a real matrix preserves its scalar quadratic form. -/
theorem quadratic_transpose (A : Mat n) (x : Index n → ℝ) :
    quadratic (fun i j => A j i) x = quadratic A x := by
  unfold quadratic
  rw [Finset.sum_comm]
  apply Finset.sum_congr rfl
  intro i _
  apply Finset.sum_congr rfl
  intro j _
  ring

/-- The admissibility equation suffices for exact scalar preservation. -/
theorem quadratic_preservation (Q A : Mat n) (h : Admissible Q A)
    (x : Index n → ℝ) : quadratic A x = quadratic Q x := by
  have hdouble : 2 * quadratic Q x =
      quadratic A x + quadratic (fun i j => A j i) x := by
    simp only [quadratic, Finset.mul_sum, ← Finset.sum_add_distrib]
    apply Finset.sum_congr rfl
    intro i _
    apply Finset.sum_congr rfl
    intro j _
    calc
      2 * (x i * Q i j * x j) = x i * (2 * Q i j) * x j := by ring
      _ = x i * A i j * x j + x i * A j i * x j := by
        rw [← h i j]
        ring
  rw [quadratic_transpose A x] at hdouble
  linarith

omit [NeZero n] in
/-- The cyclic orientation is admissible for every real symmetric matrix. -/
theorem orientation_admissible (Q : Mat n) (hQ : Symmetric Q) :
    Admissible Q (orientation Q) := by
  intro i j
  rcases lt_trichotomy (j - i).val (i - j).val with h | h | h
  · have hrev : ¬ (i - j).val < (j - i).val := by omega
    have hne : (j - i).val ≠ (i - j).val := by omega
    simp [orientation, h, hrev, Ne.symm hne]
  · simp [orientation, h]
    rw [hQ j i]
    ring
  · have hrev : ¬ (j - i).val < (i - j).val := by omega
    have hne : (j - i).val ≠ (i - j).val := by omega
    simp [orientation, h, hrev, hne]
    rw [hQ j i]

/-- The exact cyclic construction preserves the quadratic score. -/
theorem orientation_preservation (Q : Mat n) (hQ : Symmetric Q)
    (x : Index n → ℝ) : quadratic (orientation Q) x = quadratic Q x :=
  quadratic_preservation Q (orientation Q) (orientation_admissible Q hQ) x

/-- Reusing the same linear term and constant preserves the complete public score. -/
theorem affine_score_preservation (Q A : Mat n) (h : Admissible Q A)
    (x l : Index n → ℝ) (c : ℝ) :
    quadratic A x + (∑ i, l i * x i) + c =
      quadratic Q x + (∑ i, l i * x i) + c := by
  rw [quadratic_preservation Q A h x]

end FHEQuadratic
