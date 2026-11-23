(function(){
    const {node,time,day,api,clock,labelled,configureDialog}=L2EBooking;
    const form=document.getElementById('tutorSearch'),results=document.getElementById('searchResults'),feedback=document.getElementById('searchFeedback');
    const dialog=document.getElementById('tutorModal'),open=configureDialog(dialog,()=>!sending);
    let bookingClock=null,tutors=[],current=null,sending=false,searchVersion=0;
    const date=document.getElementById('dateSearch'),skill=document.getElementById('skillSearch'),bookingDate=document.getElementById('bookingDate'),slot=document.getElementById('timeSlot');
    const requested=new URLSearchParams(location.search).get('skill');if(requested)skill.value=requested.slice(0,50);
    const error=document.getElementById('bookingError');
    function review(){
        const list=document.getElementById('bookingReview');list.replaceChildren();
        labelled(list,'Tutor',current?.name||'Choose a tutor');labelled(list,'Date',bookingDate.value||'Choose a date');labelled(list,'Time',slot.selectedOptions[0]?.textContent||'No published slot selected');labelled(list,'Booking timezone',bookingClock?.timezone||'Unavailable');labelled(list,'Meeting type',document.getElementById('meetingType').selectedOptions[0].textContent);labelled(list,'Topic',document.getElementById('topic').value.trim()||'Add your topic above');
    }
    function updateSlots(){
        slot.replaceChildren();error.hidden=true;
        const selected=bookingDate.value,slots=(current?.availability||[]).filter(s=>s.day_of_week===day(selected));
        const valid=bookingClock && selected>=bookingClock.today;
        for(const s of valid?slots:[]){const from=time(s.start_time),to=time(s.end_time);if(!from||!to||from>=to)continue;if(selected===bookingClock.today && selected+' '+from<=bookingClock.now.slice(0,16))continue;const option=node('option',from+'–'+to);option.value=from+'|'+to;slot.append(option);}
        document.getElementById('availabilityDisplay').textContent=slot.options.length?'Published tutoring hours for this date · '+bookingClock.timezone:'No future published tutoring slot on this date. Choose another day.';
        document.getElementById('confirmBookingButton').disabled=!slot.options.length||sending;review();
    }
    function openTutor(tutor){
        if(!bookingClock){feedback.textContent='The shared booking clock is unavailable. Try searching again.';return;}
        current=tutor;document.getElementById('modalTitle').textContent='Study with '+tutor.name;
        const content=document.getElementById('modalContent');content.replaceChildren(node('p',tutor.bio||'No introduction added yet.'));
        if(tutor.discord_username)content.append(node('p','Discord: '+tutor.discord_username));
        bookingDate.min=bookingClock.today;bookingDate.value=date.value;document.getElementById('topic').value='';document.getElementById('meetingType').value='online';updateSlots();open();bookingDate.focus();
    }
    function render(){
        results.replaceChildren();const selectedDay=day(date.value),only=document.getElementById('availableOnly').checked;
        const visible=tutors.filter(t=>!only||(t.availability||[]).some(a=>a.day_of_week===selectedDay));
        feedback.textContent=visible.length?`${visible.length} tutor${visible.length===1?'':'s'} found. Times use ${bookingClock.timezone}.`:'No matching tutors for these filters. Try another skill/date or include tutors without hours on this date.';
        const order=document.getElementById('tutorOrder')?.value;if(order==='rating')visible.sort((a,b)=>(Number(b.rating)||0)-(Number(a.rating)||0)||String(a.name).localeCompare(String(b.name)));if(order==='name')visible.sort((a,b)=>String(a.name).localeCompare(String(b.name)));
        visible.forEach(t=>{
            const card=node('article',undefined,'tutor-card');card.append(node('h2',t.name),node('p',`${Number(t.rating)||0} / 5 · ${Number(t.total_reviews)||0} reviews`),node('p',t.bio||'No introduction added yet.'));
            const selected=(t.availability||[]).filter(a=>a.day_of_week===selectedDay);card.append(node('p',selected.length?'Tutoring hours: '+selected.map(a=>time(a.start_time)+'–'+time(a.end_time)).join(' · '):'No published tutoring hours for this date.','l2e-muted'));
            if(t.id===window.bookingUser?.id){const link=node('a','This is your profile · manage tutoring hours');link.href='/availability';card.append(link);}else{const button=node('button','View and request session');button.type='button';button.addEventListener('click',()=>openTutor(t));card.append(button);}results.append(card);
        });
    }
    async function search(event){
        event?.preventDefault();if(!form.reportValidity())return;
        const version=++searchVersion,button=document.getElementById('searchButton');button.disabled=true;results.setAttribute('aria-busy','true');feedback.textContent='Finding tutors and checking the booking clock…';
        try {
            const [found,serverClock,me]=await Promise.all([api('/search-with-availability?skill='+encodeURIComponent(skill.value.trim())),clock(),window.bookingUser?Promise.resolve(window.bookingUser):api('/me')]);
            window.bookingUser=me;
            if(version!==searchVersion)return;if(!Array.isArray(found))throw Error('Unexpected tutor results');bookingClock=serverClock;date.min=serverClock.today;if(date.value<serverClock.today)throw Error('Choose today or a future date in the shared booking clock');tutors=found;document.getElementById('bookingClock').textContent='Shared booking clock: '+serverClock.timezone+'. Your device: '+Intl.DateTimeFormat().resolvedOptions().timeZone+'. Published tutoring hours use the shared clock.';render();
        }catch(e){if(version!==searchVersion)return;results.replaceChildren();feedback.replaceChildren(document.createTextNode(e.message+' '));const retry=node('button','Retry search');retry.type='button';retry.addEventListener('click',search);feedback.append(retry);}
        finally{if(version===searchVersion){button.disabled=false;results.setAttribute('aria-busy','false');}}
    }
    document.getElementById('tutorOrder')?.addEventListener('change',()=>{if(bookingClock)render();});
    form.addEventListener('submit',search);document.getElementById('availableOnly').addEventListener('change',()=>{if(bookingClock)render();});
    bookingDate.addEventListener('change',updateSlots);for(const id of ['timeSlot','topic','meetingType'])document.getElementById(id).addEventListener(id==='topic'?'input':'change',review);
    for(const id of ['closeTutor','cancelTutor'])document.getElementById(id).addEventListener('click',()=>{if(!sending)dialog.close();});
    document.getElementById('bookingForm').addEventListener('submit',async event=>{
        event.preventDefault();if(sending||!current||!slot.value)return;const topic=document.getElementById('topic').value.trim();if(!topic){error.textContent='Add the topic you would like help with.';error.hidden=false;document.getElementById('topic').focus();return;}
        sending=true;error.hidden=true;const button=document.getElementById('confirmBookingButton');button.disabled=true;button.textContent='Sending request…';for(const field of document.getElementById('bookingForm').elements)field.disabled=true;document.getElementById('cancelTutor').disabled=true;document.getElementById('closeTutor').disabled=true;
        const [start_time,end_time]=slot.value.split('|');
        try {await api('/bookings',{method:'POST',body:JSON.stringify({tutor_id:current.id,date:bookingDate.value,start_time,end_time,topic,meeting_type:document.getElementById('meetingType').value})});dialog.close();feedback.textContent='Request sent. It is pending tutor confirmation. View it in My study sessions.';const link=node('a','View study sessions →');link.href='/my-bookings';feedback.append(' ',link);}
        catch(e){error.textContent=e.message+' Your selections are kept. Choose another slot or try again.';error.hidden=false;}
        finally {sending=false;for(const field of document.getElementById('bookingForm').elements)field.disabled=false;button.textContent='Send booking request';document.getElementById('cancelTutor').disabled=false;document.getElementById('closeTutor').disabled=false;button.disabled=!slot.options.length;}
    });
    clock().then(value=>{bookingClock=value;date.min=value.today;if(!date.value)date.value=value.today;document.getElementById('bookingClock').textContent='Shared booking clock: '+value.timezone+'. Tutoring availability is separate from your personal timetable.';if(requested)search();}).catch(()=>{date.value=new Date().toLocaleDateString('en-CA');document.getElementById('bookingClock').textContent='Booking clock unavailable. Search will retry it before offering a session.';});
})();
