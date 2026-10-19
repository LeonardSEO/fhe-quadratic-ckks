#!/usr/bin/env python3
"""Release gates runnable from any working directory in a clean checkout.
Historical timing evidence and new functional checks remain separate.
"""
import argparse, os, pathlib, re, shutil, subprocess, sys
ROOT=pathlib.Path(__file__).resolve().parents[1]
def run(argv, cwd=ROOT):
    print('+ '+' '.join(map(str,argv)),flush=True)
    subprocess.run(list(map(str,argv)),cwd=cwd,check=True)
def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--phase',choices=['all','evidence','go','lean','paper'],default='all')
    p.add_argument('--strict-ciphertexts',action='store_true',help='require extracted raw ciphertext supplement')
    p.add_argument('--smoke',action='store_true',help='execute a fresh two-process encrypted functional check')
    args=p.parse_args(); phase=lambda name:args.phase in ('all',name)
    if phase('evidence'):
        run([sys.executable,'scripts/integrity.py']+(['--strict'] if args.strict_ciphertexts else []))
        run([sys.executable,'scripts/check_correspondence.py','--check'])
        run([sys.executable,'scripts/verify_evidence.py']+(['--strict-ciphertexts'] if args.strict_ciphertexts else []))
        run([sys.executable,'scripts/generate_paper_data.py','--check'])
    if phase('go'):
        run(['go','mod','verify'],ROOT/'bench')
        run(['go','test','-count=1','./...'],ROOT/'bench')
        run(['go','vet','./...'],ROOT/'bench')
        if args.smoke:run(['bash','scripts/run_smoke.sh'])
    if phase('lean'):
        run(['lake','build'],ROOT/'lean')
        run([sys.executable,'check_axioms.py'],ROOT/'lean')
    if phase('paper'):
        epoch=(ROOT/'paper.build_epoch').read_text().strip()
        assert epoch.isdecimal() and int(epoch)>0, 'Invalid pinned PDF build epoch'
        os.environ['SOURCE_DATE_EPOCH']=epoch
        os.environ['TZ']='UTC'
        print('Pinned paper SOURCE_DATE_EPOCH='+epoch,flush=True)
        tectonic=os.environ.get('CODEX_TECTONIC_PATH') or shutil.which('tectonic')
        if tectonic:
            bundle=os.environ.get('TECTONIC_BUNDLE')
            run([tectonic]+(['--bundle',bundle] if bundle else [])+['--keep-logs','--keep-intermediates','paper.tex'])
        else:run(['latexmk','-pdf','-halt-on-error','-interaction=nonstopmode','paper.tex'])
        pdf=ROOT/'paper.pdf';assert pdf.read_bytes().startswith(b'%PDF-'), 'Missing PDF'
        log=(ROOT/'paper.log').read_text(errors='replace')
        bad=[line for line in log.splitlines() if re.search(r'(Citation|Reference).*undefined|There were undefined|Label\(s\) may have changed|! LaTeX Error|Emergency stop',line)]
        assert not bad,'Unresolved LaTeX diagnostics: '+str(bad)
        if shutil.which('pdftotext'):
            text=subprocess.run(['pdftotext','paper.pdf','-'],cwd=ROOT,check=True,text=True,capture_output=True).stdout
            assert '??' not in text,'Unresolved reference marker in extracted PDF'
            print('PASS PDF text extraction; no unresolved reference markers')
    print('PASS release gate:',args.phase,flush=True)
if __name__=='__main__':main()
