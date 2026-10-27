const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

const source = fs.readFileSync('static/js/heatmap.js', 'utf8');

function runHeatmap(counts, options = {}) {
  const container = { innerHTML: '' };
  const context = {
    container,
    counts,
    options,
    Date,
    String,
    module: {},
    window: { innerWidth: 1024 }
  };
  vm.createContext(context);
  vm.runInContext(source, context);
  vm.runInContext('renderHeatmap(container, counts, options)', context);
  return container.innerHTML;
}

test('renderHeatmap renders grid with correct activity level classes', () => {
  const counts = {
    '2026-10-27': 4,
    '2026-10-26': 2,
    '2026-10-25': 1,
  };
  const html = runHeatmap(counts, { today: '2026-10-27', weeks: 2 });
  
  assert.ok(html.includes('class="heatmap-grid"'));
  assert.ok(html.includes('data-date="2026-10-27" data-count="4"'));
  assert.ok(html.includes('class="heatmap-cell level-4" data-date="2026-10-27"'));
  assert.ok(html.includes('class="heatmap-cell level-2" data-date="2026-10-26"'));
  assert.ok(html.includes('class="heatmap-cell level-1" data-date="2026-10-25"'));
  assert.ok(html.includes('class="heatmap-cell level-0"'));
});

test('renderHeatmap renders legend and footer with configured weeks', () => {
  const html = runHeatmap({}, { today: '2026-10-27', weeks: 12 });
  assert.ok(html.includes('Past 12 weeks'));
  assert.ok(html.includes('heatmap-legend'));
});
