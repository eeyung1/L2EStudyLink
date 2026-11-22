/* Isolated fictional preview: deliberately contains no API calls or account writes. */
(() => {
  const root = document.body;
  const dashboard = document.getElementById('dashboard-content');
  const timetable = document.getElementById('timetable-content');
  const originalDashboard = dashboard.innerHTML;
  const stateInput = document.getElementById('preview-state');
  const days = ['Monday','Tuesday','Wednesday','Thursday','Friday','Saturday','Sunday'];
  let selectedDay = 0;
  let view = 'dashboard';
  const themeButton = document.getElementById('theme-toggle');
  function setTheme(theme) {
    root.dataset.l2eTheme = theme;
    themeButton.setAttribute('aria-label',`Switch to ${theme === 'dark' ? 'light' : 'dark'} appearance`);
  }
  try { setTheme(localStorage.getItem('l2e-redesign-theme') === 'dark' ? 'dark' : 'light'); } catch { setTheme('light'); }
  themeButton.addEventListener('click', () => {
    const theme = root.dataset.l2eTheme === 'dark' ? 'light' : 'dark';
    setTheme(theme);
    try { localStorage.setItem('l2e-redesign-theme',theme); } catch { /* Theme remains usable with blocked storage. */ }
  });
  function closeMore() {
    document.getElementById('more-tools').hidden = true;
    document.getElementById('more-button').setAttribute('aria-expanded','false');
  }
  function render() {
    const state = stateInput.value;
    dashboard.removeAttribute('aria-busy'); timetable.removeAttribute('aria-busy');
    if (state === 'populated') {
      dashboard.innerHTML = originalDashboard;
      timetable.innerHTML = `<div class="week-grid">${days.map((day,i)=>`<section class="week-day${i === selectedDay ? ' selected' : ''}" aria-label="${day} sample schedule"><h3>${day.slice(0,3)} <span>${23+i}</span></h3>${i < 5 ? `<div class="week-block"><span>07:00–08:00</span><strong>Daily reading</strong><small>Personal study</small></div>${i % 2 === 0 ? '<div class="week-block"><span>09:00–11:00</span><strong>Build a Go API</strong><small>Project work</small></div>' : '<div class="week-block"><span>14:00–15:00</span><strong>Peer learning</strong><small>Sample booking</small></div>'}` : '<p class="free">Room to recharge.</p>'}</section>`).join('')}</div>`;
    } else {
      const states = {
        empty: '<h2>A little space for a new plan.</h2><p>No sample blocks in this view. Start by choosing one thing you want to learn.</p><button class="primary" data-action="block">Plan a study block</button>',
        loading: '<h2>Loading your learning space…</h2><p>Sample loading state.</p><div class="skeleton" aria-hidden="true"></div><div class="skeleton" aria-hidden="true"></div>',
        error: '<h2>We couldn’t load this view.</h2><p>Your saved work is not changed. This is a simulated connection failure.</p><button class="primary" data-retry>Try again</button>',
        expired: '<h2>Your session has expired.</h2><p>Sign in again to continue. This preview does not authenticate you.</p><a href="/page/login">Go to sign in →</a>'
      };
      const html = `<div class="state-card${state === 'error' ? ' error' : ''}"${state === 'error' ? ' role="alert"' : ''}>${states[state]}</div>`;
      dashboard.innerHTML = html; timetable.innerHTML = html;
      if (state === 'loading') { dashboard.setAttribute('aria-busy','true'); timetable.setAttribute('aria-busy','true'); }
    }
    document.querySelectorAll('[data-day]').forEach(button => button.setAttribute('aria-pressed',String(Number(button.dataset.day) === selectedDay)));
  }
  document.querySelector('.day-selector').innerHTML = days.map((day,i)=>`<button type="button" data-day="${i}" aria-label="Show ${day}" aria-pressed="${i===0}">${day.slice(0,3)}</button>`).join('');
  function selectView(next) {
    view = next;
    document.getElementById('dashboard-view').hidden = view !== 'dashboard';
    document.getElementById('timetable-view').hidden = view !== 'timetable';
    document.getElementById('breadcrumb').textContent = view === 'dashboard' ? 'Today' : 'Timetable';
    document.querySelectorAll('[data-view]').forEach(button => button.setAttribute('aria-pressed',String(button.dataset.view === view)));
    closeMore();
    document.getElementById('feedback').textContent = '';
  }
  document.addEventListener('click', event => {
    const button = event.target.closest('button'); if (!button) return;
    if (button.dataset.view) selectView(button.dataset.view);
    if (button.dataset.day !== undefined) { selectedDay = Number(button.dataset.day); render(); }
    if (button.hasAttribute('data-retry')) { stateInput.value = 'populated'; render(); }
    if (button.dataset.action) { closeMore(); document.getElementById('feedback').textContent = 'Preview only: this action will open the corresponding learning flow when the redesign is integrated. Nothing has been saved.'; }
  });
  document.getElementById('more-button').addEventListener('click', () => {
    const panel = document.getElementById('more-tools'); panel.hidden = !panel.hidden;
    document.getElementById('more-button').setAttribute('aria-expanded',String(!panel.hidden));
    if (!panel.hidden) panel.querySelector('button').focus();
  });
  document.getElementById('close-more').addEventListener('click', () => { closeMore(); document.getElementById('more-button').focus(); });
  document.addEventListener('keydown', event => { if (event.key === 'Escape' && !document.getElementById('more-tools').hidden) { closeMore(); document.getElementById('more-button').focus(); } });
  stateInput.addEventListener('change', () => { document.getElementById('feedback').textContent = ''; render(); });
  render();
})();
