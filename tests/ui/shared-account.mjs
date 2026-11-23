import assert from 'node:assert/strict';
import AxeBuilder from '@axe-core/playwright';
export async function verifySharedAccount(browser,origin,output,evidence){
 evidence.sharedAccountChecks=[];
 for(const theme of ['light','dark']){
  const context=await browser.newContext({viewport:{width:390,height:844},reducedMotion:'reduce',colorScheme:theme});
  const page=await context.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
  let pending=null,requests=0;
  await page.route('**/api/v1/**',async r=>{
   if(r.request().method()!=='GET'){requests++;pending=r;return;}
   await r.fulfill({json:{id:1,name:'Fixture learner',email:'fixture@example.invalid',bio:'A learning bio',skills:[{name:'Go',proficiency:'beginner'}],product_emails:false,is_admin:false}});
  });
  for(const mode of ['login','signup','forgot-password','reset-password']){
   await page.goto(origin+'/page/'+mode,{waitUntil:'networkidle'});
   assert.equal(await page.locator('html').evaluate(n=>n.classList.contains('dark')),theme==='dark','system appearance without saved preference');
   for(const width of [320,390,768,1440]){
    await page.setViewportSize({width,height:844});
    assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false,mode+' overflow '+width);
    const header=await page.locator('.reading-header').boundingBox(),main=await page.locator('main').boundingBox();
    assert.ok(main.y>=header.y+header.height+16,mode+' header clearance');
    const small=await page.locator('.reading-header a:visible,.reading-header button:visible,main button:visible').evaluateAll(ns=>ns.filter(n=>{const r=n.getBoundingClientRect();return r.width<43.9||r.height<43.9}).map(n=>n.textContent));assert.deepEqual(small,[],mode+' touch targets');
    if(width===390||width===1440){const scan=await new AxeBuilder({page}).include('main').include('.reading-header').withTags(['wcag2a','wcag2aa','wcag21aa']).analyze();assert.deepEqual(scan.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)})),[],mode+' '+theme);const f=`first10-${mode}-${theme}-${width}.png`;await page.screenshot({path:output+'/'+f,fullPage:true});evidence.screenshots.push(f);}
   }
  }
  await page.goto(origin+'/page/login');await page.locator('#email').fill('fixture@example.invalid');await page.locator('#password').fill('fixture-password');await page.locator('#accountSubmit').click();
  await page.waitForFunction(()=>document.querySelector('#accountForm').getAttribute('aria-busy')==='true');assert.equal(requests,1);assert.equal(await page.locator('#accountSubmit').isDisabled(),true);
  await pending.fulfill({status:503,json:{error:'Fixture retry needed'}});await page.locator('#accountMessage').getByText(/Fixture retry needed/).waitFor();
  assert.equal(await page.locator('#password').inputValue(),'fixture-password');assert.equal(await page.locator('#accountForm').getAttribute('aria-busy'),'false');assert.equal(await page.evaluate(()=>document.activeElement.id),'accountMessage');
  await page.locator('#readingTheme').click();assert.equal(await page.locator('#readingTheme').getAttribute('aria-pressed'),String(theme!=='dark'));
  await page.goto(origin+'/page/signup');assert.equal(await page.locator('#readingTheme').getAttribute('aria-pressed'),String(theme!=='dark'),'explicit preference survives navigation');
  await page.locator('#readingTheme').click();
  await page.goto(origin+'/skills');await page.locator('#accountProfileSave:enabled').waitFor();assert.equal(await page.locator('.settings-layout > section').count(),3);
  assert.equal(await page.locator('.sidebar-nav a[href="/portfolio"]').count(),1);
  for(const width of [320,390,1440]){await page.setViewportSize({width,height:844});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false,'profile overflow');const f=`first10-profile-${theme}-${width}.png`;await page.screenshot({path:output+'/'+f,fullPage:true});evidence.screenshots.push(f);}
  assert.deepEqual(errors,[]);evidence.sharedAccountChecks.push({theme,checks:'system and saved appearance, account header/form clearance, touch/contrast, busy and failed submission retention/focus, separated profile sections, portfolio navigation'});await context.close();
 }
}
