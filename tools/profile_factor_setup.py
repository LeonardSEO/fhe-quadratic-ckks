"""Measure previously offline public factor construction, never hide it as inference."""
import json,pathlib,statistics,sys,time
import numpy as np
ROOT=pathlib.Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT))
from compiler_ldl import compile_ldl
models=json.loads((ROOT/'artifacts/models.json').read_text())['models'];results=[]
for m in models:
    timings={'spectral':[],'ldl':[]}
    for _ in range(9):
        t=time.perf_counter();v,u=np.linalg.eigh(np.asarray(m['Q']));rows=np.sqrt(np.abs(v))[:,None]*u.T;signs=np.sign(v);timings['spectral'].append(time.perf_counter()-t)
        t=time.perf_counter();f=compile_ldl(m['Qint'],m['normalization']);timings['ldl'].append(time.perf_counter()-t)
        assert f['rows']==m['ldl']['rows'] and f['signs']==m['ldl']['signs']
    results.append({'model':m['name'],'median_seconds':{k:statistics.median(v) for k,v in timings.items()},'samples_seconds':timings})
(ROOT/'artifacts/factor_setup_profile.json').write_text(json.dumps({'scope':'public spectral construction; complete exact LDL helper includes its pairing/residual certificates, separate from public backend plaintext encoding and affine completion certificates already timed in Go','numpy':np.__version__,'results':results},indent=2))
