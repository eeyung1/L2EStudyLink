const {test} = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');
const {webcrypto} = require('node:crypto');
const {TextEncoder} = require('node:util');

function setup() {
    const entries = new Map();
    const storage = {getItem:key=>entries.get(key)??null,setItem:(key,value)=>entries.set(key,value),removeItem:key=>entries.delete(key)};
    const requests = [];
    const context = vm.createContext({
        crypto:webcrypto,TextEncoder,Uint8Array,Map,Set,Array,JSON,Date,Promise,Error,
        sessionStorage:storage,localStorage:storage,window:{location:{href:''}},setTimeout,
        fetch:async (path,options)=>{
            requests.push({path,token:options.headers.Authorization});
            await new Promise(resolve=>setTimeout(resolve,5));
            return {status:200,ok:true,json:async()=>[{id:requests.length}]};
        }
    });
    vm.runInContext(fs.readFileSync('static/js/read-api.js','utf8'),context);
    return {context,requests,entries};
}

test('concurrent reads share one request and return cached results',async()=>{
    const {context,requests}=setup();
    const results=await Promise.all([
        context.readAPI('/api/v1/timetable','user-a'),
        context.readAPI('/api/v1/timetable','user-a')
    ]);
    assert.equal(requests.length,1);
    assert.deepEqual(JSON.stringify(results[0]),JSON.stringify(results[1]));
    await context.readAPI('/api/v1/timetable','user-a');
    assert.equal(requests.length,1);
});

test('invalidation refetches changes without crossing accounts',async()=>{
    const {context,requests,entries}=setup();
    await context.readAPI('/api/v1/reflections','user-a');
    await context.invalidateReadCache(['/api/v1/reflections'],'user-a');
    await context.readAPI('/api/v1/reflections','user-a');
    assert.equal(requests.length,2);
    await context.readAPI('/api/v1/reflections','user-b');
    assert.equal(requests.length,3);
    assert.equal(entries.size,2);
    assert.ok([...entries.keys()].every(key=>!key.includes('user-a')&&!key.includes('user-b')));
});

test('invalidation during an in-flight read does not store stale results',async()=>{
    const {context,requests}=setup();
    const first=context.readAPI('/api/v1/timetable','user-a');
    await new Promise(resolve=>setTimeout(resolve,0));
    await context.invalidateReadCache(['/api/v1/timetable'],'user-a');
    const second=context.readAPI('/api/v1/timetable','user-a');
    await Promise.all([first,second]);
    await context.readAPI('/api/v1/timetable','user-a');
    assert.equal(requests.length,2);
});

test('simultaneous account scopes stay separate', async()=>{
    const {context,requests,entries}=setup();
    await Promise.all([
        context.readAPI('/api/v1/timetable','user-a'),
        context.readAPI('/api/v1/timetable','user-b')
    ]);
    assert.equal(requests.length,2);
    assert.equal(entries.size,2);
    await context.readAPI('/api/v1/timetable','user-a');
    await context.readAPI('/api/v1/timetable','user-b');
    assert.equal(requests.length,2);
});
