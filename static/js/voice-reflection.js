let mediaRecorder;
let audioChunks = [];

async function toggleRecording() {
    const btn = document.getElementById('voice-btn');
    const status = document.getElementById('recording-status');
    const contentArea = document.getElementById('reflection-content'); // Matches your textarea ID

    if (!mediaRecorder || mediaRecorder.state === 'inactive') {
        // Start Recording
        try {
            const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
            mediaRecorder = new MediaRecorder(stream);
            audioChunks = [];

            mediaRecorder.ondataavailable = (event) => {
                audioChunks.push(event.data);
            };

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
                    
                    // Update the textarea with transcribed text
                    contentArea.value = data.text;
                    // Store the audio URL in a hidden input for the final form submission
                    document.getElementById('audio-url').value = data.audio_url;
                    
                    status.innerText = "Voice processed!";
                } catch (err) {
                    console.error("Upload failed:", err);
                    status.innerText = "Upload failed.";
                }
            };

            mediaRecorder.start();
            btn.innerHTML = "🛑 Stop Recording";
            btn.classList.add('btn-danger');
            status.innerText = "Recording...";
        } catch (err) {
            console.error("Microphone access denied:", err);
            alert("Please allow microphone access to use this feature.");
        }
    } else {
        // Stop Recording
        mediaRecorder.stop();
        btn.innerHTML = "🎤 Start Voice Reflection";
        btn.classList.remove('btn-danger');
    }
}
