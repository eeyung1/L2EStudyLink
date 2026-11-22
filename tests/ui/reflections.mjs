import assert from 'node:assert/strict';
import AxeBuilder from '@axe-core/playwright';

// Synthetic audio and permissions model test browser lifecycle races without a real microphone.
function mediaFixture() {
    window.__mic='allow';window.__stoppedTracks=0;window.__recordStarts=0;
    const stream=()=>({getTracks:()=>[{stop(){window.__stoppedTracks++;}}]});
    Object.defineProperty(navigator,'mediaDevices',{configurable:true,value:{getUserMedia:async()=>{
        if(window.__mic==='deny')throw new DOMException('Fixture denied','NotAllowedError');
        if(window.__mic==='hold')return new Promise(resolve=>window.__allowMic=()=>resolve(stream()));
        return stream();
    }}});
    function audio() {
        if(window.__oversize)return new Blob([new Uint8Array(6*1024*1024)],{type:'audio/wav'});
        const samples=8000,data=new ArrayBuffer(44+samples*2),v=new DataView(data);
        const text=(at,value)=>{for(let i=0;i<value.length;i++)v.setUint8(at+i,value.charCodeAt(i));};
        text(0,'RIFF');v.setUint32(4,36+samples*2,true);text(8,'WAVEfmt ');v.setUint32(16,16,true);v.setUint16(20,1,true);v.setUint16(22,1,true);v.setUint32(24,8000,true);v.setUint32(28,16000,true);v.setUint16(32,2,true);v.setUint16(34,16,true);text(36,'data');v.setUint32(40,samples*2,true);
        for(let i=0;i<samples;i++)v.setInt16(44+i*2,Math.sin(i/8)*1000,true);
        return new Blob([data],{type:'audio/wav'});
    }
    window.MediaRecorder=class {
        static isTypeSupported(){return true;}
        constructor(){this.state='inactive';this.mimeType='audio/wav';}
        start(){this.state='recording';window.__recordStarts++;}
        stop(){this.state='inactive';queueMicrotask(()=>{this.ondataavailable?.({data:window.__emptyAudio?new Blob():audio()});this.onstop?.();});}
    };
    const original=window.fetch;
    window.fetch=(input,options)=>{
        if(window.__ignoreUploadAbort && String(input).includes('/voice/upload')){const copy={...options};delete copy.signal;return original(input,copy);}
        return original(input,options);
    };
}
export async function verifyReflections(browser,origin,output,evidence) {
    evidence.reflectionChecks=[];evidence.reflectionAccessibility=[];
    const until=async check=>{const deadline=Date.now()+5000;while(!check()){if(Date.now()>deadline)throw Error('Fixture request did not arrive');await new Promise(r=>setTimeout(r,20));}};
    for(const theme of ['light','dark']) {
        const context=await browser.newContext({viewport:{width:390,height:900},timezoneId:'Africa/Lagos',reducedMotion:'reduce'});
        await context.addInitScript(theme=>localStorage.setItem('darkMode',String(theme==='dark')),theme);
        await context.addInitScript(mediaFixture);
        const page=await context.newPage(),errors=[],saved=[];
        page.on('pageerror',e=>errors.push(e.message));
        await page.clock.install({time:new Date('2026-11-23T10:00:00+01:00')});
        let uploadMode='fail',saveMode='fail',uploads=0,held=null,reflections=[];
        await page.route('**/api/v1/**',async route=>{
            const path=new URL(route.request().url()).pathname,method=route.request().method();
            if(path==='/api/v1/voice/upload'){
                uploads++;
                if(uploadMode==='hold'){held=route;return;}
                if(uploadMode==='fail')return route.fulfill({status:503,json:{error:'Fixture failed'}});
                return route.fulfill({status:200,json:{audio_url:'/uploads/fixture-note.webm',text:'Audio received. Transcription pending...'}});
            }
            if(method==='POST'&&path==='/api/v1/reflections'){
                saved.push(route.request().postDataJSON());
                if(saveMode==='fail')return route.fulfill({status:503,json:{error:'Fixture save failed'}});
                if(saveMode==='hold'){held=route;return;}
                reflections.push({...saved.at(-1),completed:true,id:501});return route.fulfill({status:201,json:{id:501}});
            }
            assert.equal(method,'GET','Only fixture reflection/audio writes allowed');
            let body=[];
            if(path==='/api/v1/me')body={id:9001,name:'Demo Fellow'};
            if(path==='/api/v1/timetable')body=[{id:101,day_of_week:0,start_time:'09:00',end_time:'11:00',activity:'API practice'},{id:102,day_of_week:2,start_time:'13:00',end_time:'14:00',activity:'Reading practice'}];
            if(path==='/api/v1/reflections')body=reflections;
            await route.fulfill({status:200,json:body});
        });
        await page.goto(origin+'/timetable');
        const open=async id=>{
            await page.locator(`.timetable-day-nav [data-day="${id===101?0:2}"]`).click();
            await page.locator(`[data-reflect="${id}"]`).click();
            await page.locator('#reflectionSummary').waitFor({state:'visible'});
        };
        const speak=async()=>page.locator('#modeSpeakTab').click();
        const record=async()=>{await page.locator('#voiceRecordBtn').click();await page.getByRole('button',{name:'Stop recording',exact:true}).waitFor();};
        const stop=async()=>page.getByRole('button',{name:'Stop recording',exact:true}).click();
        await open(102);
        await page.locator('#reflectionSummary').fill('Keep this written summary.');
        await page.locator('#reflectionChallenges').fill('Keep this challenge.');
        await page.locator('#reflectionLearnings').fill('Keep this learning.');
        await speak();
        assert.equal(await page.locator('#reflectionSummary').isVisible(),true);
        await page.keyboard.press('ArrowLeft');assert.equal(await page.locator('#modeWriteTab').getAttribute('aria-selected'),'true');
        await page.keyboard.press('End');assert.equal(await page.locator('#modeSpeakTab').getAttribute('aria-selected'),'true');
        await page.evaluate(()=>window.__mic='deny');
        await page.locator('#voiceRecordBtn').click();
        await page.locator('#voiceStatus').getByText(/permission was denied/).waitFor();
        assert.equal(await page.locator('#reflectionSummary').inputValue(),'Keep this written summary.');
        assert.equal(await page.locator('#saveReflectionButton').isEnabled(),true);
        evidence.reflectionChecks.push({theme,check:'denied microphone preserves editable text'});
        await page.evaluate(()=>window.__mic='allow');
        await record();
        await page.clock.runFor(1200);
        assert.match(await page.locator('#voiceTimer').textContent(),/00:01/);
        assert.equal(await page.locator('#saveReflectionButton').isDisabled(),true);
        await stop();
        await page.locator('#voiceRetryUpload').waitFor();
        assert.equal(await page.locator('#voicePreview').isVisible(),true);
        assert.equal(await page.locator('#saveReflectionButton').isDisabled(),true);
        uploadMode='success';await page.locator('#voiceRetryUpload').click();
        await page.locator('#voiceAttachedLabel').waitFor();
        assert.equal(uploads,2,'retry uploads the retained take');
        assert.equal(await page.locator('#reflectionSummary').inputValue(),'Keep this written summary.','placeholder is not a transcript');
        await page.evaluate(()=>new Promise((resolve,reject)=>{const audio=document.getElementById('voicePreviewAudio');audio.addEventListener('play',resolve,{once:true});audio.play().catch(reject);}));
        await page.locator('#voicePlayBtn').getAttribute('aria-label').then(label=>assert.ok(['Pause recording','Play recording'].includes(label)));
        await page.locator('#modeWriteTab').click();
        assert.match(await page.locator('#reflectionAttachmentStatus').textContent(),/Voice note attached/);
        await page.locator('#saveReflectionButton').click();
        await page.locator('#reflectionFormError').getByText(/Your text and attached voice note are kept/).waitFor();
        assert.equal(await page.locator('#reflectionSummary').inputValue(),'Keep this written summary.');
        assert.equal(await page.locator('#url-summary').inputValue(),'/uploads/fixture-note.webm');
        assert.equal(await page.locator('#saveReflectionButton').isEnabled(),true);
        evidence.reflectionChecks.push({theme,check:'upload retry, actual blob playback, mode switch and failed save preserve take/text'});
        await page.locator('#cancelReflectionButton').click();
        await open(101);assert.equal(await page.locator('#reflectionSummary').inputValue(),'');
        await page.locator('#reflectionSummary').fill('Another activity draft.');
        await page.locator('#cancelReflectionButton').click();
        await open(102);assert.equal(await page.locator('#reflectionSummary').inputValue(),'Keep this written summary.');
        assert.equal(await page.locator('#url-summary').inputValue(),'/uploads/fixture-note.webm');
        await speak();await page.evaluate(()=>window.__mic='deny');
        await page.locator('#voiceRerecord').click();await page.locator('#voiceStatus').getByText(/permission was denied/).waitFor();
        assert.equal(await page.locator('#url-summary').inputValue(),'/uploads/fixture-note.webm','failed re-record keeps old attachment');
        await page.evaluate(()=>{window.__mic='allow';window.__emptyAudio=true;});
        await page.locator('#voiceRerecord').click();await page.getByRole('button',{name:'Stop recording',exact:true}).waitFor();await stop();
        await page.locator('#voiceStatus').getByText(/Nothing was recorded/).waitFor();
        assert.equal(await page.locator('#url-summary').inputValue(),'/uploads/fixture-note.webm','empty replacement keeps previous attachment');
        await page.evaluate(()=>window.__emptyAudio=false);
        evidence.reflectionChecks.push({theme,check:'empty re-record keeps previous audio and text'});
        for(const width of [320,390,768,1440])for(const entry of ['write','voice']){
            await page.setViewportSize({width,height:900});await page.locator(entry==='write'?'#modeWriteTab':'#modeSpeakTab').click();
            const result=await new AxeBuilder({page}).include('#reflectionModal').withTags(['wcag2a','wcag2aa','wcag21aa']).analyze();
            evidence.reflectionAccessibility.push({theme,width,mode:entry,violations:result.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>({target:n.target,summary:n.failureSummary}))}))});
            assert.deepEqual(result.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)})),[],`${theme} ${entry} ${width} reflection accessibility`);
            assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
            const file=`reflection-${entry}-${theme}-${width}.png`;await page.screenshot({path:output+'/'+file});evidence.screenshots.push(file);
        }
        await page.setViewportSize({width:390,height:900});await speak();
        await page.locator('#voiceDelete').click();assert.equal(await page.locator('#reflectionSummary').inputValue(),'Keep this written summary.');
        assert.equal(await page.locator('#url-summary').inputValue(),'');
        // A pending permission callback must stop its stream after this draft closes.
        await page.evaluate(()=>window.__mic='hold');await page.locator('#voiceRecordBtn').click();
        await page.locator('#voiceStatus').getByText(/Allow microphone/).waitFor();
        const starts=await page.evaluate(()=>window.__recordStarts);
        await page.locator('#cancelReflectionButton').click();await open(101);
        await page.evaluate(()=>window.__allowMic());
        assert.equal(await page.evaluate(()=>window.__recordStarts),starts);
        assert.ok(await page.evaluate(()=>window.__stoppedTracks)>0);
        assert.equal(await page.locator('#reflectionSummary').inputValue(),'Another activity draft.');
        evidence.reflectionChecks.push({theme,check:'activity drafts isolated; failed re-record and late microphone permission safe'});
        // Deliberately ignore transport abort to prove the generation guard, not only AbortController.
        await speak();await page.evaluate(()=>{window.__mic='allow';window.__ignoreUploadAbort=true;});uploadMode='hold';
        await record();await stop();await page.waitForFunction(()=>document.getElementById('voiceStatus').textContent.includes('Uploading'));
        await until(()=>held);
        const late=held;held=null;
        await page.locator('#cancelReflectionButton').click();await open(102);
        await late.fulfill({status:200,json:{audio_url:'/uploads/stale-note.webm'}});
        assert.equal(await page.locator('#url-summary').inputValue(),'','late upload must not attach to another draft');
        await page.locator('#cancelReflectionButton').click();await open(101);await speak();
        await page.locator('#voiceRetryUpload').waitFor();
        uploadMode='success';await page.locator('#voiceRetryUpload').click();await page.locator('#voiceAttachedLabel').waitFor();
        await page.evaluate(()=>window.__ignoreUploadAbort=false);
        evidence.reflectionChecks.push({theme,check:'late upload ignored; interrupted draft restores local blob and retries'});
        uploadMode='hold';await page.locator('#voiceRerecord').click();await page.getByRole('button',{name:'Stop recording',exact:true}).waitFor();await stop();await until(()=>held);
        const timedOut=held;held=null;
        await page.clock.fastForward(31000);await page.locator('#voiceRetryUpload').waitFor();
        await timedOut.abort().catch(()=>{});
        assert.equal(await page.locator('#reflectionSummary').inputValue(),'Another activity draft.');
        uploadMode='success';await page.locator('#voiceRetryUpload').click();await page.locator('#voiceAttachedLabel').waitFor();
        evidence.reflectionChecks.push({theme,check:'upload timeout keeps the blob and text for retry'});
        // Stop automatically using elapsed time, without waiting three real minutes.
        await page.locator('#voiceRerecord').click();await page.getByRole('button',{name:'Stop recording',exact:true}).waitFor();
        await page.clock.fastForward(181000);await page.locator('#voiceAttachedLabel').waitFor();
        assert.match(await page.locator('#voiceTimer').textContent(),/03:00/);
        await page.waitForFunction(()=>!document.getElementById('saveReflectionButton').disabled);
        saveMode='hold';await page.locator('#saveReflectionButton').click();await until(()=>held);
        const timedSave=held;held=null;
        await page.clock.fastForward(31000);await page.locator('#reflectionFormError').getByText(/Could not confirm the save in time/).waitFor();
        await timedSave.abort().catch(()=>{});
        assert.equal(await page.locator('#reflectionSummary').inputValue(),'Another activity draft.');
        assert.equal(await page.locator('#saveReflectionButton').isEnabled(),true);
        evidence.reflectionChecks.push({theme,check:'save timeout restores editable draft and permits safe retry'});
        saveMode='hold';await page.locator('#saveReflectionButton').click();
        await until(()=>held);
        assert.equal(await page.locator('#cancelReflectionButton').isDisabled(),true);
        await page.keyboard.press('Escape');assert.equal(await page.locator('#reflectionModal').isVisible(),true);
        reflections.push({...saved.at(-1),id:501,completed:true});await held.fulfill({status:201,json:{id:501}});held=null;
        await page.locator('#reflectionModal').waitFor({state:'hidden'});await page.locator('#reflectionSavedNotice').waitFor();
        assert.equal(await page.locator('.timetable-day-nav [data-day="0"]').getAttribute('aria-pressed'),'true');
        assert.equal(saved.at(-1).summary,'Another activity draft.');
        assert.equal(saved.at(-1).audio_url,'/uploads/fixture-note.webm');
        evidence.reflectionChecks.push({theme,check:'3-minute auto-stop, save lock, correct payload and day retained'});
        // No MediaRecorder support must still leave a working text-only form.
        await open(102);await speak();await page.evaluate(()=>window.__oversize=true);
        const beforeLarge=uploads;await record();await stop();await page.locator('#voiceStatus').getByText(/exceeds the 5 MB/).waitFor();
        assert.equal(uploads,beforeLarge);assert.equal(await page.locator('#voiceRetryUpload').isVisible(),false);
        assert.equal(await page.locator('#saveReflectionButton').isDisabled(),true);
        await page.locator('#voiceDelete').click();await page.evaluate(()=>{window.__oversize=false;window.MediaRecorder=undefined;});
        evidence.reflectionChecks.push({theme,check:'oversized take is retained locally without futile upload; delete preserves text'});
        await page.locator('#voiceRecordBtn').click();await page.locator('#voiceStatus').getByText(/not supported/).waitFor();
        await page.locator('#reflectionSummary').fill('Text-only fixture reflection');saveMode='success';
        await page.locator('#saveReflectionButton').click();await page.locator('#reflectionModal').waitFor({state:'hidden'});await page.locator('#reflectionSavedNotice').waitFor();
        assert.equal(await page.locator('.timetable-day-nav [data-day="2"]').getAttribute('aria-pressed'),'true');
        evidence.reflectionChecks.push({theme,check:'unsupported recorder permits text-only save; non-current day retained'});
        assert.deepEqual(errors,[],`${theme} reflection page errors`);
        await context.close();
    }
}
