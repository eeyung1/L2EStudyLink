import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { resolve, extname } from 'node:path';
import { chromium } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { verifyReleaseAudit } from './release-audit.mjs';
import { verifyApp } from './workspace.mjs';
import { verifyReflections } from './reflections.mjs';
import { verifyHistoryAndDialogs } from './history.mjs';
import { verifyCommunity } from './community.mjs';
import { verifyLearningCoach } from './learning-coach.mjs';
import { verifyFocusIdentity } from './focus-identity.mjs';
import { verifyPublicAccountAdmin } from './public-account-admin.mjs';
import { verifyBookings } from './bookings.mjs';

const root = process.cwd(), output = 'test-results/ui';
await mkdir(output,{recursive:true});
const mime = {'.html':'text/html','.css':'text/css','.js':'text/javascript','.png':'image/png','.svg':'image/svg+xml','.webmanifest':'application/manifest+json'};
// Read-only localhost fixture host. It never starts the app, accesses a DB or forwards APIs.
const server = createServer(async (req,res) => {
  const path = new URL(req.url,'http://localhost').pathname;
  let file;
  if(path.startsWith('/static/')) file = resolve(root,'.'+path);
  else if(path === '/dashboard' || path === '/timetable' || path === '/reflections' || path === '/search-page' || path === '/my-bookings' || path === '/projects' || path === '/masterclasses' || path === '/buddies' || path === '/planner' || path === '/focus' || path === '/portfolio' || path === '/skills' || path === '/availability' || path === '/admin') file=resolve(root,'templates',({'/buddies':'buddy','/masterclasses':'masterclass'}[path]||path.slice(1))+'.html');
  else if(path.startsWith('/u/')) file=resolve(root,'templates/portfolio.html');
  else if(path === '/learn') file=resolve(root,'templates/learn.html');
  else if(path.startsWith('/learn/')) file=resolve(root,'templates/article.html');
  else if(path === '/') file=resolve(root,'templates/landing.html');
  else if(['/page/login','/page/signup','/page/forgot-password','/page/reset-password'].includes(path)) file=resolve(root,'templates',path.slice(6)+'.html');
  else {res.writeHead(404);res.end();return;}
  if(!file.startsWith(root+'/static/')&&!file.startsWith(root+'/templates/')) {res.writeHead(403);res.end();return;}
  try {let body=await readFile(file);
    if(extname(file)==='.html')body=Buffer.from(body.toString().replace('{{template "app-navigation" .}}',(await readFile(resolve(root,'templates/app-navigation.html'),'utf8')).replace('{{define "app-navigation"}}','').replace('{{end}}','')));res.writeHead(200,{'Content-Type':mime[extname(file)]||'application/octet-stream','Cache-Control':'no-store'});res.end(body);}
  catch {res.writeHead(404);res.end();}
});
await new Promise(resolve => server.listen(0,'127.0.0.1',resolve));
const origin = `http://127.0.0.1:${server.address().port}`;
const browser=await chromium.launch();
const evidence={fixtureOnly:true,sourceCommit:process.env.GITHUB_SHA || 'local',screenshots:[],appObservations:[],previewChecks:[],accessibility:[],externalAssetFailures:[]};
try {
  if(process.env.L2E_UI_SUITE!=='release'){
  // Current integrated app templates and shared navigation, with fictional API fixtures.
  for(const width of [390,1440]) for(const view of ['dashboard','timetable']) for(const state of ['populated','empty','loading','error','expired']) {
    const context=await browser.newContext({viewport:{width,height:950},timezoneId:'Africa/Lagos'});
    const page=await context.newPage();
    await page.clock.setFixedTime(new Date('2026-11-23T10:00:00+01:00'));
    page.on('requestfailed',r=>{if(!r.url().startsWith(origin))evidence.externalAssetFailures.push({url:r.url(),reason:r.failure()?.errorText});});
    await page.route('**/api/v1/**',async route=>{
      const path=new URL(route.request().url()).pathname;
      if(route.request().method()!=='GET') throw Error('Baseline test attempted a write');
      if(state==='loading') {await new Promise(r=>setTimeout(r,4000)); try{await route.fulfill({status:503,json:{error:'Delayed fixture'}});}catch{} return;}
      if(state==='expired' && path==='/api/v1/me') return route.fulfill({status:401,json:{error:'Fixture session expired'}});
      if(state==='error' && path!=='/api/v1/me') return route.fulfill({status:503,json:{error:'Fixture unavailable'}});
      let body=[];
      if(path==='/api/v1/me') body={id:9001,name:'Demo Fellow',email:'demo@example.invalid',rating:0,total_reviews:0,skills:state==='empty'?[]:[{name:'Go',level:'intermediate'}],bio:'Fictional UI fixture',is_admin:false,product_emails:false};
      if(path==='/api/v1/planner/week') body={metrics:{due_blocks:state==='empty'?0:5,reflected_blocks:state==='empty'?0:3}};
      if(path==='/api/v1/me/streaks') body={current_streak:1,longest_streak:3,total_active_days:3,today:'2026-11-23',daily_counts:{'2026-11-23':1}};
      if(path==='/api/v1/timetable' && state!=='empty')body=[{id:101,day_of_week:0,start_time:'09:00',end_time:'11:00',activity:'Build a Go API',goal:'Test validation'}];
      if(path==='/api/v1/reflections' && state!=='empty')body=[{id:201,timetable_id:101,log_date:'2026-11-23',activity_name:'Build a Go API',completed:true,summary:'Fictional reflection fixture'}];
      return route.fulfill({status:200,json:body});
    });
    await page.goto(origin+'/'+view,{waitUntil:'domcontentloaded'});
    if(state==='expired') await page.waitForURL('**/page/login',{waitUntil:'domcontentloaded'});
    else if(state==='populated'||state==='empty') {if(view==='dashboard')await page.locator('#greeting').filter({hasText:'Demo Fellow'}).waitFor();else await page.locator('#day-0').waitFor({state:'attached'});}
    else if(state==='error') {if(view==='dashboard')await page.getByText('Projects are unavailable right now').scrollIntoViewIfNeeded();else await page.getByText(/Could not load timetable/).scrollIntoViewIfNeeded();}
    else await page.waitForTimeout(200);
    const file=`app-${view}-${state}-${width}.png`;
    await page.screenshot({path:output+'/'+file,fullPage:false});
    evidence.screenshots.push(file);
    evidence.appObservations.push({view,state,width,url:new URL(page.url()).pathname,overflow:await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),pageErrors:'Read-only state capture; authenticated server integration remains separate.'});
    await context.close();
  }
  await verifyApp(browser,origin,output,evidence);
  await verifyReflections(browser,origin,output,evidence);
  await verifyHistoryAndDialogs(browser,origin,output,evidence);
  await verifyBookings(browser,origin,output,evidence);
  await verifyCommunity(browser,origin,output,evidence);
  await verifyLearningCoach(browser,origin,output,evidence);
  await verifyFocusIdentity(browser,origin,output,evidence);
  await verifyPublicAccountAdmin(browser,origin,output,evidence);
  for(const theme of ['light','dark']) {
    const context=await browser.newContext({viewport:{width:1440,height:1000},reducedMotion:'reduce'});
    const page=await context.newPage();
    const errors=[];let apiRequests=0;
    page.on('pageerror',e=>errors.push(e.message));
    await page.route('**/api/**',route=>{apiRequests++;return route.abort();});
    await page.goto(origin+'/static/redesign/index.html');
    if(theme==='dark')await page.getByRole('button',{name:'Switch to dark appearance'}).click();
    assert.equal(await page.locator('body').getAttribute('data-l2e-theme'),theme);
    for(const view of ['dashboard','timetable']) {
      await page.locator(`.preview-sidebar [data-view="${view}"]`).click();
      for(const width of [320,360,375,390,412,480,768,820,1024,1280,1440,1920,3840]) {
        await page.setViewportSize({width,height:1000});
        assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,`${theme} ${view} ${width}px overflow`);
        const header=await page.locator('.topbar').boundingBox();
        const heading=await page.locator(`#${view}-view h1`).boundingBox();
        assert.ok(heading.y>=header.y+header.height,`${theme} ${view} header clearance`);
        const controls=await page.locator('button:visible, select:visible').evaluateAll(nodes=>nodes.map(node=>({label:node.textContent.trim(),height:node.getBoundingClientRect().height,width:node.getBoundingClientRect().width})));
        assert.ok(controls.every(c=>c.height>=43.9 && c.width>=43.9),`touch targets: ${JSON.stringify(controls.filter(c=>c.height<43.9||c.width<43.9))}`);
        evidence.previewChecks.push({theme,view,width,overflow:false,headerClearance:true,touchTargets:true});
        if(width===390||width===1440){const file=`preview-${view}-${theme}-${width}.png`;await page.screenshot({path:output+'/'+file,fullPage:width===1440});evidence.screenshots.push(file);}
      }
      await page.setViewportSize({width:390,height:844});
      for(const state of ['populated','empty','loading','error','expired']) {
        await page.selectOption('#preview-state',state);
        const result=await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21aa']).analyze();
        evidence.accessibility.push({theme,view,state,violations:result.violations.map(v=>({id:v.id,impact:v.impact,nodes:v.nodes.map(n=>n.target)}))});
        assert.deepEqual(result.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)})),[],`${theme} ${view} ${state} accessibility`);
        if(state==='error') {await page.getByRole('button',{name:'Try again',exact:true}).click();assert.equal(await page.locator('#preview-state').inputValue(),'populated');}
      }
      await page.selectOption('#preview-state','populated');
      if(view==='timetable'){await page.getByRole('button',{name:'Show Wednesday'}).click();assert.equal(await page.locator('.week-day.selected').getAttribute('aria-label'),'Wednesday sample schedule');}
      // Switch via mobile navigation before the next desktop loop.
      await page.setViewportSize({width:1440,height:1000});
    }
    await page.setViewportSize({width:390,height:844});
    await page.getByRole('button',{name:'More',exact:true}).click();
    await page.keyboard.press('Escape');assert.equal(await page.locator('#more-button').getAttribute('aria-expanded'),'false');
    assert.equal(await page.evaluate(()=>document.activeElement.id),'more-button');
    await page.locator('.bottom-nav [data-action="reflection"]').click();
    await page.getByRole('status').filter({hasText:'Preview only'}).waitFor();
    await page.reload();assert.equal(await page.locator('body').getAttribute('data-l2e-theme'),theme,'theme persistence');
    assert.equal(apiRequests,0,'preview must not call account APIs');assert.deepEqual(errors,[]);
    await context.close();
  }
  }
  await verifyReleaseAudit(browser,origin,output,evidence);
  console.log('UI checks passed: baseline state captures, responsive previews, touch targets, clearance, WCAG scans, state recovery, menus, themes and zero preview API calls.');
} finally {
  await writeFile(output+'/evidence.json',JSON.stringify(evidence,null,2));
  await browser.close();await new Promise(resolve=>server.close(resolve));
}
