"""Public compiler API for terminal real scalar quadratic computations.

No secret key, ciphertext input, or client feature extraction is involved.
Public neural weights are folded once. The execution backend is bench/packed.go.
"""
from fractions import Fraction
import math

def fold_network(W, v, bias, output_bias=0, normalization=1):
    """Exact rational public expansion, then explicitly certified binary64 storage."""
    # Exact integer accumulation avoids Fraction overhead on every multiply.
    # Python integers cannot overflow. The final dyadic storage is still checked.
    if all(isinstance(x,int) for row in W for x in row) and all(isinstance(x,int) for x in list(v)+list(bias)+[output_bias]):
        h=len(W); d=len(W[0]); scale=Fraction(normalization)
        if len(v)!=h or len(bias)!=h or any(len(row)!=d for row in W):
            raise ValueError('inconsistent network dimensions')
        Q=[[scale*sum(v[k]*W[k][i]*W[k][j] for k in range(h)) for j in range(d)] for i in range(d)]
        linear=[scale*(2*sum(v[k]*bias[k]*W[k][i] for k in range(h))) for i in range(d)]
        constant=scale*(sum(v[k]*bias[k]*bias[k] for k in range(h))+output_bias)
        qf=[[float(x) for x in row] for row in Q]; lf=list(map(float,linear));cf=float(constant)
        if not all(math.isfinite(x) for row in qf for x in row) or not all(map(math.isfinite,lf+[cf])):
            raise ValueError('public coefficient overflow')
        qr=sum(abs(Q[i][j]-Fraction(qf[i][j])) for i in range(d) for j in range(d))
        lr=sum(abs(linear[i]-Fraction(lf[i])) for i in range(d));cr=abs(constant-Fraction(cf))
        return {'Q':qf,'linear':lf,'constant':cf,'storage_residual_box_exact':str(qr+lr+cr)}
    W=[[Fraction(w) for w in row] for row in W]
    v=list(map(Fraction,v)); b=list(map(Fraction,bias)); scale=Fraction(normalization)
    h=len(W); d=len(W[0])
    if len(v)!=h or len(b)!=h or any(len(row)!=d for row in W):
        raise ValueError('inconsistent network dimensions')
    Q=[[scale*sum(v[k]*W[k][i]*W[k][j] for k in range(h)) for j in range(d)] for i in range(d)]
    linear=[2*scale*sum(v[k]*b[k]*W[k][i] for k in range(h)) for i in range(d)]
    constant=scale*(sum(v[k]*b[k]*b[k] for k in range(h))+Fraction(output_bias))
    qf=[[float(x) for x in row] for row in Q]; lf=list(map(float,linear)); cf=float(constant)
    if not all(math.isfinite(x) for row in qf for x in row) or not all(map(math.isfinite,lf+[cf])):
        raise ValueError('public coefficient overflow')
    qr=sum(abs(Q[i][j]-Fraction(qf[i][j])) for i in range(d) for j in range(d))
    lr=sum(abs(linear[i]-Fraction(lf[i])) for i in range(d)); cr=abs(constant-Fraction(cf))
    return {'Q':qf,'linear':lf,'constant':cf,'storage_residual_box_exact':str(qr+lr+cr)}

def cyclic_representative(Q):
    """Return A with sym(A)=Q and <=floor(d/2)+1 cyclic diagonals.

    Retain offsets 0..floor(d/2), double non-self-inverse off-diagonals.
    Legal only if the sole consumer is the real scalar sum_i x_i(Ax)_i.
    """
    d=len(Q)
    if not d or any(len(row)!=d for row in Q): raise ValueError('Q must be square')
    if any(not math.isfinite(float(Q[i][j])) or Q[i][j]!=Q[j][i] for i in range(d) for j in range(d)):
        raise ValueError('finite symmetric Q required')
    A=[[0.]*d for _ in range(d)]
    for i in range(d):
        for j in range(d):
            r=(j-i)%d
            if r==0 or d%2==0 and r==d//2: A[i][j]=float(Q[i][j])
            elif 0<r<=(d-1)//2: A[i][j]=2.*float(Q[i][j])
            if not math.isfinite(A[i][j]): raise ValueError('orientation coefficient overflow')
    for i in range(d):
        for j in range(d):
            assert (Fraction(A[i][j])+Fraction(A[j][i]))/2==Fraction(Q[i][j])
    return A
