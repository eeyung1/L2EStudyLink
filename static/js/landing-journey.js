/* Public practice only: no API writes, storage of demo content, or HTML injection. */
(()=>{
 const body=document.querySelector('.landing-page');if(!body)return;
 const el=id=>document.getElementById(id),tabs=[...document.querySelectorAll('[data-step]')];
 const reduced=matchMedia('(prefers-reduced-motion: reduce)');
 function saved(key){try{return localStorage.getItem(key);}catch{return null;}}
 function save(key,value){try{localStorage.setItem(key,value);}catch{}}
 document.documentElement.classList.add('landing-enhanced');
 const menu=el('landingMenu'),header=menu.closest('header');menu.hidden=false;
 function closeMenu(focus=false){header.classList.remove('landing-nav-open');menu.setAttribute('aria-expanded','false');if(focus)menu.focus();}
 menu.addEventListener('click',()=>{const open=header.classList.toggle('landing-nav-open');menu.setAttribute('aria-expanded',String(open));});
 header.addEventListener('keydown',e=>{if(e.key==='Escape'&&menu.getAttribute('aria-expanded')==='true'){e.preventDefault();closeMenu(true);}});
 el('landingNav').addEventListener('click',e=>{if(e.target.closest('a'))closeMenu();});
 document.addEventListener('click',e=>{if(!header.contains(e.target))closeMenu();});
 matchMedia('(min-width: 901px)').addEventListener('change',()=>closeMenu());
 const theme=el('readingTheme');function syncTheme(){const dark=document.documentElement.classList.contains('dark');theme.setAttribute('aria-pressed',String(dark));theme.title=dark?'Switch to light appearance':'Switch to dark appearance';document.querySelector('meta[name="theme-color"]').content=dark?'#102337':'#0369a1';}syncTheme();new MutationObserver(syncTheme).observe(document.documentElement,{attributes:true,attributeFilter:['class']});
 const motion=el('landingMotion');let paused=saved('l2e-motion')==='paused';motion.hidden=false;
 function syncMotion(){const stop=paused||reduced.matches;body.dataset.motion=stop?'paused':'running';motion.disabled=reduced.matches;motion.setAttribute('aria-pressed',String(stop));motion.textContent=reduced.matches?'Motion reduced':paused?'Resume motion':'Pause motion';motion.title=motion.textContent;}
 motion.addEventListener('click',()=>{paused=!paused;save('l2e-motion',paused?'paused':'running');syncMotion();});reduced.addEventListener('change',syncMotion);syncMotion();
 let interval=null,remaining=10,chosen=25,running=false,finished=false;
 const status=message=>el('demoStatus').textContent=message;
 function stop(){clearInterval(interval);interval=null;running=false;}
 function display(){const seconds=finished?0:Math.ceil(chosen*60*remaining/10);el('demoTimer').textContent=String(Math.floor(seconds/60)).padStart(2,'0')+':'+String(seconds%60).padStart(2,'0');el('demoStart').textContent=running?'Pause preview':remaining===10?'Start preview':'Resume preview';el('demoStart').disabled=finished;}
 function select(tab,focus=false){stop();display();tabs.forEach(t=>{const active=t===tab;t.setAttribute('aria-selected',String(active));t.tabIndex=active?0:-1;el('journey-'+t.dataset.step).hidden=!active;});if(focus)tab.focus();}
 tabs.forEach((t,i)=>{t.addEventListener('click',()=>select(t));t.addEventListener('keydown',e=>{let next;if(e.key==='ArrowRight')next=(i+1)%tabs.length;if(e.key==='ArrowLeft')next=(i+tabs.length-1)%tabs.length;if(e.key==='Home')next=0;if(e.key==='End')next=tabs.length-1;if(next!==undefined){e.preventDefault();select(tabs[next],true);}});});
 el('demoPlan').addEventListener('submit',e=>{e.preventDefault();const goal=el('demoGoal').value.trim();if(!goal){el('demoGoal').setCustomValidity('Enter a practice goal.');el('demoGoal').reportValidity();return;}el('demoStudyGoal').textContent=goal;chosen=Number(el('demoDuration').value);remaining=10;finished=false;select(tabs[1],true);display();status('Your '+chosen+'-minute practice plan is ready. Try a 10-second timer preview.');});el('demoGoal').addEventListener('input',()=>el('demoGoal').setCustomValidity(''));
 el('demoStart').addEventListener('click',()=>{if(running){stop();display();status('Timer preview paused.');return;}if(finished)return;running=true;display();status('Preview running. Ten seconds represents your chosen study block.');interval=setInterval(()=>{remaining--;if(remaining<=0){finished=true;stop();status('Preview complete. Take a moment to reflect.');}display();},1000);});
 el('demoReset').addEventListener('click',()=>{stop();remaining=10;finished=false;display();status('Timer preview reset.');});
 el('demoNext').addEventListener('click',()=>{select(tabs[2],true);el('demoThought').focus();status('What did you notice? This practice reflection is not saved.');});
 el('demoThought').addEventListener('input',()=>{el('demoThought').setCustomValidity('');el('demoCount').textContent=el('demoThought').value.length+' / 280 characters';});
 el('demoReflection').addEventListener('submit',e=>{e.preventDefault();if(!el('demoThought').value.trim()){el('demoThought').setCustomValidity('Add a thought before finishing.');el('demoThought').reportValidity();return;}stop();el('demoReflection').hidden=true;el('demoComplete').hidden=false;el('demoRestart').hidden=false;status('You tried the full learning cycle. Nothing was saved or sent.');el('demoRestart').focus();});
 el('demoRestart').addEventListener('click',()=>{stop();el('demoPlan').reset();el('demoGoal').setCustomValidity('');el('demoReflection').reset();el('demoThought').setCustomValidity('');el('demoReflection').hidden=false;el('demoComplete').hidden=true;el('demoRestart').hidden=true;el('demoCount').textContent='0 / 280 characters';el('demoStudyGoal').textContent=el('demoGoal').value;chosen=25;remaining=10;finished=false;select(tabs[0],true);display();status('A fresh practice cycle. Nothing from the previous demo was saved.');});
 document.addEventListener('visibilitychange',()=>{body.dataset.pageHidden=String(document.hidden);if(document.hidden&&running){stop();display();status('Timer preview paused while you were away.');}});window.addEventListener('pagehide',stop);
 // Content remains visible without observer support, JavaScript, or animation.
 if('IntersectionObserver' in window){const reveal=new IntersectionObserver(entries=>entries.forEach(entry=>{if(entry.isIntersecting){entry.target.removeAttribute('data-reveal-pending');reveal.unobserve(entry.target);}}),{threshold:.08});for(const node of document.querySelectorAll('.section-intro,.pathway-card,.demo-copy,.landing-faq')){node.dataset.reveal='';if(!reduced.matches&&!paused&&node.getBoundingClientRect().top>innerHeight)node.dataset.revealPending='';reveal.observe(node);}const art=new IntersectionObserver(entries=>{body.dataset.offscreen=String(!entries[0].isIntersecting);});art.observe(document.querySelector('.learning-universe'));}
})();
