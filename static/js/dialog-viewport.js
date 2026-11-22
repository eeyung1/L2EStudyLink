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
    update();
})();
