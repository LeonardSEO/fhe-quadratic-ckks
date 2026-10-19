import FHEQuadratic.Preservation

namespace FHEQuadratic

variable {n : ℕ} [NeZero n]

/-- Canonical representatives are precisely the numerical offsets 0 through floor(n/2). -/
theorem canonical_iff_val_le_half (r : Index n) :
    r ∈ canonicalOffsets n ↔ r.val ≤ n / 2 := by
  rw [mem_canonicalOffsets]
  by_cases hr : r = 0
  · simp [hr]
  · rw [ZMod.neg_val, if_neg hr]
    have hv := ZMod.val_lt r
    omega

/-- Exact number of inversion orbits, valid for odd and even positive dimensions. -/
theorem canonicalOffsets_card (n : ℕ) [NeZero n] :
    (canonicalOffsets n).card = n / 2 + 1 := by
  classical
  have hpos : 0 < n := NeZero.pos n
  have hhalf : n / 2 < n := by omega
  calc
    (canonicalOffsets n).card = (Finset.range (n / 2 + 1)).card := by
      apply Finset.card_bij (fun r _ => r.val)
      · intro r hr
        exact Finset.mem_range.mpr (by have := (canonical_iff_val_le_half r).mp hr; omega)
      · intro a ha b hb heq
        exact ZMod.val_injective n heq
      · intro k hk
        have hk' : k < n := by have := Finset.mem_range.mp hk; omega
        refine ⟨(k : ZMod n), ?_, ?_⟩
        · apply (canonical_iff_val_le_half _).mpr
          simpa [ZMod.val_natCast, Nat.mod_eq_of_lt hk'] using (show k ≤ n / 2 by
            have := Finset.mem_range.mp hk; omega)
        · simp [ZMod.val_natCast, Nat.mod_eq_of_lt hk']
    _ = n / 2 + 1 := Finset.card_range _

/-- The orientation never uses the larger member of a nontrivial offset pair. -/
theorem orientation_active_subset (Q : Mat n) :
    activeOffsets (orientation Q) ⊆ canonicalOffsets n := by
  intro r hr
  obtain ⟨i, hi⟩ := mem_activeOffsets _ _ |>.mp hr
  apply mem_canonicalOffsets _ |>.mpr
  by_contra hn
  have hlt : (-r).val < r.val := by omega
  have hid : i - (i + r) = -r := by abel
  have hjd : (i + r) - i = r := by abel
  simp [orientation, hid, hjd, not_lt.mpr (Nat.le_of_lt hlt), ne_of_gt hlt] at hi

/-- General support upper bound, with fixed offsets counted exactly once. -/
theorem orientation_support_upper_bound (Q : Mat n) :
    (activeOffsets (orientation Q)).card ≤ n / 2 + 1 := by
  calc
    _ ≤ (canonicalOffsets n).card := Finset.card_le_card (orientation_active_subset Q)
    _ = _ := canonicalOffsets_card n

end FHEQuadratic
