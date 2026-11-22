'use strict';
const byId=id=>document.getElementById(id);
const token=sessionToken();
function logout(){return endSession(token);}
function toggleDark(){localStorage.setItem('darkMode',String(document.documentElement.classList.toggle('dark')));}
let currentMetrics=null, busy=false, requestSequence=0;
const initialScope=sessionScope();
function checkAccount(){if(sessionScope()!==initialScope){location.reload();throw Error('Your account changed. Reload this page.');}}
const headers=()=>token?{'Authorization':'Bearer '+token}:{};
const localDate=date=>`${date.getFullYear()}-${String(date.getMonth()+1).padStart(2,'0')}-${String(date.getDate()).padStart(2,'0')}`;
const today=new Date();today.setHours(12,0,0,0);today.setDate(today.getDate()-(today.getDay()+6)%7);
byId('week-start').value=localDate(today);
byId('timezone').value=Intl.DateTimeFormat().resolvedOptions().timeZone||'UTC';
function message(text,error=false){byId('status').textContent=text;byId('status').classList.toggle('coach-error',error);}
function setBusy(value){busy=value;byId('load-week').disabled=value;byId('week-start').disabled=value;byId('timezone').disabled=value;byId('analyze').disabled=value||!currentMetrics||byId('analyze').dataset.available!=='true';byId('analyze').textContent=value?'Working…':'Get weekly coaching';}
async function readResponse(response){checkAccount();let data;try{data=await response.json();}catch(_){throw new Error('We could not load your weekly review. Please try again.');}if(response.status===401){window.location.href='/page/login';throw new Error('Please sign in again.');}if(!response.ok)throw new Error(data.error||'We could not load your weekly review. Please try again.');return data;}
function renderMetrics(metrics,available){
 currentMetrics=metrics;byId('weekly-summary').hidden=false;byId('week-caption').textContent=metrics.week_start+' to '+metrics.week_end+' (end excluded) · '+metrics.timezone+' · Snapshot: '+metrics.as_of;
 byId('scheduled').textContent=metrics.scheduled_minutes+' min';byId('reflected').textContent=metrics.reflected_minutes+' min';byId('coverage').textContent=metrics.adherence_percent+'%';byId('missing').textContent=String(metrics.missing_reflections);
 byId('coverage').textContent=metrics.due_blocks?metrics.adherence_percent+'%':'No due blocks';
 const delta=metrics.streak_change;byId('streak').textContent=`${metrics.active_days} active days · Reflection streak: ${metrics.streak_at_end} days (${delta>=0?'+':''}${delta} this week) · ${metrics.pending_reflections} blocks not yet due.`;
 byId('unmatched').textContent=metrics.unmatched_reflections?`${metrics.unmatched_reflections} reflections have no matching timetable duration. They still contribute to your reflection habit.`:'';
 const chart=byId('daily-chart');chart.replaceChildren();
 for(const day of metrics.days){
  const row=document.createElement('div');row.className='coach-day';row.setAttribute('role','listitem');
  const label=document.createElement('h3');label.textContent=new Intl.DateTimeFormat(undefined,{weekday:'long',month:'short',day:'numeric',timeZone:'UTC'}).format(new Date(day.date+'T12:00:00Z'));
  const values=document.createElement('p');values.textContent=`${day.reflected_minutes} of ${day.scheduled_minutes} estimated minutes covered`;
  const track=document.createElement('div');track.className='coach-track';const bar=document.createElement('div');bar.className='coach-bar';bar.style.width=(day.scheduled_minutes?Math.min(100,100*day.reflected_minutes/day.scheduled_minutes):0)+'%';track.append(bar);track.setAttribute('aria-hidden','true');
  const detail=document.createElement('p');detail.className='coach-muted';detail.textContent=`${day.reflected_blocks} reflected blocks · ${day.missing_reflections} missing reflections · ${day.pending_reflections} not yet due`;
  row.append(label,values,track,detail);chart.append(row);
 }
 byId('analyze').dataset.available=available&&metrics.active_days>0?'true':'false';byId('coaching-note').textContent=!available?'AI coaching is not available yet.':metrics.active_days===0?'Add a reflection for this week to begin.':'';
}
async function loadWeek(event){
 if(event)event.preventDefault();if(busy)return;
 checkAccount();byId('retry-week').hidden=true;const sequence=++requestSequence;currentMetrics=null;byId('weekly-summary').hidden=true;byId('review').hidden=true;setBusy(true);message('Loading your week…');
 try{
  const selected=new Date(byId('week-start').value+'T12:00:00');if(Number.isNaN(selected.getTime()))throw new Error('Choose a valid week.');selected.setDate(selected.getDate()-(selected.getDay()+6)%7);byId('week-start').value=localDate(selected);
  const query=new URLSearchParams({week_start:byId('week-start').value,timezone:byId('timezone').value.trim()});
  const data=await readResponse(await fetch('/api/v1/planner/week?'+query,{headers:headers(),cache:'no-store'}));if(sequence!==requestSequence)return;
  renderMetrics(data.metrics,data.coaching_available);message(data.metrics.active_days?'Your week is ready.':'No reflections for this week yet. Your timetable is shown below.');
 }catch(error){message(error.message,true);byId('retry-week').hidden=false;}finally{setBusy(false);}
}
function renderList(id,items,empty){const list=byId(id);list.replaceChildren();for(const text of items.length?items:[empty]){const item=document.createElement('li');item.textContent=text;list.append(item);}}
byId('week-form').addEventListener('submit',loadWeek);byId('retry-week').addEventListener('click',()=>loadWeek());
for(const id of ['week-start','timezone'])byId(id).addEventListener('change',()=>{currentMetrics=null;byId('weekly-summary').hidden=true;byId('review').hidden=true;message('Select View week to refresh your metrics.');});
byId('analyze').addEventListener('click',async()=>{
 if(busy||!currentMetrics)return;checkAccount();setBusy(true);byId('review').hidden=true;message('Your coach is reviewing this week’s reflections…');
 try{
  const data=await readResponse(await fetch('/api/v1/planner/analyze',{method:'POST',headers:{...headers(),'Content-Type':'application/json'},body:JSON.stringify({week_start:currentMetrics.week_start,timezone:currentMetrics.timezone})}));
  renderMetrics(data.metrics,true);const review=data.coaching;byId('review-summary').textContent=review.summary;renderList('wins',review.wins,'No specific wins identified in this week’s reflections.');renderList('bottlenecks',review.bottlenecks,'No specific bottlenecks identified.');renderList('priorities',review.priorities,'');byId('question').textContent=review.coaching_question;byId('context-note').textContent=data.context_truncated?'Some reflections were shortened or omitted to keep this review focused.':'';byId('review').hidden=false;byId('review').tabIndex=-1;byId('review').focus();message('Your coaching is ready. Use what helps as you plan your next week.');
 }catch(error){message(error.message,true);}finally{setBusy(false);}
});
loadWeek();

fetch('/api/v1/me',{headers:headers(),cache:'no-store'}).then(r=>r.ok?r.json():null).then(me=>{if(me?.is_admin)byId('adminNavLink')?.classList.remove('hidden');}).catch(()=>{});
