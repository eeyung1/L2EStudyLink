'use strict';
function notificationSafeLink(value) {
    if (typeof value !== 'string' || !value.startsWith('/') || value.startsWith('//')) return '/dashboard';
    try { const decoded = decodeURIComponent(value); if (decoded.startsWith('//') || /[\\\r\n\t]/.test(decoded)) return '/dashboard'; } catch (_) { return '/dashboard'; }
    return value;
}
function notificationBadge(count) { return count > 99 ? '99+' : String(Math.max(0, count)); }
function notificationReadFrames(buffer, receive) {
    buffer = buffer.replace(/\r\n/g, '\n'); let end;
    while ((end = buffer.indexOf('\n\n')) !== -1) {
        const frame = buffer.slice(0, end); buffer = buffer.slice(end + 2);
        if (frame.length > 131072) throw new Error('Update too large');
        let event = 'message'; const data = [];
        for (const line of frame.split('\n')) { if (line.startsWith('event:')) event = line.slice(6).trim(); if (line.startsWith('data:')) data.push(line.slice(5).replace(/^ /, '')); }
        if (event === 'notifications' && data.length) receive(JSON.parse(data.join('\n')));
    }
    if (buffer.length > 131072) throw new Error('Update too large'); return buffer;
}
if (typeof module !== 'undefined' && module.exports) module.exports = { notificationSafeLink, notificationBadge, notificationReadFrames };
if (typeof document !== 'undefined') {
    (async () => {
        if (document.getElementById('notification-widget')) return;
        const host = document.querySelector('.sidebar-heading'); if (!host) return;
        const token = sessionToken(), initialScope = sessionScope();
        const headers = () => ({ Authorization: 'Bearer ' + token });
        let disposed = false, controller = new AbortController(), items = [], lastID = 0, hasMore = false, unread = 0, busy = false, showingOlder = false;
        const accountChanged = () => { if (sessionScope() === initialScope) return false; controller.abort(); window.location.reload(); return true; };
        const request = async (path, method = 'GET', body, keepalive = false) => {
            if (accountChanged()) throw new Error('Account changed');
            const options = { method, headers: headers(), credentials: 'same-origin', cache: 'no-store', keepalive };
            if (body !== undefined) { options.headers['Content-Type'] = 'application/json'; options.body = JSON.stringify(body); }
            const response = await fetch('/api/v1/notifications' + path, options);
            if (accountChanged()) throw new Error('Account changed');
            if (response.status === 204) return null;
            let data; try { data = await response.json(); } catch (_) { throw new Error('Unable to load notifications. Please try again.'); }
            if (!response.ok) throw new Error(data.error || 'Notifications are temporarily unavailable.'); return data;
        };
        let fragment;
        try {
            const response = await fetch('/api/v1/notifications/widget', { headers: headers(), credentials: 'same-origin', cache: 'no-store' });
            if (!response.ok || accountChanged()) return;
            // This fragment is static application markup, never notification text.
            fragment = new DOMParser().parseFromString(await response.text(), 'text/html');
        } catch (_) { return; }
        for (const style of fragment.querySelectorAll('style')) document.head.append(document.importNode(style, true));
        const widget = fragment.getElementById('notification-widget'); if (!widget || document.getElementById('notification-widget')) return;
        host.append(document.importNode(widget, true));
        const el = id => document.getElementById(id), panel = el('notification-panel');
        const status = text => { el('notification-status').textContent = text; };
        function position() { const rect = el('notification-toggle').getBoundingClientRect(); const width = Math.min(370, window.innerWidth - 32); panel.style.left = Math.max(16, Math.min(rect.left, window.innerWidth - width - 16)) + 'px'; panel.style.top = Math.max(12, Math.min(rect.bottom + 8, window.innerHeight - 180)) + 'px'; }
        function close(restoreFocus = false) { panel.hidden = true; el('notification-toggle').setAttribute('aria-expanded', 'false'); if (restoreFocus) el('notification-toggle').focus(); }
        function render() {
            const badge = el('notification-badge'); badge.hidden = unread === 0; badge.textContent = notificationBadge(unread);
            el('notification-toggle').setAttribute('aria-label', unread ? `Notifications: ${unread} unread` : 'Notifications: all read');
            el('notification-read-all').disabled = busy || unread === 0;
            const list = el('notification-items'); list.replaceChildren();
            for (const item of items) {
                const row = document.createElement('li'); row.className = 'notification-item' + (item.is_read ? '' : ' unread');
                const link = document.createElement('a'), title = document.createElement('strong'), message = document.createElement('p'), time = document.createElement('time');
                link.href = notificationSafeLink(item.link_url); title.textContent = item.title; message.textContent = item.message; time.dateTime = item.created_at;
                const date = new Date(item.created_at); time.textContent = Number.isNaN(date.getTime()) ? '' : date.toLocaleString(); link.append(title, message, time);
                link.addEventListener('click', () => { if (!item.is_read) request('/' + item.id + '/read', 'PUT', undefined, true).catch(() => {}); }); row.append(link);
                if (!item.is_read) { const button = document.createElement('button'); button.type = 'button'; button.textContent = 'Mark read'; button.disabled = busy; button.addEventListener('click', () => mark(item.id)); row.append(button); }
                list.append(row);
            }
            el('notification-older').hidden = !hasMore; el('notification-older').disabled = busy;
        }
        function receive(data) {
            if (disposed || accountChanged()) return;
            unread = data.unread_count; lastID = data.last_event_id;
            // Preserve an explicitly paged history while new live events arrive.
            if (!showingOlder) { items = data.items; hasMore = data.has_more; }
            else { const latest = new Map(data.items.map(item => [item.id, item])); items = items.map(item => latest.get(item.id) || item); for (const item of data.items) if (!items.some(existing => existing.id === item.id)) items.unshift(item); items.sort((a, b) => b.id - a.id); }
            render(); status(items.length ? 'Your latest updates.' : 'You are all caught up. New updates will appear here.');
        }
        async function refresh() { try { showingOlder = false; receive(await request('')); } catch (error) { status(error.message); } }
        async function mark(id) { if (busy) return; busy = true; render(); try { await request('/' + id + '/read', 'PUT'); const item = items.find(item => item.id === id); if (item) item.is_read = true; await refresh(); } catch (error) { status(error.message); } finally { busy = false; render(); } }
        el('notification-toggle').addEventListener('click', () => { if (panel.hidden) { panel.hidden = false; position(); el('notification-toggle').setAttribute('aria-expanded', 'true'); panel.focus(); refresh(); } else close(); });
        el('notification-close').addEventListener('click', () => close(true));
        document.addEventListener('click', event => { if (!panel.hidden && !el('notification-widget').contains(event.target)) close(); });
        document.addEventListener('keydown', event => { if (event.key === 'Escape' && !panel.hidden) close(true); });
        window.addEventListener('resize', () => { if (!panel.hidden) position(); });
        el('notification-refresh').addEventListener('click', refresh);
        el('notification-read-all').addEventListener('click', async () => { if (busy) return; busy = true; render(); try { await request('/read', 'PUT', { through_id: lastID }); await refresh(); } catch (error) { status(error.message); } finally { busy = false; render(); } });
        el('notification-older').addEventListener('click', async () => {
            if (busy || !items.length) return; busy = true; render();
            try { const data = await request('?before_id=' + items[items.length - 1].id); showingOlder = true; for (const item of data.items) if (!items.some(old => old.id === item.id)) items.push(item); hasMore = data.has_more; unread = data.unread_count; lastID = data.last_event_id; }
            catch (error) { status(error.message); } finally { busy = false; render(); }
        });
        const sleep = ms => new Promise(resolve => { if (controller.signal.aborted) return resolve(); const done = () => { clearTimeout(timer); controller.signal.removeEventListener('abort', done); resolve(); }; const timer = setTimeout(done, ms); controller.signal.addEventListener('abort', done, { once: true }); });
        async function stream() {
            let retry = 2000;
            while (!disposed && !controller.signal.aborted) {
                let reader;
                try {
                    const response = await fetch('/api/v1/notifications/stream', { headers: headers(), credentials: 'same-origin', cache: 'no-store', signal: controller.signal });
                    if (accountChanged()) return;
                    if ([401, 403].includes(response.status)) { controller.abort(); return; }
                    if (!response.ok || !response.body || !response.headers.get('content-type')?.includes('text/event-stream')) throw new Error('Live updates interrupted');
                    retry = 2000; reader = response.body.getReader(); const decoder = new TextDecoder(); let buffer = '';
                    while (!disposed && !controller.signal.aborted) { const { value, done } = await reader.read(); if (done) break; buffer = notificationReadFrames(buffer + decoder.decode(value, { stream: true }), receive); }
                } catch (_) { if (disposed || controller.signal.aborted) return; }
                finally { if (reader) { try { await reader.cancel(); } catch (_) {} } }
                if (disposed || controller.signal.aborted) return;
                status('Reconnecting live updates. You can still refresh notifications.');
                try { receive(await request('')); } catch (_) {}
                await sleep(retry); retry = Math.min(30000, retry * 2);
            }
        }
        window.addEventListener('pagehide', () => { disposed = true; controller.abort(); });
        window.addEventListener('pageshow', event => { if (event.persisted) window.location.reload(); });
        document.addEventListener('visibilitychange', () => { if (!document.hidden && !accountChanged()) refresh(); });
        await refresh(); stream();
    })();
}
