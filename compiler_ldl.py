"""Compile an integer symmetric quadratic form into sparse signed CKKS factors.

The default path uses exact Fraction arithmetic with symmetric largest-diagonal
pivoting. A nonzero, zero-diagonal Schur block needs a 2x2 pivot; this compact
compiler explicitly falls back to NumPy's spectral factorization in that case.
Every returned binary64 factorization receives an exact rational residual
certificate, including the fallback. No random tests underlie that certificate.
"""

from fractions import Fraction
import math
from typing import Any


def _up(value: Fraction) -> float:
    """Smallest representable float >= a nonnegative rational (or infinity)."""
    try:
        result = float(value)
    except OverflowError:
        return math.inf
    if math.isfinite(result) and Fraction.from_float(result) < value:
        result = math.nextafter(result, math.inf)
    return result


def _ldl(q: list[list[Fraction]]):
    n = len(q)
    a = [row[:] for row in q]
    lower = [[Fraction(i == j) for j in range(n)] for i in range(n)]
    diagonal = [Fraction(0) for _ in range(n)]
    order = list(range(n))
    for k in range(n):
        pivot = max(range(k, n), key=lambda j: abs(a[j][j]))
        if not a[pivot][pivot]:
            if any(a[i][j] for i in range(k, n) for j in range(k, n)):
                raise ArithmeticError("nonzero zero-diagonal Schur block requires a 2x2 pivot")
            break  # Exact zero tail: retain zero D entries, omit their factors.
        a[k], a[pivot] = a[pivot], a[k]
        for row in a:
            row[k], row[pivot] = row[pivot], row[k]
        order[k], order[pivot] = order[pivot], order[k]
        for j in range(k):
            lower[k][j], lower[pivot][j] = lower[pivot][j], lower[k][j]
        diagonal[k] = a[k][k]
        for i in range(k + 1, n):
            lower[i][k] = a[i][k] / diagonal[k]
        for i in range(k + 1, n):
            for j in range(k + 1, n):
                a[i][j] -= lower[i][k] * diagonal[k] * lower[j][k]
    # A programming error cannot silently become a factorization result.
    for i in range(n):
        for j in range(n):
            reconstructed = sum(lower[i][k] * diagonal[k] * lower[j][k]
                                for k in range(n))
            if reconstructed != q[order[i]][order[j]]:
                raise AssertionError("exact LDL reconstruction failed")
    return lower, diagonal, order


def _maximum_overlap_pairs(rows: list[list[float]], signs: list[int]):
    """Maximum-cardinality, maximum-overlap opposite-sign matching.

    Rectangular Hungarian algorithm with integer costs; no SciPy dependency.
    Unpaired negative rows are represented by purely imaginary complex forms.
    """
    positive = [i for i, sign in enumerate(signs) if sign > 0]
    negative = [i for i, sign in enumerate(signs) if sign < 0]
    supports = [{j for j, coefficient in enumerate(row) if coefficient != 0.0}
                for row in rows]
    left, right = ((positive, negative) if len(positive) <= len(negative)
                   else (negative, positive))
    matched = []
    if left:
        nr, nc = len(left), len(right)
        cost = [[-len(supports[i] & supports[j]) for j in right] for i in left]
        u, v = [0] * (nr + 1), [0] * (nc + 1)
        assignment, way = [0] * (nc + 1), [0] * (nc + 1)
        for i in range(1, nr + 1):
            assignment[0], column = i, 0
            minimum = [math.inf] * (nc + 1)
            used = [False] * (nc + 1)
            while True:
                used[column] = True
                current_row, delta, next_column = assignment[column], math.inf, 0
                for j in range(1, nc + 1):
                    if not used[j]:
                        reduced = cost[current_row - 1][j - 1] - u[current_row] - v[j]
                        if reduced < minimum[j]:
                            minimum[j], way[j] = reduced, column
                        if minimum[j] < delta:
                            delta, next_column = minimum[j], j
                for j in range(nc + 1):
                    if used[j]:
                        u[assignment[j]] += delta
                        v[j] -= delta
                    else:
                        minimum[j] -= delta
                column = next_column
                if assignment[column] == 0:
                    break
            while True:
                previous = way[column]
                assignment[column] = assignment[previous]
                column = previous
                if column == 0:
                    break
        for j in range(1, nc + 1):
            if assignment[j]:
                a, b = left[assignment[j] - 1], right[j - 1]
                matched.append([a, b] if signs[a] > 0 else [b, a])
    matched.sort(key=lambda pair: (pair[0], pair[1]))
    used_rows = {index for pair in matched for index in pair}
    pairs = matched + [[i, None] for i in positive if i not in used_rows]
    pairs += [[None, i] for i in negative if i not in used_rows]
    return pairs, supports


