import assert from 'node:assert/strict';
import AxeBuilder from '@axe-core/playwright';
export async function verifyApp(browser,origin,output,evidence) {
    evidence.workspaceChecks=[];
    for(const theme of ['light','dark']) {
        const context=await browser.newContext({viewport:{width:390,height:900},timezoneId:'Africa/Lagos'});
        await context.addInitScript(theme=>localStorage.setItem('darkMode',String(theme==='dark')),theme);
        const page=await context.newPage(),errors=[];
        page.on('pageerror',e=>errors.push(e.message));
        await page.clock.setFixedTime(new Date('2026-11-23T10:00:00+01:00'));
        let blocks=[{id:101,day_of_week:0,start_time:'09:00',end_time:'11:00',activity:'Build a Go API',goal:'Test validation'},{id:102,day_of_week:2,start_time:'13:00',end_time:'14:00',activity:"Review O'Reilly <API> notes",goal:'Practice safely'}];
        let failCoverage=false,failBookings=false;
        await page.route('**/api/v1/**',async route=>{
            const path=new URL(route.request().url()).pathname,method=route.request().method();
            if(method==='POST' && path==='/api/v1/timetable'){const b=route.request().postDataJSON();blocks.push({...b,id:103});return route.fulfill({status:201,json:{id:103}});}
            if(method==='DELETE' && path==='/api/v1/timetable/103'){blocks=blocks.filter(b=>b.id!==103);return route.fulfill({status:200,json:{message:'Deleted fixture'}});}
            assert.equal(method,'GET','Only explicit fixture writes are permitted');
            if(path==='/api/v1/bookings' && failBookings)return route.fulfill({status:503,json:{error:'Fixture unavailable'}});
            if(path==='/api/v1/planner/week' && failCoverage)return route.fulfill({status:503,json:{error:'Fixture unavailable'}});
            let body=[];
            if(path==='/api/v1/me')body={id:9001,name:'Demo Fellow',email:'demo@example.invalid',skills:[],rating:0,total_reviews:0};
            if(path==='/api/v1/timetable')body=blocks;
            if(path==='/api/v1/me/streaks')body={current_streak:0,longest_streak:0,total_active_days:0,daily_counts:{}};
            if(path==='/api/v1/planner/week')body={metrics:{due_blocks:5,reflected_blocks:3}};
            if(path==='/api/v1/bookings')body=[{id:1,date:'2026-11-22',start_time:'12:00',end_time:'13:00',topic:'Past session',status:'confirmed'}, {id:2,date:'2026-11-24',start_time:'14:00',end_time:'15:00',topic:'Cancelled session',status:'cancelled'}, {id:3,date:'2026-11-24',start_time:'12:00',end_time:'13:00',topic:'Peer API review',status:'confirmed',role:'student',tutor_name:'Demo Tutor'}];
            await route.fulfill({status:200,json:body});
        });
        for(const view of ['dashboard','timetable']) {
            await page.setViewportSize({width:390,height:900});
            await page.goto(origin+'/'+view);
            if(view==='dashboard'){
                await page.locator('#todayAgenda').getByText('Build a Go API',{exact:true}).waitFor();
                await page.locator('#nextSession').getByText('Peer API review',{exact:true}).waitFor();
                await page.locator('#weeklyCoverage').getByText('3 of 5').waitFor();
                assert.equal(await page.locator('#nextSession').getByText('Past session').count(),0);
                assert.ok(await page.locator('.daily-workspace').evaluate(n=>!!(n.compareDocumentPosition(document.querySelector('.activation-banner')) & Node.DOCUMENT_POSITION_FOLLOWING)));
            } else {
                await page.locator('#day-0').waitFor();
                assert.equal(await page.locator('.timetable-day-card:visible').count(),1);
            }
            await page.getByRole('button',{name:'Open navigation',exact:true}).click();
            assert.equal(await page.locator('main').getAttribute('inert'),'');
            assert.equal(await page.locator('.sidebar-nav [aria-current="page"]').getAttribute('href'),'/'+view);
            assert.equal(await page.locator('.workspace-nav-group').count(),3);
            const navResult=await new AxeBuilder({page}).include('.sidebar').withTags(['wcag2a','wcag2aa','wcag21aa']).analyze();
            assert.deepEqual(navResult.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)})),[],`${theme} navigation accessibility`);
            await page.keyboard.press('Escape');
            assert.equal(await page.locator('.mobile-menu-toggle').getAttribute('aria-expanded'),'false');
            assert.equal(await page.evaluate(()=>document.activeElement.classList.contains('mobile-menu-toggle')),true);
            for(const width of [320,390,768,1024,1440,1920]){
                await page.setViewportSize({width,height:1000});
                assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,`${theme} ${view} ${width} app overflow`);
                const controls=await page.locator('main button:visible').evaluateAll(nodes=>nodes.map(n=>({text:n.textContent,height:n.getBoundingClientRect().height,width:n.getBoundingClientRect().width})));
                assert.ok(controls.every(n=>n.height>=43.9 && n.width>=43.9),JSON.stringify(controls));
                if(view==='timetable' && width>=1440){await page.waitForFunction(()=>document.querySelector('.timetable-day-nav [data-day="all"]').getAttribute('aria-pressed')==='true');assert.equal(await page.locator('.timetable-day-card:visible').count(),7);}
                if(width===390||width===1440){const file=`integrated-${view}-${theme}-${width}.png`;await page.screenshot({path:output+'/'+file,fullPage:width===1440});evidence.screenshots.push(file);}
                evidence.workspaceChecks.push({view,theme,width,overflow:false,touchTargets:true});
            }
            await page.setViewportSize({width:390,height:900});
            const result=await new AxeBuilder({page}).include(view==='dashboard'?'.daily-workspace':'main').withTags(['wcag2a','wcag2aa','wcag21aa']).analyze();
            assert.deepEqual(result.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)})),[],`${theme} ${view} integrated accessibility`);
        }
        await page.locator('.timetable-day-nav [data-day="2"]').click();
        await page.locator('#day-2 [data-reflect="102"]').click();
        await page.getByRole('heading',{name:"Reflection: Review O'Reilly <API> notes",exact:true}).waitFor();
        await page.getByRole('button',{name:'Cancel',exact:true}).last().click();
        // Existing mutation handlers operate only against this in-memory fixture.
        await page.getByRole('button',{name:'+ Add time block',exact:true}).click();
        await page.locator('#dayPicker [data-day="2"]').click();
        await page.locator('#blockActivity').fill('New fixture block');
        await page.locator('#saveBlockButton').click();
        await page.locator('#day-2').getByText('New fixture block',{exact:false}).waitFor();
        assert.equal(await page.locator('.timetable-day-nav [data-day="2"]').getAttribute('aria-pressed'),'true');
        await page.locator('#day-2 .bg-gray-50').filter({hasText:'New fixture block'}).getByRole('button',{name:'Delete',exact:true}).click();
        await page.locator('#confirmDeleteBlock').click();
        await page.getByText('Time block and its linked reflections deleted.',{exact:true}).waitFor();
        assert.equal(await page.locator('.timetable-day-nav [data-day="2"]').getAttribute('aria-pressed'),'true');
        await page.getByRole('button',{name:'+ Add time block',exact:true}).click();
        await page.keyboard.press('Escape');
        assert.equal(await page.locator('#addModal').isVisible(),false);
        assert.equal(await page.locator('main').getAttribute('inert'),null);
        await page.goto(origin+'/timetable?reflect=today&block=101');
        await page.getByRole('heading',{name:'Reflection: Build a Go API',exact:true}).waitFor();
        await page.getByRole('button',{name:'Cancel',exact:true}).last().click();
        failCoverage=true;failBookings=true;
        await page.goto(origin+'/dashboard');
        await page.locator('#weeklyCoverage [data-daily-retry]').waitFor();
        await page.locator('#nextSession [data-daily-retry]').waitFor();
        await page.locator('#todayAgenda').getByText('Build a Go API',{exact:true}).waitFor();
        failCoverage=false;failBookings=false;
        await page.locator('#weeklyCoverage [data-daily-retry]').click();
        await page.locator('#nextSession [data-daily-retry]').click();
        await page.locator('#weeklyCoverage').getByText('3 of 5').waitFor();
        await page.locator('#nextSession').getByText('Peer API review',{exact:true}).waitFor();
        assert.deepEqual(errors,[],`${theme} app page errors`);
        await context.close();
    }
}
