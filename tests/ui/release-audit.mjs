import assert from 'node:assert/strict';
import AxeBuilder from '@axe-core/playwright';

const widths=[320,360,375,390,412,480,768,820,1024,1280,1440,1920,2560,3840];
const views=['/','/page/login','/page/signup','/page/forgot-password','/page/reset-password','/learn','/learn/audit-guide','/dashboard','/timetable','/reflections','/search-page','/my-bookings','/projects','/masterclasses','/buddies','/planner','/focus','/portfolio','/u/audit_learner','/skills','/availability','/admin'];
const metrics={week_start:'2026-11-16',week_end:'2026-11-23',timezone:'UTC',as_of:'2026-11-23T12:00:00Z',scheduled_minutes:0,reflected_minutes:0,due_blocks:0,adherence_percent:0,missing_reflections:0,active_days:0,streak_change:0,streak_at_end:0,pending_reflections:0,unmatched_reflections:0,days:[]};
export async function verifyReleaseAudit(browser,origin,output,evidence){
 const findings=[];evidence.releaseFindings=findings;
 evidence.releaseMatrix=[];evidence.releaseStates=[];evidence.releaseInteractions=[];
 for(const theme of ['light','dark']){
 const context=await browser.newContext({viewport:{width:390,height:950},reducedMotion:'reduce'});await context.addInitScript(t=>localStorage.setItem('darkMode',String(t==='dark')),theme);const page=await context.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
 let mode='ready',profileWrites=0,savedBio='A saved learning bio',holds=[],admin=true;
 const projects=[1,2].map(id=>({id,title:'Audit project '+id,description:'A useful shared practice project.',roles_needed:'Learners',time_commitment:'One hour',owner_id:1,owner_name:'Audit learner',status:'open',member_count:0,members:[]}));
 await page.route('**/api/v1/**',async r=>{const p=new URL(r.request().url()).pathname,m=r.request().method();
 if(mode==='expired')return r.fulfill({status:401,json:{error:'Sign in required'}});
 if(mode==='offline')return r.abort('internetdisconnected');
 if(mode==='error'&&p!=='/api/v1/me')return r.fulfill({status:503,json:{error:'Audit service unavailable'}});
 if(mode==='slow'&&p!=='/api/v1/me'){await new Promise(resolve=>holds.push(resolve));}
 if(m!=='GET'){
 if(p==='/api/v1/profile'){profileWrites++;await new Promise(resolve=>setTimeout(resolve,300));if(mode==='save-error')return r.fulfill({status:503,json:{error:'Audit save unavailable'}});savedBio=r.request().postDataJSON().bio;return r.fulfill({json:{message:'Saved'}});}
 throw Error('Unlisted release mutation '+m+' '+p);
 }
 let body=[];
 if(p==='/api/v1/me')body={id:1,name:'Audit learner',email:'audit@example.invalid',is_admin:admin,product_emails:false,bio:savedBio,skills:[{name:'Go',proficiency:'beginner'}]};
 else if(p==='/api/v1/me/streaks')body={current_streak:0,longest_streak:0,total_active_days:0,today:'2026-11-23',daily_counts:{}};
 else if(p==='/api/v1/planner/week')body={metrics,coaching_available:false};
 else if(p==='/api/v1/booking-clock')body={timezone:'UTC',today:'2026-11-23',now:'2026-11-23T10:00:00Z'};
 else if(p==='/api/v1/reflections/retention')body={retains_after_block_delete:true};
 else if(p==='/api/v1/buddies')body={me:{current_streak:0,longest_streak:0,total_active_days:0},active:[],incoming:[],outgoing:[],suggestions:[]};
 else if(p==='/api/v1/masterclasses')body={user_id:1,items:[],has_more:false};
 else if(p==='/api/v1/focus/rooms')body={rooms:[],current_room_id:''};
 else if(p.startsWith('/api/v1/portfolio/'))body={is_owner:p.endsWith('/me'),portfolio_public:true,handle:'audit_learner',name:'Audit learner',bio:savedBio,rating:0,total_reviews:0,completed_sessions:0,scheduled_session_hours:0,current_streak:0,longest_streak:0,skills:[],reviews:[],projects:[],activity_days:{}};
 else if(p==='/api/v1/collaboration/preferences')body={open_to_invites:false};
 else if(p==='/api/v1/projects')body=projects;
 else if(/^\/api\/v1\/projects\/\d+$/.test(p))body=projects.find(v=>v.id===Number(p.split('/').at(-1)));
 else if(p==='/api/v1/articles')body={articles:[{slug:'audit-guide',title:'A useful practice guide',summary:'A small public learning exercise.',category:'Practice',published_at:'2026-11-23T10:00:00Z'}]};
 else if(p.endsWith('/comments'))body={comments:[]};
 else if(p==='/api/v1/articles/audit-guide')body={title:'A useful practice guide',summary:'A small public learning exercise.',category:'Practice',published_at:'2026-11-23T10:00:00Z',body:'Choose one outcome. Practice and reflect.'};
 else if(p==='/api/v1/admin/stats')body={total_users:1,total_bookings:0,total_skills:1};
 else if(p==='/api/v1/admin/product-email-audience')body={eligible:0};
 return r.fulfill({json:body});
 });
 async function geometry(view,width,zoom=false){
 assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false,`${view} ${theme} ${width} overflow`);
 const main=await page.locator('main').boundingBox();assert.ok(main&&main.width>0,view+' main visible');
 if(view==='/'){
  const nav=await page.locator('.reading-header').boundingBox(),title=await page.locator('#heroTitle').boundingBox(),eyebrow=await page.locator('.landing-eyebrow').boundingBox();
  assert.ok(eyebrow.y>=nav.y+nav.height+20,'landing header/hero clearance');
  assert.ok(title.y>=eyebrow.y+eyebrow.height,'landing title/eyebrow separation');
  const cta=await page.locator('.landing-copy .reading-button:visible').boundingBox();
  if(!zoom)assert.ok(cta.y+cta.height<=950,'landing primary action visible on first screen');
  const tiny=await page.locator('.reading-header a:visible,.reading-header button:visible,.journey-tabs button:visible').evaluateAll(ns=>ns.filter(n=>{const r=n.getBoundingClientRect();return r.width<43.9||r.height<43.9}).map(n=>n.textContent));
  assert.deepEqual(tiny,[],'landing navigation and preview touch targets');
 }
 const header=page.locator('.sidebar-heading:visible');if(await header.count()&&width<=860){const b=await header.boundingBox(),h=await page.locator('main h1').first().boundingBox();if(h)assert.ok(h.y>=b.y+b.height-1,view+' header clearance');}
 const bad=await page.locator('main input:visible:not([type=checkbox]):not([type=radio]),main select:visible,main textarea:visible').evaluateAll(ns=>ns.filter(n=>parseFloat(getComputedStyle(n).fontSize)<16).map(n=>n.id));assert.deepEqual(bad,[],view+' input text size');
 const small=await page.locator('main button:visible:not([role=tab])').evaluateAll(ns=>ns.filter(n=>{const b=n.getBoundingClientRect();return b.width<43.9||b.height<43.9}).map(n=>({id:n.id,text:n.textContent.trim()})));assert.deepEqual(small,[],view+' primary button touch size');
 evidence.releaseMatrix.push({view,theme,width,zoom,overflow:false,headerClearance:true,inputSize:true});
 }
 for(const view of views){mode='ready';await page.goto(origin+view,{waitUntil:'networkidle'});
 if(view==='/dashboard')assert.equal(await page.locator('.learning-actions a').count(),3);
 if(view==='/timetable'){await page.locator('#showToday').click();const today=await page.evaluate(()=>(new Date().getDay()+6)%7);assert.equal(await page.locator('.timetable-day-nav [data-day="'+today+'"]').getAttribute('aria-pressed'),'true');}
 if(view==='/reflections'){await page.locator('#reflectionOrder').selectOption('oldest');await page.locator('[data-reflection-format="voice"]').click();assert.equal(await page.locator('#reflectionMediaFilter').inputValue(),'voice');await page.locator('[data-reflection-format="all"]').click();}
 if(view==='/search-page'){await page.locator('#searchButton:enabled').waitFor();await page.locator('#skillSearch').fill('Temporary filter');await page.locator('#clearTutorSearch').click();assert.equal(await page.locator('#skillSearch').inputValue(),'');}
 if(view==='/my-bookings'){await page.locator('[data-session-status="pending"]').click();assert.equal(await page.locator('#sessionStatus').inputValue(),'pending');await page.locator('[data-session-status="all"]').click();}
 if(view==='/availability'){await page.locator('#availDay:enabled').waitFor();await page.locator('[data-availability-day="2"]').click();assert.equal(await page.locator('#availDay').inputValue(),'2');await page.locator('[data-availability-day="0"]').click();}
 if(view==='/planner'){await page.locator('#week-start:enabled').waitFor();const before=await page.locator('#week-start').inputValue();await page.locator('[data-week-shift="-1"]').click();const shifted=await page.locator('#week-start').inputValue();assert.equal((Date.parse(before)-Date.parse(shifted))/86400000,7);await page.locator('#week-start:enabled').waitFor();await page.locator('[data-week-shift="1"]').click();await page.locator('#week-start:enabled').waitFor();assert.equal(await page.locator('#week-start').inputValue(),before);}
 if(view==='/focus')assert.equal(await page.locator('#focusQuiet').isDisabled(),true);
 if(view==='/buddies'){await page.locator('#buddySuggestionSearch').fill('No matching suggestion');assert.equal(await page.locator('#buddySearchStatus').isVisible(),true);await page.locator('#buddySuggestionSearch').fill('');}
 if(view==='/masterclasses'){await page.locator('#classOpenOnly').check();assert.equal(await page.locator('#classOpenOnly').isChecked(),true);await page.locator('#classOpenOnly').uncheck();}
 for(const width of widths){await page.setViewportSize({width,height:950});try{await geometry(view,width);}catch(e){findings.push({view,theme,width,error:e.message});}}
 await page.setViewportSize({width:390,height:950});
 const scan=await new AxeBuilder({page}).include('main').withTags(['wcag2a','wcag2aa','wcag21aa']).analyze();for(const v of scan.violations)findings.push({view,theme,rule:v.id,nodes:v.nodes.map(n=>n.target)});
 // 200% reflow: the 1440x1000 desktop content area becomes 720x500 CSS pixels.
 await page.setViewportSize({width:720,height:500});try{await geometry(view,720,true);}catch(e){findings.push({view,theme,width:720,reflow:true,error:e.message});}
 if(['/','/page/login','/admin'].includes(view)){const f=`release-${view==='/'?'landing':view.split('/').at(-1)}-${theme}-720.png`;await page.screenshot({path:output+'/'+f,fullPage:true});evidence.screenshots.push(f);}
 // Keyboard focus must remain visibly indicated on an actionable main control.
 const control=page.locator('main a[href]:visible,main button:visible:not([disabled]),main input:visible:not([disabled])').first();if(await control.count()){await control.focus();await page.keyboard.press('Shift+Tab');await page.keyboard.press('Tab');const focused=await page.evaluate(()=>{const n=document.activeElement,s=getComputedStyle(n);return {tag:n.tagName,visible:n.getClientRects().length>0,indicated:s.outlineStyle!=='none'&&parseFloat(s.outlineWidth)>0||s.boxShadow!=='none'};});if(!focused.visible||!focused.indicated)findings.push({view,theme,error:'Keyboard focus missing',focused});}
 }
 console.log('Release matrix findings so far: '+JSON.stringify(findings));
 // Each settings read state must recover through an explicit retry without losing a draft.
 await page.setViewportSize({width:390,height:844});mode='error';await page.goto(origin+'/availability');await page.locator('#availability').getByRole('button',{name:'Try again'}).waitFor();assert.equal(await page.locator('#availDay').isDisabled(),true);mode='ready';await page.locator('#availability').getByRole('button',{name:'Try again'}).click();await page.locator('#availDay:enabled').waitFor();evidence.releaseStates.push({theme,state:'error → retry → empty',view:'availability'});
 mode='slow';await page.goto(origin+'/availability',{waitUntil:'domcontentloaded'});await page.waitForFunction(()=>document.querySelector('#availability').getAttribute('aria-busy')==='true');assert.equal(await page.locator('#availDay').isDisabled(),true);await page.waitForTimeout(100);mode='ready';holds.splice(0).forEach(resolve=>resolve());await page.locator('#availDay:enabled').waitFor();evidence.releaseStates.push({theme,state:'slow → ready',view:'availability'});
 mode='offline';await page.goto(origin+'/skills');await page.locator('#retryProfile').waitFor();mode='ready';await page.locator('#retryProfile').click();await page.locator('#accountProfileSave:enabled').waitFor();evidence.releaseStates.push({theme,state:'offline → retry',view:'settings'});
 mode='save-error';await page.locator('#accountBio').fill('My retained draft');await page.locator('#accountProfileSave').click();await page.locator('#accountProfileStatus').getByText(/entries are kept/).waitFor();assert.equal(await page.locator('#accountBio').inputValue(),'My retained draft');mode='ready';const count=profileWrites;await page.evaluate(()=>{const f=document.getElementById('accountProfileForm');f.requestSubmit();f.requestSubmit();});await page.locator('#accountProfileStatus').getByText('Profile saved.',{exact:true}).waitFor();assert.equal(profileWrites,count+1);await page.locator('#accountBio').fill('A second deliberate save');await page.locator('#accountProfileSave').click();await page.locator('#accountProfileStatus').getByText('Profile saved.',{exact:true}).waitFor();assert.equal(profileWrites,count+2);evidence.releaseInteractions.push({theme,checks:'failure draft retention, pending double-submit blocked, second deliberate save accepted'});
 await page.goto(origin+'/projects');await page.locator('#projectList').getByRole('button',{name:'View project'}).first().click();await page.waitForURL('**/projects?project=1');await page.locator('#detailsHeading').getByText('Audit project 1',{exact:true}).waitFor();await page.locator('#projectList').getByRole('button',{name:'View project'}).nth(1).click();await page.waitForURL('**/projects?project=2');await page.goBack();await page.locator('#detailsHeading').getByText('Audit project 1',{exact:true}).waitFor();await page.goForward();await page.locator('#detailsHeading').getByText('Audit project 2',{exact:true}).waitFor();evidence.releaseInteractions.push({theme,checks:'project deep-link back and forward restore the selected record'});
 await page.goto(origin+'/skills');await page.locator('#skills').getByRole('button',{name:'Remove Go',exact:true}).click();await page.getByRole('dialog').waitFor();assert.equal(await page.getByRole('button',{name:'Keep unchanged',exact:true}).evaluate(n=>document.activeElement===n),true);await page.keyboard.press('Shift+Tab');assert.equal(await page.getByRole('dialog').evaluate(d=>d.contains(document.activeElement)),true);await page.keyboard.press('Tab');await page.keyboard.press('Escape');await page.getByRole('dialog').waitFor({state:'hidden'});assert.equal(await page.getByRole('button',{name:'Remove Go',exact:true}).evaluate(n=>document.activeElement===n),true);evidence.releaseInteractions.push({theme,checks:'dialog keyboard containment, Escape cancellation and opener restoration'});
 for(const view of ['/skills','/availability','/admin','/projects','/masterclasses','/buddies','/planner','/focus']){mode='expired';await page.goto(origin+view,{waitUntil:'domcontentloaded'});await page.waitForURL('**/page/login',{waitUntil:'domcontentloaded'});evidence.releaseStates.push({theme,state:'expired → login',view});}mode='ready';
 assert.deepEqual(errors,[],theme+' release page errors');await context.close();
 }
 assert.deepEqual(findings,[],'Release geometry and accessibility findings');
}
