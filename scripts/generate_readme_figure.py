#!/usr/bin/env python3
"""Generate a deterministic SVG from checked, frozen paper aggregates (stdlib only)."""
import argparse, json
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
def render():
    records=json.loads((ROOT/'generated/paper_data.json').read_text())['separate_process']
    lookup={(v['dimension'],v['method']):v for v in records}
    plans=[(8,'triangular_direct','oriented_direct'),(16,'triangular_direct','oriented_direct'),(32,'symmetric_bsgs3','oriented_bsgs3')]
    out=['<svg xmlns="http://www.w3.org/2000/svg" width="960" height="430" viewBox="0 0 960 430" role="img" aria-labelledby="title desc">',
         '<title id="title">Complete encrypted inference: baseline versus cyclic orientation</title>',
         '<desc id="desc">Separate client and server processes, batch 128. Baseline and oriented full inference times in milliseconds at dimensions 8, 16 and 32. Bars begin at zero; lower is better.</desc>',
         '<rect width="960" height="430" fill="#fff"/>',
         '<style>text{font-family:Arial,Helvetica,sans-serif;fill:#17212b}.title{font-size:23px;font-weight:700}.sub{font-size:14px;fill:#526170}.label{font-size:16px;font-weight:700}.value{font-size:14px}.tick{font-size:12px;fill:#64717d}.gain{font-size:15px;font-weight:700;fill:#16634b}</style>',
         '<text x="36" y="39" class="title">Complete encrypted inference, less work</text>',
         '<text x="36" y="65" class="sub">Full loopback chain in milliseconds · same model, parameters and error requirement</text>',
         '<rect x="36" y="86" width="15" height="11" fill="#8c99a5"/><text x="59" y="97" class="sub">Optimized baseline</text>',
         '<rect x="242" y="86" width="15" height="11" fill="#176b9a"/><text x="265" y="97" class="sub">Cyclic orientation</text>']
    origin=132; scale=30
    for tick in (0,5,10,15,20):
        x=origin+tick*scale
        out += [f'<line x1="{x}" y1="124" x2="{x}" y2="345" stroke="#e7ebee"/>',f'<text x="{x}" y="367" class="tick" text-anchor="middle">{tick}</text>']
    for i,(n,bname,cname) in enumerate(plans):
        b,c=lookup[n,bname],lookup[n,cname];y=137+i*76
        improvement=100*(1-c['chain_ms']/b['chain_ms'])
        out += [f'<text x="36" y="{y+22}" class="label">d = {n}</text>']
        for j,(record,color) in enumerate(((b,'#8c99a5'),(c,'#176b9a'))):
            width=record['chain_ms']*scale; yy=y+j*24
            out += [f'<rect x="{origin}" y="{yy}" width="{width:.5f}" height="17" rx="2" fill="{color}"/>',f'<text x="{origin+width+8:.5f}" y="{yy+13}" class="value">{record["chain_ms"]:.3f}</text>']
        out += [f'<text x="845" y="{y+28}" class="gain" text-anchor="end">−{improvement:.1f}%</text>']
    out += ['<text x="132" y="390" class="sub">Lower is better</text>',
            '<text x="36" y="416" class="tick">Apple M4 · batch 128 · five process/key rounds × 31 inferences · median of process medians</text>','</svg>']
    return '\n'.join(out)+'\n'
def main():
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--check',action='store_true');a=p.parse_args()
    target=ROOT/'generated/results.svg';payload=render()
    if a.check:assert target.read_text()==payload,'Run scripts/generate_readme_figure.py'
    else:target.write_text(payload)
    print('PASS deterministic README figure')
if __name__=='__main__':main()
