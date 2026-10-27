let mediaRecorder = null;
let audioChunks = [];
let activeStream = null;

async function toggleRecording(targetId = 'reflectionSummary', suffix = 'summary') {
    const btn = document.getElementById(`voice-btn-${suffix}`) || document.getElementById('voice-btn');
    const status = document.getElementById(`status-${suffix}`) || document.getElementById('recording-status');
    const contentArea = document.getElementById(targetId) || document.getElementById('reflection-content');
    const urlInput = document.getElementById(`url-${suffix}`) || document.getElementById('audio-url');

    if (!mediaRecorder || mediaRecorder.state === 'inactive') {
        // Start Recording
        try {
            if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
                alert("Audio recording is not supported on this browser.");
                return;
            }

            const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
            activeStream = stream;

            let options = {};
            if (typeof MediaRecorder.isTypeSupported === 'function') {
                if (MediaRecorder.isTypeSupported('audio/webm;codecs=opus')) {
                    options = { mimeType: 'audio/webm;codecs=opus' };
                } else if (MediaRecorder.isTypeSupported('audio/webm')) {
                    options = { mimeType: 'audio/webm' };
                } else if (MediaRecorder.isTypeSupported('audio/mp4')) {
                    options = { mimeType: 'audio/mp4' };
                }
            }

            try {
                mediaRecorder = new MediaRecorder(stream, options);
            } catch (e) {
                mediaRecorder = new MediaRecorder(stream);
            }

            audioChunks = [];
            mediaRecorder.ondataavailable = (event) => {
                if (event.data && event.data.size > 0) {
                    audioChunks.push(event.data);
                }
            };

            mediaRecorder.onstop = async () => {
                if (activeStream) {
                    activeStream.getTracks().forEach(track => track.stop());
                    activeStream = null;
                }

                const mimeType = mediaRecorder.mimeType || 'audio/webm';
                const audioBlob = new Blob(audioChunks, { type: mimeType });
                if (status) status.innerText = "Transcribing...";

                const ext = mimeType.includes('mp4') ? 'mp4' : 'webm';
                const formData = new FormData();
                formData.append('audio', audioBlob, `voice-reflection.${ext}`);

                try {
                    const headers = {};
                    const token = (typeof sessionToken === 'function' ? sessionToken() : '') || localStorage.getItem('token') || '';
                    if (token) {
                        headers['Authorization'] = 'Bearer ' + token;
                    }

                    const response = await fetch('/api/v1/voice/upload', {
                        method: 'POST',
                        headers: headers,
                        body: formData
                    });

                    if (!response.ok) {
                        const errData = await response.json().catch(() => ({}));
                        throw new Error(errData.error || `Upload failed with status ${response.status}`);
                    }

                    const data = await response.json();

                    if (contentArea && data.text) {
                        const current = contentArea.value.trim();
                        contentArea.value = current ? current + " " + data.text : data.text;
                        contentArea.dispatchEvent(new Event('input', { bubbles: true }));
                    }

                    if (urlInput && data.audio_url) {
                        urlInput.value = data.audio_url;
                    }

                    if (status) status.innerText = "Voice processed!";
                } catch (err) {
                    console.error("Voice upload failed:", err);
                    if (status) status.innerText = "Upload failed.";
                }
            };

            mediaRecorder.start();
            if (btn) {
                btn.innerHTML = "🛑 Stop";
                btn.classList.add('recording-active');
            }
            if (status) status.innerText = "Recording...";
        } catch (err) {
            console.error("Microphone access denied:", err);
            alert("Please allow microphone access to use voice reflection.");
            if (status) status.innerText = "";
        }
    } else {
        // Stop Recording
        mediaRecorder.stop();
        if (btn) {
            btn.innerHTML = "🎤 Voice note";
            btn.classList.remove('recording-active');
        }
    }
}
