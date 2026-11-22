#!/usr/bin/env python3
"""Generate a deterministic source inventory; --check rejects stale audit output."""
from html.parser import HTMLParser
from pathlib import Path
import json,re,sys
class Screen(HTMLParser):
    def __init__(self):
        super().__init__(); self.links=[]; self.styles=[]; self.scripts=[]; self.forms=0; self.dialogs=[]; self.controls=0; self.inline_styles=0
    def handle_starttag(self,tag,attrs):
        a=dict(attrs)
        if tag=='a' and a.get('href'): self.links.append(a['href'])
        if tag=='link' and a.get('rel')=='stylesheet': self.styles.append(a.get('href',''))
        if tag=='script' and a.get('src'): self.scripts.append(a['src'])
        if tag=='form': self.forms+=1
        if tag=='dialog' or a.get('role')=='dialog' or 'modal' in a.get('id','').lower(): self.dialogs.append(a.get('id','unnamed'))
        if tag in ('input','textarea','select','button'): self.controls+=1
        if tag=='style': self.inline_styles+=1
screens={}
for p in sorted(Path('templates').glob('*.html')):
    text=p.read_text()
    if '{{template "app-navigation" .}}' in text: text=text.replace('{{template "app-navigation" .}}',Path('templates/app-navigation.html').read_text())
    parser=Screen(); parser.feed(text); screens[str(p)]={k:v for k,v in vars(parser).items() if k in ('links','styles','scripts','forms','dialogs','controls','inline_styles')}
source=Path('main.go').read_text()
routes=[{'method':m,'path':p,'scope':scope} for scope,m,p in re.findall(r'\b(router|api|protected|admin)\.(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\("([^"]+)"',source)]
# Registered route groups live outside main.go too; capture them explicitly.
registrations=[]
for p in sorted(Path('handlers').glob('*.go')):
    if p.name.endswith('_test.go'): continue
    for method,path in re.findall(r'\b(?:protected|group|api|routes|router)\.(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\("([^"]+)"',p.read_text()): registrations.append({'source':str(p),'method':method,'path':path})
report={'foundation_baseline_commit':'5e2309e14a98ed7f03aca5c8620b599cbdb1c581','notes':['Paths retain the route-group spelling in source; api/protected groups use /api/v1 and admin uses /api/v1/admin.','Source inventory is not a production authorization or device audit.','Generic /page/:name and article/portfolio dynamic routes require representative runtime coverage.'], 'main_routes':routes,'handler_routes':registrations,'screens':screens}
output=json.dumps(report,indent=2,ensure_ascii=False)+'\n'; target=Path('docs/redesign/source-inventory.json')
if '--check' in sys.argv:
    if not target.exists() or target.read_text()!=output: sys.exit('UI inventory is stale: run python3 scripts/audit-ui.py')
else: target.write_text(output)
print(f'Inventoried {len(screens)} templates, {len(routes)} main route declarations and {len(registrations)} grouped handler routes.')
