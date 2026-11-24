(()=>{
    const {node,api,el,json,tell,button,state,card,badge,decide}=L2ECommunity;
    let loading=false;
    let actionInProgress=false;
    let currentData={me:{current_streak:0,longest_streak:0,total_active_days:0},active:[],incoming:[],outgoing:[],suggestions:[]};
    const boxes=['activeList','incomingList','outgoingList','suggestionList'];

    function renderStats(me){
        if(!me)return;
        if(el('myCurrent'))el('myCurrent').textContent=me.current_streak+' days';
        if(el('myLongest'))el('myLongest').textContent=me.longest_streak+' days';
        if(el('myActiveDays'))el('myActiveDays').textContent=String(me.total_active_days);
    }

    function renderActive(list,myStreak){
        const box=el('activeList');
        if(!box)return;
        box.replaceChildren();
        if(!list||!list.length){
            state(box,'No accepted buddies yet. Send a request or accept one below.');
            return;
        }
        for(const person of list){
            const c=card(person.name);
            if(person.handle)c.append(node('p','@'+person.handle,'community-handle'));
            if(person.bio)c.append(node('p',person.bio,'community-bio'));
            c.append(badge('Accepted buddy'));
            if(person.streak){
                const compare=node('div',undefined,'buddy-comparison');
                compare.append(
                    node('p','Your current streak: '+(myStreak?.current_streak??0)+' days'),
                    node('p','Their current streak: '+(person.streak.current_streak??0)+' days')
                );
                c.append(compare);
            }
            box.append(c);
        }
    }

    function renderIncoming(list){
        const box=el('incomingList');
        if(!box)return;
        box.replaceChildren();
        if(!list||!list.length){
            state(box,'No requests for you right now.');
            return;
        }
        for(const person of list){
            const c=card(person.name);
            if(person.handle)c.append(node('p','@'+person.handle,'community-handle'));
            if(person.bio)c.append(node('p',person.bio,'community-bio'));
            c.append(badge('Awaiting your decision'));
            const actions=node('div',undefined,'community-actions');
            for(const status of ['active','declined']){
                const isDecline=status==='declined';
                const label=isDecline?'Decline buddy':'Accept buddy';
                actions.append(button(label,()=>{
                    if(actionInProgress)return;
                    decide(
                        status==='active'?'Accept buddy request?':'Decline buddy request?',
                        person.name+'. '+(status==='active'?'Both of you will be able to compare activity streaks.':'No buddy connection will be made.'),
                        async()=>{
                            if(actionInProgress)return;
                            actionInProgress=true;
                            try{
                                await api('/buddies/respond',{
                                    method:'POST',
                                    body:json({connection_id:person.connection_id,status})
                                });
                                tell('buddyMessage',status==='active'?'Buddy accepted.':'Request declined.');
                                await load();
                            }finally{
                                actionInProgress=false;
                            }
                        }
                    );
                },isDecline));
            }
            c.append(actions);
            box.append(c);
        }
    }

    function renderOutgoing(list){
        const box=el('outgoingList');
        if(!box)return;
        box.replaceChildren();
        if(!list||!list.length){
            state(box,'No requests waiting for a response.');
            return;
        }
        for(const person of list){
            const c=card(person.name);
            if(person.handle)c.append(node('p','@'+person.handle,'community-handle'));
            if(person.bio)c.append(node('p',person.bio,'community-bio'));
            c.append(badge('Request sent · awaiting response'));
            box.append(c);
        }
    }

    function renderSuggestions(list,searchActive=false){
        const box=el('suggestionList');
        if(!box)return;
        box.replaceChildren();
        if(!list||!list.length){
            state(box,searchActive?'No suggested fellows match your search. Try different terms or clear the search input.':'No new fellows available right now.');
            return;
        }
        for(const person of list){
            const c=card(person.name);
            if(person.handle)c.append(node('p','@'+person.handle,'community-handle'));
            if(person.bio)c.append(node('p',person.bio,'community-bio'));
            c.append(badge('Available for connection'));
            const actions=node('div',undefined,'community-actions');
            actions.append(button('Send buddy request',()=>{
                if(actionInProgress)return;
                decide(
                    'Send buddy request?',
                    person.name+' will decide whether to connect. Streak comparisons are shared only after acceptance.',
                    async()=>{
                        if(actionInProgress)return;
                        actionInProgress=true;
                        try{
                            await api('/buddies/request',{
                                method:'POST',
                                body:json({buddy_id:person.user_id})
                            });
                            tell('buddyMessage','Buddy request sent.');
                            await load();
                        }finally{
                            actionInProgress=false;
                        }
                    }
                );
            }));
            c.append(actions);
            box.append(c);
        }
    }

    function applySearch(){
        const searchInput=el('buddySuggestionSearch');
        const searchStatus=el('buddySearchStatus');
        const allSuggestions=currentData?.suggestions||[];
        const query=(searchInput?.value||'').trim();

        if(!query){
            if(searchStatus){
                searchStatus.hidden=true;
                searchStatus.textContent='';
            }
            renderSuggestions(allSuggestions,false);
            return;
        }

        const lower=query.toLowerCase();
        const matches=allSuggestions.filter(person=>{
            const name=(person.name||'').toLowerCase();
            const handle=(person.handle||'').toLowerCase();
            const bio=(person.bio||'').toLowerCase();
            return name.includes(lower)||handle.includes(lower)||bio.includes(lower);
        });

        if(searchStatus){
            searchStatus.hidden=false;
            if(!matches.length){
                searchStatus.textContent=`No fellows found matching "${query}".`;
            }else{
                searchStatus.textContent=`Showing ${matches.length} of ${allSuggestions.length} suggested fellows.`;
            }
        }
        renderSuggestions(matches,true);
    }

    function render(data){
        currentData=data;
        renderStats(data.me);
        renderActive(data.active,data.me);
        renderIncoming(data.incoming);
        renderOutgoing(data.outgoing);
        applySearch();
    }

    async function load(){
        if(loading)return;
        loading=true;
        if(el('refreshButton'))el('refreshButton').disabled=true;
        if(el('retryBuddies'))el('retryBuddies').hidden=true;
        boxes.forEach(id=>{
            const box=el(id);
            if(box)box.setAttribute('aria-busy','true');
        });
        try{
            const d=new Date();
            const today=`${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`;
            const data=await api('/buddies?today='+today);
            render(data);
        }catch(e){
            tell('buddyMessage',e.message||'Buddy information could not be refreshed.',true);
            if(el('retryBuddies'))el('retryBuddies').hidden=false;
            boxes.forEach(id=>{
                const box=el(id);
                if(box)state(box,'Buddy information could not be refreshed. Retry to see the current state.');
            });
            for(const id of ['myCurrent','myLongest','myActiveDays']){
                if(el(id))el(id).textContent='—';
            }
        }finally{
            loading=false;
            if(el('refreshButton'))el('refreshButton').disabled=false;
            boxes.forEach(id=>{
                const box=el(id);
                if(box)box.setAttribute('aria-busy','false');
            });
        }
    }

    function setTab(tabName){
        document.querySelectorAll('[data-buddy-tab]').forEach(btn=>{
            btn.setAttribute('aria-pressed',String(btn.dataset.buddyTab===tabName));
        });
        const layout=document.querySelector('.buddy-layout');
        if(layout)layout.setAttribute('data-active-tab',tabName);

        const colLeft=el('buddyColLeft');
        const colRight=el('buddyColRight');

        document.querySelectorAll('.buddy-section').forEach(sec=>{
            const secType=sec.dataset.buddySection;
            if(tabName==='all'||secType===tabName){
                sec.hidden=false;
            }else{
                sec.hidden=true;
            }
        });

        if(colLeft&&colRight){
            if(tabName==='connections'||tabName==='discover'){
                colLeft.hidden=false;
                colRight.hidden=true;
                colLeft.classList.add('col-span-full');
                colRight.classList.remove('col-span-full');
            }else if(tabName==='requests'){
                colLeft.hidden=true;
                colRight.hidden=false;
                colRight.classList.add('col-span-full');
                colLeft.classList.remove('col-span-full');
            }else{
                colLeft.hidden=false;
                colRight.hidden=false;
                colLeft.classList.remove('col-span-full');
                colRight.classList.remove('col-span-full');
            }
        }
    }

    document.querySelectorAll('[data-buddy-tab]').forEach(btn=>{
        btn.addEventListener('click',()=>setTab(btn.dataset.buddyTab));
    });

    document.querySelectorAll('.learning-jumps a').forEach(a=>{
        a.addEventListener('click',()=>{
            const target=a.getAttribute('href')?.replace('#','');
            if(target==='activeHeading')setTab('connections');
            else if(target==='discoverHeading')setTab('discover');
            else if(target==='incomingHeading'||target==='outgoingHeading')setTab('requests');
        });
    });

    const searchInput=el('buddySuggestionSearch');
    if(searchInput){
        searchInput.addEventListener('input',applySearch);
        searchInput.addEventListener('search',applySearch);
    }

    if(el('refreshButton')){
        el('refreshButton').addEventListener('click',()=>{
            tell('buddyMessage','');
            load();
        });
    }

    if(el('retryBuddies')){
        el('retryBuddies').addEventListener('click',()=>{
            tell('buddyMessage','');
            load();
        });
    }

    load();
})();
