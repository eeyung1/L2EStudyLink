// Retry only safe GET requests when the server or network briefly fails.
async function readAPI(path, token) {
    for (let attempt = 0; attempt < 3; attempt++) {
        try {
            const response = await fetch(path, {
                headers: { 'Authorization': 'Bearer ' + token }
            });
            if (response.status === 401) {
                localStorage.removeItem('token');
                window.location.href = '/page/login';
                throw new Error('Session expired');
            }
            if (!response.ok) {
                if (response.status >= 500 && attempt < 2) {
                    await new Promise(resolve => setTimeout(resolve, 500 * (attempt + 1)));
                    continue;
                }
                throw new Error('Server returned ' + response.status);
            }
            const data = await response.json();
            if (!Array.isArray(data)) throw new Error('Unexpected server response');
            return data;
        } catch (error) {
            if (error.message === 'Session expired' || error.message.startsWith('Server returned ') ||
                error.message === 'Unexpected server response' || attempt === 2) throw error;
            await new Promise(resolve => setTimeout(resolve, 500 * (attempt + 1)));
        }
    }
}
