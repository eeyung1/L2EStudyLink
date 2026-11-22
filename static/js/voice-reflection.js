/* Reflection draft/recorder lifecycle. Drafts stay in this page's memory only. */
(function () {
    'use strict';
    const MAX_SECONDS=180, MAX_BYTES=5*1024*1024;
    const el=id=>document.getElementById(id);
    const drafts=new Map();
    let key=null,version=0,mode='write',state='ready',saving=false;
    let recorder=null,stream=null,timer=null,startedAt=0,seconds=0,blobSeconds=0;
    let blob=null,blobURL='',audioURL='',upload=null,requesting=false;
    let opener=null;
    const takeState=()=>audioURL?'attached':blob?(blob.size>MAX_BYTES?'too-large':'upload-error'):'ready';
    const busy=()=>['requesting','recording','stopping','uploading'].includes(state);
    const clock=s=>`${String(Math.floor(s/60)).padStart(2,'0')}:${String(Math.floor(s%60)).padStart(2,'0')}`;
    function status(text,tone='') {
        el('voiceStatus').textContent=text;
        el('voiceStatus').classList.toggle('is-error',tone==='error');
        el('voiceStatus').classList.toggle('is-success',tone==='success');
    }
    function counters() {
        [['reflectionSummary','summaryCount'],['reflectionChallenges','challengesCount'],['reflectionLearnings','learningsCount']].forEach(([field,count])=>el(count).textContent=String(el(field).value.length));
    }
    function render() {
        const record=el('voiceRecordBtn');
        record.disabled=saving||['requesting','stopping','uploading'].includes(state);
        record.textContent=state==='recording'?'Stop recording':state==='requesting'?'Waiting for microphone…':'Start recording';
        record.setAttribute('aria-label',state==='recording'?'Stop recording':'Start recording');
        record.classList.toggle('is-recording',state==='recording');
        el('voiceWaveform').classList.toggle('is-active',state==='recording');
        el('voiceTimer').textContent=clock(seconds)+' / 03:00';
        el('voicePreview').hidden=!blob;
        el('voiceAttachedLabel').hidden=!audioURL;
        el('url-summary').value=audioURL;
        el('voiceRetryUpload').hidden=state!=='upload-error';
        el('voiceRetryUpload').disabled=saving||busy();
        el('voiceRerecord').disabled=saving||busy();
        el('voiceDelete').disabled=saving||['requesting','recording','stopping'].includes(state);
        el('saveReflectionButton').disabled=saving||busy()||!!(blob&&!audioURL);
        el('saveReflectionButton').textContent=saving?'Saving reflection…':'Save reflection';
        el('cancelReflectionButton').disabled=saving;
        el('voicePlayBtn').disabled=saving||['recording','requesting','stopping'].includes(state);
        ['reflectionSummary','reflectionChallenges','reflectionLearnings','modeWriteTab','modeSpeakTab','voiceScrubber'].forEach(id=>el(id).disabled=saving);
        el('reflectionAttachmentStatus').textContent=state==='uploading'?'Voice note uploading. Saving becomes available after attachment.':state==='recording'?'Recording is active. Stop it before saving.':state==='requesting'?'Waiting for microphone permission.':state==='too-large'?'Voice note exceeds the 5 MB limit. Re-record or delete it before saving.':blob&&!audioURL?'Voice note is not attached. Retry its upload in Speak mode or delete the take before saving.':audioURL?'Voice note attached. Your written fields and audio will be saved together.':'No voice note attached. You can save a written reflection.';
        el('reflectionModal').setAttribute('aria-busy',String(saving));
        el('reflectionVoicePanel').setAttribute('aria-busy',String(state==='uploading'||state==='requesting'));
    }
    function stopTracks(s=stream) {if(s)s.getTracks().forEach(track=>track.stop());if(s===stream)stream=null;}
    function stopTimer() {if(timer)clearInterval(timer);timer=null;}
    function elapsed() {return Math.min(MAX_SECONDS,Math.floor((performance.now()-startedAt)/1000));}
    function revokePreview() {
        const audio=el('voicePreviewAudio');audio.pause();audio.removeAttribute('src');audio.load();
        if(blobURL)URL.revokeObjectURL(blobURL);blobURL='';
        el('voiceScrubber').value='0';el('voicePlayBtn').textContent='▶';el('voicePlayBtn').setAttribute('aria-label','Play recording');
    }
    function preview() {
        revokePreview();
        if(blob){blobURL=URL.createObjectURL(blob);el('voicePreviewAudio').src=blobURL;el('voicePreviewAudio').load();}
        el('voiceDuration').textContent=clock(blobSeconds);el('voiceCurrentTime').textContent='00:00';
    }
    function stash() {
        if(!key)return;
        drafts.set(key,{summary:el('reflectionSummary').value,challenges:el('reflectionChallenges').value,learnings:el('reflectionLearnings').value,blob,audioURL,seconds:blob?blobSeconds:0,mode});
    }
    function cancelAsync() {
        version++;requesting=false;
        if(upload)upload.abort();upload=null;
        stopTimer();
        if(recorder){recorder.onstop=null;recorder.ondataavailable=null;recorder.onerror=null;try{if(recorder.state!=='inactive')recorder.stop();}catch{}recorder=null;}
        stopTracks();
    }
    function setMode(next,focus=false) {
        if(saving)return;
        mode=next==='voice'?'voice':'write';
        for(const [value,tab,panel] of [['write','modeWriteTab','reflectionWritePanel'],['voice','modeSpeakTab','reflectionVoicePanel']]){
            el(tab).setAttribute('aria-selected',String(mode===value));el(tab).tabIndex=mode===value?0:-1;el(panel).hidden=mode!==value;
        }
        // Switching to Write finishes the take instead of silently discarding it.
        if(mode==='write'&&state==='recording')stopRecording();
        if(mode==='write'&&state==='requesting'){cancelAsync();state=takeState();render();}
        const panel=document.querySelector('#reflectionModal .reflection-modal');if(panel)panel.scrollTop=0;
        if(focus)el(mode==='write'?'modeWriteTab':'modeSpeakTab').focus();
    }
    function openDraft(nextKey) {
        stash();cancelAsync();revokePreview();key=nextKey;saving=false;
        const draft=drafts.get(key)||{};
        blob=draft.blob||null;audioURL=draft.audioURL||'';seconds=draft.seconds||0;blobSeconds=seconds;
        for(const [id,name] of [['reflectionSummary','summary'],['reflectionChallenges','challenges'],['reflectionLearnings','learnings']])el(id).value=draft[name]||'';
        state=takeState();
        setMode(draft.mode||'write');preview();counters();
        el('reflectionFormError').hidden=true;el('reflectionFormError').textContent='';
        status(state==='too-large'?'Recording exceeds the 5 MB limit. Re-record a shorter take or delete it.':blob?(audioURL?'Voice note attached. Add a written summary before saving.':'Recording kept on this page. Retry its upload or delete the take.'):'Record a voice note or write your reflection below.',blob&&!audioURL?'error':'');
        render();
    }
    async function uploadTake() {
        if(!blob||saving||state==='uploading')return;
        if(blob.size>MAX_BYTES){state='too-large';status('This recording exceeds the 5 MB upload limit. Re-record a shorter take or delete it to save text only.','error');render();return;}
        const take=blob,id=version;
        upload=new AbortController();const controller=upload;
        state='uploading';status('Uploading your voice note… Keep this page open.');render();
        const form=new FormData();form.append('audio',take,'voice-reflection.'+(take.type.includes('mp4')?'mp4':'webm'));
        const timeout=setTimeout(()=>controller.abort(),30000);
        try {
            const token=typeof sessionToken==='function'?sessionToken():'';
            const response=await fetch('/api/v1/voice/upload',{method:'POST',headers:token?{Authorization:'Bearer '+token}:{},body:form,signal:controller.signal});
            if(!response.ok)throw Error('Upload unavailable');
            const data=await response.json();
            if(id!==version||take!==blob)return;
            // The server does not transcribe audio. Never insert its placeholder text.
            if(typeof data.audio_url!=='string'||!/^\/uploads\/[a-zA-Z0-9_-]+\.(webm|ogg|mp4|wav|m4a|mp3)$/.test(data.audio_url))throw Error('Upload returned no valid audio attachment');
            audioURL=data.audio_url;state='attached';status('Voice note attached. Transcription is not available; add a written summary below.','success');
        } catch(error) {
            if(id!==version||take!==blob)return;
            state='upload-error';status('Upload interrupted or unavailable. Your recording and text are kept on this page. Retry upload, or delete the take to save text only.','error');
        } finally {
            clearTimeout(timeout);
            if(id===version){upload=null;stash();render();}
        }
    }
    function mimeType() {
        return ['audio/webm;codecs=opus','audio/webm','audio/mp4'].find(type=>typeof MediaRecorder.isTypeSupported==='function'&&MediaRecorder.isTypeSupported(type))||'';
    }
    async function startRecording() {
        if(saving||busy()||requesting)return;
        if(typeof MediaRecorder==='undefined'||!navigator.mediaDevices?.getUserMedia){status('Recording is not supported in this browser. You can still write and save your reflection.','error');return;}
        el('voicePreviewAudio').pause();
        const id=version;requesting=true;state='requesting';status('Allow microphone access to begin recording.');render();
        let acquired;
        try {
            acquired=await navigator.mediaDevices.getUserMedia({audio:true});
            if(id!==version){stopTracks(acquired);return;}
            const type=mimeType(),next=type?new MediaRecorder(acquired,{mimeType:type}):new MediaRecorder(acquired);
            const chunks=[];let interrupted=false;
            next.ondataavailable=event=>{if(id===version&&event.data?.size)chunks.push(event.data);};
            next.onerror=()=>{if(id!==version)return;interrupted=true;stopRecording();};
            next.onstop=()=>{
                if(id!==version)return;
                stopTimer();stopTracks();recorder=null;
                const take=new Blob(chunks,{type:next.mimeType||'audio/webm'});
                if(!take.size){seconds=blobSeconds;state=takeState();status('Nothing was recorded. Your text is unchanged; try recording again.','error');render();return;}
                blob=take;blobSeconds=seconds;audioURL='';preview();stash();
                if(take.size>MAX_BYTES){state='too-large';status('This recording exceeds the 5 MB upload limit. Re-record a shorter take or delete it to save text only.','error');render();return;}
                state='ready';
                if(interrupted)status('Recording was interrupted. Uploading the available audio…');
                uploadTake();
            };
            next.start();
            stream=acquired;recorder=next;seconds=0;startedAt=performance.now();state='recording';
            // Retain any earlier take until the replacement has actually stopped.
            timer=setInterval(()=>{seconds=elapsed();render();if(seconds>=MAX_SECONDS)stopRecording();},200);
            status('Recording… Stop when you are done. Maximum duration: 3 minutes.');
        } catch(error) {
            if(acquired)stopTracks(acquired);
            if(id!==version)return;
            recorder=null;state=takeState();
            status(error.name==='NotAllowedError'?'Microphone permission was denied. Allow it in browser settings and try again, or write below.':'The microphone could not start. Check that it is connected and available, or write below.','error');
        } finally {if(id===version){requesting=false;render();}}
    }
    function stopRecording() {
        if(state!=='recording'||!recorder)return;
        seconds=elapsed();stopTimer();state='stopping';status('Finishing your recording…');render();
        try {recorder.stop();}catch {stopTracks();recorder=null;state=takeState();status('Recording stopped unexpectedly. Your existing text and take are unchanged.','error');render();}
    }
    function deleteTake() {
        if(saving||['recording','stopping','requesting'].includes(state))return;
        cancelAsync();blob=null;audioURL='';seconds=0;blobSeconds=0;state='ready';preview();status('Voice take deleted. Your written reflection is unchanged.');stash();render();
    }
    function playback() {
        const audio=el('voicePreviewAudio');if(!audio.src)return;
        if(audio.paused)audio.play().catch(()=>status('Playback could not start. The take is still kept; try again or re-record.','error'));else audio.pause();
    }
    function syncPlayer() {
        const audio=el('voicePreviewAudio'),duration=Number.isFinite(audio.duration)&&audio.duration>0?audio.duration:blobSeconds;
        el('voiceCurrentTime').textContent=clock(audio.currentTime||0);el('voiceDuration').textContent=clock(duration);
        if(duration>0&&!el('voiceScrubber').matches(':active'))el('voiceScrubber').value=String(Math.min(100,(audio.currentTime||0)/duration*100));
    }
    function closeDraft() {
        if(saving)return false;
        if(state==='recording')status('Recording cancelled on close. Your previous take and written text were kept.');
        stash();cancelAsync();seconds=blobSeconds;state=takeState();
        el('voicePreviewAudio').pause();
        return true;
    }
    document.addEventListener('DOMContentLoaded',()=>{
        for(const id of ['reflectionSummary','reflectionChallenges','reflectionLearnings'])el(id).addEventListener('input',()=>{counters();stash();});
        el('voiceRetryUpload').addEventListener('click',uploadTake);
        el('modeWriteTab').parentElement.addEventListener('keydown',event=>{
            if(!['ArrowLeft','ArrowRight','Home','End'].includes(event.key))return;
            event.preventDefault();setMode(event.key==='Home'?'write':event.key==='End'?'voice':mode==='write'?'voice':'write',true);
        });
        const audio=el('voicePreviewAudio');
        for(const name of ['loadedmetadata','timeupdate','ended'])audio.addEventListener(name,syncPlayer);
        for(const name of ['play','pause','ended'])audio.addEventListener(name,()=>{el('voicePlayBtn').textContent=audio.paused?'▶':'⏸';el('voicePlayBtn').setAttribute('aria-label',audio.paused?'Play recording':'Pause recording');});
        el('voiceScrubber').addEventListener('input',()=>{const duration=Number.isFinite(audio.duration)?audio.duration:blobSeconds;if(duration>0){audio.currentTime=Number(el('voiceScrubber').value)/100*duration;syncPlayer();}});
        el('reflectionModal').addEventListener('keydown',event=>{
            if(event.key==='Escape'){event.preventDefault();if(!saving)window.closeReflectionModal();return;}
            if(event.key!=='Tab')return;
            const items=Array.from(el('reflectionModal').querySelectorAll('button:not([disabled]):not([tabindex="-1"]),textarea:not([disabled]),input:not([type="hidden"]):not([disabled])')).filter(n=>n.getClientRects().length);
            const first=items[0],last=items[items.length-1];
            if(event.shiftKey&&document.activeElement===first){event.preventDefault();last.focus();}
            else if(!event.shiftKey&&document.activeElement===last){event.preventDefault();first.focus();}
        });
        render();
    });
    window.openReflectionDraft=openDraft;
    window.setReflectionMode=setMode;
    window.toggleVoiceRecording=()=>state==='recording'?stopRecording():startRecording();
    window.toggleVoicePlayback=playback;
    window.rerecordVoiceTake=startRecording;
    window.deleteVoiceTake=deleteTake;
    window.abortVoiceRecording=closeDraft;
    window.resetReflectionModal=()=>{if(key)drafts.delete(key);key=null;openDraft(null);};
    window.setReflectionSaving=value=>{saving=value;render();};
    window.reflectionCanSave=()=>!saving&&!busy()&&!(blob&&!audioURL);
    window.markReflectionSaved=()=>{drafts.delete(key);key=null;cancelAsync();blob=null;audioURL='';seconds=0;blobSeconds=0;state='ready';revokePreview();};
    window.onReflectionModalOpened=()=>{opener=document.activeElement;document.querySelector('.app-shell').inert=true;el(mode==='voice'?'voiceRecordBtn':'reflectionSummary').focus();};
    window.onReflectionModalClosed=()=>{document.querySelector('.app-shell').inert=false;if(opener?.isConnected)opener.focus();opener=null;};
    window.addEventListener('pagehide',()=>{cancelAsync();revokePreview();drafts.clear();});
})();
