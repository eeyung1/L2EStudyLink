// Short-lived, per-session cache for the signed-in user's own planning data.
// Only these GET responses are cached; changes must explicitly invalidate them.
const planningPaths = new Set(['/api/v1/timetable', '/api/v1/reflections']);
const planningInFlight = new Map();
const planningGeneration = new Map();
const planningTTL = 20 * 1000;
let planningToken = null;
let planningScope = null;

async function planningCacheKey(path, token) {
    if (!planningPaths.has(path)) throw new Error('Unsupported cached path');
    if (planningToken !== token) {
        planningToken = token;
        planningScope = crypto.subtle.digest('SHA-256', new TextEncoder().encode(token))
            .then(bytes => Array.from(new Uint8Array(bytes), n => n.toString(16).padStart(2, '0')).join(''));
    }
    const scope = planningScope;
    return 'l2e:planning:v1:' + await scope + ':' + path;
}

async function invalidateReadCache(paths, token) {
    for (const path of paths) {
        const key = await planningCacheKey(path, token);
        planningGeneration.set(key, (planningGeneration.get(key) || 0) + 1);
        planningInFlight.delete(key);
        try { sessionStorage.removeItem(key); } catch (_) { /* Storage can be disabled. */ }
    }
}

async function clearPlanningCache(token) {
    await invalidateReadCache([...planningPaths], token);
}

async function readAPI(path, token) {
    const key = await planningCacheKey(path, token);
    try {
        const cached = JSON.parse(sessionStorage.getItem(key));
        if (cached && Date.now() - cached.savedAt < planningTTL && Array.isArray(cached.data)) return cached.data;
    } catch (_) { /* Network still works when storage is unavailable. */ }
    if (planningInFlight.has(key)) return planningInFlight.get(key);

    const generation = planningGeneration.get(key) || 0;
    const pending = (async () => {
        for (let attempt = 0; attempt < 3; attempt++) {
            try {
                const response = await fetch(path, { headers: { Authorization: 'Bearer ' + token } });
                if (response.status === 401) {
                    await clearPlanningCache(token);
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
                if (planningGeneration.get(key) === undefined || planningGeneration.get(key) === generation) {
                    try { sessionStorage.setItem(key, JSON.stringify({ savedAt: Date.now(), data })); } catch (_) { /* Storage can be disabled. */ }
                }
                return data;
            } catch (error) {
                if (error.message === 'Session expired' || error.message.startsWith('Server returned ') ||
                    error.message === 'Unexpected server response' || attempt === 2) throw error;
                await new Promise(resolve => setTimeout(resolve, 500 * (attempt + 1)));
            }
        }
    })();
    planningInFlight.set(key, pending);
    try { return await pending; }
    finally { if (planningInFlight.get(key) === pending) planningInFlight.delete(key); }
}
