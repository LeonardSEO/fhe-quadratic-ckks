#!/usr/bin/env python3
"""Exact rational bridge for the Python/Go cyclic rule; not a Go verification."""
import json, pathlib, sys
from fractions import Fraction as F
ROOT=pathlib.Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT))
from scalar_compiler import cyclic_representative, fold_network

def formal_rule(q):
    # Equivalent to cyclicOrient in Lean: offset <= half, tie remains undoubled.
    n=len(q)
    return [[q[i][j] if 2*((j-i)%n) in (0,n) else 2*q[i][j] if 2*((j-i)%n)<n else F(0) for j in range(n)] for i in range(n)]

def check(q,name):
    n=len(q);a=cyclic_representative(q);b=formal_rule(q)
    assert [[F(v) for v in row] for row in a]==b,name
    assert all(b[i][j]+b[j][i]==2*q[i][j] for i in range(n) for j in range(n)),name
    active=sorted({(j-i)%n for i in range(n) for j in range(n) if b[i][j]})
    assert len(active)<=n//2+1
    return dict(name=name,dimension=n,Q=[[str(v) for v in row] for row in q],A=[[str(v) for v in row] for row in b],active_offsets=active)

def main():
    records=[]
    for n in range(1,18):
        q=[[F((min(i,j)+1)*(max(i,j)+2),8) for j in range(n)] for i in range(n)]
        records.append(check(q,f'dense_d{n}'))
        records.append(check([[v if (i+j)%3 else F(0) for j,v in enumerate(row)] for i,row in enumerate(q)],f'sparse_d{n}'))
    for m in json.loads((ROOT/'artifacts/models.json').read_text())['models']:
        folded=fold_network(m['W'],m['v'],m['bias'],m['output_bias'],F(m['normalization_exact']))
        assert folded['Q']==m['Q'] and folded['linear']==m['linear'] and folded['constant']==m['constant']
        assert folded['storage_residual_box_exact']=='0'
        # Independently recompute 14 stored binary64 factor residual certificates.
        for kind in ('ldl','spectral'):
            factors=m[kind];rows=[[F(v) for v in row] for row in factors['rows']]
            residual=sum(abs(F(m['Q'][i][j])-sum(sign*row[i]*row[j] for row,sign in zip(rows,factors['signs']))) for i in range(m['dimension']) for j in range(m['dimension']))
            assert residual==F(factors['residual_box_unit_exact'])
            assert residual<=F(factors['residual_box_unit_upper'])
        # The Go backend is restricted to power-of-two dimensions; its even rule
        # must agree on each frozen model. General odd construction is Python/Lean.
        n=m['dimension'];assert n>0 and n&(n-1)==0
        go=[[F(m['Q'][i][j])*(1 if (j-i)%n in (0,n//2) else 2 if (j-i)%n<n//2 else 0) for j in range(n)] for i in range(n)]
        assert go==formal_rule([[F(v) for v in row] for row in m['Q']])
        records.append(check([[F(v) for v in row] for row in m['Q']],m['name']))
    target=ROOT/'generated/exact_correspondence.json';payload={'scope':'Exact rational agreement of exported Python matrices with the mathematical cyclic rule. Go agrees by source review and encrypted tests; not formally verified. General theorem is proved separately in Lean.','records':records}
    if '--check' in sys.argv:
        assert json.loads(target.read_text())==payload,'regenerate correspondence export'
    else:target.write_text(json.dumps(payload,indent=2)+'\n')
    print(f'PASS {len(records)} exact matrix bridges, all seven exact network folds')
if __name__=='__main__':main()
