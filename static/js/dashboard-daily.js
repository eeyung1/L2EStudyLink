/* Read-only daily workspace. APIs remain responsible for account authorization. */
(function () {
    const today = new Date();
    const localDate = date => `${date.getFullYear()}-${String(date.getMonth()+1).padStart(2,'0')}-${String(date.getDate()).padStart(2,'0')}`;
    const date = localDate(today), day = (today.getDay()+6)%7;
    const monday = new Date(today); monday.setDate(today.getDate()-day);
    const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
    document.getElementById('dailyDate').textContent = today.toLocaleDateString(undefined,{weekday:'long',day:'numeric',month:'long'}) + ' · ' + timezone;
    const escape = value => String(value ?? '').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
    const time = value => String(value || '').slice(0,5);
    const get = async path => {
        const response = await fetch(path,{headers:{Authorization:'Bearer '+token}});
        if (!response.ok) throw Error(response.status===401?'Session expired':'Request unavailable');
        return response.json();
    };
    function finish(id,html) { const node=document.getElementById(id);node.innerHTML=html;node.setAttribute('aria-busy','false'); }
    function failure(id,action) {finish(id,`<div class="l2e-feedback" role="status"><p>This information is unavailable. Other daily actions are still ready.</p><button type="button" class="l2e-button secondary" data-daily-retry="${action}">Try again</button></div>`);}
    function loading(id,text) {const node=document.getElementById(id);node.setAttribute('aria-busy','true');node.innerHTML=`<p class="l2e-loading">${text}</p>`;}
    async function agenda() {
        loading('todayAgenda',"Loading today's blocks…");
        const [blocksResult,reflectionsResult]=await Promise.allSettled([readAPI('/api/v1/timetable',token),readAPI('/api/v1/reflections',token)]);
        if(blocksResult.status==='rejected'){failure('todayAgenda','agenda');return;}
        const blocks=blocksResult.value.filter(b=>b.day_of_week===day).sort((a,b)=>a.start_time.localeCompare(b.start_time));
        const reflections=reflectionsResult.status==='fulfilled'?reflectionsResult.value:null;
        const warning=reflections===null?'<p class="l2e-feedback">Reflection status could not be checked. <button class="l2e-button secondary" data-daily-retry="agenda">Try again</button></p>':'';
        finish('todayAgenda',warning+(blocks.length?'<ol class="daily-agenda">'+blocks.map(b=>{
            const reflected=reflections?.some(r=>r.timetable_id===b.id && r.log_date===date);
            return `<li><span class="agenda-time">${escape(time(b.start_time))}–${escape(time(b.end_time))}</span><div><h4>${escape(b.activity)}</h4>${b.goal?`<p>${escape(b.goal)}</p>`:''}${reflected?'<span class="l2e-badge">Reflection added</span>':`<a href="/timetable?reflect=today&amp;block=${Number(b.id)}">Add reflection →</a>`}</div></li>`;
        }).join('')+'</ol>':'<p>No blocks planned for today. <a href="/timetable">Plan a time block</a> or start a focus session.</p>'));
        const first=blocks.find(b=>!reflections?.some(r=>r.timetable_id===b.id && r.log_date===date));
        document.getElementById('dailyReflection').href='/timetable?reflect=today'+(first?'&block='+Number(first.id):'');
    }
    async function bookings() {
        loading('nextSession','Loading sessions…');
        try {
            const items=await get('/api/v1/bookings');
            if(!Array.isArray(items)) throw Error('Unexpected sessions');
            // Bookings store a local date and clock time, without an offset.
            // Keep that representation; do not invent a UTC conversion.
            const now=`${date} ${String(today.getHours()).padStart(2,'0')}:${String(today.getMinutes()).padStart(2,'0')}`;
            const upcoming=items.filter(b=>['pending','confirmed'].includes(b.status) && `${String(b.date).slice(0,10)} ${time(b.start_time)}`>=now)
                .sort((a,b)=>(String(a.date).slice(0,10)+' '+time(a.start_time)).localeCompare(String(b.date).slice(0,10)+' '+time(b.start_time)))[0];
            finish('nextSession',upcoming?`<h4>${escape(upcoming.topic || 'Study session')}</h4><p>${escape(String(upcoming.date).slice(0,10))} · ${escape(time(upcoming.start_time))}–${escape(time(upcoming.end_time))}</p><p>With ${escape(upcoming.role==='tutor'?upcoming.student_name:upcoming.tutor_name)}</p><span class="l2e-badge">${upcoming.status==='confirmed'?'Confirmed':'Pending confirmation'}</span><p class="l2e-muted">Booking date and local time as scheduled.</p>`:'<p>No upcoming booked sessions. <a href="/search-page">Find a study partner</a>.</p>');
        } catch (_) {failure('nextSession','bookings');}
    }
    async function coverage() {
        loading('weeklyCoverage','Loading coverage…');
        try {
            const data=await get('/api/v1/planner/week?'+new URLSearchParams({week_start:localDate(monday),timezone}));
            const m=data.metrics;
            if(!m || !Number.isFinite(m.due_blocks)||!Number.isFinite(m.reflected_blocks))throw Error('Unexpected coverage');
            finish('weeklyCoverage',m.due_blocks>0?`<p class="coverage-number">${escape(m.reflected_blocks)} of ${escape(m.due_blocks)}</p><p>Due blocks have completed reflections.</p><progress max="${m.due_blocks}" value="${Math.min(m.due_blocks,m.reflected_blocks)}" aria-label="Due blocks with completed reflections"></progress>`:'<p>No blocks are due yet this week. Coverage will appear as scheduled blocks end or you add completed reflections.</p>');
        } catch (_) {failure('weeklyCoverage','coverage');}
    }
    const actions={agenda,bookings,coverage};
    document.addEventListener('click',event=>{const retry=event.target.closest('[data-daily-retry]');if(retry)actions[retry.dataset.dailyRetry]?.();});
    agenda();bookings();coverage();
})();
