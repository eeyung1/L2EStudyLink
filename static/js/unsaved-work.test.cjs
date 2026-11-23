const {test}=require('node:test');
const assert=require('node:assert/strict');
const {registry}=require('./unsaved-work.js');
test('independent unsaved sources survive saving one entry and can be discarded',()=>{const r=registry();let text='',voice=false;r.watch('text',()=>!!text);r.watch('voice',()=>voice);assert.equal(r.dirty(),false);text='A private draft';voice=true;assert.equal(r.dirty(),true);text='';assert.equal(r.dirty(),true);r.remove('voice');assert.equal(r.dirty(),false);});
test('registering a source again replaces its previous state',()=>{const r=registry();r.watch('form',()=>true);r.watch('form',()=>false);assert.equal(r.dirty(),false);});
