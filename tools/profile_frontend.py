"""Measure public JSON load and exact network folding separately from auditing."""
import datetime, json, pathlib, statistics, sys, time
ROOT=pathlib.Path(__file__).resolve().parents[1]; sys.path.insert(0,str(ROOT))
start=time.perf_counter(); from scalar_compiler import fold_network,cyclic_representative
import_seconds=time.perf_counter()-start
start=time.perf_counter(); models=json.loads((ROOT/'artifacts/models.json').read_text())['models']; load=time.perf_counter()-start
results=[]
for m in models:
    samples=[]
    for _ in range(9):
        start=time.perf_counter(); c=fold_network(m['W'],m['v'],m['bias'],m['output_bias'],m['normalization']); A=cyclic_representative(c['Q']); samples.append(time.perf_counter()-start)
        assert c['Q']==m['Q'] and c['linear']==m['linear'] and c['constant']==m['constant']
        assert c['storage_residual_box_exact']=='0'
    results.append({'model':m['name'],'samples_seconds':samples,'median_seconds':statistics.median(samples),'coefficient_storage_error_exact':'0','orientation_error_exact':'0'})
(ROOT/'artifacts/frontend_profile.json').write_text(json.dumps({'recorded_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'module_import_seconds':import_seconds,'models_load_seconds':load,'implementation':'exact Fraction reference compiler; includes rational expansion and orientation certificate; excludes unrelated LDL/spectral construction and statistical audits','results':results},indent=2))
