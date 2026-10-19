"""Standalone scientific figure from measured process-level records."""
import argparse,json,pathlib,statistics
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
ROOT=pathlib.Path(__file__).resolve().parents[1]
p=argparse.ArgumentParser();p.add_argument('--separated',action='store_true');args=p.parse_args()
source='separated_confirmation_final' if args.separated else 'confirmation'
rows=json.loads((ROOT/f'artifacts/{source}/records.json').read_text())['records']
fig,axes=plt.subplots(1,2,figsize=(10,4.5),layout='constrained')
styles=[('baseline','Tuned baseline','#727a84'),('oriented_direct','Oriented: direct','#2269a7'),('oriented_bsgs3','Oriented: BSGS3','#3e8d76')]
for ax,metric,label in [(axes[0],'time','Full encrypted HTTP chain (ms)'),(axes[1],'rss','Server peak RSS (MB)' if args.separated else 'Peak process RSS (MB)')]:
 for k,(method,legend,color) in enumerate(styles):
  heights=[]; lows=[]; highs=[]
  for n in [8,16,32]:
   chosen=[r for r in rows if r['run']['dimension']==n and (r['method']==r['baseline'] if method=='baseline' else r['method']==method)]
   values=[r['run']['results'][0]['median']['complete_local_chain_seconds']*1000 if metric=='time' else (r['run']['results'][0]['server_process']['exit_peak_rss_bytes'] if args.separated else r['peak_rss_bytes'])/1e6 for r in chosen]
   m=statistics.median(values);heights.append(m);lows.append(m-min(values));highs.append(max(values)-m)
  ax.bar([j+(k-1)*.25 for j in range(3)],heights,width=.23,color=color,label=legend,yerr=[lows,highs],capsize=3,error_kw={'linewidth':.9})
 ax.set_xticks(range(3),['8','16','32']);ax.set_xlabel('Input features');ax.set_ylabel(label);ax.set_ylim(bottom=0);ax.grid(axis='y',alpha=.18);ax.set_axisbelow(True);ax.spines[['top','right']].set_visible(False)
axes[0].legend(frameon=False,fontsize=8,loc='upper left');fig.suptitle('Terminal quadratic compilation: batch 128, identical security and error targets',fontsize=11)
fig.supxlabel('Five fresh process/key rounds × 31 encrypted repetitions; bars: median; whiskers: range. '+('Client and server in distinct processes.' if args.separated else 'RSS includes client + server.'),fontsize=8)
dest=ROOT/'artifacts/figures';dest.mkdir(exist_ok=True)
name='separated_time_memory' if args.separated else 'confirmed_time_memory'
fig.savefig(dest/f'{name}.png',dpi=180);fig.savefig(dest/f'{name}.svg')
print(dest/f'{name}.png')
