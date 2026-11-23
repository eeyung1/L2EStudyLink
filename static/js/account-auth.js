(()=>{
 const el=id=>document.getElementById(id),mode=document.body.dataset.accountMode,form=el('accountForm');let busy=false;
 const next=learningReturnPath();el('accountReturn').textContent=next==='/learn'?'Continue to the Learning Hub after signing in.':'After signing in, return to the guide you opened.';
 for(const link of document.querySelectorAll('[data-auth-route]'))link.href='/page/'+link.dataset.authRoute+'?next='+encodeURIComponent(next);
 if(mode==='reset-password')el('email').value=new URLSearchParams(location.search).get('email')||'';
 function message(copy,error=false){el('accountMessage').textContent=copy;el('accountMessage').classList.toggle('is-error',error);if(error)el('accountMessage').focus();}
 for(const button of document.querySelectorAll('[data-password]'))button.addEventListener('click',()=>{const input=el(button.dataset.password),show=input.type==='password';input.type=show?'text':'password';button.textContent=show?'Hide':'Show';button.setAttribute('aria-label',show?'Hide password':'Show password');button.setAttribute('aria-pressed',String(show));});
 form.addEventListener('submit',async e=>{e.preventDefault();if(busy)return;if(mode==='reset-password'&&el('newPassword').value!==el('confirmPassword').value){message('Passwords do not match. Check both fields.',true);el('confirmPassword').focus();return;}
  const body={email:el('email').value.trim()};if(mode==='login'||mode==='signup')body.password=el('password').value;if(mode==='signup'){body.name=el('name').value.trim();body.product_emails=el('productEmails').checked;}if(mode==='reset-password'){body.code=el('code').value.trim();body.new_password=el('newPassword').value;}
  busy=true;form.setAttribute('aria-busy','true');form.querySelectorAll('input,button').forEach(n=>n.disabled=true);const label=el('accountSubmit').textContent;el('accountSubmit').textContent='Please wait…';message('');
  try{const response=await fetch('/api/v1/'+mode,{method:'POST',credentials:'same-origin',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});const data=await response.json().catch(()=>({}));if(!response.ok)throw Error(data.error||'The request could not be completed. Please try again.');
   if(mode==='login'||mode==='reset-password'&&data.user){localStorage.removeItem('token');location.href=next;return;}
   if(mode==='forgot-password'){location.href='/page/reset-password?email='+encodeURIComponent(body.email)+'&next='+encodeURIComponent(next);return;}
   form.hidden=true;el('accountSuccess').hidden=false;el('accountSuccessCopy').textContent=mode==='signup'?'Account created. Sign in to continue.':'Password updated. Sign in to continue.';el('accountContinue').href='/page/login?next='+encodeURIComponent(next);el('accountContinue').focus();
  }catch(error){message(error.message+' Your form entries are kept on this page.',true);}finally{busy=false;form.setAttribute('aria-busy','false');form.querySelectorAll('input,button').forEach(n=>n.disabled=false);el('accountSubmit').textContent=label;}
 });
})();
