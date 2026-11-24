/* Personal history: text-only DOM rendering and same-origin, authorized audio. */
(function () {
    'use strict';
    const list = document.getElementById('reflectionsList');
    let rows = [], failed = false;
    const filter = document.getElementById('reflectionFilters');

    function node(tag, text, className) {
        const item = document.createElement(tag);
        if (text !== undefined) item.textContent = text;
        if (className) item.className = className;
        return item;
    }

    function clock(s) {
        if (!Number.isFinite(s) || s < 0) s = 0;
        return `${String(Math.floor(s / 60)).padStart(2, '0')}:${String(Math.floor(s % 60)).padStart(2, '0')}`;
    }

    function audioPath(value) {
        return typeof value === 'string' && /^\/uploads\/(?:[1-9][0-9]*-)?[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}\.(webm|ogg|mp4|wav|m4a|mp3)$/.test(value) ? value : null;
    }

    function prettyDate(value) {
        if (!/^\d{4}-\d{2}-\d{2}$/.test(value || '')) return 'Date unavailable';
        const date = new Date(value + 'T12:00:00');
        return Number.isNaN(date.getTime()) ? 'Date unavailable' : date.toLocaleDateString(undefined, { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' });
    }

    function detail(parent, label, value) {
        if (value) {
            const p = node('p');
            p.append(node('strong', label + ' '), document.createTextNode(value));
            parent.append(p);
        }
    }

    function syncChips(val) {
        const isAud = val === 'voice' || val === 'audio';
        document.querySelectorAll('[data-reflection-format]').forEach(btn => {
            const fmt = btn.dataset.reflectionFormat;
            const pressed = (fmt === 'all' && val === 'all') ||
                            ((fmt === 'voice' || fmt === 'audio') && isAud) ||
                            (fmt === 'text' && val === 'text');
            btn.setAttribute('aria-pressed', String(pressed));
        });
    }

    function render() {
        list.querySelectorAll('audio').forEach(audio => audio.pause());
        list.replaceChildren();

        const search = (document.getElementById('reflectionSearch')?.value || '').trim().toLocaleLowerCase();
        const date = document.getElementById('reflectionDateFilter')?.value || '';
        const media = document.getElementById('reflectionMediaFilter')?.value || 'all';
        const isAudio = media === 'voice' || media === 'audio';

        const visible = rows.filter(row => {
            if (date && row.log_date !== date) return false;
            if (search) {
                const haystack = [row.activity_name || row.activity, row.summary, row.challenges, row.learnings].join(' ').toLocaleLowerCase();
                if (!haystack.includes(search)) return false;
            }
            if (media === 'all') return true;
            if (isAudio) return !!row.audio_url;
            return !row.audio_url;
        });

        const historyCount = document.getElementById('historyCount');
        if (historyCount) {
            historyCount.textContent = failed ? 'History could not be loaded.' : `${visible.length} of ${rows.length} reflection${rows.length === 1 ? '' : 's'} shown`;
        }

        if (failed) {
            const errorBox = node('section', undefined, 'workspace-empty l2e-feedback');
            errorBox.id = 'historyErrorState';
            errorBox.setAttribute('role', 'alert');
            errorBox.append(
                node('h2', 'Reflection history unavailable'),
                node('p', 'History is unavailable. Your filters are kept.')
            );
            const retry = node('button', 'Try again', 'workspace-action');
            retry.type = 'button';
            retry.addEventListener('click', () => load(true));
            errorBox.append(retry);
            list.append(errorBox);
            return;
        }

        if (!rows.length) {
            const empty = node('section', undefined, 'workspace-empty');
            empty.id = 'emptyHistoryState';
            empty.append(
                node('h2', 'Begin your reflection habit'),
                node('p', 'Plan a study block on your timetable to start capturing your learning reflections.')
            );
            const link = node('a', 'Plan a study block →', 'workspace-action');
            link.href = '/timetable';
            empty.append(link);
            list.append(empty);
            return;
        }

        if (!visible.length) {
            const noMatch = node('section', undefined, 'workspace-empty');
            noMatch.id = 'noMatchState';
            noMatch.append(
                node('h2', 'No matching reflections'),
                node('p', 'No reflections match your current search and filter criteria. Adjust your filters or clear them to view your history.')
            );
            const clearBtn = node('button', 'Clear filters', 'workspace-action');
            clearBtn.type = 'button';
            clearBtn.addEventListener('click', () => {
                const clearFilterBtn = document.getElementById('clearReflectionFilter');
                if (clearFilterBtn) {
                    clearFilterBtn.click();
                } else if (filter) {
                    filter.reset();
                } else {
                    const searchEl = document.getElementById('reflectionSearch');
                    const dateEl = document.getElementById('reflectionDateFilter');
                    const mediaEl = document.getElementById('reflectionMediaFilter');
                    if (searchEl) searchEl.value = '';
                    if (dateEl) dateEl.value = '';
                    if (mediaEl) mediaEl.value = 'all';
                    syncChips('all');
                    render();
                }
            });
            noMatch.append(clearBtn);
            list.append(noMatch);
            return;
        }

        visible.sort((a, b) => {
            const order = document.getElementById('reflectionOrder')?.value === 'oldest' ? 1 : -1;
            return order * (String(a.log_date).localeCompare(String(b.log_date)) || Number(a.id) - Number(b.id));
        });

        visible.forEach((row, index) => {
            const card = node('article', undefined, 'history-card');
            const activity = row.activity_name || row.activity || 'Previous activity';
            card.append(node('h2', activity));

            const dateLine = node('p', prettyDate(row.log_date), 'history-date');
            const time = node('time', row.log_date);
            time.dateTime = row.log_date;
            dateLine.append(' · ', time);
            card.append(dateLine);

            if (row.timetable_id == null) {
                card.append(node('p', 'Time block removed · reflection kept in your history', 'history-detached'));
            } else if (row.start_time && row.end_time) {
                card.append(node('p', `${String(row.start_time).slice(0, 5)}–${String(row.end_time).slice(0, 5)} · current recurring timetable time`, 'l2e-muted'));
            }

            detail(card, 'What I did:', row.summary || 'Not specified');
            detail(card, 'Challenges:', row.challenges);
            detail(card, 'Learned:', row.learnings);

            if (row.audio_url) {
                const path = audioPath(row.audio_url);
                const title = node('h3', 'Voice note');
                title.id = 'history-audio-' + index;
                card.append(title);

                if (path) {
                    const preview = node('div', undefined, 'reflection-voice-preview');

                    const audio = node('audio');
                    audio.preload = 'none';
                    audio.src = path;
                    audio.setAttribute('aria-labelledby', title.id);

                    const scrubberRow = node('div', undefined, 'reflection-scrubber');
                    const playBtn = node('button', '▶', 'reflection-play-btn');
                    playBtn.type = 'button';
                    playBtn.setAttribute('aria-label', 'Play recording');

                    const scrubber = node('input', undefined, 'reflection-scrubber-range');
                    scrubber.type = 'range';
                    scrubber.min = '0';
                    scrubber.max = '100';
                    scrubber.step = '0.1';
                    scrubber.value = '0';
                    scrubber.setAttribute('aria-label', 'Audio scrubber');

                    const timeDisplay = node('span', undefined, 'reflection-scrub-time');
                    const curTime = node('span', '00:00');
                    const sep = document.createTextNode(' / ');
                    const durTime = node('span', '00:00');
                    timeDisplay.append(curTime, sep, durTime);

                    scrubberRow.append(playBtn, scrubber, timeDisplay);

                    const feedback = node('p', 'Private recording · only your signed-in account can play it.', 'l2e-muted');
                    feedback.setAttribute('role', 'status');

                    const retry = node('button', 'Retry audio', 'reflection-voice-action');
                    retry.type = 'button';
                    retry.hidden = true;

                    function syncPlayer() {
                        const cur = audio.currentTime || 0;
                        const dur = Number.isFinite(audio.duration) && audio.duration > 0 ? audio.duration : 0;
                        curTime.textContent = clock(cur);
                        durTime.textContent = clock(dur);
                        if (dur > 0 && !scrubber.matches(':active')) {
                            scrubber.value = String(Math.min(100, (cur / dur) * 100));
                        }
                    }

                    playBtn.addEventListener('click', () => {
                        if (audio.paused) {
                            list.querySelectorAll('audio').forEach(other => { if (other !== audio) other.pause(); });
                            audio.play().catch(() => {});
                        } else {
                            audio.pause();
                        }
                    });

                    scrubber.addEventListener('input', () => {
                        const dur = Number.isFinite(audio.duration) && audio.duration > 0 ? audio.duration : 0;
                        if (dur > 0) {
                            audio.currentTime = (Number(scrubber.value) / 100) * dur;
                            curTime.textContent = clock(audio.currentTime);
                        }
                    });

                    audio.addEventListener('play', () => {
                        list.querySelectorAll('audio').forEach(other => { if (other !== audio) other.pause(); });
                        playBtn.textContent = '⏸';
                        playBtn.setAttribute('aria-label', 'Pause recording');
                    });

                    audio.addEventListener('pause', () => {
                        playBtn.textContent = '▶';
                        playBtn.setAttribute('aria-label', 'Play recording');
                    });

                    audio.addEventListener('ended', () => {
                        playBtn.textContent = '▶';
                        playBtn.setAttribute('aria-label', 'Play recording');
                        audio.currentTime = 0;
                        syncPlayer();
                    });

                    audio.addEventListener('loadedmetadata', syncPlayer);
                    audio.addEventListener('timeupdate', syncPlayer);
                    audio.addEventListener('durationchange', syncPlayer);

                    audio.addEventListener('canplay', () => {
                        feedback.textContent = 'Recording ready to play.';
                        retry.hidden = true;
                        playBtn.disabled = false;
                        scrubber.disabled = false;
                        syncPlayer();
                    });

                    audio.addEventListener('error', () => {
                        feedback.textContent = 'Recording unavailable. Your session may have expired, or the file may no longer exist. Retry or sign in again.';
                        retry.hidden = false;
                        playBtn.disabled = true;
                        scrubber.disabled = true;
                    });

                    retry.addEventListener('click', () => {
                        feedback.textContent = 'Reloading recording…';
                        retry.hidden = true;
                        audio.load();
                        playBtn.disabled = false;
                        scrubber.disabled = false;
                    });

                    preview.append(audio, scrubberRow, feedback, retry);
                    card.append(preview);
                } else {
                    card.append(node('p', 'This older recording link cannot be played safely. Your written reflection is still available.', 'l2e-muted'));
                }
            }
            list.append(card);
        });
    }

    async function load(refresh = false) {
        list.setAttribute('aria-busy', 'true');
        const authToken = typeof token !== 'undefined' ? token : (typeof sessionToken === 'function' ? sessionToken() : '');
        try {
            if (refresh && typeof invalidateReadCache === 'function') {
                await invalidateReadCache(['/api/v1/reflections'], authToken);
            }
            rows = await readAPI('/api/v1/reflections', authToken);
            failed = false;
            render();
        } catch (error) {
            if (error.message === 'Session expired') return;
            failed = true;
            render();
        } finally {
            list.setAttribute('aria-busy', 'false');
        }
    }

    if (filter) {
        filter.addEventListener('submit', event => event.preventDefault());
        filter.addEventListener('reset', event => {
            event.preventDefault();
            const searchInput = document.getElementById('reflectionSearch');
            const dateInput = document.getElementById('reflectionDateFilter');
            const mediaInput = document.getElementById('reflectionMediaFilter');
            if (searchInput) searchInput.value = '';
            if (dateInput) dateInput.value = '';
            if (mediaInput) mediaInput.value = 'all';
            syncChips('all');
            render();
        });
    }

    for (const id of ['reflectionSearch', 'reflectionDateFilter', 'reflectionMediaFilter']) {
        const el = document.getElementById(id);
        if (el) {
            el.addEventListener(id === 'reflectionSearch' ? 'input' : 'change', () => {
                if (id === 'reflectionMediaFilter') syncChips(el.value);
                render();
            });
        }
    }

    document.getElementById('reflectionOrder')?.addEventListener('change', render);

    document.querySelectorAll('[data-reflection-format]').forEach(btn => {
        btn.addEventListener('click', () => {
            const mediaFilter = document.getElementById('reflectionMediaFilter');
            if (!mediaFilter) return;
            const format = btn.dataset.reflectionFormat;
            mediaFilter.value = format;
            syncChips(format);
            render();
        });
    });

    load();
})();
