#!/usr/bin/env python3
"""Deterministic stdlib aggregation from frozen raw records; no benchmark reruns."""
import json,math,pathlib,statistics,sys
ROOT=pathlib.Path(__file__).resolve().parents[1];A=ROOT/'artifacts';OUT=ROOT/'generated';OUT.mkdir(exist_ok=True)
med=statistics.median

def load(p):return json.loads((A/p).read_text())
def setup(r,separated):
    run=r['run'];v=run['results'][0]
    return math.fsum(run.get(k,0) for k in ('parameter_construction_seconds','client_keypair_generation_seconds','public_key_serialize_seconds'))+math.fsum(v.get(k,0) for k in ('model_prepare_seconds','evaluation_key_generation_seconds','key_serialize_seconds','http_setup_seconds','warmup_client_encode_encrypt_seconds','warmup_total_seconds'))+v['http_key_distribution']['roundtrip_seconds']+(v['server_process']['shutdown_seconds'] if separated else 0)
def profiles(arm):
    records=load(arm+'/records.json')['records'];out=[];separated=arm=='separated_confirmation_final'
    for n in (8,16,32):
        rows=[r for r in records if r['run']['dimension']==n]
        base={r['round']:r for r in rows if r['method']==r['baseline']}
        for method in dict.fromkeys(r['method'] for r in rows):
            rs=[r for r in rows if r['method']==method];v=rs[0]['run']['results'][0]
            ratios=[base[r['round']]['run']['results'][0]['median']['complete_local_chain_seconds']/r['run']['results'][0]['median']['complete_local_chain_seconds'] for r in rs]
            out.append(dict(dimension=n,method=method,server_ms=med(r['run']['results'][0]['median']['server_compute_seconds'] for r in rs)*1000,chain_ms=med(r['run']['results'][0]['median']['complete_local_chain_seconds'] for r in rs)*1000,rss_MB=med(r['run']['results'][0]['server_process']['exit_peak_rss_bytes'] if separated else r['peak_rss_bytes'] for r in rs)/1e6,keys_MB=v['evaluation_key_serialized_bytes']/1e6,setup_ms=med(setup(r,separated) for r in rs)*1000,diagonals=v['diagonals'],rotations=v['total_rotations'],rotation_keys=v['rotation_key_count'],pt_muls=v['plaintext_multiplications'],ct_muls=v['ciphertext_multiplications'],depth=v['multiplicative_depth'],paired_median=med(ratios),paired_min=min(ratios),paired_max=max(ratios),paired_wins=sum(x>1 for x in ratios)))
    return out

def tex_escape(s):return s.replace('_',r'\_')
def table(rows):
    lines=[r'\begin{tabular}{rlrrrrr}',r'\toprule',r'$d$ & Plan & Server ms & Chain ms & RSS MB & Keys MB & Setup ms \\',r'\midrule']
    for p in rows:lines.append(f"{p['dimension']} & "+tex_escape(p['method'])+f" & {p['server_ms']:.3f} & {p['chain_ms']:.3f} & {p['rss_MB']:.2f} & {p['keys_MB']:.3f} & {p['setup_ms']:.2f}"+r' \\')
    return '\n'.join(lines+[r'\bottomrule',r'\end{tabular}'])+'\n'

def main():
    sep=profiles('separated_confirmation_final');combined=profiles('confirmation');sweep={}
    for mode in ('packed','factors','network','feature_matched','ci'):
        for r in load('final_v2/'+mode+'/aggregate.json')['runs']:
            for v in r['results']:
                if not v['method'].startswith('oriented_'):sweep.setdefault(r['model'],[]).append((mode,v))
    broad=[]
    for r in load('final_v2/packed/aggregate.json')['runs']:
        mode,b=min(sweep[r['model']],key=lambda z:z[1]['median']['complete_local_chain_seconds']);c=min((v for v in r['results'] if v['oriented']),key=lambda v:v['median']['complete_local_chain_seconds'])
        broad.append(dict(model=r['model'],baseline=mode+'/'+b['method'],candidate=c['method'],baseline_ms=b['median']['complete_local_chain_seconds']*1000,candidate_ms=c['median']['complete_local_chain_seconds']*1000,speedup=b['median']['complete_local_chain_seconds']/c['median']['complete_local_chain_seconds']))
    data=dict(separate_process=sep,combined_process=combined,broad_sweep=broad,validation_batches=len(load('heldout_inputs/summary.json')['records']),validation_max_error=load('heldout_inputs/summary.json')['max_error'])
    outputs={'paper_data.json':json.dumps(data,indent=2)+'\n','results_tables.tex':table(sep),'combined_table.tex':table(combined)}
    lines=[r'\begin{tabular}{llrrr}',r'\toprule',r'Model & Baseline & Base ms & Cand. ms & Speedup \\',r'\midrule']
    labels=['Archive 8/64','Development 8/64','Development 16/128','Development 32/128','Held-out 8/96','Held-out 16/96','SPD 16/64']
    for label,p in zip(labels,broad):lines.append(label+' & '+tex_escape(p['baseline'])+f" & {p['baseline_ms']:.3f} & {p['candidate_ms']:.3f} & {p['speedup']:.3f}"+r' \\')
    outputs['sweep_table.tex']='\n'.join(lines+[r'\bottomrule',r'\end{tabular}'])+'\n'
    # pgfplots source is a vector figure compiled by the paper's TeX engine.
    fig=[r'\begin{tikzpicture}',r'\begin{axis}[ybar,bar width=9pt,width=0.95\linewidth,height=4cm,ylabel={Ratio to baseline},symbolic x coords={d8 direct,d8 BSGS,d16 direct,d16 BSGS,d32 direct,d32 BSGS},xtick=data,x tick label style={rotate=30,anchor=east,font=\scriptsize},ymin=0,ymax=1.6,legend style={font=\scriptsize,at={(0.5,1.03)},anchor=south,legend columns=2}]']
    for metric,label in [('chain_ms','Recurring chain time'),('rss_MB','Server peak RSS')]:
        coords=[]
        for p in sep:
            if not p['method'].startswith('oriented'):continue
            b=next(z for z in sep if z['dimension']==p['dimension'] and not z['method'].startswith('oriented'))
            x=f"d{p['dimension']} "+('direct' if p['method']=='oriented_direct' else 'BSGS');coords.append(f"({x},{p[metric]/b[metric]:.8f})")
        fig.append(r'\addplot coordinates {'+' '.join(coords)+'};');fig.append(r'\addlegendentry{'+label+'}')
    fig.extend([r'\end{axis}',r'\end{tikzpicture}']);outputs['tradeoffs_figure.tex']='\n'.join(fig)+'\n'
    for name,content in outputs.items():
        if '--check' in sys.argv:assert (OUT/name).read_text()==content,name
        else:(OUT/name).write_text(content)
    print('PASS deterministic paper data: 9 separated profiles, 9 combined profiles, 7 sweep rows')
if __name__=='__main__':main()
