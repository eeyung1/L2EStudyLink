import {launchWithLeaveDefaults} from './leave-defaults.mjs';
import assert from 'node:assert/strict';
import {chromium} from '@playwright/test';
import {mkdir,writeFile} from 'node:fs/promises';
import {resolve} from 'node:path';
const origin=process.env.L2E_TEST_ORIGIN;assert.ok(origin?.startsWith('http://127.0.0.1:'));
const output=resolve('../test-results/ui');await mkdir(output,{recursive:true});const browser=await launchWithLeaveDefaults(chromium);const evidence={realGin:true,isolatedPostgres:true,checks:[]};
try{
    const make=async(token,scope)=>{const context=await browser.newContext({viewport:{width:390,height:844},timezoneId:'Africa/Lagos'});await context.addInitScript(()=>localStorage.setItem('darkMode','false'));await context.addCookies([{name:'l2e_session',value:token,url:origin,httpOnly:true,sameSite:'Lax'},{name:'l2e_scope',value:scope.repeat(32),url:origin,sameSite:'Lax'}]);return context;};
    const student=await make(process.env.L2E_STUDENT_TOKEN,'a'),tutor=await make(process.env.L2E_TUTOR_TOKEN,'b');const page=await student.newPage(),errors=[];page.on('pageerror',error=>errors.push(error.message));
    await page.goto(origin+'/search-page');await page.locator('#bookingClock').getByText(/Shared booking clock/).waitFor();
    await page.locator('#skillSearch').fill('Go');await page.locator('#dateSearch').fill(process.env.L2E_BOOKING_DATE);await page.locator('#searchButton').click();
    await page.getByRole('heading',{name:'Go mentor',exact:true}).waitFor();await page.getByRole('button',{name:'View and request session',exact:true}).click();
    await page.locator('#topic').fill('Browser booking fixture');await page.selectOption('#meetingType','in_person');
    const clock=await(await student.request.get(origin+'/api/v1/booking-clock')).json();assert.match(await page.locator('#bookingReview').textContent(),new RegExp(clock.timezone));assert.match(await page.locator('#bookingReview').textContent(),/In person/);
    await page.screenshot({path:output+'/real-booking-review-390.png'});
    await page.locator('#confirmBookingButton').click();await page.locator('#searchFeedback').getByText(/Request sent/).waitFor();
    await page.goto(origin+'/my-bookings');await page.locator('.session-status').getByText('Pending confirmation',{exact:true}).waitFor();
    assert.equal(await page.getByRole('button',{name:'Accept request',exact:true}).count(),0);
    evidence.checks.push('Search published availability, review the shared timezone and send a pending request through real APIs');
    const host=await tutor.newPage();await host.goto(origin+'/my-bookings');await host.getByRole('button',{name:'Accept request',exact:true}).click();await host.locator('#bookingActionConfirm').click();await host.locator('.session-status').getByText('Confirmed',{exact:true}).waitFor();
    await page.reload();await page.locator('.session-status').getByText('Confirmed',{exact:true}).waitFor();await page.getByRole('button',{name:'Cancel session',exact:true}).click();await page.locator('#bookingActionConfirm').click();await page.locator('.session-status').getByText('Cancelled',{exact:true}).waitFor();
    await host.reload();await host.locator('.session-status').getByText('Cancelled',{exact:true}).waitFor();assert.equal(await host.locator('.session-actions button').count(),0);
    await host.screenshot({path:output+'/real-booking-cancelled-390.png'});
    const guest=await browser.newContext();assert.equal((await guest.request.get(origin+'/api/v1/bookings')).status(),401);await guest.close();
    evidence.checks.push('Tutor accepts, student cancels a confirmed session, both accounts see the same terminal state, guest reads denied');
    assert.deepEqual(errors,[]);await student.close();await tutor.close();console.log('Real booking browser journey passed.');
}finally{await writeFile(output+'/real-booking-evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
