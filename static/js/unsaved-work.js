/* Draft state stays in memory. No field content is written to browser storage. */
(()=>{
 function registry(){const checks=new Map();return {watch:(key,check)=>checks.set(key,check),remove:key=>checks.delete(key),dirty:()=>[...checks.values()].some(check=>check())};}
 if(typeof module!=='undefined')module.exports={registry};
 if(typeof window==='undefined')return;
 const entries=registry(),forms=new WeakMap();let leaving=false;
 const scope=()=>document.cookie.match(/(?:^|; )l2e_scope=([a-f0-9]+)/)?.[1]||'',initialScope=scope();
 function values(form){return JSON.stringify([...form.querySelectorAll('input,textarea,select')].filter(n=>!['password','submit','button'].includes(n.type)).map(n=>[n.name||n.id,n.type==='checkbox'||n.type==='radio'?n.checked:n.value]));}
 function watch(form){if(!form)return;if(forms.has(form))return forms.get(form);let baseline=values(form);const handle={saved(){baseline=values(form);},dirty(){return form.isConnected&&values(form)!==baseline;}};forms.set(form,handle);entries.watch(form,handle.dirty);form.addEventListener('reset',()=>queueMicrotask(handle.saved));form.addEventListener('input',()=>leaving=false);form.addEventListener('change',()=>leaving=false);return handle;}
 function dirty(){return scope()===initialScope&&entries.dirty();}
 function confirmLeave(){if(!dirty())return true;return window.confirm('You have unsaved changes on this page. Leave and discard them?');}
 window.L2EUnsaved={watch,saved:form=>watch(form)?.saved(),register:(key,check)=>entries.watch(key,check),hasChanges:dirty,confirmLeave,allowLeave:()=>leaving=true,resume:()=>leaving=false};
 window.addEventListener('beforeunload',e=>{if(!leaving&&dirty()){e.preventDefault();e.returnValue='';}});
 document.addEventListener('input',()=>leaving=false,true);
 window.addEventListener('pageshow',()=>leaving=false);
 document.addEventListener('click',e=>{if(e.defaultPrevented||e.button!==0||e.metaKey||e.ctrlKey||e.shiftKey||e.altKey)return;const a=e.target.closest?.('a[href]');if(!a||a.hasAttribute('download')||a.target&&a.target!=='_self')return;let u;try{u=new URL(a.href,location.href);}catch{return;}if(!['http:','https:'].includes(u.protocol)||u.pathname===location.pathname&&u.search===location.search&&u.hash)return;if(!confirmLeave()){e.preventDefault();e.stopImmediatePropagation();return;}leaving=true;},true);
 function mount(){document.querySelectorAll('form[data-unsaved-watch]').forEach(watch);}if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',mount,{once:true});else mount();
})();
