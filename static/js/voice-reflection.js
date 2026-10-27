/* OPay-standard reflection recorder controller.
 *
 * Drives the timetable reflection modal's segmented Write/Speak switch,
 * audio recording with a 3-minute cap, direct upload to
 * /api/v1/voice/upload (attaching the returned audio_url to the
 * reflection payload), and an in-modal preview with scrubber,
 * re-record and delete controls. Plain ES2017+, no build step. */
(function () {
    'use strict';

    const MAX_RECORDING_SECONDS = 180; // 03:00 cap
    const UPLOAD_ENDPOINT = '/api/v1/voice/upload';

    let mediaRecorder = null;
    let activeStream = null;
    let audioChunks = [];
    let timerHandle = null;
    let recordingSeconds = 0;
    let isRecording = false;
    let discardOnStop = false;

    let takeBlobUrl = '';        // local object URL used for preview playback
    let takeAudioUrl = '';       // persisted URL returned by the upload API
    let appendedTranscript = ''; // transcript auto-filled into the summary field

    let modalOpener = null;      // element focused before the modal opened

    function el(id) {
        return document.getElementById(id);
    }

    function formatClock(totalSeconds) {
        const seconds = Math.max(0, Math.floor(totalSeconds || 0));
        const mm = String(Math.floor(seconds / 60)).padStart(2, '0');
        const ss = String(seconds % 60).padStart(2, '0');
        return mm + ':' + ss;
    }

    function formatPlayerTime(totalSeconds) {
        const seconds = Math.max(0, Math.floor(totalSeconds || 0));
        return Math.floor(seconds / 60) + ':' + String(seconds % 60).padStart(2, '0');
    }

    function setStatus(message, tone) {
        const status = el('voiceStatus');
        if (!status) return;
        status.textContent = message || '';
        status.classList.toggle('is-error', tone === 'error');
        status.classList.toggle('is-success', tone === 'success');
    }

    function setTimer() {
        const timer = el('voiceTimer');
        if (timer) timer.innerHTML = formatClock(recordingSeconds) + ' <span>/ 03:00</span>';
    }

    function stopTimer() {
        if (timerHandle) {
            clearInterval(timerHandle);
            timerHandle = null;
        }
    }

    function startTimer() {
        stopTimer();
        recordingSeconds = 0;
        setTimer();
        timerHandle = setInterval(function () {
            recordingSeconds += 1;
            setTimer();
            if (recordingSeconds >= MAX_RECORDING_SECONDS) {
                stopVoiceRecording(); // auto-stop at 03:00
            }
        }, 1000);
    }

    // States: ready | recording | uploading | preview
    function setRecorderState(state) {
        const btn = el('voiceRecordBtn');
        const waveform = el('voiceWaveform');
        const preview = el('voicePreview');
        if (btn) {
            btn.classList.toggle('is-recording', state === 'recording');
            btn.textContent = state === 'recording' ? '⏹' : '🎤';
            btn.setAttribute('aria-label', state === 'recording' ? 'Stop recording' : 'Start recording');
            btn.disabled = state === 'uploading';
        }
        if (waveform) waveform.classList.toggle('is-active', state === 'recording');
        if (preview) preview.hidden = state !== 'preview';
        if (state !== 'recording') stopTimer();
    }

    function updateCounter(fieldId, counterId) {
        const field = el(fieldId);
        const counter = el(counterId);
        if (!field || !counter) return;
        counter.textContent = String(field.value.length);
        const limit = Number(field.maxLength) || 0;
        if (counter.parentElement) {
            counter.parentElement.classList.toggle('near-limit', limit > 0 && field.value.length >= limit * 0.9);
        }
    }

    function updateCounters() {
        updateCounter('reflectionSummary', 'summaryCount');
        updateCounter('reflectionChallenges', 'challengesCount');
        updateCounter('reflectionLearnings', 'learningsCount');
    }

    function releaseStream() {
        if (activeStream) {
            activeStream.getTracks().forEach(function (track) { track.stop(); });
            activeStream = null;
        }
    }

    function clearAppendedTranscript() {
        const summary = el('reflectionSummary');
        if (summary && appendedTranscript) {
            const marker = appendedTranscript;
            if (summary.value.endsWith(marker)) {
                summary.value = summary.value.slice(0, -marker.length).replace(/[ \t]+$/, '');
            } else if (summary.value.indexOf(marker) !== -1) {
                summary.value = summary.value.replace(marker, '').replace(/[ \t]{2,}/g, ' ');
            }
            summary.dispatchEvent(new Event('input', { bubbles: true }));
        }
        appendedTranscript = '';
    }

    // Drop the current take: clear the attached audio_url, the auto-filled
    // transcript and the local preview, without touching typed text.
    function releaseTake() {
        clearAppendedTranscript();
        if (takeBlobUrl) {
            URL.revokeObjectURL(takeBlobUrl);
            takeBlobUrl = '';
        }
        takeAudioUrl = '';
        const urlInput = el('url-summary');
        if (urlInput) urlInput.value = '';
        const audio = el('voicePreviewAudio');
        if (audio) {
            audio.pause();
            audio.removeAttribute('src');
            audio.load();
        }
        const scrub = el('voiceScrubber');
        if (scrub) scrub.value = '0';
        const attached = el('voiceAttachedLabel');
        if (attached) attached.hidden = true;
        updatePlayerTime(0, 0);
        updatePlayButton(false);
    }

    function pickMimeType() {
        if (typeof MediaRecorder === 'undefined' || typeof MediaRecorder.isTypeSupported !== 'function') return '';
        if (MediaRecorder.isTypeSupported('audio/webm;codecs=opus')) return 'audio/webm;codecs=opus';
        if (MediaRecorder.isTypeSupported('audio/webm')) return 'audio/webm';
        if (MediaRecorder.isTypeSupported('audio/mp4')) return 'audio/mp4';
        return '';
    }

    function updatePlayButton(playing) {
        const btn = el('voicePlayBtn');
        if (!btn) return;
        btn.textContent = playing ? '⏸' : '▶';
        btn.setAttribute('aria-label', playing ? 'Pause recording' : 'Play recording');
    }

    function updatePlayerTime(current, duration) {
        const cur = el('voiceCurrentTime');
        const dur = el('voiceDuration');
        if (cur) cur.textContent = formatPlayerTime(current);
        if (dur) dur.textContent = formatPlayerTime(duration);
    }

    function syncScrubber() {
        const audio = el('voicePreviewAudio');
        const scrub = el('voiceScrubber');
        if (!audio || !scrub) return;
        const duration = audio.duration || 0;
        if (duration > 0 && !scrub.matches(':active')) {
            scrub.value = String((audio.currentTime / duration) * 100);
        }
        updatePlayerTime(audio.currentTime, duration);
    }

    function openPreview(audioBlob) {
        if (takeBlobUrl) URL.revokeObjectURL(takeBlobUrl);
        takeBlobUrl = URL.createObjectURL(audioBlob);
        const audio = el('voicePreviewAudio');
        if (audio) {
            audio.src = takeBlobUrl;
            audio.load();
        }
        const scrub = el('voiceScrubber');
        if (scrub) scrub.value = '0';
        updatePlayerTime(0, 0);
        updatePlayButton(false);
    }

    function startVoiceRecording() {
        if (isRecording || mediaRecorder) return Promise.resolve();
        if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
            setStatus('Audio recording is not supported in this browser.', 'error');
            return Promise.resolve();
        }
        releaseTake(); // a new take replaces any previous one
        return navigator.mediaDevices.getUserMedia({ audio: true }).then(function (stream) {
            activeStream = stream;
            const mimeType = pickMimeType();
            try {
                mediaRecorder = mimeType ? new MediaRecorder(stream, { mimeType: mimeType }) : new MediaRecorder(stream);
            } catch (err) {
                console.error('Could not start recorder:', err);
                releaseStream();
                setStatus('Could not start recording on this device.', 'error');
                return;
            }

            audioChunks = [];
            discardOnStop = false;
            mediaRecorder.ondataavailable = function (event) {
                if (event.data && event.data.size > 0) audioChunks.push(event.data);
            };
            mediaRecorder.onstop = handleRecordingStop;

            mediaRecorder.start();
            isRecording = true;
            setRecorderState('recording');
            setStatus('Recording… tap stop when you are done.');
            startTimer();
        }).catch(function (err) {
            console.error('Microphone access denied:', err);
            setStatus('Microphone access denied. Allow the mic to record.', 'error');
        });
    }

    function stopVoiceRecording() {
        if (!isRecording || !mediaRecorder) return;
        isRecording = false;
        stopTimer();
        setTimer();
        try {
            mediaRecorder.stop();
        } catch (err) {
            console.error('Could not stop recorder:', err);
            mediaRecorder = null;
            releaseStream();
            setRecorderState('ready');
        }
    }

    // Discard-in-flight take (mode switch, modal close, reset).
    function stopVoiceRecordingWithDiscard() {
        discardOnStop = true;
        stopVoiceRecording();
    }

    async function handleRecordingStop() {
        const recorder = mediaRecorder;
        mediaRecorder = null;
        releaseStream();

        if (discardOnStop) {
            discardOnStop = false;
            setRecorderState('ready');
            setStatus('');
            return;
        }

        const mimeType = (recorder && recorder.mimeType) || 'audio/webm';
        const audioBlob = new Blob(audioChunks, { type: mimeType });
        audioChunks = [];
        if (!audioBlob.size) {
            setRecorderState('ready');
            setStatus('Nothing was recorded. Try again.', 'error');
            return;
        }

        setRecorderState('uploading');
        setStatus('Uploading your voice note…');

        const ext = mimeType.indexOf('mp4') !== -1 ? 'mp4' : 'webm';
        const formData = new FormData();
        formData.append('audio', audioBlob, 'voice-reflection.' + ext);

        let uploadedUrl = '';
        try {
            const headers = {};
            const token = (typeof sessionToken === 'function' ? sessionToken() : '') || localStorage.getItem('token') || '';
            if (token) headers['Authorization'] = 'Bearer ' + token;
            const response = await fetch(UPLOAD_ENDPOINT, { method: 'POST', headers: headers, body: formData });
            if (!response.ok) {
                const errData = await response.json().catch(function () { return {}; });
                throw new Error(errData.error || ('Upload failed with status ' + response.status));
            }
            const data = await response.json();
            uploadedUrl = data.audio_url || '';
            if (data.text) {
                const summary = el('reflectionSummary');
                if (summary) {
                    appendedTranscript = data.text;
                    const current = summary.value.trim();
                    summary.value = current ? current + ' ' + data.text : data.text;
                    summary.dispatchEvent(new Event('input', { bubbles: true }));
                }
            }
        } catch (err) {
            console.error('Voice upload failed:', err);
            setRecorderState('preview');
            openPreview(audioBlob);
            setStatus('Upload failed — listen back, re-record or delete.', 'error');
            return;
        }

        takeAudioUrl = uploadedUrl;
        const urlInput = el('url-summary');
        if (urlInput) urlInput.value = uploadedUrl;
        const attached = el('voiceAttachedLabel');
        if (attached) attached.hidden = !uploadedUrl;
        setRecorderState('preview');
        openPreview(audioBlob);
        setStatus(uploadedUrl ? 'Voice note attached to your reflection.' : 'Upload finished.', 'success');
    }

    function toggleVoicePlayback() {
        const audio = el('voicePreviewAudio');
        if (!audio || !audio.src) return;
        if (audio.paused) {
            audio.play().catch(function (err) { console.error('Playback failed:', err); });
        } else {
            audio.pause();
        }
    }

    function rerecordVoiceTake() {
        releaseTake();
        startVoiceRecording();
    }

    function deleteVoiceTake() {
        releaseTake();
        setRecorderState('ready');
        recordingSeconds = 0;
        setTimer();
        setStatus('Take deleted. Record a new one whenever you are ready.');
    }

    function setReflectionMode(mode) {
        const write = mode !== 'voice';
        const writeTab = el('modeWriteTab');
        const speakTab = el('modeSpeakTab');
        const writePanel = el('reflectionWritePanel');
        const voicePanel = el('reflectionVoicePanel');
        if (writeTab) writeTab.setAttribute('aria-selected', String(write));
        if (speakTab) speakTab.setAttribute('aria-selected', String(!write));
        if (writePanel) writePanel.hidden = !write;
        if (voicePanel) voicePanel.hidden = write;
        if (write) {
            const audio = el('voicePreviewAudio');
            if (audio) audio.pause();
            if (isRecording) {
                stopVoiceRecordingWithDiscard();
                setStatus('');
            }
        }
    }

    function resetReflectionModal() {
        stopVoiceRecordingWithDiscard();
        releaseTake();
        ['reflectionSummary', 'reflectionChallenges', 'reflectionLearnings'].forEach(function (id) {
            const field = el(id);
            if (field) field.value = '';
        });
        updateCounters();
        const urlInput = el('url-summary');
        if (urlInput) urlInput.value = '';
        const error = el('reflectionFormError');
        if (error) {
            error.hidden = true;
            error.textContent = '';
        }
        recordingSeconds = 0;
        setTimer();
        setStatus('');
        setRecorderState('ready');
        setReflectionMode('write');
    }

    function onReflectionModalOpened() {
        modalOpener = document.activeElement;
        const firstField = el('reflectionSummary');
        if (firstField) firstField.focus();
    }

    function onReflectionModalClosed() {
        if (modalOpener && typeof modalOpener.focus === 'function' && modalOpener.isConnected) {
            modalOpener.focus();
        }
        modalOpener = null;
    }

    function modalFocusables() {
        const modal = document.querySelector('#reflectionModal .reflection-modal');
        if (!modal) return [];
        return Array.prototype.slice.call(
            modal.querySelectorAll('button, [href], input:not([type="hidden"]), textarea, select, [tabindex]:not([tabindex="-1"])')
        ).filter(function (node) {
            return !node.disabled && node.offsetParent !== null;
        });
    }

    function closeModalSafely() {
        if (typeof window.closeReflectionModal === 'function') {
            window.closeReflectionModal();
        }
    }

    function wireModal() {
        const modal = el('reflectionModal');
        if (!modal) return;
        modal.addEventListener('click', function (event) {
            if (event.target === modal) closeModalSafely(); // backdrop tap
        });
        modal.addEventListener('keydown', function (event) {
            if (event.key === 'Escape') {
                event.preventDefault();
                closeModalSafely();
                return;
            }
            if (event.key !== 'Tab') return;
            const focusables = modalFocusables();
            if (!focusables.length) return;
            const first = focusables[0];
            const last = focusables[focusables.length - 1];
            if (event.shiftKey && document.activeElement === first) {
                event.preventDefault();
                last.focus();
            } else if (!event.shiftKey && document.activeElement === last) {
                event.preventDefault();
                first.focus();
            }
        });
    }

    function wireFields() {
        [['reflectionSummary', 'summaryCount'], ['reflectionChallenges', 'challengesCount'], ['reflectionLearnings', 'learningsCount']]
            .forEach(function (pair) {
                const field = el(pair[0]);
                if (field) {
                    field.addEventListener('input', function () { updateCounter(pair[0], pair[1]); });
                }
            });
        updateCounters();
    }

    function wirePreview() {
        const audio = el('voicePreviewAudio');
        const scrub = el('voiceScrubber');
        if (audio) {
            audio.addEventListener('timeupdate', syncScrubber);
            audio.addEventListener('loadedmetadata', syncScrubber);
            audio.addEventListener('play', function () { updatePlayButton(true); });
            audio.addEventListener('pause', function () { updatePlayButton(false); });
            audio.addEventListener('ended', function () {
                updatePlayButton(false);
                syncScrubber();
            });
        }
        if (scrub && audio) {
            scrub.addEventListener('input', function () {
                if (!audio.duration) return;
                audio.currentTime = (Number(scrub.value) / 100) * audio.duration;
                updatePlayerTime(audio.currentTime, audio.duration);
            });
        }
    }

    document.addEventListener('DOMContentLoaded', function () {
        wireModal();
        wireFields();
        wirePreview();
        setTimer();
    });

    window.setReflectionMode = setReflectionMode;
    window.toggleVoiceRecording = toggleVoiceRecordingWrapper;
    window.toggleVoicePlayback = toggleVoicePlayback;
    window.rerecordVoiceTake = rerecordVoiceTake;
    window.deleteVoiceTake = deleteVoiceTake;
    window.resetReflectionModal = resetReflectionModal;
    window.abortVoiceRecording = stopVoiceRecordingWithDiscard;
    window.onReflectionModalOpened = onReflectionModalOpened;
    window.onReflectionModalClosed = onReflectionModalClosed;

    function toggleVoiceRecordingWrapper() {
        if (isRecording) {
            stopVoiceRecording();
        } else {
            startVoiceRecording();
        }
    }
})();
