(() => {
  if ('serviceWorker' in navigator) window.addEventListener('load', () => navigator.serviceWorker.register('/sw.js').catch(() => {}));
  let installPrompt;
  const button = document.getElementById('installApp');
  if (!button) return;
  window.addEventListener('beforeinstallprompt', event => {
    event.preventDefault(); installPrompt = event; button.hidden = false;
  });
  window.addEventListener('appinstalled', () => { button.hidden = true; installPrompt = null; });
  button.addEventListener('click', async () => {
    if (!installPrompt) return;
    installPrompt.prompt();
    await installPrompt.userChoice;
    installPrompt = null; button.hidden = true;
  });
})();
