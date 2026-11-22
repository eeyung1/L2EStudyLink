/* Accessible focus lifecycle for the existing add-block sheet. */
(function () {
    const modal=document.querySelector('[data-workspace-dialog]');
    if(!modal)return;
    let opener=null,wasOpen=false;
    const items=()=>Array.from(modal.querySelectorAll('button:not([disabled]),input,textarea')).filter(n=>n.getClientRects().length);
    new MutationObserver(()=>{
        const open=!modal.classList.contains('hidden');
        if(open===wasOpen)return;
        wasOpen=open;
        if(open){opener=document.activeElement;document.querySelector('.app-shell').inert=true;items()[0]?.focus();}
        else {document.querySelector('.app-shell').inert=false;if(opener?.isConnected)opener.focus();}
    }).observe(modal,{attributes:true,attributeFilter:['class']});
    modal.addEventListener('keydown',event=>{
        if(event.key==='Escape' && !document.getElementById('saveBlockButton').disabled){event.preventDefault();closeAddModal();}
        if(event.key==='Tab'){
            const nodes=items(),first=nodes[0],last=nodes[nodes.length-1];
            if(event.shiftKey && document.activeElement===first){event.preventDefault();last.focus();}
            else if(!event.shiftKey && document.activeElement===last){event.preventDefault();first.focus();}
        }
    });
    document.getElementById('dayPicker').addEventListener('click',()=>{
        document.querySelectorAll('#dayPicker .day-chip').forEach(n=>n.setAttribute('aria-pressed',String(n.classList.contains('selected'))));
    });
    new MutationObserver(()=>{
        document.querySelectorAll('#dayPicker .day-chip').forEach(n=>n.setAttribute('aria-pressed',String(n.classList.contains('selected'))));
    }).observe(document.getElementById('dayPicker'),{subtree:true,attributes:true,attributeFilter:['class']});
    document.querySelectorAll('#dayPicker .day-chip').forEach(n=>n.setAttribute('aria-pressed','false'));
})();
