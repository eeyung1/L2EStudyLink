import assert from 'node:assert/strict';
import AxeBuilder from '@axe-core/playwright';

function wav() {
    const b=Buffer.alloc(16044);b.write('RIFF',0);b.writeUInt32LE(b.length-8,4);b.write('WAVEfmt ',8);b.writeUInt32LE(16,16);b.writeUInt16LE(1,20);b.writeUInt16LE(1,22);b.writeUInt32LE(8000,24);b.writeUInt32LE(16000,28);b.writeUInt16LE(2,32);b.writeUInt16LE(16,34);b.write('data',36);b.writeUInt32LE(16000,40);return b;
}
export async function verifyHistoryAndDialogs(browser,origin,output,evidence) {
    evidence.historyChecks=[];evidence.dialogChecks=[];
    for(const theme of ['light','dark']){
        const context=await browser.newContext({viewport:{width:390,height:844},reducedMotion:'reduce'});
        await context.addInitScript(theme=>{
            localStorage.setItem('darkMode',String(theme==='dark'));
            const viewport=new EventTarget();viewport.height=844;viewport.offsetTop=0;
            Object.defineProperty(window,'visualViewport',{value:viewport,configurable:true});
        },theme);
        const page=await context.newPage(),errors=[],external=[];page.on('pageerror',e=>errors.push(e.message));
        page.on('request',request=>{if(request.url().includes('example.invalid'))external.push(request.url());});
        await page.clock.setFixedTime(new Date('2026-11-24T10:00:00Z'));
        let failHistory=false,retention=true;
        const rows=[{id:1,timetable_id:101,log_date:'2026-11-23',activity_name:'Learn Go',summary:'Go practice\n<script>window.pwned=true</script>',challenges:'Validation',audio_url:'/uploads/9001-00000000-0000-4000-8000-000000000001.wav',start_time:'09:00',end_time:'10:00'}, {id:2,timetable_id:null,log_date:'2026-11-22',activity_name:'Previous reading',summary:'Detached history retained',audio_url:'',day_of_week:0,start_time:'',end_time:''},{id:3,timetable_id:null,log_date:'2026-11-21',activity_name:'Older voice',summary:'Written summary remains',audio_url:'https://example.invalid/track.wav'}];
        await page.route('**/api/v1/**',async route=>{
            const path=new URL(route.request().url()).pathname;
            assert.equal(route.request().method(),'GET','History/dialog fixtures never mutate data');
            if(path==='/api/v1/reflections'&&failHistory)return route.fulfill({status:503,json:{error:'Fixture unavailable'}});
            if(path==='/api/v1/reflections/retention')return route.fulfill({status:retention?200:503,json:{verified:retention,policy:'preserve_reflections'}});
            let body=[];
            if(path==='/api/v1/reflections')body=rows;
            if(path==='/api/v1/timetable')body=[{id:101,day_of_week:1,start_time:'09:00',end_time:'10:00',activity:'Dialog fixture'}];
            if(path==='/api/v1/me')body={id:9001,name:'Fixture learner'};
            if(path==='/api/v1/me/streaks')body={current_streak:0,longest_streak:0,total_active_days:0,daily_counts:{}};
            await route.fulfill({status:200,json:body});
        });
        await page.route('**/uploads/**',route=>route.fulfill({status:200,contentType:'audio/wav',body:wav()}));
        await page.goto(origin+'/reflections');await page.locator('.history-card').first().waitFor();
        assert.equal(await page.locator('.history-card').count(),3);assert.equal(await page.evaluate(()=>window.pwned),undefined);
        await page.getByText('Time block removed · reflection kept in your history',{exact:true}).first().waitFor();
        await page.locator('#reflectionSearch').fill('Learn Go');assert.equal(await page.locator('.history-card').count(),1);
        await page.selectOption('#reflectionMediaFilter','voice');assert.equal(await page.locator('.history-card').count(),1);
        await page.locator('#reflectionDateFilter').fill('2026-11-22');await page.locator('#reflectionDateFilter').dispatchEvent('change');await page.getByRole('heading',{name:'No matching reflections'}).waitFor();
        await page.locator('#clearReflectionFilter').click();await page.locator('.history-card').nth(2).waitFor();
        assert.equal(await page.locator('#reflectionSearch').inputValue(),'');
        assert.equal(await page.locator('audio').count(),1,'unsafe historic audio URL is not loaded');
        await page.locator('audio').evaluate(audio=>audio.dispatchEvent(new Event('error')));
        await page.getByRole('button',{name:'Retry audio',exact:true}).click();
        await page.locator('audio').evaluate(audio=>audio.play());
        for(const width of [320,390,768,1440]){
            await page.setViewportSize({width,height:844});
            assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,`${theme} ${width} history overflow`);
            const scan=await new AxeBuilder({page}).include('main').withTags(['wcag2a','wcag2aa','wcag21aa']).analyze();
            assert.deepEqual(scan.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)})),[],`${theme} ${width} history accessibility`);
            evidence.historyChecks.push({theme,width,overflow:false,violations:[]});
            if(width===390||width===1440){const file=`history-${theme}-${width}.png`;await page.screenshot({path:output+'/'+file});evidence.screenshots.push(file);}
        }
        await page.locator('#reflectionSearch').fill('Go');
        await page.evaluate(()=>sessionStorage.clear());failHistory=true;await page.reload();
        await page.getByRole('button',{name:'Try again',exact:true}).waitFor();
        await page.locator('#reflectionSearch').fill('Go');failHistory=false;await page.getByRole('button',{name:'Try again',exact:true}).click();await page.locator('.history-card').waitFor();
        assert.equal(await page.locator('#reflectionSearch').inputValue(),'Go');assert.deepEqual(external,[]);
        await page.setViewportSize({width:390,height:844});await page.goto(origin+'/timetable');
        const day=1;await page.locator(`.timetable-day-nav [data-day="${day}"]`).click();
        // Fixture reflection dates are unrelated to today's clock, so this block remains reflectable.
        await page.locator('[data-reflect="101"]').click();
        await page.locator('#modeWriteTab').focus();await page.keyboard.press('Shift+Tab');assert.equal(await page.evaluate(()=>document.activeElement.id),'cancelReflectionButton');
        await page.keyboard.press('Tab');assert.equal(await page.evaluate(()=>document.activeElement.id),'modeWriteTab');
        for(const width of [320,390,768,1440]){
            await page.setViewportSize({width,height:844});await page.locator('#reflectionLearnings').focus();
            await page.evaluate(()=>{visualViewport.height=420;visualViewport.offsetTop=70;visualViewport.dispatchEvent(new Event('resize'));});
            await page.waitForFunction(()=>document.documentElement.style.getPropertyValue('--l2e-visible-height')==='420px');
            const frame=await page.locator('#reflectionModal .reflection-modal').boundingBox(),footer=await page.locator('#reflectionModal .dialog-footer').boundingBox();
            assert.ok(frame.y>=69 && frame.y+frame.height<=491,`${width} reflection frame within keyboard viewport: ${JSON.stringify(frame)}`);
            assert.ok(footer.y+footer.height<=491,`${width} reflection actions remain visible`);
            const body=await page.locator('#reflectionModal .dialog-body').boundingBox(),field=await page.locator('#reflectionLearnings').boundingBox();
            assert.ok(field.y>=body.y-1&&field.y+field.height<=body.y+body.height+1,`${width} focused field scrolls into view`);
            evidence.dialogChecks.push({theme,width,simulatedKeyboard:true,actionsVisible:true,focusVisible:true});
            if(width===390){const file=`keyboard-reflection-${theme}.png`;await page.screenshot({path:output+'/'+file});evidence.screenshots.push(file);}
            await page.evaluate(()=>{visualViewport.height=844;visualViewport.offsetTop=0;visualViewport.dispatchEvent(new Event('resize'));});
        }
        await page.keyboard.press('Escape');assert.equal(await page.locator('#reflectionModal').isVisible(),false);
        assert.equal(await page.evaluate(()=>document.activeElement.dataset.reflect),'101');
        await page.getByRole('button',{name:'+ Add time block',exact:true}).click();await page.locator('#blockGoal').focus();
        await page.evaluate(()=>{visualViewport.height=420;visualViewport.offsetTop=70;visualViewport.dispatchEvent(new Event('resize'));});
        await page.waitForFunction(()=>document.documentElement.style.getPropertyValue('--l2e-visible-height')==='420px');
        const addFooter=await page.locator('#addModal .dialog-footer').boundingBox();assert.ok(addFooter.y+addFooter.height<=491);
        await page.keyboard.press('Escape');assert.equal(await page.locator('#addModal').isVisible(),false);
        await page.evaluate(()=>{visualViewport.height=844;visualViewport.offsetTop=0;visualViewport.dispatchEvent(new Event('resize'));});
        retention=false;await page.locator('.timetable-day-card:visible').getByRole('button',{name:'Delete',exact:true}).click();
        await page.locator('#deleteBlockError').waitFor();assert.equal(await page.locator('#confirmDeleteBlock').isDisabled(),true);
        await page.keyboard.press('Escape');assert.equal(await page.locator('#deleteBlockDialog').isVisible(),false);
        assert.deepEqual(errors,[]);await context.close();
    }
}
