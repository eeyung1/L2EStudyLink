const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const html = fs.readFileSync('templates/timetable.html', 'utf8');
const source = html.slice(html.indexOf('        const compactAgenda ='), html.indexOf('        window.logout ='));
function page(jsDay, compact = true) {
    const cards = Array.from({length:7}, (_, day) => ({id:`day-${day}`, hidden:false}));
    const buttons = [...Array.from({length:7}, (_, day) => String(day)), 'all'].map(day => ({dataset:{day}, setAttribute(name,value){this[name]=value;}}));
    const media={matches:compact,addEventListener(event,listener){this.change=listener;}};
    const context = {matchMedia:()=>media,Date:class {getDay(){return jsDay;}}, document:{getElementById(){return {classList:{toggle(){}}};},querySelectorAll(selector){return selector.includes('card') ? cards : buttons;}}};
    vm.createContext(context);
    vm.runInContext(source, context);
    return {cards, buttons, resize(compact){media.matches=compact;media.change();}, choose(day){vm.runInContext(`selectTimetableDay(${JSON.stringify(day)})`, context);}};
}
test('defaults to device weekday including Sunday and Monday mapping', () => {
    for (let jsDay=0; jsDay<7; jsDay++) {
        const p=page(jsDay);
        assert.deepEqual(p.cards.filter(c=>!c.hidden).map(c=>c.id), [`day-${(jsDay+6)%7}`]);
        assert.equal(p.buttons[(jsDay+6)%7]['aria-pressed'], 'true');
    }
});
test('another day and All update cards and selected controls without fetching', () => {
    const p=page(3);
    p.choose(4);
    assert.deepEqual(p.cards.filter(c=>!c.hidden).map(c=>c.id), ['day-4']);
    p.choose('all');
    assert.equal(p.cards.filter(c=>!c.hidden).length, 7);
    assert.equal(p.buttons[7]['aria-pressed'], 'true');
    p.choose(0);
    assert.equal(p.cards.filter(c=>!c.hidden).length, 1);
    assert.equal(p.buttons[7]['aria-pressed'], 'false');
});
for (const [,script] of html.matchAll(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/g)) new vm.Script(script);

test('desktop defaults to week and an explicit selection survives resizing', () => {
 const p=page(1,false);assert.equal(p.cards.filter(c=>!c.hidden).length,7);
 p.resize(true);assert.deepEqual(p.cards.filter(c=>!c.hidden).map(c=>c.id),['day-0']);
 p.choose(3);p.resize(false);assert.deepEqual(p.cards.filter(c=>!c.hidden).map(c=>c.id),['day-3']);
});
