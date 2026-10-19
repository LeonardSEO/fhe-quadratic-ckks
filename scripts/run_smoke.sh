#!/usr/bin/env bash
# Fresh encrypted functional check; this does not reproduce historical timings.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p "$ROOT/artifacts/reruns"
DEST="$(mktemp -d "$ROOT/artifacts/reruns/smoke-XXXXXXXX")"
BUILD="$(mktemp -d)"
trap 'rm -rf "$BUILD"' EXIT
(cd "$ROOT/bench" && go build -o "$BUILD/fhe-bench" .)
for METHOD in triangular_direct oriented_direct; do
  FHE_PACKED_TRANSPORT=http FHE_PACKED_SERVER_PROCESS=1 FHE_PACKED_VALIDATION=1 \
    FHE_PACKED_METHOD="$METHOD" "$BUILD/fhe-bench" \
    -models "$ROOT/artifacts/models.json" -model synthetic_d8h64seed20261020 \
    -batch 128 -repeats 3 -out "$DEST/$METHOD"
done
python3 - "$DEST" <<'PY'
import hashlib,json,pathlib,sys
root=pathlib.Path(sys.argv[1]); records=[]
for path in sorted(root.rglob('run.json')):
    run=json.loads(path.read_text()); assert run['server_process_requested']
    assert run['batch']==128 and run['repeats']==3 and run['logN']==13
    assert run['log_default_scale']==40 and run['thread_count']==1
    assert len(run['results'])==1
    v=run['results'][0]; s=v['server_process']
    assert s['server_pid']!=s['client_pid'] and s['exit_peak_rss_bytes']>0
    assert v['http_key_distribution']['server_process_id']==s['server_pid']
    assert v['quality_passed'] and v['max_abs_error_all_lanes_repeats']<=.001
    assert v['bootstraps']==0
    for kind in ('input','output'):
        data=(path.parent/(v['method']+'_'+kind+'.ct')).read_bytes()
        assert hashlib.sha256(data).hexdigest()==v[kind+'_sha256_last_repeat']
    records.append({'method':v['method'],'maximum_absolute_error':v['max_abs_error_all_lanes_repeats'],
                    'client_pid':s['client_pid'],'server_pid':s['server_pid'],
                    'server_peak_rss_bytes':s['exit_peak_rss_bytes']})
assert len(records)==2
report={'status':'PASS','scope':'fresh encrypted two-process functional smoke; not an Apple M4 benchmark reproduction', 'records':records}
(root/'validation.json').write_text(json.dumps(report,indent=2)+'\n'); print(json.dumps(report,indent=2))
print('Artifacts:',root)
PY
