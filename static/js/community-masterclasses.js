(()=>{
    'use strict';
    const {node,api,el,json,tell,button,state,card,badge,decide}=L2ECommunity;
    let classes=[],currentUser=0,loading=false,busy=false,hasMore=false,failed=false;

    const tzEl = el('classTimezone');
    if (tzEl) {
        tzEl.textContent='Class times use your device timezone: '+(Intl.DateTimeFormat().resolvedOptions().timeZone||'local time')+'. Host inputs are converted to an absolute time when published.';
    }

    const request=(path,method='GET',body)=>api('/masterclasses'+path,{method,...(body?{body:json(body)}:{})});

    function controls(){
        for(const id of ['classSearch','mineOnly','refreshClasses','olderClasses','classOpenOnly']) {
            const element = el(id);
            if (element) element.disabled=loading||busy;
        }
        const searchSubmit = el('searchForm')?.querySelector('button[type=submit]');
        if (searchSubmit) searchSubmit.disabled=loading||busy;
        const createBtn = el('createClass');
        if (createBtn) createBtn.disabled=busy;
    }

    function render(){
        const list=el('classList');
        list.replaceChildren();
        list.setAttribute('aria-busy',String(loading));

        if(!classes.length) {
            state(list,loading?'Loading classes…':failed?'Classes could not be loaded. Retry below.':'No matching classes yet. Change your filters or host the first one.');
        }

        const visible=classes.filter(item=>!el('classOpenOnly').checked||(item.status!=='cancelled'&&new Date(item.starts_at)>new Date()&&item.reserved<item.capacity));

        if(classes.length&&!visible.length) {
            state(list,'No upcoming classes with seats among those loaded. Turn off the filter or load more classes.');
        }

        for(const item of visible){
            const c=card(item.title);
            const date=node('time',new Date(item.starts_at).toLocaleString());
            date.dateTime=item.starts_at;

            const hostInfo = node('p', 'Hosted by '+item.tutor_name, 'class-host');
            const desc = node('p', item.description||'No additional details shared.', 'class-description');
            const availableSeats = Math.max(0, item.capacity - item.reserved);
            const seatInfo = node('p', `${availableSeats} of ${item.capacity} learner seats available · ${item.duration_minutes} minutes`, 'class-seats');

            c.append(hostInfo, desc, date, seatInfo);

            const future=new Date(item.starts_at)>new Date();
            const badgeText = item.status==='cancelled'?'Class cancelled':!future?'Started / past class':item.tutor_id===currentUser?'You are hosting':item.reservation_status==='reserved'?'Your seat is reserved':item.reservation_status==='cancelled'?'Your reservation was cancelled':item.reserved>=item.capacity?'Class full':'Open for reservations';
            c.append(badge(badgeText));

            if(item.status==='scheduled'&&future){
                const action=item.tutor_id===currentUser?'cancel':item.reservation_status==='reserved'?'leave':'reserve';
                const label=action==='cancel'?'Cancel class':action==='leave'?'Cancel my reservation':item.reserved>=item.capacity?'Class full':'Reserve a seat';
                const b=button(label,()=>change(item,action),action!=='reserve');
                b.disabled=busy||loading||(action==='reserve'&&item.reserved>=item.capacity);
                const actions=node('div',undefined,'community-actions');
                actions.append(b);
                c.append(actions);
            }
            list.append(c);
        }

        const olderBtn = el('olderClasses');
        if (olderBtn) olderBtn.hidden=!hasMore;
        controls();
    }

    async function load(append=false){
        if(loading)return;
        loading=true;
        failed=false;
        const retryBtn = el('retryClasses');
        if (retryBtn) retryBtn.hidden=true;
        render();
        try{
            const query=new URLSearchParams({
                q:(el('classSearch')?.value || '').trim(),
                mine:String(!!el('mineOnly')?.checked)
            });
            if(append&&classes.length) {
                query.set('before_id',classes[classes.length-1].id);
            }
            const data=await request('?'+query);
            currentUser=data.user_id;
            classes=append?classes.concat(data.items.filter(item=>!classes.some(old=>old.id===item.id))):data.items;
            hasMore=data.has_more;
        }catch(e){
            failed=true;
            tell('classMessage',e.message,true);
            if (retryBtn) retryBtn.hidden=false;
        }finally{
            loading=false;
            render();
        }
    }

    function change(item,action){
        if(busy||loading)return;
        const label=action==='cancel'?'Cancel this class?':action==='leave'?'Cancel your reservation?':'Reserve a seat?';
        const desc=item.title+' · '+new Date(item.starts_at).toLocaleString()+'. '+(action==='cancel'?'This cancels every learner reservation.':action==='leave'?'Your seat becomes available to other learners.':'One learner seat will be reserved for you, subject to availability.');
        decide(label,desc,async()=>{
            if(busy)return;
            busy=true;
            render();
            try{
                await request('/'+item.id+(action==='cancel'?'/cancel':'/rsvp'),action==='reserve'?'POST':action==='cancel'?'PUT':'DELETE');
                tell('classMessage',action==='reserve'?'Your seat is reserved.':'Cancellation saved.');
                await load();
            }catch(e){
                await load();
                throw e;
            }finally{
                busy=false;
                render();
            }
        });
    }

    el('classOpenOnly')?.addEventListener('change',render);

    el('hostForm')?.addEventListener('submit',async event=>{
        event.preventDefault();
        if(busy)return;

        const titleInput = el('classTitle');
        const descInput = el('classDescription');
        const startsInput = el('classStarts');
        const durationInput = el('classDuration');
        const capacityInput = el('classCapacity');

        const title = (titleInput?.value || '').trim();
        const description = (descInput?.value || '').trim();
        const startsVal = startsInput?.value || '';
        const duration = Number(durationInput?.value);
        const capacity = Number(capacityInput?.value);

        if (title.length < 3 || title.length > 100) {
            tell('hostError', 'Class title must be between 3 and 100 characters.', true);
            titleInput?.focus();
            return;
        }

        if (!startsVal) {
            tell('hostError', 'Choose a valid future start time.', true);
            startsInput?.focus();
            return;
        }

        const starts = new Date(startsVal);
        if (Number.isNaN(starts.getTime()) || starts <= new Date()) {
            tell('hostError', 'Choose a valid future start time.', true);
            startsInput?.focus();
            return;
        }

        if (Number.isNaN(duration) || duration < 15 || duration > 240) {
            tell('hostError', 'Duration must be between 15 and 240 minutes.', true);
            durationInput?.focus();
            return;
        }

        if (Number.isNaN(capacity) || capacity < 2 || capacity > 100) {
            tell('hostError', 'Learner seats must be between 2 and 100.', true);
            capacityInput?.focus();
            return;
        }

        busy=true;
        render();
        tell('hostError','');

        const createBtn = el('createClass');
        const originalText = createBtn ? createBtn.textContent : 'Publish class';
        if (createBtn) createBtn.textContent = 'Publishing class…';

        try{
            await request('','POST',{
                title,
                description,
                starts_at:starts.toISOString(),
                duration_minutes:duration,
                capacity
            });
            el('hostForm').reset();
            window.L2EUnsaved?.saved(el('hostForm'));
            tell('classMessage','Your class is published.');
            await load();
        }catch(e){
            tell('hostError',e.message,true);
        }finally{
            busy=false;
            if (createBtn) createBtn.textContent = originalText;
            render();
        }
    });

    el('searchForm')?.addEventListener('submit',e=>{e.preventDefault();tell('classMessage','');load();});
    el('mineOnly')?.addEventListener('change',()=>{tell('classMessage','');load();});
    el('refreshClasses')?.addEventListener('click',()=>{tell('classMessage','');load();});
    el('retryClasses')?.addEventListener('click',()=>{tell('classMessage','');load();});
    el('olderClasses')?.addEventListener('click',()=>load(true));

    load();
})();
