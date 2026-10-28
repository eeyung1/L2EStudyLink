'use strict';

// Pure helpers are also used by the Go-driven Node regression test.
function focusReadEvents(buffer, receive) {
    buffer = buffer.replace(/\r\n/g, '\n');
    let end;
    while ((end = buffer.indexOf('\n\n')) !== -1) {
        const frame = buffer.slice(0, end); buffer = buffer.slice(end + 2);
        if (frame.length > 131072) throw new Error('Room update too large');
        let event = 'message'; const data = [];
        for (const line of frame.split('\n')) {
            if (line.startsWith('event:')) event = line.slice(6).trim();
            if (line.startsWith('data:')) data.push(line.slice(5).replace(/^ /, ''));
        }
        if (data.length) receive(event, JSON.parse(data.join('\n')));
    }
    if (buffer.length > 131072) throw new Error('Room update too large');
    return buffer;
}
function focusSeconds(anchor, now) { return Math.max(0, Math.ceil((anchor.duration - (now - anchor.at)) / 1000)); }
function focusPhaseKey(timer) { return timer.cycle + ':' + timer.phase; }
if (typeof module !== 'undefined' && module.exports) module.exports = { focusReadEvents, focusSeconds, focusPhaseKey };

if (typeof document !== 'undefined' && document.getElementById('focus-app')) {
    (() => {
        const el = id => document.getElementById(id);
        const token = sessionToken();
        let room = null, anchor = null, phase = null, stream = null, generation = 0, busy = false, disposed = false;
        let audio = null, phaseKey = null;
        const auth = () => ({ Authorization: 'Bearer ' + token });
        const status = (text, error = false) => { el('focus-status').textContent = text; el('focus-status').classList.toggle('error', error); };
        function setBusy(value) {
            busy = value;
            for (const id of ['focus-create-button', 'focus-refresh', 'focus-leave', 'focus-save-target']) el(id).disabled = value;
            for (const button of el('focus-rooms').querySelectorAll('button')) button.disabled = value;
        }
        async function api(path, method = 'GET', body) {
            const options = { method, credentials: 'same-origin', headers: auth(), cache: 'no-store' };
            if (body !== undefined) { options.headers['Content-Type'] = 'application/json'; options.body = JSON.stringify(body); }
            const response = await fetch('/api/v1/focus/' + path, options);
            if (response.status === 204) return null;
            let value; try { value = await response.json(); } catch (_) { throw new Error('We could not load the room. Please try again.'); }
            if (!response.ok) { const error = new Error(value.error || 'We could not load the room. Please try again.'); error.status = response.status; throw error; }
            return value;
        }
        function stopStream() { generation++; if (stream) stream.abort(); stream = null; }
        function resetRoom() {
            stopStream(); room = null; anchor = null; phaseKey = null;
            el('focus-session').hidden = true; el('focus-roster-panel').hidden = true;
            el('focus-lobby').hidden = false; el('focus-discovery').hidden = false; el('focus-roster').replaceChildren();
        }
        function chime() {
            if (!el('focus-chime').checked || !audio || audio.state !== 'running') return;
            const oscillator = audio.createOscillator(), gain = audio.createGain();
            oscillator.connect(gain); gain.connect(audio.destination); oscillator.frequency.value = 660;
            gain.gain.setValueAtTime(0, audio.currentTime); gain.gain.linearRampToValueAtTime(0.12, audio.currentTime + 0.03); gain.gain.exponentialRampToValueAtTime(0.001, audio.currentTime + 0.5);
            oscillator.start(); oscillator.stop(audio.currentTime + 0.55);
        }
        function renderClock(timer) {
            const key = focusPhaseKey(timer); if (phaseKey && key !== phaseKey) chime(); phaseKey = key; phase = timer.phase;
            anchor = { duration: Math.max(0, Date.parse(timer.ends_at) - Date.parse(timer.server_time)), at: performance.now() };
            el('focus-phase').textContent = phase === 'break' ? 'Take a breath' : phase === 'focus' ? 'Time to focus' : 'Starting soon';
            el('focus-cycle').textContent = 'Cycle ' + timer.cycle + ' · 25 minutes focus / 5 minutes break';
            el('focus-progress').max = phase === 'break' ? 300 : 1500; paintClock();
        }
        function paintClock() {
            if (!anchor) return;
            const seconds = focusSeconds(anchor, performance.now());
            el('focus-time').textContent = String(Math.floor(seconds / 60)).padStart(2, '0') + ':' + String(seconds % 60).padStart(2, '0');
            el('focus-progress').value = Math.min(el('focus-progress').max, seconds);
            if (!seconds) el('focus-connection').textContent = 'Waiting for the next room update…';
        }
        function renderRoom(value) {
            room = value; el('focus-active-title').textContent = value.title; el('focus-count').textContent = '(' + value.participants.length + ')';
            const list = el('focus-roster'); list.replaceChildren();
            for (const person of value.participants) {
                const item = document.createElement('li'); item.className = 'focus-person';
                const name = document.createElement('strong'), target = document.createElement('p');
                name.textContent = person.display_name; target.textContent = person.target; item.append(name, target); list.append(item);
            }
            renderClock(value.timer);
        }
        const delay = (ms, signal) => new Promise(resolve => {
            if (signal.aborted) { resolve(); return; }
            const done = () => { clearTimeout(timer); signal.removeEventListener('abort', done); resolve(); };
            const timer = setTimeout(done, ms); signal.addEventListener('abort', done, { once: true });
        });
        async function followRoom(id) {
            stopStream(); const current = generation; stream = new AbortController(); const controller = stream;
            let backoff = 1500;
            while (!disposed && !controller.signal.aborted && current === generation && room && room.id === id) {
                let reader;
                try {
                    const response = await fetch('/api/v1/focus/stream?' + new URLSearchParams({ room_id: id }), { headers: auth(), credentials: 'same-origin', cache: 'no-store', signal: controller.signal });
                    if ([401, 403, 404].includes(response.status)) { resetRoom(); status('Your room membership ended. Join a room to continue.', true); refreshRooms(); return; }
                    if (response.status === 429) { const data = await response.json().catch(() => ({})); status(data.error || 'Live rooms are busy. Close another focus tab or try again shortly.', true); }
                    if (!response.ok || !response.body || !response.headers.get('content-type')?.includes('text/event-stream')) throw new Error('Room connection interrupted');
                    el('focus-connection').textContent = 'Live · Your timer is synced with the room';
                    status('You are in. Protect this one small intention.');
                    phaseKey = null; backoff = 1500;
                    reader = response.body.getReader(); const decoder = new TextDecoder(); let buffer = '';
                    while (!controller.signal.aborted && current === generation) {
                        const { value, done } = await reader.read(); if (done) break;
                        buffer = focusReadEvents(buffer + decoder.decode(value, { stream: true }), (event, data) => {
                            if (current !== generation) return;
                            if (event === 'presence') renderRoom(data);
                            if (event === 'clock') renderClock(data);
                            if (event === 'room_closed' || event === 'membership_expired') { resetRoom(); status('Your room membership ended. Join a room to continue.'); refreshRooms(); }
                        });
                    }
                } catch (error) { if (controller.signal.aborted || current !== generation) return; }
                finally { if (reader) { try { await reader.cancel(); } catch (_) {} } }
                if (controller.signal.aborted || current !== generation) return;
                el('focus-connection').textContent = 'Reconnecting… Your displayed timer may be out of date.';
                await delay(backoff, controller.signal); backoff = Math.min(10000, backoff * 2);
            }
        }
        async function enter(id) {
            if (busy) return;
            const target = el('focus-goal').value.trim(); if (!target) { status('Set a study target before joining.', true); el('focus-goal').focus(); return; }
            setBusy(true);
            try {
                const value = id ? await api('rooms/' + id + '/join', 'POST', { target }) : await api('rooms', 'POST', { title: el('focus-room-title').value.trim(), target });
                renderRoom(value); el('focus-active-goal').value = target; el('focus-session').hidden = false; el('focus-roster-panel').hidden = false;
                el('focus-lobby').hidden = true; el('focus-discovery').hidden = true;
                status('You are in. Protect this one small intention.'); followRoom(value.id);
            } catch (error) { status(error.message, true); } finally { setBusy(false); }
        }
        async function refreshRooms() {
            if (disposed || room) return;
            try {
                const data = await api('rooms'); if (disposed || room) return;
                const list = el('focus-rooms'); list.replaceChildren();
                if (!data.rooms.length) { const empty = document.createElement('li'); empty.textContent = 'A quiet start. Create the first room.'; empty.className = 'focus-muted'; list.append(empty); }
                for (const item of data.rooms) {
                    const row = document.createElement('li'); row.className = 'focus-room';
                    const text = document.createElement('div'), title = document.createElement('strong'), detail = document.createElement('p'), button = document.createElement('button');
                    title.textContent = item.title; detail.textContent = item.participant_count + ' here · ' + (item.timer.phase === 'break' ? 'On a break' : 'Focusing'); detail.className = 'focus-count';
                    text.append(title, detail); button.type = 'button'; button.textContent = data.current_room_id === item.id ? 'Return to room' : 'Join room'; button.disabled = busy; button.addEventListener('click', () => enter(item.id)); row.append(text, button); list.append(row);
                }
                status('Choose your target, then join a room or create one.');
            } catch (error) { status(error.message, true); }
        }
        el('focus-create').addEventListener('submit', event => { event.preventDefault(); enter(); });
        el('focus-refresh').addEventListener('click', refreshRooms);
        el('focus-leave').addEventListener('click', async () => {
            if (busy || !room) return; setBusy(true);
            try { await api('rooms/' + room.id + '/membership', 'DELETE'); resetRoom(); status('Session complete. Take your progress with you.'); await refreshRooms(); }
            catch (error) { if (error.status === 404) { resetRoom(); refreshRooms(); } else status(error.message, true); }
            finally { setBusy(false); }
        });
        el('focus-target-form').addEventListener('submit', async event => {
            event.preventDefault(); if (busy || !room) return; setBusy(true);
            try { renderRoom(await api('rooms/' + room.id + '/target', 'PUT', { target: el('focus-active-goal').value.trim() })); status('Your intention is updated.'); }
            catch (error) { status(error.message, true); } finally { setBusy(false); }
        });
        el('focus-chime').addEventListener('change', async () => {
            if (!el('focus-chime').checked) return;
            try { const Audio = window.AudioContext || window.webkitAudioContext; if (!Audio) throw new Error('Audio unavailable'); audio = audio || new Audio(); await audio.resume(); }
            catch (_) { el('focus-chime').checked = false; status('Chimes are unavailable in this browser. Your timer will still update.'); }
        });
        el('focus-theme').addEventListener('click', () => localStorage.setItem('darkMode', String(document.documentElement.classList.toggle('dark'))));
        el('focus-logout').addEventListener('click', () => { resetRoom(); endSession(token); });
        const clockInterval = setInterval(paintClock, 250), roomsInterval = setInterval(refreshRooms, 15000);
        window.addEventListener('pagehide', () => { disposed = true; stopStream(); clearInterval(clockInterval); clearInterval(roomsInterval); if (audio) audio.close().catch(() => {}); });
        window.addEventListener('pageshow', event => { if (event.persisted) window.location.reload(); });
        refreshRooms();
    })();
}
