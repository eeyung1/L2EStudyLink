/* Shared presentation only: existing page controllers retain action ownership. */
(()=>{
 const sun='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" aria-hidden="true"><circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5m11 11L19 19M5 19l1.5-1.5m11-11L19 5"/></svg>';
 function sync(){const dark=document.documentElement.classList.contains('dark');for(const button of document.querySelectorAll('#readingTheme,#darkToggle')){if(!button.querySelector('svg'))button.innerHTML=sun;button.setAttribute('aria-label','Dark appearance');button.setAttribute('aria-pressed',String(dark));button.title=dark?'Switch to light appearance':'Switch to dark appearance';}}
 function boot(){sync();new MutationObserver(sync).observe(document.documentElement,{attributes:true,attributeFilter:['class']});}
 if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',boot,{once:true});else boot();
})();
