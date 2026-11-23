/* Keep modal chrome inside the visible viewport when a phone keyboard opens. */
(function () {
    const viewport=window.visualViewport;
    let scheduled=false;
    function update() {
        scheduled=false;
        document.documentElement.style.setProperty('--l2e-visible-height',(viewport?.height||innerHeight)+'px');
        document.documentElement.style.setProperty('--l2e-visible-top',(viewport?.offsetTop||0)+'px');
        const open=document.querySelector('#reflectionModal:not(.hidden), #addModal:not(.hidden), #deleteBlockDialog[open], [data-viewport-dialog][open]');
        document.body.classList.toggle('workspace-dialog-open',!!open);
        if(open?.contains(document.activeElement))document.activeElement.scrollIntoView({block:'nearest',inline:'nearest'});
    }
    function schedule(){if(!scheduled){scheduled=true;requestAnimationFrame(update);}}
    viewport?.addEventListener('resize',schedule);viewport?.addEventListener('scroll',schedule);
    window.addEventListener('resize',schedule);
    for(const dialog of document.querySelectorAll('#reflectionModal, #addModal, #deleteBlockDialog, [data-viewport-dialog]'))new MutationObserver(schedule).observe(dialog,{attributes:true,attributeFilter:['class','open']});
    // Keep keyboard traversal within the active native modal, including the
    // first/last boundary where Chromium can otherwise move to browser chrome.
    document.addEventListener('keydown',event=>{
        if(event.key!=='Tab'||event.defaultPrevented)return;
        const dialog=document.querySelector('dialog[open]');if(!dialog)return;
        const items=[...dialog.querySelectorAll('a[href],button,input,select,textarea,[tabindex]')].filter(n=>!n.disabled&&n.tabIndex>=0&&n.getClientRects().length);
        if(!items.length){event.preventDefault();dialog.focus();return;}
        const first=items[0],last=items[items.length-1],active=document.activeElement;
        if(event.shiftKey&&(active===first||!dialog.contains(active))){event.preventDefault();last.focus();}
        else if(!event.shiftKey&&(active===last||!dialog.contains(active))){event.preventDefault();first.focus();}
    });
    update();
})();
