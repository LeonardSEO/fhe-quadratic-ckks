"""Reserved-seed encrypted validation; never used for parameter/plan tuning."""
import argparse, json, math, os, pathlib, subprocess
ROOT=pathlib.Path(__file__).resolve().parents[1]
p=argparse.ArgumentParser(); p.add_argument('--seeds',type=int,default=40); p.add_argument('--out',default='reruns/heldout_inputs'); a=p.parse_args()
models=json.loads((ROOT/'artifacts/models.json').read_text())['models']
profiles=json.loads((ROOT/'artifacts/final_v2/packed/aggregate.json').read_text())['runs']
dest=ROOT/'artifacts'/a.out; dest.mkdir(parents=True,exist_ok=True)
records=[]; max_error=0.; random_count=0
for model in models:
    name=model['name']; n=len(model['Q']); batch=4096//n
    profile=next(v for v in profiles if v['model']==name)
    comparator=min((v for v in profile['results'] if not v['oriented']),key=lambda v:v['median']['complete_local_chain_seconds'])['method']
    for seed_index in range(a.seeds):
        seed=731000+seed_index
        for method in [comparator,'oriented_direct']:
            out=dest/name/str(seed)/method
            env=os.environ.copy(); env.update(FHE_PACKED_METHOD=method,FHE_PACKED_INPUT_SEED=str(seed),FHE_PACKED_VALIDATION='1')
            command=[str(ROOT/'bench/fhebench'),'-model',name,'-batch',str(batch),'-repeats','1','-out',str(out)]
            run_process=subprocess.run(command,cwd=ROOT/'bench',env=env,capture_output=True,text=True)
            if run_process.returncode: raise RuntimeError(run_process.stdout+run_process.stderr)
            run=json.loads((out/name/'standard/run.json').read_text()); result=run['results'][0]
            fixture=json.loads((out/name/'standard/fixture.json').read_text())
            network_error=0.
            for x,y in zip(fixture['inputs'],result['decoded_first_lane']):
                network=model['normalization']*(sum(v*(sum(wj*xj for wj,xj in zip(w,x))+b)**2 for w,v,b in zip(model['W'],model['v'],model['bias']))+model['output_bias'])
                network_error=max(network_error,abs(network-y))
            assert math.isfinite(network_error) and network_error<=1e-3
            assert result['quality_passed']
            err=max(network_error,result['max_abs_error_all_lanes_repeats']); max_error=max(max_error,err)
            records.append({'model':name,'seed':seed,'batch':batch,'method':method,'input_kinds':run['input_kinds'],'network_error_first_block':network_error,'max_error_all_slots':err,'input_sha256':result['input_sha256_last_repeat'],'quality_passed':True})
            if method=='oriented_direct': random_count+=run['input_kinds'].get('uniform_dyadic_box',0)
        if seed_index%10==0: print(name,'seed',seed,'max_error',max_error,flush=True)
        (dest/'summary.json').write_text(json.dumps({'seed_range':[731000,731000+a.seeds-1],'not_used_for_tuning':True,'records':records,'max_error':max_error,'uniform_input_model_pairs':random_count,'statistical_scope':'Empirical zero failures only. Same-dimensional models reuse input vectors, and outputs within a batch share a fresh key. No pooled IID binomial confidence claim and no uniform-box CKKS error theorem.'},indent=2))
print('all validation passed',len(records),'batches; maxerror',max_error,flush=True)
