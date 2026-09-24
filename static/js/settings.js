const token = sessionToken();
if (!token) location.href = '/page/login';
const headers = {Authorization: 'Bearer ' + token};
const escapeHtml = value => String(value ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
async function api(path, options = {}) {
    const response = await fetch('/api/v1' + path, {...options, headers: {...headers, ...(options.body ? {'Content-Type':'application/json'} : {})}});
    if (response.status === 401) { location.href = '/page/login'; throw Error('Session expired'); }
    const result = await response.json();
    if (!response.ok) { const error = Error(result.error || 'Please try again'); error.status = response.status; throw error; }
    return result;
}
function notice(message, error = false) {
    const box = document.getElementById('settingsNotice');
    box.textContent = message;
    box.className = error ? 'settings-notice error' : 'settings-notice';
}
async function logout() { await endSession(token); }
function toggleDark() {
    const dark = document.documentElement.classList.toggle('dark');
    localStorage.setItem('darkMode', dark);
    document.getElementById('darkToggle').textContent = dark ? '☀️' : '🌙';
}
if (localStorage.getItem('darkMode') === 'true') document.getElementById('darkToggle').textContent = '☀️';
api('/me').then(me => { if (me.is_admin) document.getElementById('adminNavLink').classList.remove('hidden'); }).catch(() => {});
if (document.getElementById('skillsSection')) {
    let skills = [];
    async function loadSkills() {
        const me = await api('/me'); skills = me.skills || [];
        document.getElementById('skills').innerHTML = skills.length ? skills.map((skill, index) => `<div class="settings-skill"><div class="settings-skill-info"><span class="settings-skill-icon" aria-hidden="true">✦</span><div><strong>${escapeHtml(skill.name)}</strong><small>${escapeHtml(skill.proficiency)}</small></div></div><div class="settings-actions"><button type="button" class="settings-action" data-edit="${index}" aria-label="Edit ${escapeHtml(skill.name)}"><svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="m15 5 4 4M4 20l4-.8L19 8a2.8 2.8 0 0 0-4-4L4 15z"/></svg>Edit</button><button type="button" class="settings-action settings-action-danger" data-remove="${index}" aria-label="Remove ${escapeHtml(skill.name)}"><svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M3 6h18M8 6V4h8v2m3 0-1 14H6L5 6m5 4v6m4-6v6"/></svg>Remove</button></div></div>`).join('') : '<p class="page-sub">No skills yet. Add one below to start sharing what you know.</p>';
    }
    const form = document.getElementById('skillForm');
    let editing = null;
    document.getElementById('skills').addEventListener('click', async event => {
        const edit = event.target.closest('[data-edit]');
        if (edit) { editing = skills[Number(edit.dataset.edit)].name; document.getElementById('newSkillName').value=editing; document.getElementById('newSkillProficiency').value=skills[Number(edit.dataset.edit)].proficiency; document.getElementById('skillSubmit').textContent='Save changes'; document.getElementById('cancelSkillEdit').hidden=false; document.getElementById('newSkillName').focus(); notice('Editing '+editing); return; }
        const remove = event.target.closest('[data-remove]');
        if (remove) { const name = skills[Number(remove.dataset.remove)]?.name; if (!name || !confirm('Remove '+name+'?')) return; try { await api('/skills/'+encodeURIComponent(name),{method:'DELETE'}); notice('Skill removed'); await loadSkills(); } catch (error) {notice(error.message,true);} }
    });
    function clearEdit() { editing=null; form.reset(); document.getElementById('skillSubmit').textContent='Add skill'; document.getElementById('cancelSkillEdit').hidden=true; }
    document.getElementById('cancelSkillEdit').addEventListener('click',()=>{clearEdit();notice('Edit cancelled');});
    form.addEventListener('submit',async event => {
        event.preventDefault(); const button=document.getElementById('skillSubmit'); button.disabled=true;
        try { const body=JSON.stringify({skill_name:document.getElementById('newSkillName').value.trim(),proficiency:document.getElementById('newSkillProficiency').value});
            await api(editing ? '/skills/'+encodeURIComponent(editing) : '/skills',{method:editing?'PUT':'POST',body});
            notice(editing?'Skill updated':'Skill added');clearEdit();await loadSkills();
        } catch(error){notice(error.message,true);} finally {button.disabled=false;}
    });
    loadSkills().catch(error=>notice(error.message,true));
}
if (document.getElementById('availabilitySection')) {
    let slots = [];
    const days = ['Monday','Tuesday','Wednesday','Thursday','Friday','Saturday','Sunday'];
    // Retry reads only; failed writes must never be submitted twice automatically.
    async function readAvailability() {
        for (let attempt = 0; attempt < 3; attempt++) {
            try { return await api('/availability'); }
            catch (error) {
                if (attempt === 2 || error.message === 'Session expired' || error.status && error.status < 500) throw error;
                await new Promise(resolve => setTimeout(resolve, 400 * (attempt + 1)));
            }
        }
    }
    async function loadAvailability() {
        const list = document.getElementById('availability');
        list.textContent = 'Loading your weekly hours…';
        document.getElementById('availabilityForm').querySelectorAll('select, input, button').forEach(control => control.disabled = true);
        try {
            slots = await readAvailability();
            document.getElementById('availabilityForm').querySelectorAll('select, input, button').forEach(control => control.disabled = false);
            const count = document.getElementById('weeklyCount');
            count.textContent = slots.length === 1 ? '1 weekly slot' : `${slots.length} weekly slots`;
            list.innerHTML = slots.length ? slots.map((slot,index)=>`<div class="avail-slot weekly-slot"><div class="weekly-day" aria-hidden="true">${days[slot.day_of_week].slice(0,3)}</div><div class="weekly-slot-text"><strong>${days[slot.day_of_week]}</strong><span>${escapeHtml(slot.start_time)} – ${escapeHtml(slot.end_time)}</span></div><button type="button" data-remove="${index}" class="settings-action settings-action-danger" aria-label="Remove ${days[slot.day_of_week]} ${escapeHtml(slot.start_time)} to ${escapeHtml(slot.end_time)}"><svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M3 6h18M8 6V4h8v2m3 0-1 14H6L5 6m5 4v6m4-6v6"/></svg>Remove</button></div>`).join('') : '<div class="weekly-empty"><span aria-hidden="true">✦</span><strong>Your week is full of possibility</strong><p>Choose a day and time below to make your first session available.</p></div>';
        } catch (error) {
            notice('Your hours could not be loaded yet. Try again without refreshing the page.', true);
            list.innerHTML = '<div class="weekly-empty"><strong>Could not load your hours</strong><p>Your saved hours are still there. Try loading them again.</p><button type="button" class="settings-action" id="retryAvailability">Try again</button></div>';
            document.getElementById('retryAvailability').addEventListener('click', () => {notice('');loadAvailability();});
        }
    }
    document.getElementById('availability').addEventListener('click',async event=>{
        const button=event.target.closest('[data-remove]'); if(!button)return;
        if(!confirm('Remove this time slot?'))return;
        button.disabled=true;
        try {await api('/availability',{method:'PUT',body:JSON.stringify(slots.filter((_,i)=>i!==Number(button.dataset.remove)))});notice('Time slot removed');await loadAvailability();}catch(error){notice(error.message,true);button.disabled=false;}
    });
    document.getElementById('availabilityForm').addEventListener('submit',async event=>{
        event.preventDefault(); const button=event.target.querySelector('button[type="submit"]');button.disabled=true;
        const day=Number(document.getElementById('availDay').value), start=document.getElementById('availStartTime').value, end=document.getElementById('availEndTime').value;
        if(start>=end){notice('Choose an end time after the start time',true);button.disabled=false;return;}
        try {await api('/availability',{method:'PUT',body:JSON.stringify([...slots,{day_of_week:day,start_time:start,end_time:end}])});notice('Availability saved');await loadAvailability();}catch(error){notice(error.message,true);}finally{button.disabled=false;}
    });
    loadAvailability();
}
