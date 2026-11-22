/* Personal history: text-only DOM rendering and same-origin, authorized audio. */
(function () {
    const list=document.getElementById('reflectionsList');let rows=[],failed=false;
    const filter=document.getElementById('reflectionFilters');
    function node(tag,text,className) {const item=document.createElement(tag);if(text!==undefined)item.textContent=text;if(className)item.className=className;return item;}
    function audioPath(value) {
        return typeof value==='string' && /^\/uploads\/(?:[1-9][0-9]*-)?[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}\.(webm|ogg|mp4|wav|m4a|mp3)$/.test(value)?value:null;
    }
    function prettyDate(value) {
        if(!/^\d{4}-\d{2}-\d{2}$/.test(value||''))return 'Date unavailable';
        const date=new Date(value+'T12:00:00');return Number.isNaN(date.getTime())?'Date unavailable':date.toLocaleDateString(undefined,{weekday:'long',day:'numeric',month:'long',year:'numeric'});
    }
    function detail(parent,label,value) {if(value){const p=node('p');p.append(node('strong',label+' '),document.createTextNode(value));parent.append(p);}}
    function render() {
        list.querySelectorAll('audio').forEach(audio=>audio.pause());list.replaceChildren();
        const search=document.getElementById('reflectionSearch').value.trim().toLocaleLowerCase();
        const date=document.getElementById('reflectionDateFilter').value,media=document.getElementById('reflectionMediaFilter').value;
        const visible=rows.filter(row=>(!date||row.log_date===date)&&(!search||[row.activity_name||row.activity,row.summary,row.challenges,row.learnings].join(' ').toLocaleLowerCase().includes(search))&&(media==='all'||(media==='voice'?!!row.audio_url:!row.audio_url)));
        document.getElementById('historyCount').textContent=failed?'History could not be loaded.':`${visible.length} of ${rows.length} reflection${rows.length===1?'':'s'} shown`;
        if(failed){const p=node('p','History is unavailable. Your filters are kept.','l2e-feedback');const retry=node('button','Try again');retry.type='button';retry.addEventListener('click',()=>load(true));p.append(retry);list.append(p);return;}
        if(!visible.length){const empty=node('section',undefined,'workspace-empty');empty.append(node('h2',rows.length?'No matching reflections':'Begin your reflection habit'),node('p',rows.length?'Change your filters or clear them to see more.':'Reflect on a timetable block to see your progress here.'));const link=node('a','Go to timetable →');link.href='/timetable';empty.append(link);list.append(empty);return;}
        visible.forEach((row,index)=>{
            const card=node('article',undefined,'history-card');const activity=row.activity_name||row.activity||'Previous activity';card.append(node('h2',activity));
            const dateLine=node('p',prettyDate(row.log_date),'history-date');const time=node('time',row.log_date);time.dateTime=row.log_date;dateLine.append(' · ',time);card.append(dateLine);
            if(row.timetable_id==null)card.append(node('p','Time block removed · reflection kept in your history','history-detached'));
            else if(row.start_time&&row.end_time)card.append(node('p',`${String(row.start_time).slice(0,5)}–${String(row.end_time).slice(0,5)} · current recurring timetable time`,'l2e-muted'));
            detail(card,'What I did:',row.summary||'Not specified');detail(card,'Challenges:',row.challenges);detail(card,'Learned:',row.learnings);
            if(row.audio_url){
                const title=node('h3','Voice note');title.id='history-audio-'+index;card.append(title);
                const path=audioPath(row.audio_url);
                if(path){const audio=node('audio');audio.controls=true;audio.preload='none';audio.src=path;audio.setAttribute('aria-labelledby',title.id);const feedback=node('p','Private recording · only your signed-in account can play it.','l2e-muted');feedback.setAttribute('role','status');const retry=node('button','Retry audio');retry.type='button';retry.hidden=true;
                    audio.addEventListener('play',()=>list.querySelectorAll('audio').forEach(other=>{if(other!==audio)other.pause();}));
                    audio.addEventListener('error',()=>{feedback.textContent='Recording unavailable. Your session may have expired, or the file may no longer exist. Retry or sign in again.';retry.hidden=false;});
                    audio.addEventListener('canplay',()=>{feedback.textContent='Recording ready to play.';retry.hidden=true;});
                    retry.addEventListener('click',()=>{feedback.textContent='Reloading recording…';retry.hidden=true;audio.load();});card.append(audio,feedback,retry);
                }else card.append(node('p','This older recording link cannot be played safely. Your written reflection is still available.','l2e-muted'));
            }
            list.append(card);
        });
    }
    async function load(refresh=false) {
        list.setAttribute('aria-busy','true');
        try {if(refresh)await invalidateReadCache(['/api/v1/reflections'],token);rows=await readAPI('/api/v1/reflections',token);failed=false;render();}
        catch(error){if(error.message==='Session expired')return;failed=true;render();}
        finally {list.setAttribute('aria-busy','false');}
    }
    filter.addEventListener('submit',event=>event.preventDefault());
    for(const id of ['reflectionSearch','reflectionDateFilter','reflectionMediaFilter'])document.getElementById(id).addEventListener(id==='reflectionSearch'?'input':'change',render);
    filter.addEventListener('reset',event=>{event.preventDefault();document.getElementById('reflectionSearch').value='';document.getElementById('reflectionDateFilter').value='';document.getElementById('reflectionMediaFilter').value='all';render();});
    load();
})();
