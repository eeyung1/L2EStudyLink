(()=>{
    const {node,api,el,json,tell,button,state,card,badge,decide}=L2ECommunity;
    let loading=false;
    const boxes=['activeList','incomingList','outgoingList','suggestionList'];
    function render(data){el('myCurrent').textContent=data.me.current_streak+' days';el('myLongest').textContent=data.me.longest_streak+' days';el('myActiveDays').textContent=data.me.total_active_days;
        for(const [key,id,empty] of [['active','activeList','No accepted buddies yet. Send a request or accept one below.'],['incoming','incomingList','No requests for you right now.'],['outgoing','outgoingList','No requests waiting for a response.'],['suggestions','suggestionList','No new fellows available right now.']]){
            const box=el(id);box.replaceChildren();if(!data[key].length)state(box,empty);
            for(const person of data[key]){const c=card(person.name);if(person.handle)c.append(node('p','@'+person.handle));if(person.bio)c.append(node('p',person.bio));if(key==='incoming'||key==='outgoing')c.append(badge(key==='incoming'?'Awaiting your decision':'Request sent · awaiting response'));
                if(key==='active'&&person.streak){const compare=node('div',undefined,'buddy-comparison');compare.append(node('p','Your current streak: '+data.me.current_streak+' days'),node('p','Their current streak: '+person.streak.current_streak+' days'));c.append(badge('Accepted buddy'),compare);}
                const actions=node('div',undefined,'community-actions');
                if(key==='suggestions')actions.append(button('Send buddy request',()=>decide('Send buddy request?',person.name+' will decide whether to connect. Streak comparisons are shared only after acceptance.',async()=>{await api('/buddies/request',{method:'POST',body:json({buddy_id:person.user_id})});tell('buddyMessage','Buddy request sent.');await load();})));
                if(key==='incoming')for(const status of ['active','declined'])actions.append(button(status==='active'?'Accept buddy':'Decline buddy',()=>decide(status==='active'?'Accept buddy request?':'Decline buddy request?',person.name+'. '+(status==='active'?'Both of you will be able to compare activity streaks.':'No buddy connection will be made.'),async()=>{await api('/buddies/respond',{method:'POST',body:json({connection_id:person.connection_id,status})});tell('buddyMessage',status==='active'?'Buddy accepted.':'Request declined.');await load();}),status==='declined'));
                c.append(actions);box.append(c);
            }
        }
    }
    async function load(){if(loading)return;loading=true;el('refreshButton').disabled=true;el('retryBuddies').hidden=true;boxes.forEach(id=>el(id).setAttribute('aria-busy','true'));try{const d=new Date(),today=`${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`;render(await api('/buddies?today='+today));}catch(e){tell('buddyMessage',e.message,true);el('retryBuddies').hidden=false;boxes.forEach(id=>state(el(id),'Buddy information could not be refreshed. Retry to see the current state.'));for(const id of ['myCurrent','myLongest','myActiveDays'])el(id).textContent='—';}finally{loading=false;el('refreshButton').disabled=false;boxes.forEach(id=>el(id).setAttribute('aria-busy','false'));}}
    el('refreshButton').addEventListener('click',()=>{tell('buddyMessage','');load();});el('retryBuddies').addEventListener('click',()=>{tell('buddyMessage','');load();});load();
})();
