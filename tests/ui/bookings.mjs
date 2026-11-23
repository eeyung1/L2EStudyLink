import assert from 'node:assert/strict';
import AxeBuilder from '@axe-core/playwright';
export async function verifyBookings(browser,origin,output,evidence){
    evidence.bookingChecks=[];evidence.bookingAccessibility=[];
    for(const theme of ['light','dark']){
        const context=await browser.newContext({viewport:{width:390,height:844},timezoneId:'Africa/Lagos',reducedMotion:'reduce'});
        await context.addInitScript(theme=>localStorage.setItem('darkMode',String(theme==='dark')),theme);
        const page=await context.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));await page.clock.setFixedTime(new Date('2026-11-23T11:00:00+01:00'));
        const clock={timezone:'UTC',today:'2026-11-23',now:'2026-11-23 10:00:00'};
        let conflict=true,postCount=0,failList=false;
        const sessions=[{id:1,status:'pending',role:'tutor',can_accept:true,can_decline:true,can_cancel:true},{id:2,status:'confirmed',role:'student',can_cancel:true},{id:3,status:'confirmed',role:'tutor',can_complete:true,can_no_show:true},{id:4,status:'cancelled',role:'student'},{id:5,status:'completed',role:'student',can_review:true},{id:6,status:'no_show',role:'tutor'}].map(b=>({...b,tutor_name:'Go mentor',student_name:'Fictional learner',topic:'API practice',date:b.can_complete||['completed','no_show'].includes(b.status)?'2026-11-21':'2026-11-24',start_time:'09:00',end_time:'11:00',timezone:'UTC',meeting_type:'online'}));
        await page.route('**/api/v1/**',async route=>{
            const path=new URL(route.request().url()).pathname,method=route.request().method();let body=[];
            if(method==='POST'&&path==='/api/v1/bookings'){postCount++;return route.fulfill({status:conflict?409:201,json:conflict?{error:'Slot already booked'}:{booking_id:99}});}
            if(method==='PUT'&&path==='/api/v1/bookings/1/status'){sessions[0].status='confirmed';sessions[0].can_accept=false;sessions[0].can_decline=false;return route.fulfill({status:200,json:{status:'confirmed'}});}
            assert.equal(method,'GET','Only explicit booking fixture writes permitted');
            if(path==='/api/v1/booking-clock')body=clock;
            if(path==='/api/v1/me')body={id:9001,name:'Own profile'};
            if(path==='/api/v1/search-with-availability')body=[{id:9001,name:'Own profile',availability:[{day_of_week:1,start_time:'09:00',end_time:'11:00'}]},{id:9002,name:'Go <mentor>',rating:4.5,total_reviews:3,bio:'Learn <script>window.pwned=true</script> safely',availability:[{day_of_week:1,start_time:'09:00',end_time:'11:00'}]},{id:9003,name:'Wednesday tutor',availability:[{day_of_week:2,start_time:'09:00',end_time:'10:00'}]}];
            if(path==='/api/v1/bookings'){if(failList)return route.fulfill({status:503,json:{error:'Fixture unavailable'}});body=sessions;}
            await route.fulfill({status:200,json:body});
        });
        await page.goto(origin+'/search-page');await page.locator('#bookingClock').getByText(/Shared booking clock/).waitFor();await page.locator('#skillSearch').fill('Go');await page.locator('#dateSearch').fill('2026-11-24');await page.locator('#searchButton').click();await page.getByRole('heading',{name:'Go <mentor>',exact:true}).waitFor();
        await page.locator('#tutorOrder').selectOption('rating');assert.equal(await page.locator('.tutor-card h2').first().textContent(),'Go <mentor>');await page.locator('#tutorOrder').selectOption('name');assert.equal(await page.locator('.tutor-card h2').first().textContent(),'Go <mentor>');assert.equal(await page.locator('.tutor-card').count(),2);assert.equal(await page.getByRole('button',{name:'View and request session',exact:true}).count(),1);assert.equal(await page.evaluate(()=>window.pwned),undefined);
        await page.locator('#availableOnly').uncheck();assert.equal(await page.locator('.tutor-card').count(),3);await page.locator('#availableOnly').check();
        for(const width of [320,390,768,1440]){
            await page.setViewportSize({width,height:844});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,`${theme} ${width} search overflow`);
            const result=await new AxeBuilder({page}).include('main').withTags(['wcag2a','wcag2aa','wcag21aa']).analyze();assert.deepEqual(result.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)})),[],`${theme} search accessibility`);evidence.bookingAccessibility.push({theme,view:'search',width,violations:[]});
            if(width===390||width===1440){const file=`tutors-${theme}-${width}.png`;await page.screenshot({path:output+'/'+file,fullPage:width===1440});evidence.screenshots.push(file);}
        }
        await page.setViewportSize({width:390,height:844});await page.getByRole('button',{name:'View and request session',exact:true}).click();await page.locator('#topic').fill('Practice validation');await page.selectOption('#meetingType','in_person');assert.match(await page.locator('#bookingReview').textContent(),/Go <mentor>.*2026-11-24.*09:00.*UTC.*In person.*Practice validation/s);
        const scan=await new AxeBuilder({page}).include('#tutorModal').withTags(['wcag2a','wcag2aa','wcag21aa']).analyze();assert.deepEqual(scan.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)})),[],`${theme} booking review accessibility`);
        await page.locator('#confirmBookingButton').click();await page.locator('#bookingError').getByText(/Slot already booked/).waitFor();assert.equal(await page.locator('#topic').inputValue(),'Practice validation');assert.equal(postCount,1);conflict=false;await page.locator('#confirmBookingButton').click();await page.locator('#searchFeedback').getByText(/Request sent/).waitFor();assert.equal(postCount,2);
        evidence.bookingChecks.push({theme,check:'safe tutor cards, availability filter, self-profile link, explicit review and conflict retry preserve selection'});
        await page.goto(origin+'/my-bookings');await page.locator('.session-card').nth(5).waitFor();
        for(const width of [320,390,768,1440]){
            await page.setViewportSize({width,height:844});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,`${theme} ${width} sessions overflow`);
            const result=await new AxeBuilder({page}).include('main').withTags(['wcag2a','wcag2aa','wcag21aa']).analyze();assert.deepEqual(result.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)})),[],`${theme} sessions accessibility`);evidence.bookingAccessibility.push({theme,view:'sessions',width,violations:[]});
            if(width===390||width===1440){const file=`sessions-${theme}-${width}.png`;await page.screenshot({path:output+'/'+file,fullPage:width===1440});evidence.screenshots.push(file);}
        }
        await page.selectOption('#sessionStatus','confirmed');assert.equal(await page.locator('.session-card').count(),2);await page.selectOption('#sessionRole','tutor');assert.equal(await page.locator('.session-card').count(),1);assert.equal(await page.getByRole('button',{name:'Mark completed',exact:true}).count(),1);
        await page.selectOption('#sessionStatus','pending');await page.selectOption('#sessionRole','all');await page.getByRole('button',{name:'Accept request',exact:true}).click();await page.locator('#bookingActionConfirm').click();await page.getByRole('heading',{name:'No sessions match these filters',exact:true}).waitFor();
        assert.equal(await page.locator('#sessionStatus').inputValue(),'pending');await page.selectOption('#sessionStatus','all');await page.locator('.session-card').nth(5).waitFor();
        failList=true;await page.reload();await page.getByRole('button',{name:'Retry sessions',exact:true}).waitFor();failList=false;await page.getByRole('button',{name:'Retry sessions',exact:true}).click();await page.locator('.session-card').nth(5).waitFor();
        evidence.bookingChecks.push({theme,check:'all five session states, server-permitted actions, role/status filtering, action confirmation and retry'});
        assert.deepEqual(errors,[]);await context.close();
    }
}
