#!/usr/bin/env python3
"""Read-only validation of frozen evidence; optional complete ciphertext supplement."""
import argparse,hashlib,json,math,pathlib,statistics
from fractions import Fraction as F
ROOT=pathlib.Path(__file__).resolve().parents[1];A=ROOT/'artifacts'
def load(p):return json.loads(p.read_text())
def digest(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def close(a,b):return math.isclose(a,b,rel_tol=1e-12,abs_tol=1e-15)
def prime64(n):
    # Deterministic Miller-Rabin for unsigned 64-bit integers.
    if n<2:return False
    for p in (2,3,5,7,11,13,17,19,23,29,31,37):
        if n%p==0:return n==p
    d=n-1;s=0
    while d%2==0:d//=2;s+=1
    for a in (2,325,9375,28178,450775,9780504,1795265022):
        if a%n==0:continue
        x=pow(a,d,n)
        if x in (1,n-1):continue
        for _ in range(s-1):
            x=x*x%n
            if x==n-1:break
        else:return False
    return True
FIXED={
 'Standard':([288230376151760897,1099511480321,1099511922689],[1152921504606830593]),
 'ConjugateInvariant':([288230376152137729,1099511922689,1099512938497],[1152921504606748673])}
def check_parameters(run):
    assert (run['Q'],run['P'])==FIXED[run['ring_type']]
    assert run['logN']==13 and run['log_default_scale']==40 and run['thread_count']==1
    assert close(run['logQP'],sum(math.log2(p) for p in run['Q']+run['P']))
    assert run['logQP']<214 # published table screen, not a new estimator calculation

def check_result(v,folder,strict,counts):
    assert v.get('quality_passed',v.get('quality_gate_passed')) and v['max_abs_error_all_lanes_repeats']<=v.get('required_max_absolute_error',v.get('quality_gate_absolute_error',.001))<=.001
    assert v['bootstraps']==0
    for key,values in v.get('samples',{}).items():
        assert values and all(math.isfinite(x) and x>=0 for x in values),(folder,key)
        if key in v['median']:assert close(statistics.median(values),v['median'][key]),(folder,key)
    for kind in ('input','output'):
        hashes=v[kind+'_sha256_last_repeat']
        paths=[folder/(v['method']+'_'+kind+(f'_{i:02d}' if isinstance(hashes,list) else '')+'.ct') for i in range(len(hashes) if isinstance(hashes,list) else 1)]
        for p,expected in zip(paths,hashes if isinstance(hashes,list) else [hashes]):
          if p.exists():
            assert digest(p)==expected,p
            counts['ciphertexts']+=1
          else:
            counts['omitted_ciphertexts']+=1
            assert not strict, f'Missing supplementary ciphertext {p}'
def main():
    p=argparse.ArgumentParser();p.add_argument('--strict-ciphertexts',action='store_true');args=p.parse_args()
    counts=dict(run_files=0,method_records=0,ciphertexts=0,omitted_ciphertexts=0,fixture_outputs=0)
    max_error=0.
    affine_checked=set()
    for ring,(q,p) in FIXED.items():
        for modulus in q+p:
            assert modulus<2**64 and prime64(modulus)
            assert (modulus-1)%((4 if ring=='ConjugateInvariant' else 2)*8192)==0
    assert 'github.com/tuneinsight/lattigo/v6 v6.2.0' in (ROOT/'bench/go.mod').read_text()
    models={m['name']:m for m in load(A/'models.json')['models']}
    for arm in ('final_v2','confirmation','separated_confirmation_final','heldout_inputs'):
        for path in (A/arm).rglob('run.json'):
            run=load(path);counts['run_files']+=1
            assert run['batch']>0
            check_parameters(run)
            fixture_path=next(path.parent.glob('*fixture.json'));fixture=load(fixture_path);assert digest(fixture_path)==run['fixture_sha256']
            model=models[run['model']]
            # The exact dyadic fold is independently checked by check_correspondence.py.
            # Recompute original model scores via expanded coefficients for all saved inputs.
            reference=[sum(x[i]*sum(q*xj for q,xj in zip(row,x)) for i,row in enumerate(model['Q']))+sum(a*b for a,b in zip(model['linear'],x))+model['constant'] for x in fixture['inputs']]
            for y,z in zip(reference,fixture['reference']):assert abs(y-z)<1e-11
            for v in run['results']:
                assert len(v['decoded_first_lane'])==len(reference)==run['batch']
                assert all(len(values)==run['repeats'] for values in v.get('samples',{}).values())
                counts['method_records']+=1;check_result(v,path.parent,args.strict_ciphertexts,counts)
                if 'coefficient_residual_certificate' in v:
                    kind=v['factorization'];identity=(model['name'],kind)
                    if identity not in affine_checked:
                        factors=model[kind];rows=[[F(z) for z in row] for row in factors['rows']];signs=factors['signs'];bias=list(map(F,v['affine_bias_real']));d=run['dimension']
                        quadratic=sum(abs(F(model['Q'][i][j])-sum(sign*row[i]*row[j] for sign,row in zip(signs,rows))) for i in range(d) for j in range(d))
                        linear=sum(abs(F(model['linear'][i])-2*sum(sign*b*row[i] for sign,b,row in zip(signs,bias,rows))) for i in range(d))
                        constant=abs(F(v['shifted_constant'])+sum(sign*b*b for sign,b in zip(signs,bias))-F(model['constant']))
                        cert=v['coefficient_residual_certificate']
                        assert quadratic+linear+constant==F(cert['unit_box_absolute_error_exact'])
                        assert quadratic<=F(cert['quadratic_entry_l1_upper']) and linear<=F(cert['linear_l1_upper']) and constant<=F(cert['constant_abs_upper'])
                        affine_checked.add(identity)
                for y,z in zip(reference,v['decoded_first_lane']):
                    assert abs(y-z)<=.001;counts['fixture_outputs']+=1
                max_error=max(max_error,v['max_abs_error_all_lanes_repeats'])
                if arm=='separated_confirmation_final':
                    s=v['server_process'];assert s['server_pid']!=s['client_pid'];assert s['exit_peak_rss_bytes']>0
                    assert v['http_key_distribution']['server_process_id']==s['server_pid']
                    assert run['server_process_requested'] and run['batch']==128 and run['repeats']==31
                    assert run['logN']==13 and run['thread_count']==1 and run['log_default_scale']==40
    for arm in ('confirmation','separated_confirmation_final'):
        records=load(A/arm/'records.json')['records'];assert len(records)==45
        for r in records:
            path=A/arm/r['model']/f"round{r['round']}"/r['method']/r['model']/'standard/run.json'
            assert load(path)==r['run']
    h=load(A/'heldout_inputs/summary.json');assert len(h['records'])==560
    assert h['not_used_for_tuning'] and h['max_error']<=.001
    assert len(affine_checked)==14
    result={'status':'PASS','affine_certificates':len(affine_checked),'checked':counts,'maximum_stored_encrypted_error':max_error,'ciphertext_scope':'all required' if args.strict_ciphertexts else 'present binaries checked; omitted binary evidence requires supplement'}
    print(json.dumps(result,indent=2))
if __name__=='__main__':main()
