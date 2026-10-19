"""Counterbalanced fresh-process confirmation, including real loopback HTTP/RSS.

Selects comparator from the fixed development sweep, never from held-out inputs.
Each process uses fresh keys and encryption; times and OS resource logs retained.
"""
import argparse, datetime, json, os, pathlib, re, subprocess

ROOT=pathlib.Path(__file__).resolve().parents[1]
parser=argparse.ArgumentParser()
parser.add_argument('--rounds',type=int,default=5)
parser.add_argument('--repeats',type=int,default=31)
parser.add_argument('--out',default='reruns/confirmation')
parser.add_argument('--separated',action='store_true')
args=parser.parse_args()
records=[]
names=['synthetic_d8h64seed20261020','synthetic_d16h128seed20261021','synthetic_d32h128seed20261022']
base=ROOT/'artifacts'/args.out
base.mkdir(parents=True,exist_ok=True)
profile=json.loads((ROOT/'artifacts/final_v2/packed/aggregate.json').read_text())
for name in names:
    r=next(r for r in profile['runs'] if r['model']==name)
    baseline=min((v for v in r['results'] if not v['oriented']),key=lambda v:v['median']['complete_local_chain_seconds'])['method']
    methods=[baseline,'oriented_direct','oriented_bsgs3']
    for round_index in range(args.rounds):
        order=methods[:] if round_index%2==0 else list(reversed(methods))
        for method in order:
            dest=base/name/f'round{round_index}'/method
            dest.mkdir(parents=True,exist_ok=True)
            env=os.environ.copy()
            env.update(FHE_PACKED_METHOD=method,FHE_PACKED_TRANSPORT='http')
            if args.separated: env['FHE_PACKED_SERVER_PROCESS']='1'
            command=['/usr/bin/time','-l',str(ROOT/'bench/fhebench'),'-model',name,'-batch','128','-repeats',str(args.repeats),'-out',str(dest)]
            started=datetime.datetime.now(datetime.timezone.utc).isoformat()
            p=subprocess.run(command,cwd=ROOT/'bench',env=env,capture_output=True,text=True)
            (dest/'stdout.log').write_text(p.stdout)
            (dest/'resource.log').write_text(p.stderr)
            if p.returncode: raise RuntimeError(p.stderr+p.stdout)
            run=json.loads((dest/name/'standard/run.json').read_text())
            value=run['results'][0]
            match=re.search(r'(\d+)\s+maximum resident set size',p.stderr)
            if not match: raise RuntimeError('missing process peak RSS')
            record={'model':name,'method':method,'baseline':baseline,'round':round_index,'started':started,'command':command,'environment':{'FHE_PACKED_METHOD':method,'FHE_PACKED_TRANSPORT':'http','FHE_PACKED_SERVER_PROCESS':'1' if args.separated else '0'},'peak_rss_bytes':int(match.group(1)),'resource_scope':'OS time resource report for process tree; not isolated client RSS or simultaneous sum. Isolated server peak is server_process.exit_peak_rss_bytes.' if args.separated else 'one process containing client and server','run':run}
            records.append(record)
            (base/'records.json').write_text(json.dumps({'records':records},indent=2))
            print(name,round_index,method,'server_ms',round(value['median']['server_compute_seconds']*1e3,3),'HTTP_chain_ms',round(value['median']['complete_local_chain_seconds']*1e3,3),'RSS_MB',round(record['peak_rss_bytes']/1e6,1),flush=True)
