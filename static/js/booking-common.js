window.L2EBooking=(function(){
    function node(tag,text,className){const n=document.createElement(tag);if(text!==undefined)n.textContent=text;if(className)n.className=className;return n;}
    function time(value){const match=String(value||'').match(/(?:T|^)(\d{2}:\d{2})/);return match?match[1]:'';}
    function day(date){return (new Date(date+'T12:00:00').getDay()+6)%7;}
    async function api(path,options={}){
        const response=await fetch('/api/v1'+path,{...options,headers:{Authorization:'Bearer '+sessionToken(),...(options.body?{'Content-Type':'application/json'}:{}),...options.headers}});
        const value=await response.json().catch(()=>({}));if(!response.ok){const error=Error(value.error||'Request unavailable. Please try again.');error.status=response.status;throw error;}return value;
    }
    async function clock(){const value=await api('/booking-clock');if(!value.timezone||!value.today)throw Error('Booking clock unavailable');return value;}
    function labelled(parent,label,value){parent.append(node('dt',label),node('dd',value));}
    function configureDialog(dialog,canClose=()=>true){let opener=null;dialog.addEventListener('cancel',event=>{if(!canClose())event.preventDefault();});dialog.addEventListener('close',()=>{if(opener?.isConnected)opener.focus();opener=null;});return ()=>{opener=document.activeElement;dialog.showModal();};}
    return {node,time,day,api,clock,labelled,configureDialog};
})();
