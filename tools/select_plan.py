"""Deterministic setup/throughput/key-memory planner from measured profiles."""
import argparse,json,pathlib
ROOT=pathlib.Path(__file__).resolve().parents[1]
p=argparse.ArgumentParser();p.add_argument('--dimension',type=int,choices=[8,16,32],required=True);p.add_argument('--inferences',type=int,default=100);p.add_argument('--bandwidth-mbps',type=float,default=0,help='0 uses measured loopback only; positive adds conservative payload/BW time');p.add_argument('--max-eval-key-mb',type=float,default=0);p.add_argument('--max-rss-mb',type=float,default=0);a=p.parse_args()
if a.inferences<1:raise ValueError('positive inference horizon required')
profile_path=ROOT/'artifacts/cost_profiles_separated.json'
if not profile_path.exists():profile_path=ROOT/'artifacts/cost_profiles.json'
profiles=json.loads(profile_path.read_text())['profiles'];rows=[]
for v in profiles:
 if v['dimension']!=a.dimension:continue
 if a.max_eval_key_mb and v['evaluation_key_bytes']>a.max_eval_key_mb*1e6:continue
 if a.max_rss_mb and v['peak_rss_range_bytes'][1]>a.max_rss_mb*1e6:continue
 total=v['public_frontend_seconds']+v['setup_seconds']+a.inferences*v['chain_seconds']
 if a.bandwidth_mbps:total+=(v['evaluation_key_bytes']+a.inferences*(v['input_bytes']+v['output_bytes']))/(a.bandwidth_mbps*1e6/8)
 rows.append({'method':v['method'],'estimated_total_seconds':total,'profile':v})
if not rows:raise ValueError('no measured plan meets resource budgets')
rows.sort(key=lambda v:v['estimated_total_seconds']);print(json.dumps({'selected':rows[0]['method'],'profile':str(profile_path),'all_feasible':rows,'qualification':'measured M4/B128 profile, same input format; bandwidth is an explicit model, no WAN RTT estimate; RSS budget uses worst of5 observed server peaks in separated profile; no performance guarantee on other hardware'},indent=2))
