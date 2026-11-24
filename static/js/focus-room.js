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
        const token = sessionToken(), initialScope = sessionScope();
        const checkAccount = () => { if (sessionScope() !== initialScope) { stopStream(); window.location.reload(); throw Error("Your account changed. Reload this page."); } };
        let room = null, anchor = null, phase = null, stream = null, generation = 0, busy = false, disposed = false;
        let audio = null, phaseKey = null, loading = false, myID = 0;
        const auth = () => ({ Authorization: 'Bearer ' + token });
        const status = (text, error = false) => {
            const statusEl = el('focus-status');
            if (!statusEl) return;
            statusEl.textContent = text;
            statusEl.classList.toggle('error', error);
        };
        function setBusy(value) {
            busy = value;
            for (const id of ['focus-create-button', 'focus-refresh', 'focus-leave', 'focus-save-target', 'focus-reflect', 'focus-goal', 'focus-room-title', 'focus-active-goal', 'startBtn', 'pauseBtn', 'resetBtn']) {
                const targetEl = el(id);
                if (targetEl) targetEl.disabled = value;
            }
            const roomsEl = el('focus-rooms');
            if (roomsEl) {
                for (const button of roomsEl.querySelectorAll('button')) button.disabled = value;
            }
        }
        async function api(path, method = 'GET', body) {
            checkAccount();
            const options = { method, credentials: 'same-origin', headers: auth(), cache: 'no-store' };
            if (body !== undefined) { options.headers['Content-Type'] = 'application/json'; options.body = JSON.stringify(body); }
            const response = await fetch('/api/v1/focus/' + path, options); checkAccount();
            if (response.status === 204) return null;
            let value; try { value = await response.json(); } catch (_) { throw new Error('We could not load the room. Please try again.'); }
            checkAccount();
            if (!response.ok) { const error = new Error(value.error || 'We could not load the room. Please try again.'); error.status = response.status; throw error; }
            return value;
        }
        function stopStream() { generation++; if (stream) stream.abort(); stream = null; }
        function resetRoom() {
            stopStream(); room = null; anchor = null; phaseKey = null;
            el('focus-session').hidden = true; el('focus-roster-panel').hidden = true;
            el('focus-lobby').hidden = false; el('focus-discovery').hidden = false;
            el('focus-roster').replaceChildren();
            if (el('attendeesList')) el('attendeesList').replaceChildren();
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
            el('focus-progress').max = phase === 'break' ? 300 : 1500;
            paintClock();
        }
        function paintClock() {
            if (!anchor) return;
            const seconds = focusSeconds(anchor, performance.now());
            const timeStr = String(Math.floor(seconds / 60)).padStart(2, '0') + ':' + String(seconds % 60).padStart(2, '0');
            el('focus-time').textContent = timeStr;
            if (el('timerDisplay')) el('timerDisplay').textContent = timeStr;
            el('focus-progress').value = Math.min(el('focus-progress').max, seconds);
            if (!seconds) el('focus-connection').textContent = 'Waiting for the next room update…';
        }
        function renderRoom(value) {
            room = value;
            el('focus-active-title').textContent = value.title;
            el('focus-count').textContent = '(' + value.participants.length + ')';
            if (el('attendeeCount')) el('attendeeCount').textContent = String(value.participants.length);
            const list = el('focus-roster'); list.replaceChildren();
            const altList = el('attendeesList'); if (altList) altList.replaceChildren();
            for (const person of value.participants) {
                const item = document.createElement('li'); item.className = 'focus-person';
                const name = document.createElement('strong'), target = document.createElement('p');
                name.textContent = person.display_name + (person.user_id === myID ? ' (you)' : '');
                target.textContent = person.target;
                item.append(name, target);
                list.append(item);
                if (altList) {
                    const altItem = document.createElement('li');
                    altItem.textContent = person.display_name + ': ' + person.target;
                    altList.append(altItem);
                }
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
                    checkAccount(); if ([401, 403, 404].includes(response.status)) { resetRoom(); status('Your room membership ended. Join a room to continue.', true); refreshRooms(); return; }
                    if (response.status === 429) { const data = await response.json().catch(() => ({})); status(data.error || 'Live rooms are busy. Close another focus tab or try again shortly.', true); }
                    if (!response.ok || !response.body || !response.headers.get('content-type')?.includes('text/event-stream')) throw new Error('Room connection interrupted');
                    el('focus-connection').textContent = 'Live · Your timer is synced with the room';
                    status('You are in. Protect this one small intention.');
                    phaseKey = null; backoff = 1500;
                    reader = response.body.getReader(); const decoder = new TextDecoder(); let buffer = '';
                    while (!controller.signal.aborted && current === generation) {
                        const { value, done } = await reader.read(); if (done) break;
                        buffer = focusReadEvents(buffer + decoder.decode(value, { stream: true }), (event, data) => {
                            checkAccount(); if (current !== generation) return;
                            if (event === 'presence') renderRoom(data);
                            if (event === 'clock') renderClock(data);
                            if (event === 'room_closed' || event === 'membership_expired') { resetRoom(); status('Your room membership ended. Join a room to continue.'); refreshRooms(); }
                        });
                    }
                } catch (error) { if (controller.signal.aborted || current !== generation) return; }
                finally { if (reader) { try { await reader.cancel(); } catch (_) { } } }
                if (controller.signal.aborted || current !== generation) return;
                el('focus-connection').textContent = 'Reconnecting… Your displayed timer may be out of date.';
                await delay(backoff, controller.signal); backoff = Math.min(10000, backoff * 2);
            }
        }
        async function enter(id) {
            if (busy) return;
            const target = el('focus-goal').value.trim();
            if (!target) {
                status('Set a study target before joining.', true);
                el('focus-goal').focus();
                return;
            }
            if (!id) {
                const titleInput = el('focus-room-title');
                const roomTitle = titleInput ? titleInput.value.trim() : '';
                if (!roomTitle) {
                    status('Enter a room title to start a fresh room.', true);
                    titleInput?.focus();
                    return;
                }
            }
            setBusy(true);
            try {
                const value = id ? await api('rooms/' + id + '/join', 'POST', { target }) : await api('rooms', 'POST', { title: el('focus-room-title').value.trim(), target });
                renderRoom(value);
                el('focus-active-goal').value = target;
                if (el('timerGoal')) el('timerGoal').textContent = target;
                el('focus-session').hidden = false;
                el('focus-roster-panel').hidden = false;
                el('focus-lobby').hidden = true;
                el('focus-discovery').hidden = true;
                el('focus-finished').hidden = true;
                el('focus-active-title').focus();
                status('You are in. Protect this one small intention.');
                followRoom(value.id);
            } catch (error) {
                status(error.message, true);
            } finally {
                setBusy(false);
            }
        }
        async function refreshRooms() {
            if (disposed || room || loading) return;
            loading = true;
            el('focus-rooms').setAttribute('aria-busy', 'true');
            try {
                const data = await api('rooms');
                if (disposed || room) return;
                const list = el('focus-rooms');
                list.replaceChildren();
                const total = (data.rooms || []).reduce((sum, r) => sum + (r.participant_count || 0), 0);
                const summary = el('activeRoomsSummary');
                if (summary) {
                    if (!data.rooms.length) {
                        summary.textContent = '0 active rooms';
                    } else {
                        const roomLabel = data.rooms.length === 1 ? '1 active room' : data.rooms.length + ' active rooms';
                        const peerLabel = total === 1 ? '1 learner' : total + ' learners';
                        summary.textContent = `${roomLabel} · ${peerLabel}`;
                    }
                }
                if (!data.rooms.length) {
                    const empty = document.createElement('li');
                    empty.textContent = 'A quiet start. Create the first room.';
                    empty.className = 'focus-muted';
                    list.append(empty);
                }
                for (const item of data.rooms) {
                    const row = document.createElement('li');
                    row.className = 'focus-room';
                    const text = document.createElement('div');
                    const title = document.createElement('strong');
                    const detail = document.createElement('p');
                    const button = document.createElement('button');
                    title.textContent = item.title;
                    const phaseText = item.timer.phase === 'break' ? 'On a break' : 'Focusing';
                    const countText = item.participant_count === 1 ? '1 here' : item.participant_count + ' here';
                    detail.textContent = countText + ' · ' + phaseText;
                    detail.className = 'focus-count';
                    text.append(title, detail);
                    button.type = 'button';
                    button.textContent = data.current_room_id === item.id ? 'Return to room' : 'Join room';
                    button.disabled = busy;
                    button.addEventListener('click', () => enter(item.id));
                    row.append(text, button);
                    list.append(row);
                }
                status('Choose your target, then join a room or create one.');
            } catch (error) {
                const list = el('focus-rooms');
                list.replaceChildren();
                const item = document.createElement('li');
                item.textContent = 'Rooms could not be loaded. Your target is kept; use Refresh rooms to retry.';
                list.append(item);
                status(error.message, true);
            } finally {
                loading = false;
                el('focus-rooms').setAttribute('aria-busy', 'false');
            }
        }
        el('focus-create').addEventListener('submit', event => { event.preventDefault(); enter(); });
        el('focus-refresh').addEventListener('click', refreshRooms);
        const leaveDialog = el('focus-leave-dialog'); let leaveOpener = null, wantsReflection = false;
        function askLeave(reflect) { if (busy || !room) return; wantsReflection = reflect; leaveOpener = document.activeElement; el('focus-leave-title').textContent = reflect ? 'Leave and choose a reflection block?' : 'Leave this focus room?'; el('focus-leave-error').textContent = ''; leaveDialog.showModal(); el('focus-leave-cancel').focus(); }
        leaveDialog.addEventListener('cancel', e => { if (busy) e.preventDefault(); });
        leaveDialog.addEventListener('close', () => { if (leaveOpener?.isConnected && !leaveOpener.closest('[hidden]')) leaveOpener.focus(); else el('focus-status').focus(); });
        el('focus-leave').addEventListener('click', () => askLeave(false));
        el('focus-reflect').addEventListener('click', () => askLeave(true));
        el('focus-leave-cancel').addEventListener('click', () => { if (!busy) leaveDialog.close(); });
        el('focus-leave-confirm').addEventListener('click', async () => {
            if (busy || !room) return; setBusy(true); el('focus-leave-cancel').disabled = el('focus-leave-confirm').disabled = true;
            try {
                await api('rooms/' + room.id + '/membership', 'DELETE'); resetRoom(); leaveDialog.close(); await refreshRooms(); status('You left the room. Nothing was recorded as study time.'); el('focus-finished').hidden = false; if (wantsReflection) el('focus-reflection-link').focus();
            }
            catch (error) { if (error.status === 404) { resetRoom(); leaveDialog.close(); status('Your membership has already ended. You can choose a block to reflect.'); el('focus-finished').hidden = false; refreshRooms(); } else el('focus-leave-error').textContent = error.message + ' Leaving could not be confirmed. Retry, or refresh to check your membership.'; }
            finally { setBusy(false); el('focus-leave-cancel').disabled = el('focus-leave-confirm').disabled = false; }
        });
        el('focus-target-form').addEventListener('submit', async event => {
            event.preventDefault(); if (busy || !room) return;
            const newTarget = el('focus-active-goal').value.trim();
            if (!newTarget) return;
            setBusy(true);
            try {
                const updated = await api('rooms/' + room.id + '/target', 'PUT', { target: newTarget });
                renderRoom(updated);
                if (el('timerGoal')) el('timerGoal').textContent = newTarget;
                status('Your intention is updated.');
            }
            catch (error) { status(error.message, true); }
            finally { setBusy(false); }
        });
        el('focus-chime').addEventListener('change', async () => {
            if (!el('focus-chime').checked) return;
            try { const Audio = window.AudioContext || window.webkitAudioContext; if (!Audio) throw new Error('Audio unavailable'); audio = audio || new Audio(); await audio.resume(); }
            catch (_) { el('focus-chime').checked = false; status('Chimes are unavailable in this browser. Your timer will still update.'); }
        });
        if (el('startBtn')) {
            el('startBtn').addEventListener('click', async () => {
                if (audio && audio.state === 'suspended') await audio.resume().catch(() => {});
                paintClock();
                status('Timer active and synchronized with room.');
            });
        }
        if (el('pauseBtn')) {
            el('pauseBtn').addEventListener('click', () => {
                if (el('focus-chime')) el('focus-chime').checked = false;
                status('Chime paused. Room timer continues silently.');
            });
        }
        if (el('resetBtn')) {
            el('resetBtn').addEventListener('click', () => {
                paintClock();
                status('Timer display resynced with room clock.');
            });
        }
        el('focus-theme').addEventListener('click', () => localStorage.setItem('darkMode', String(document.documentElement.classList.toggle('dark'))));
        el('focus-logout').addEventListener('click', () => { resetRoom(); endSession(token); });
        const clockInterval = setInterval(paintClock, 250), roomsInterval = setInterval(refreshRooms, 15000);
        window.addEventListener('pagehide', () => { disposed = true; stopStream(); clearInterval(clockInterval); clearInterval(roomsInterval); if (audio) audio.close().catch(() => { }); });
        window.addEventListener('pageshow', event => { if (event.persisted) window.location.reload(); });
        fetch('/api/v1/me', { headers: auth(), cache: 'no-store' }).then(r => r.ok ? r.json() : null).then(me => { checkAccount(); myID = me?.id || 0; if (me?.is_admin) el('adminNavLink')?.classList.remove('hidden'); }).catch(() => { });
        refreshRooms();
    })();
}
