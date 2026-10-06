let mediaRecorder;
let audioChunks = [];

async function toggleRecording(targetId, suffix) {
    const btn = document.getElementById(`voice-btn-${suffix}`) || document.getElementById('voice-btn');
    const status = document.getElementById(`status-${suffix}`) || document.getElementById('recording-status');
    const contentArea = document.getElementById(targetId);
    const urlInput = document.getElementById(`url-${suffix}`) || document.getElementById('audio-url');

    if (!mediaRecorder || mediaRecorder.state === 'inactive') {
        try {
            const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
            mediaRecorder = new MediaRecorder(stream);
            audioChunks = [];

            mediaRecorder.ondataavailable = (event) => audioChunks.push(event.data);

            mediaRecorder.onstop = async () => {
                const audioBlob = new Blob(audioChunks, { type: 'audio/webm' });
                status.innerText = "Transcribing...";
                
                const formData = new FormData();
                formData.append('audio', audioBlob);

                try {
                    const response = await fetch('/api/voice/upload', {
                        method: 'POST',
                        body: formData
                    });
                    const data = await response.json();
                    
                    // SMART PART: Append to existing text or replace
                    contentArea.value = (contentArea.value ? contentArea.value + " " : "") + data.text;
                    if(urlInput) urlInput.value = data.audio_url;
                    
                    status.innerText = "Done!";
                } catch (err) {
                    status.innerText = "Error.";
                }
            };

            mediaRecorder.start();
            btn.innerHTML = "🛑 Stop";
            status.innerText = "Recording...";
        } catch (err) {
            alert("Microphone access denied.");
        }
    } else {
        mediaRecorder.stop();
        btn.innerHTML = "🎤 Voice";
    }
}
