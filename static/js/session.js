// The authorization cookie is HttpOnly. This readable scope is random and
// grants no access; it only keeps each account's browser cache separate.
function sessionScope() {
    const match = document.cookie.match(/(?:^|; )l2e_scope=([a-f0-9]+)/);
    return match ? match[1] : '';
}

function sessionToken() {
    const old = localStorage.getItem('token');
    if (old && !sessionScope()) {
        fetch('/api/v1/session/migrate', {
            method: 'POST',
            headers: {Authorization: 'Bearer ' + old},
            credentials: 'same-origin'
        }).then(response => { if (response.ok) localStorage.removeItem('token'); }).catch(() => {});
        return old;
    }
    if (old) localStorage.removeItem('token');
    return 'cookie-session-' + sessionScope();
}

const sessionFetch = window.fetch.bind(window);
window.fetch = async function(input, options) {
    const response = await sessionFetch(input, options);
    const path = typeof input === 'string' ? input : input.url;
    if (response.status === 401 && path.startsWith('/api/v1/') && path !== '/api/v1/session/migrate') {
        window.location.href = '/page/login';
    }
    return response;
};

async function endSession(token) {
    if (typeof clearPlanningCache === 'function') await clearPlanningCache(token);
    try {
        const response = await fetch('/api/v1/logout', {method:'POST',credentials:'same-origin'});
        if (!response.ok) throw Error('Could not sign out');
    } catch (_) {
        alert('Could not sign out. Please try again.');
        return;
    }
    localStorage.removeItem('token');
    window.location.href = '/page/login';
}