def compile_ldl(Qint, scale=1.0) -> dict[str, Any]:
    """Return JSON-safe signed factors for ``scale * Qint``.

    ``Qint`` must be a finite integer symmetric matrix. ``scale`` is interpreted
    as its exact binary64 value, rather than as an unstated decimal rational.
    ``rows[i]`` is a real factor row in ORIGINAL input-coordinate order and
    ``signs[i]`` is +1 or -1. ``pairs`` contains ``[positive_index,negative_index]``
    with None on an unmatched side. For each pair, ``complex_real`` and
    ``complex_imag`` give the coefficients of the single complex square form.

    The residual certificate bounds the real quadratic error for every point
    in ``|x_j| <= R`` by ``R**2 * residual_box_unit_upper``. It covers coefficient
    storage/compilation error only; CKKS encryption, arithmetic and rescaling
    error must be added by the execution harness.
    """
    q = [[Fraction(value) for value in row] for row in Qint]
    n = len(q)
    if not n or any(len(row) != n for row in q):
        raise ValueError("Qint must be a nonempty square matrix")
    if any(value.denominator != 1 for row in q for value in row):
        raise ValueError("Qint must contain integer coefficients")
    if any(q[i][j] != q[j][i] for i in range(n) for j in range(n)):
        raise ValueError("Qint must be symmetric")
    scale_float = float(scale)
    if not math.isfinite(scale_float):
        raise ValueError("scale must be finite")
    scale_exact = Fraction.from_float(scale_float)
    method, fallback_reason = "exact_pivoted_fraction_ldl", None
    exact_verified, order, diagonal_strings = False, list(range(n)), []
    rows, signs = [], []
    try:
        lower, diagonal, order = _ldl(q)
        diagonal_strings = [str(value) for value in diagonal]
        exact_verified = True
        for k, value in enumerate(diagonal):
            if not value or not scale_exact:
                continue
            magnitude = math.sqrt(float(abs(value * scale_exact)))
            row = [0.0] * n
            for j in range(n):
                row[order[j]] = magnitude * float(lower[j][k])
            if not all(math.isfinite(coefficient) for coefficient in row):
                raise ArithmeticError("binary64 LDL coefficient overflow")
            rows.append(row)
            signs.append(1 if value * scale_exact > 0 else -1)
    except ArithmeticError as error:
        # NumPy is deliberately optional on the normal exact-LDL path.
        try:
            import numpy as np
        except ImportError as missing:
            raise RuntimeError("spectral fallback requires NumPy: " + str(error)) from missing
        method, fallback_reason = "spectral_fallback", str(error)
        matrix = np.array([[float(value * scale_exact) for value in row] for row in q])
        eigenvalues, eigenvectors = np.linalg.eigh(matrix)
        rows, signs = [], []
        for k, value in enumerate(eigenvalues):
            # No tolerance truncation: retain every nonzero binary64 eigenvalue.
            if value != 0.0:
                rows.append((math.sqrt(abs(float(value))) * eigenvectors[:, k]).tolist())
                signs.append(1 if value > 0.0 else -1)
        order, diagonal_strings, exact_verified = list(range(n)), [], False
    pairs, supports = _maximum_overlap_pairs(rows, signs)
    complex_real = [[0.0] * n if positive is None else rows[positive][:]
                    for positive, negative in pairs]
    complex_imag = [[0.0] * n if negative is None else rows[negative][:]
                    for positive, negative in pairs]
    exact_rows = [[Fraction.from_float(value) for value in row] for row in rows]
    residual = [[sum(signs[k] * exact_rows[k][i] * exact_rows[k][j]
                     for k in range(len(rows))) - scale_exact * q[i][j]
                 for j in range(n)] for i in range(n)]
    qabs = [[sum(row[i] * row[j] for row in exact_rows)
             for j in range(n)] for i in range(n)]
    residual_l1 = sum(abs(value) for row in residual for value in row)
    residual_infinity = max(sum(abs(value) for value in row) for row in residual)
    growth = max(sum(abs(value) for value in row) for row in qabs)
    complex_nnz = sum(len((supports[a] if a is not None else set()) |
                         (supports[b] if b is not None else set())) for a, b in pairs)
    return {
        "method": method,
        "fallback_reason": fallback_reason,
        "dimension": n,
        "scale": scale_float,
        "scale_exact_binary64": str(scale_exact),
        "rows": rows,
        "signs": signs,
        "pairs": pairs,
        "complex_real": complex_real,
        "complex_imag": complex_imag,
        "complex_coefficients": [
            {str(j): {"real": real[j], "imag": imag[j]}
             for j in range(n) if real[j] != 0.0 or imag[j] != 0.0}
            for real, imag in zip(complex_real, complex_imag)
        ],
        "permutation": order,
        "diagonal_exact_unscaled": diagonal_strings,
        "exact_ldl_reconstruction_verified": exact_verified,
        "inertia": {"positive": signs.count(1), "negative": signs.count(-1)},
        "inertia_certified": exact_verified,
        "inertia_assessment": ("exact rational pivots, including exact zero tail"
                               if exact_verified else
                               "binary64 spectral signs; not an exact inertia certificate"),
        "real_nonzero_coefficients": sum(map(len, supports)),
        "complex_nonzero_coefficients": complex_nnz,
        "matching_saved_public_multiplications": sum(map(len, supports)) - complex_nnz,
        "growth_bound": _up(growth),
        "growth_bound_exact": str(growth),
        "growth_bound_meaning": "upper bound for norm_inf(F^T F) of stored real factor rows",
        "residual_box_unit_upper": _up(residual_l1),
        "residual_box_unit_exact": str(residual_l1),
        "residual_infinity_upper": _up(residual_infinity),
        "residual_certificate": {
            "arithmetic": "exact rational arithmetic on stored IEEE-754 binary64 coefficients",
            "target": "Qint multiplied by exact binary64 scale",
            "domain": "all real x with max_j abs(x_j) <= R",
            "bound": "absolute quadratic coefficient error <= R^2 * residual_box_unit_upper",
            "excludes": "CKKS encoding, encryption, arithmetic, rescaling and decryption error",
        },
    }
