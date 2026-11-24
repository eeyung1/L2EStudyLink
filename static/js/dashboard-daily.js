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
        if(blocksResult.status==='rejected' || !Array.isArray(blocksResult.value)){failure('todayAgenda','agenda');return;}
        const blocks=blocksResult.value.filter(b=>b.day_of_week===day).sort((a,b)=>a.start_time.localeCompare(b.start_time));
        const reflections=reflectionsResult.status==='fulfilled'&&Array.isArray(reflectionsResult.value)?reflectionsResult.value:null;
        const warning=reflections===null?'<div class="l2e-feedback warning" role="status"><p>Reflection status could not be checked. Other daily actions are still ready.</p><button type="button" class="l2e-button secondary" data-daily-retry="agenda">Try again</button></div>':'';
        finish('todayAgenda',warning+(blocks.length?'<ol class="daily-agenda">'+blocks.map(b=>{
            const reflected=reflections?.some(r=>r.timetable_id===b.id && r.log_date===date);
            return `<li><span class="agenda-time">${escape(time(b.start_time))}–${escape(time(b.end_time))}</span><div class="agenda-item-body"><h4>${escape(b.activity)}</h4>${b.goal?`<p class="agenda-goal">${escape(b.goal)}</p>`:''}<div class="agenda-item-actions">${reflected?'<span class="l2e-badge badge-reflected"><span aria-hidden="true">✓ </span>Reflection added</span>':`<a href="/timetable?reflect=today&amp;block=${Number(b.id)}" class="l2e-button small agenda-reflect-btn">Add reflection <span aria-hidden="true">→</span></a>`}</div></div></li>`;
        }).join('')+'</ol>':'<p class="daily-empty-text">No blocks planned for today. <a href="/timetable" class="l2e-text-link">Plan a time block</a> or <a href="/focus" class="l2e-text-link">start a focus session</a>.</p>'));
        const first=blocks.find(b=>!reflections?.some(r=>r.timetable_id===b.id && r.log_date===date));
        const dailyReflectBtn=document.getElementById('dailyReflection');
        if(dailyReflectBtn){
            dailyReflectBtn.href='/timetable?reflect=today'+(first?'&block='+Number(first.id):'');
        }
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
            if(upcoming) {
                const isConfirmed=upcoming.status==='confirmed';
                const statusBadge=isConfirmed
                    ? '<span class="l2e-badge badge-confirmed" data-session-status="confirmed"><span aria-hidden="true">✓ </span>Confirmed</span>'
                    : '<span class="l2e-badge badge-pending" data-session-status="pending"><span aria-hidden="true">⏳ </span>Pending confirmation</span>';
                const partnerName=upcoming.role==='tutor'?upcoming.student_name:upcoming.tutor_name;
                const partnerRole=upcoming.role==='tutor'?'Student':'Tutor';
                finish('nextSession',`
                    <div class="next-session-card-body">
                        <div class="next-session-top">
                            <h4>${escape(upcoming.topic || 'Study session')}</h4>
                            ${statusBadge}
                        </div>
                        <p class="next-session-timing"><strong>${escape(String(upcoming.date).slice(0,10))}</strong> · ${escape(time(upcoming.start_time))}–${escape(time(upcoming.end_time))}</p>
                        <p class="next-session-with">With ${escape(partnerName || 'Fellow')} <span class="l2e-muted">(${partnerRole})</span></p>
                        <div class="next-session-links">
                            <a href="/my-bookings" class="l2e-button secondary small next-session-direct-btn">Manage in My Bookings <span aria-hidden="true">→</span></a>
                        </div>
                        <p class="l2e-muted next-session-footnote">Booking date and local time as scheduled.</p>
                    </div>
                `);
            } else {
                finish('nextSession','<p class="next-session-empty">No upcoming booked sessions. <a href="/search-page">Find a study partner</a> or view <a href="/my-bookings">all bookings</a>.</p>');
            }
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
