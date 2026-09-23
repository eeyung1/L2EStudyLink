const token = localStorage.getItem('token');
if (!token) location.href = '/page/login';
const headers = {Authorization: 'Bearer ' + token};
const escapeHtml = value => String(value ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
async function api(path, options = {}) {
    const response = await fetch('/api/v1' + path, {...options, headers: {...headers, ...(options.body ? {'Content-Type':'application/json'} : {})}});
    if (response.status === 401) { location.href = '/page/login'; throw Error('Session expired'); }
    const result = await response.json();
    if (!response.ok) throw Error(result.error || 'Please try again');
    return result;
}
function notice(message, error = false) {
    const box = document.getElementById('settingsNotice');
    box.textContent = message;
    box.className = error ? 'settings-notice error' : 'settings-notice';
}
async function logout() { localStorage.removeItem('token'); location.href = '/page/login'; }
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
        document.getElementById('skills').innerHTML = skills.length ? skills.map((skill, index) => `<div class="settings-skill"><div><strong>${escapeHtml(skill.name)}</strong><small>${escapeHtml(skill.proficiency)}</small></div><div><button type="button" data-edit="${index}">Edit</button><button type="button" data-remove="${index}">Remove</button></div></div>`).join('') : '<p class="page-sub">No skills yet. Add one below to start sharing what you know.</p>';
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
    async function loadAvailability() {
        slots = await api('/availability');
        document.getElementById('availability').innerHTML = slots.length ? slots.map((slot,index)=>`<div class="avail-slot"><span>${days[slot.day_of_week]} · ${escapeHtml(slot.start_time)}–${escapeHtml(slot.end_time)}</span><button type="button" data-remove="${index}" class="btn-delete">Remove</button></div>`).join('') : '<p class="page-sub">No weekly hours yet. Add your first slot below.</p>';
    }
    document.getElementById('availability').addEventListener('click',async event=>{
        const button=event.target.closest('[data-remove]'); if(!button)return;
        if(!confirm('Remove this time slot?'))return;
        button.disabled=true;
        try {await api('/availability',{method:'PUT',body:JSON.stringify(slots.filter((_,i)=>i!==Number(button.dataset.remove)))});notice('Time slot removed');await loadAvailability();}catch(error){notice(error.message,true);button.disabled=false;}
    });
    document.getElementById('availabilityForm').addEventListener('submit',async event=>{
        event.preventDefault(); const button=event.target.querySelector('button[type="submit"]');button.disabled=true;
        const day=Number(document.getElementById('availDay').value), start=document.getElementById('availStartHour').value+':00', end=document.getElementById('availEndHour').value+':00';
        if(start>=end){notice('Choose an end time after the start time',true);button.disabled=false;return;}
        try {await api('/availability',{method:'PUT',body:JSON.stringify([...slots,{day_of_week:day,start_time:start,end_time:end}])});notice('Availability saved');await loadAvailability();}catch(error){notice(error.message,true);}finally{button.disabled=false;}
    });
    loadAvailability().catch(error=>notice(error.message,true));
}
