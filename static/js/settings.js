(()=>{
 const {api,node,el,tell,button,state,decide}=L2ECommunity;let skills=[],editing=null,slots=[],revision=null,ready=false,busy=false;
 const notice=(text,error=false)=>tell('settingsNotice',text,error);
 function lock(form,value){form.querySelectorAll('input,select,textarea,button').forEach(n=>n.disabled=value);}
 async function loadProfile(){
  el('retryProfile').hidden=true;
  try{
   const me=await api('/me');
   el('profileIdentity').textContent=me.name+' · '+me.email;
   el('accountDiscord').value=me.discord_username||'';
   el('accountBio').value=me.bio||'';
   lock(el('accountProfileForm'),false);
   L2EUnsaved.saved(el('accountProfileForm'));
   el('settingsProductEmails').checked=!!me.product_emails;
   el('settingsProductEmails').disabled=false;
  }catch(e){
   el('profileIdentity').textContent=e.message;
   el('retryProfile').hidden=false;
  }
 }
 async function loadSkills(){
  const box=el('skills');
  box.setAttribute('aria-busy','true');
  try{
   skills=(await api('/me')).skills||[];
   box.replaceChildren();
   if(!skills.length)state(box,'No skills shared yet. Add one below.');
   for(const s of skills){
    const c=node('article',undefined,'settings-record');
    c.append(node('h3',s.name),node('p','Self-reported proficiency: '+s.proficiency));
    const actions=node('div',undefined,'community-actions');
    actions.append(button('Edit '+s.name,()=>{
     if(busy)return;
     editing=s.name;
     el('newSkillName').value=s.name;
     el('newSkillProficiency').value=s.proficiency;
     el('skillSubmit').textContent='Save changes';
     el('cancelSkillEdit').hidden=false;
     L2EUnsaved.saved(el('skillForm'));
     el('newSkillName').focus();
    }),button('Remove '+s.name,()=>{
     if(busy)return;
     decide('Remove this skill?',s.name+' will no longer appear in your skills list.',async()=>{
      busy=true;
      try{
       await api('/skills/'+encodeURIComponent(s.name),{method:'DELETE'});
       notice('Skill removed.');
       await loadSkills();
      }finally{
       busy=false;
      }
     });
    }));
    c.append(actions);
    box.append(c);
   }
  }catch(e){
   state(box,e.message,loadSkills);
  }finally{
   box.setAttribute('aria-busy','false');
  }
 }
 function clearEdit(){
  editing=null;
  el('skillForm').reset();
  el('skillSubmit').textContent='Add skill';
  el('cancelSkillEdit').hidden=true;
 }

 if(el('skillsSection')){
  let portfolioHandle='';
  async function loadPortfolioPreference(){
   if(!el('settingsPortfolioPublic'))return;
   try{
    const port=await api('/portfolio/me');
    portfolioHandle=port.handle||'';
    el('settingsPortfolioPublic').checked=!!port.portfolio_public;
    el('settingsPortfolioPublic').disabled=false;
   }catch(_){}
  }

  loadProfile();
  loadSkills();
  loadPortfolioPreference().catch(()=>{});
  el('retryProfile').addEventListener('click',()=>{
   loadProfile();
   loadPortfolioPreference().catch(()=>{});
  });

  let profileSaving=false;
  el('accountProfileForm').addEventListener('submit',async e=>{
   e.preventDefault();
   const form=e.currentTarget;
   if(profileSaving||el('accountProfileSave').disabled)return;
   profileSaving=true;
   const saveBtn=el('accountProfileSave');
   saveBtn.disabled=true;
   const body={discord_username:el('accountDiscord').value.trim(),bio:el('accountBio').value.trim()};
   lock(form,true);
   tell('accountProfileStatus','Saving…');
   try{
    await api('/profile',{method:'PUT',body:JSON.stringify(body)});
    L2EUnsaved.saved(form);
    tell('accountProfileStatus','Profile saved.');
   }catch(e){
    tell('accountProfileStatus',e.message+' Your entries are kept.',true);
   }finally{
    profileSaving=false;
    lock(form,false);
   }
  });

  let emailSaving=false;
  el('settingsProductEmails').addEventListener('change',async function(){
   if(emailSaving)return;
   emailSaving=true;
   const selected=this.checked;
   this.disabled=true;
   try{
    await api('/me/product-emails',{method:'PUT',body:JSON.stringify({enabled:selected})});
    tell('emailPreferenceStatus',selected?'Product emails enabled.':'Product emails turned off.');
   }catch(e){
    this.checked=!selected;
    tell('emailPreferenceStatus',e.message,true);
   }finally{
    emailSaving=false;
    this.disabled=false;
   }
  });

  if(el('settingsPortfolioPublic')){
   let portSaving=false;
   el('settingsPortfolioPublic').addEventListener('change',async function(){
    if(portSaving)return;
    portSaving=true;
    const selected=this.checked;
    this.disabled=true;
    tell('portfolioPreferenceStatus','Updating sharing preferences…');
    try{
     const res=await api('/portfolio/visibility',{method:'PUT',body:JSON.stringify({public:selected})});
     this.checked=!!res.portfolio_public;
     tell('portfolioPreferenceStatus',selected?'Your public portfolio is published.':'Your portfolio is now private.');
    }catch(e){
     this.checked=!selected;
     tell('portfolioPreferenceStatus',e.message,true);
    }finally{
     portSaving=false;
     this.disabled=false;
    }
   });
  }

  let skillSaving=false;
  el('skillForm').addEventListener('submit',async e=>{
   e.preventDefault();
   if(skillSaving||busy||el('skillSubmit').disabled)return;
   const name=el('newSkillName').value.trim();
   if(!name){notice('Please enter a skill name.',true);return;}
   skillSaving=true;
   busy=true;
   const form=e.currentTarget;
   lock(form,true);
   try{
    await api(editing?'/skills/'+encodeURIComponent(editing):'/skills',{method:editing?'PUT':'POST',body:JSON.stringify({skill_name:name,proficiency:el('newSkillProficiency').value})});
    notice(editing?'Skill updated.':'Skill added.');
    clearEdit();
    L2EUnsaved.saved(form);
    await loadSkills();
   }catch(e){
    notice(e.message+' Your entries are kept.',true);
   }finally{
    skillSaving=false;
    busy=false;
    lock(form,false);
   }
  });

  el('cancelSkillEdit').addEventListener('click',()=>{
   if(!busy){
    clearEdit();
    L2EUnsaved.saved(el('skillForm'));
   }
  });
 }

 if(el('availabilitySection')){
  function conflict(e){if(e.status===412){ready=false;lock(el('availabilityForm'),true);state(el('availability'),e.message,load);el('weeklyCount').textContent='Reload required';}return e;}
  async function saveSlots(next){try{return await api('/availability',{method:'PUT',headers:revision?{'If-Match':revision}:{},body:JSON.stringify(next)});}catch(e){throw conflict(e);}}
  const days=['Monday','Tuesday','Wednesday','Thursday','Friday','Saturday','Sunday'];
  async function load(){ready=false;const form=el('availabilityForm'),box=el('availability');lock(form,true);box.setAttribute('aria-busy','true');try{const snapshot=await api('/availability?revision=1');slots=Array.isArray(snapshot)?snapshot:snapshot.slots;revision=Array.isArray(snapshot)?null:snapshot.revision;box.replaceChildren();el('weeklyCount').textContent=slots.length+' published weekly slot(s)';if(!slots.length)state(box,'No tutoring hours published yet.');for(const s of slots){const c=node('article',undefined,'settings-record');c.append(node('h3',days[s.day_of_week]),node('p',s.start_time+'–'+s.end_time),button('Remove slot',()=>{if(busy)return;decide('Remove this tutoring slot?',days[s.day_of_week]+' '+s.start_time+'–'+s.end_time+'. Existing bookings keep their own status.',async()=>{busy=true;try{const next=slots.filter(x=>!(x.day_of_week===s.day_of_week&&x.start_time===s.start_time&&x.end_time===s.end_time));await saveSlots(next);notice('Time slot removed.');await load();}finally{busy=false;}});}));box.append(c);}ready=true;lock(form,false);}catch(e){state(box,e.message+' Load the saved hours before editing.',load);el('weeklyCount').textContent='Hours unavailable';}finally{box.setAttribute('aria-busy','false');}}
  el('availabilityForm').addEventListener('submit',async e=>{e.preventDefault();if(busy||!ready)return;const form=e.currentTarget;const start=el('availStartTime').value,end=el('availEndTime').value;if(start>=end){notice('Choose an end time after the start time.',true);return;}busy=true;lock(form,true);try{await saveSlots([...slots,{day_of_week:Number(el('availDay').value),start_time:start,end_time:end}]);L2EUnsaved.saved(form);notice('Availability saved.');await load();}catch(e){notice(e.message+' Your entries are kept.',true);}finally{busy=false;if(ready)lock(form,false);}});
  L2EBooking.clock().then(c=>el('bookingClock').textContent='Published tutoring hours use the shared booking clock: '+c.timezone).catch(()=>el('bookingClock').textContent='Booking clock unavailable. Check it before publishing hours.');load();
 }
})();
