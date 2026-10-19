#!/usr/bin/env python3
"""Create/check immutable evidence manifests. No benchmark or Lean cache scanning."""
import argparse,hashlib,json,pathlib
ROOT=pathlib.Path(__file__).resolve().parents[1]
def sha(p):
 h=hashlib.sha256()
 with p.open('rb') as f:
  for b in iter(lambda:f.read(1<<20),b''):h.update(b)
 return h.hexdigest()
def main():
 p=argparse.ArgumentParser();p.add_argument('--write',action='store_true');p.add_argument('--strict',action='store_true');a=p.parse_args()
 target=ROOT/'docs/EVIDENCE_MANIFEST.json'
 if a.write:
  entries=[]
  for path in sorted((ROOT/'artifacts').rglob('*')):
   if not path.is_file() or 'reruns' in path.parts:continue
   relative=path.relative_to(ROOT/'artifacts')
   if len(relative.parts)>1 and relative.parts[0] not in {'final_v2','confirmation','separated_confirmation_final','heldout_inputs'}:continue
   entries.append(dict(path=path.relative_to(ROOT).as_posix(),bytes=path.stat().st_size,sha256=sha(path),supplement=path.suffix in {'.ct','.bin'}))
  target.write_text(json.dumps(dict(scope='Frozen public fixtures, ciphertext evidence, decoded outputs and archived measurement records. Excludes new reruns. Binary .ct/.bin files are distributed separately as a supplement.',files=entries),indent=2)+'\n')
 manifest=json.loads(target.read_text());ok=omitted=0
 for e in manifest['files']:
  path=ROOT/e['path']
  if not path.exists() and e['supplement'] and not a.strict:omitted+=1;continue
  assert path.exists(),e['path'];assert path.stat().st_size==e['bytes'],e['path'];assert sha(path)==e['sha256'],e['path'];ok+=1
 print(f'PASS integrity: {ok} files, {omitted} supplementary binaries absent')
if __name__=='__main__':main()
