import FHEQuadratic.Orientation

namespace FHEQuadratic

variable {n : ℕ} [NeZero n]

/-- Canonical cyclic offsets are unique modulo sign, including self-inverse offsets. -/
theorem canonicalOffsets_neg_unique {r s : Index n}
    (hr : r ∈ canonicalOffsets n) (hs : s ∈ canonicalOffsets n)
    (h : r = -s) : r = s := by
  have hr' := (mem_canonicalOffsets r).mp hr
  have hs' := (mem_canonicalOffsets s).mp hs
  have hn : -r = s := by rw [h]; simp
  apply ZMod.val_injective n
  apply le_antisymm
  · simpa only [hn] using hr'
  · simpa only [h] using hs'

/-- Density forces every sign orbit to meet the support of every admissible representative. -/
theorem active_or_neg_active (Q A : Mat n) (hQ : CyclicDense Q)
    (hA : Admissible Q A) (r : Index n) :
    r ∈ activeOffsets A ∨ -r ∈ activeOffsets A := by
  obtain ⟨i, hi⟩ := hQ r
  by_cases ha : A i (i + r) ≠ 0
  · exact Or.inl ((mem_activeOffsets A r).mpr ⟨i, ha⟩)
  · right
    apply (mem_activeOffsets A (-r)).mpr
    refine ⟨i + r, ?_⟩
    simp only [add_neg_cancel_right]
    intro hz
    have heq := hA i (i + r)
    have haz : A i (i + r) = 0 := Classical.not_not.mp ha
    rw [haz, hz] at heq
    apply hi
    linarith

/-- The support lower bound first counts sign-orbit representatives directly. -/
theorem canonicalOffsets_card_le_activeOffsets_card (Q A : Mat n)
    (hQ : CyclicDense Q) (hA : Admissible Q A) :
    (canonicalOffsets n).card ≤ (activeOffsets A).card := by
  classical
  let pick : Index n → Index n := fun r => if r ∈ activeOffsets A then r else -r
  have hpick : ∀ r, pick r ∈ activeOffsets A := by
    intro r
    by_cases hr : r ∈ activeOffsets A
    · simpa only [pick, if_pos hr] using hr
    · simpa only [pick, if_neg hr] using (active_or_neg_active Q A hQ hA r).resolve_left hr
  apply Finset.card_le_card_of_injOn pick
  · intro r hr
    exact hpick r
  · intro r hr s hs heq
    by_cases hra : r ∈ activeOffsets A <;> by_cases hsa : s ∈ activeOffsets A
    · simpa only [pick, if_pos hra, if_pos hsa] using heq
    · have h : r = -s := by simpa only [pick, if_pos hra, if_neg hsa] using heq
      exact canonicalOffsets_neg_unique hr hs h
    · have h : -r = s := by simpa only [pick, if_neg hra, if_pos hsa] using heq
      apply canonicalOffsets_neg_unique hr hs
      simpa using congrArg Neg.neg h
    · have h : -r = -s := by simpa only [pick, if_neg hra, if_neg hsa] using heq
      exact neg_injective h

/-- For every positive dimension, cyclic density gives the matching support lower bound. -/
theorem activeOffsets_card_lower_bound (Q A : Mat n)
    (hQ : CyclicDense Q) (hA : Admissible Q A) :
    n / 2 + 1 ≤ (activeOffsets A).card := by
  rw [← canonicalOffsets_card n]
  exact canonicalOffsets_card_le_activeOffsets_card Q A hQ hA

/-- Under density the cyclic orientation attains the exact support optimum. -/
theorem orientation_support_exact (Q : Mat n)
    (hQ : Symmetric Q) (hd : CyclicDense Q) :
    (activeOffsets (orientation Q)).card = n / 2 + 1 := by
  exact le_antisymm (orientation_support_upper_bound Q)
    (activeOffsets_card_lower_bound Q (orientation Q) hd (orientation_admissible Q hQ))

/-- The optimum is restricted to admissible matrix representatives. -/
theorem orientation_support_optimal (Q A : Mat n)
    (hQ : Symmetric Q) (hd : CyclicDense Q) (hA : Admissible Q A) :
    (activeOffsets (orientation Q)).card ≤ (activeOffsets A).card := by
  rw [orientation_support_exact Q hQ hd]
  exact activeOffsets_card_lower_bound Q A hd hA

end FHEQuadratic
