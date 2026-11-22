window.L2ECommunity=(()=>{
    const {node}=L2EBooking,scope=sessionScope();
    function checkAccount(){if(sessionScope()!==scope){location.reload();throw Error('Your account changed. Reload this page.');}}
    async function api(path,options={}){checkAccount();const response=await fetch('/api/v1'+path,{...options,credentials:'same-origin',cache:'no-store',headers:{Authorization:'Bearer '+sessionToken(),...(options.body?{'Content-Type':'application/json'}:{})}});checkAccount();if(response.status===401){location.href='/page/login';throw Error('Your session expired. Sign in again.');}const data=await response.json().catch(()=>null);if(!response.ok)throw Error(data?.error||'Service unavailable. Please try again.');if(data===null)throw Error('Unexpected response. Please try again.');return data;}
    const el=id=>document.getElementById(id),json=data=>JSON.stringify(data);
    function tell(id,copy,error=false){const n=el(id);n.textContent=copy;n.classList.toggle('is-error',error);}
    function button(label,run,secondary=false){const b=node('button',label,secondary?'secondary':'');b.type='button';b.addEventListener('click',()=>run(b));return b;}
    function state(box,copy,retry){const section=node('div',undefined,'community-state');section.append(node('p',copy));if(retry)section.append(button('Try again',retry));box.replaceChildren(section);}
    function card(title){const c=node('article',undefined,'community-item');c.append(node('h3',title));return c;}
    function badge(copy){return node('span',copy,'community-badge');}
    // One reusable native confirmation dialog; no action occurs before confirmation.
    const dialog=node('dialog',undefined,'booking-dialog');dialog.setAttribute('data-viewport-dialog','');dialog.setAttribute('aria-labelledby','communityConfirmTitle');
    const head=node('header',undefined,'dialog-header'),title=node('h2');title.id='communityConfirmTitle';head.append(title);
    const body=node('div',undefined,'dialog-body'),copy=node('p'),error=node('p',undefined,'l2e-feedback is-error');error.setAttribute('role','status');body.append(copy,error);
    const foot=node('footer',undefined,'dialog-footer'),cancel=button('Keep unchanged',()=>dialog.close(),true),confirm=button('Confirm',async()=>{if(!pending||busy)return;busy=true;cancel.disabled=confirm.disabled=true;try{await pending();dialog.close();}catch(e){error.textContent=e.message;}finally{busy=false;cancel.disabled=confirm.disabled=false;}});foot.append(cancel,confirm);dialog.append(head,body,foot);document.querySelector('main').append(dialog);
    let pending=null,busy=false;const open=L2EBooking.configureDialog(dialog,()=>!busy);
    function decide(label,description,run){pending=run;title.textContent=label;copy.textContent=description;error.textContent='';open();cancel.focus();}
    window.logout=()=>endSession(sessionToken());window.toggleDark=()=>{const dark=document.documentElement.classList.toggle('dark');localStorage.setItem('darkMode',String(dark));};
    el('logoutButton')?.addEventListener('click',window.logout);el('darkToggle')?.addEventListener('click',()=>{if(!el('darkToggle').hasAttribute('onclick'))window.toggleDark();});
    api('/me').then(me=>{if(me.is_admin)el('adminNavLink')?.classList.remove('hidden');}).catch(()=>{});
    return {node,api,el,json,tell,button,state,card,badge,decide};
})();
