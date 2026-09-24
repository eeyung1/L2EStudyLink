const {test}=require('node:test');
const assert=require('node:assert/strict');
const vm=require('node:vm');
const fs=require('node:fs');

function page({cookie='',legacy=null}={}) {
 const entries=new Map(legacy?[['token',legacy]]:[]), calls=[];
 const fetch=async (path,options)=>{calls.push({path,options});return {ok:true,status:200};};
 const context=vm.createContext({
   document:{cookie},window:{location:{href:''},fetch},fetch,
   localStorage:{getItem:key=>entries.get(key)||null,removeItem:key=>entries.delete(key)},
   alert:()=>{throw Error('Unexpected alert')},
   Promise
 });
 vm.runInContext(fs.readFileSync('static/js/session.js','utf8'),context);
 return {context,calls,entries};
}

test('cookie scope is a non-secret cache key',()=>{
 const {context,entries}=page({cookie:'l2e_scope=abc123'});
 assert.equal(context.sessionToken(),'cookie-session-abc123');
 assert.equal(entries.has('token'),false);
});

test('legacy bearer token is exchanged and removed from storage',async()=>{
 const {context,calls,entries}=page({legacy:'old-jwt'});
 assert.equal(context.sessionToken(),'old-jwt');
 await new Promise(resolve=>setTimeout(resolve,0));
 assert.equal(calls[0].path,'/api/v1/session/migrate');
 assert.equal(calls[0].options.headers.Authorization,'Bearer old-jwt');
 assert.equal(entries.has('token'),false);
});

test('logout clears cookie through the server before redirecting',async()=>{
 const {context,calls}=page({cookie:'l2e_scope=abc123'});
 await context.endSession(context.sessionToken());
 assert.equal(calls[0].path,'/api/v1/logout');
 assert.equal(context.window.location.href,'/page/login');
});
