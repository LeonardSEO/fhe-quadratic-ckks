#!/usr/bin/env python3
"""Cache pinned official-proxy archives with curl when Go TLS is unavailable.
Go still verifies h1 module sums pinned from the already authenticated Lattigo
archive and saved public Go checksum-database record. No version is upgraded.
"""
from pathlib import Path
import concurrent.futures
import hashlib
import json
import os
import subprocess

root=Path(__file__).resolve().parents[1]
bench=root/'bench'
cache=Path(subprocess.check_output(['go','env','GOMODCACHE'],text=True).strip())
# The shipped lockfile works even before any module has been extracted.
expected=(bench/'go.sum').read_text()
sumdb=(root/'artifacts/lattigo-sumdb.txt').read_text()
main_sums='\n'.join(x for x in sumdb.splitlines() if x.startswith('github.com/tuneinsight/lattigo/v6 '))+'\n'+expected
(bench/'go.sum').write_text(main_sums)
modules=[('github.com/tuneinsight/lattigo/v6','v6.2.0')]
modules+=sorted({(line.split()[0],line.split()[1]) for line in expected.splitlines() if '/go.mod' not in line.split()[1]})
def escape(module):return ''.join('!'+c.lower() if c.isupper() else c for c in module)
def fetch(item):
    module,version,suffix=item
    dest=cache/'cache/download'/escape(module)/'@v'/f'{version}.{suffix}'
    dest.parent.mkdir(parents=True,exist_ok=True)
    if not dest.exists():
        tmp=dest.with_suffix(dest.suffix+'.curltmp')
        
        for base in ('https://proxy.golang.org','https://proxy.golang.com.cn','https://goproxy.io'):
            result=subprocess.run(['curl','--fail','--silent','--show-error','--location','--max-time','15','--retry','1',f'{base}/{escape(module)}/@v/{version}.{suffix}','--output',str(tmp)],capture_output=True,text=True)
            if result.returncode==0:break
        else:raise RuntimeError(f'{module}@{version}.{suffix}: '+result.stderr)
        print(f'cached {module}@{version}.{suffix}',flush=True)
        tmp.rename(dest)
    return {'module':module,'version':version,'file':str(dest),'sha256':hashlib.sha256(dest.read_bytes()).hexdigest()}
with concurrent.futures.ThreadPoolExecutor(max_workers=4) as ex:
    jobs={(m,v,s) for m,v in modules for s in ('mod','info','zip')}
    jobs.update((line.split()[0],line.split()[1].removesuffix('/go.mod'),'mod') for line in expected.splitlines() if line.split()[1].endswith('/go.mod'))
    manifest=list(ex.map(fetch,sorted(jobs)))
for module,version in modules:
    list_file=cache/'cache/download'/escape(module)/'@v'/'list'
    versions=set(list_file.read_text().splitlines()) if list_file.exists() else set()
    versions.add(version)
    list_file.write_text('\n'.join(sorted(versions))+'\n')
(root/'artifacts/dependency_archive_sha256.json').write_text(json.dumps(manifest,indent=2)+'\n')
env={**os.environ,'GOSUMDB':'off','GOPROXY':'file://'+str(cache/'cache/download')}
subprocess.run(['go','mod','edit','-require=github.com/tuneinsight/lattigo/v6@v6.2.0'],cwd=bench,env=env,check=True)
subprocess.run(['go','mod','tidy'],cwd=bench,env=env,check=True)
# h1 values consumed by Go must agree with the authenticated pinned archive.
expected_lines=set(main_sums.splitlines())
actual_lines=set((bench/'go.sum').read_text().splitlines())
unknown=actual_lines-expected_lines
if unknown:raise RuntimeError('Unexpected checksums: '+repr(unknown))
print('Offline module resolution succeeded; every resulting h1 is in the pinned source sums.')
