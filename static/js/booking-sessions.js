(function(){
    const {node,time,api,configureDialog}=L2EBooking,list=document.getElementById('bookingsList'),notice=document.getElementById('bookingNotice');
    const names={pending:'Pending confirmation',confirmed:'Confirmed',cancelled:'Cancelled',completed:'Completed',no_show:'No-show'};
    const dialog=document.getElementById('bookingActionDialog');let rows=[],pending=null,busy=false;
    const drafts=new Map();let reviewBusy=false,targetHandled=false;
    const open=configureDialog(dialog,()=>!busy);
    function tell(message,error=false){notice.textContent=message;notice.classList.toggle('is-error',error);}
    function describe(b){return `${String(b.date).slice(0,10)} · ${time(b.start_time)}–${time(b.end_time)} · ${b.timezone||'shared booking clock'}`;}
    function action(b,label,kind){const button=node('button',label);button.type='button';button.addEventListener('click',()=>{
        pending={b,kind,label};document.getElementById('bookingActionTitle').textContent=label+'?';document.getElementById('bookingActionDescription').textContent=`${b.topic} · ${describe(b)}. ${kind==='no_show'?'Confirm that the learner did not attend.':kind==='completed'?'Confirm that the session took place.':'The participants will see the updated session status.'}`;document.getElementById('bookingActionError').hidden=true;open();document.getElementById('bookingActionCancel').focus();
    });return button;}
    function render(){
        list.replaceChildren();const status=document.getElementById('sessionStatus').value,role=document.getElementById('sessionRole').value;
        const visible=rows.filter(b=>(status==='all'||b.status===status)&&(role==='all'||b.role===role));
        if(!visible.length){const section=node('section',undefined,'workspace-empty');section.append(node('h2',rows.length?'No sessions match these filters':'Your first session is waiting'),node('p','Find a study partner or change the status and role filters.'));const link=node('a','Find a study partner →');link.href='/search-page';section.append(link);list.append(section);return;}
        visible.sort((a,b)=>Number(!!b.can_cancel)-Number(!!a.can_cancel)|| (a.can_cancel?(String(a.date)+a.start_time).localeCompare(String(b.date)+b.start_time):(String(b.date)+b.start_time).localeCompare(String(a.date)+a.start_time))).forEach(b=>{
            const card=node('article',undefined,'session-card');card.dataset.session=String(b.id);
            card.append(node('p',b.role==='tutor'?'You are tutoring':'You are learning','workspace-eyebrow'),node('h2',b.role==='tutor'?b.student_name:b.tutor_name),node('p',b.topic),node('p',describe(b),'l2e-muted'),node('p',b.meeting_type==='in_person'?'In person':'Online'),node('span',names[b.status]||'Status unavailable','session-status status-'+b.status));
            if(b.status==='pending')card.append(node('p',b.role==='tutor'?'Review this request, then accept or decline.':'Waiting for the tutor to accept this request.','l2e-muted'));
            if(b.status==='confirmed'&&b.role==='tutor'&&!b.can_complete)card.append(node('p','Completion and no-show actions become available after the session ends.','l2e-muted'));
            const actions=node('div',undefined,'session-actions');
            if(b.can_accept)actions.append(action(b,'Accept request','confirmed'));
            if(b.can_decline)actions.append(action(b,'Decline request','decline'));
            else if(b.can_cancel)actions.append(action(b,'Cancel session','cancel'));
            if(b.can_complete)actions.append(action(b,'Mark completed','completed'));
            if(b.can_no_show)actions.append(action(b,'Report no-show','no_show'));
            card.append(actions);
            if(b.can_review){const draft=drafts.get(b.id)||{rating:'',comment:'',again:''};drafts.set(b.id,draft);
                const form=node('form',undefined,'booking-review-form');const heading=node('h3','Review this completed session');form.append(heading,node('p','Your rating contributes to the tutor’s profile. Write about the session and avoid personal or sensitive information.'));
                const ratingLabel=node('label','Rating (required)'),rating=node('select');rating.name='rating';rating.required=true;for(const n of [0,5,4,3,2,1]){const option=node('option',n?n+' '+(n===1?'star':'stars'):'Choose a rating');option.value=n?String(n):'';rating.append(option);}rating.value=draft.rating;ratingLabel.append(rating);
                const commentLabel=node('label','Your review (10–1000 characters)'),comment=node('textarea');comment.name='comment';comment.minLength=10;comment.maxLength=1000;comment.required=true;comment.rows=4;comment.value=draft.comment;commentLabel.append(comment);
                const againLabel=node('label','Would you book again? (optional)'),again=node('select');again.name='would_book_again';for(const [value,text] of [['','Prefer not to say'],['yes','Yes'],['no','No']]){const o=node('option',text);o.value=value;again.append(o);}again.value=draft.again;againLabel.append(again);
                const feedback=node('p',undefined,'l2e-feedback');feedback.setAttribute('role','status');const button=node('button','Publish review');button.type='submit';button.disabled=reviewBusy;
                for(const input of [rating,comment,again])input.addEventListener('input',()=>{draft.rating=rating.value;draft.comment=comment.value;draft.again=again.value;});
                form.append(ratingLabel,commentLabel,againLabel,feedback,button);form.addEventListener('submit',async event=>{event.preventDefault();if(reviewBusy)return;const text=comment.value.trim();if(text.length<10){feedback.textContent='Write at least 10 characters about the session.';comment.focus();return;}reviewBusy=true;form.querySelectorAll('input,textarea,select,button').forEach(n=>n.disabled=true);button.textContent='Publishing…';feedback.textContent='';
                    const body={rating:Number(rating.value),comment:text};if(again.value)body.would_book_again=again.value==='yes';
                    try{await api(`/bookings/${b.id}/reviews`,{method:'POST',body:JSON.stringify(body)});drafts.delete(b.id);b.can_review=false;reviewBusy=false;tell('Review published. Thank you for sharing your experience.');render();}
                    catch(e){feedback.textContent=e.message+' Your review is kept here. Try again or refresh sessions to check whether it was saved.';feedback.classList.add('is-error');form.querySelectorAll('input,textarea,select,button').forEach(n=>n.disabled=false);button.textContent='Publish review';}
                    finally{reviewBusy=false;}
                });card.append(form);
            }else if(b.status==='completed'&&b.role==='student')card.append(node('p','This session is not awaiting a review.','l2e-muted'));
            list.append(card);
        });
    }
    async function load(){list.setAttribute('aria-busy','true');if(!rows.length)list.replaceChildren(node('p','Loading your sessions…','l2e-feedback'));try{const value=await api('/bookings');if(!Array.isArray(value))throw Error('Unexpected session response');rows=value;render();
        if(!targetHandled){targetHandled=true;const id=Number(new URLSearchParams(location.search).get('session'));if(Number.isSafeInteger(id)&&id>0){const card=Array.from(list.querySelectorAll('[data-session]')).find(c=>Number(c.dataset.session)===id);if(card){card.tabIndex=-1;card.classList.add('session-selected');card.scrollIntoView({block:'start'});card.focus({preventScroll:true});}else tell('This session is no longer available in your account.');}}}catch(e){list.replaceChildren(node('p','Sessions could not be loaded. Your filters are kept.','l2e-feedback'));const retry=node('button','Retry sessions');retry.type='button';retry.addEventListener('click',load);list.append(retry);tell(e.message,true);}finally{list.setAttribute('aria-busy','false');}}
    for(const id of ['sessionStatus','sessionRole'])document.getElementById(id).addEventListener('change',render);
    document.getElementById('sessionFilters').addEventListener('submit',e=>e.preventDefault());
    document.getElementById('bookingActionCancel').addEventListener('click',()=>{if(!busy)dialog.close();});
    document.getElementById('bookingActionConfirm').addEventListener('click',async()=>{
        if(busy||!pending)return;busy=true;const {b,kind}=pending;const confirm=document.getElementById('bookingActionConfirm'),cancel=document.getElementById('bookingActionCancel');confirm.disabled=cancel.disabled=true;confirm.textContent='Updating…';
        let path=`/bookings/${b.id}`,options={method:'DELETE'};
        if(kind==='confirmed'||kind==='decline'){path+='/status';options={method:'PUT',body:JSON.stringify({status:kind==='decline'?'cancelled':'confirmed'})};}
        if(kind==='completed'||kind==='no_show'){path+='/outcome';options={method:'PUT',body:JSON.stringify({status:kind})};}
        try{await api(path,options);dialog.close();tell('Session updated.');await load();notice.tabIndex=-1;notice.focus();}
        catch(e){const error=document.getElementById('bookingActionError');error.textContent=e.message;error.hidden=false;}
        finally{busy=false;confirm.disabled=cancel.disabled=false;confirm.textContent='Confirm';}
    });
    load();
})();
