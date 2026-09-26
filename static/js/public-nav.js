(() => {
  // Public pages remain readable for everyone. A valid session only changes
  // the account controls; it does not redirect the reader away from the hub.
  fetch('/api/v1/me', {credentials:'same-origin'}).then(response => {
    if (response.ok) document.documentElement.classList.add('is-signed-in');
  }).catch(() => {});
})();
