const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

const source = fs.readFileSync('static/js/learning-return.js','utf8');
function destination(search) {
  const context = {window:{location:{search}},URLSearchParams};
  vm.createContext(context);
  vm.runInContext(source,context);
  return vm.runInContext('learningReturnPath()',context);
}

test('returns a reader to the guide they opened', () => {
  assert.equal(destination('?next=%2Flearn%2Fstart-an-agentic-workflow'),'/learn/start-an-agentic-workflow');
  assert.equal(destination(''),'/learn');
});

test('rejects external and non-learning return URLs', () => {
  for (const next of ['//evil.example','https://evil.example','/dashboard','/learn/../admin','/learn/%2F%2Fevil.example']) {
    assert.equal(destination('?next='+encodeURIComponent(next)),'/learn');
  }
});
