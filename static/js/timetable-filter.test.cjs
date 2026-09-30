const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const html = fs.readFileSync('templates/timetable.html', 'utf8');
const source = html.slice(html.indexOf('        let timetableDay ='), html.indexOf('        window.logout ='));
function page(jsDay) {
    const cards = Array.from({length:7}, (_, day) => ({id:`day-${day}`, hidden:false}));
    const buttons = [...Array.from({length:7}, (_, day) => String(day)), 'all'].map(day => ({dataset:{day}, setAttribute(name,value){this[name]=value;}}));
    const context = {Date:class {getDay(){return jsDay;}}, document:{querySelectorAll(selector){return selector.includes('card') ? cards : buttons;}}};
    vm.createContext(context);
    vm.runInContext(source, context);
    return {cards, buttons, choose(day){vm.runInContext(`selectTimetableDay(${JSON.stringify(day)})`, context);}};
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
