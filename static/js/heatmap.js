// Renders an interactive, responsive GitHub-style activity contribution heatmap
function renderHeatmap(containerElement, dailyCounts, options = {}) {
    if (!containerElement) return;

    const weeks = options.weeks || 24;
    const today = options.today ? new Date(options.today + 'T00:00:00Z') : new Date();
    
    // Normalize to current day midnight UTC
    const currentDayOfWeek = (today.getUTCDay() + 6) % 7; // 0 = Mon, 6 = Sun
    
    // Calculate start date (Monday of weeks-ago)
    const totalDays = (weeks * 7) - (6 - currentDayOfWeek);
    const startDate = new Date(today);
    startDate.setUTCDate(today.getUTCDate() - totalDays + 1);
    const startDayOfWeek = (startDate.getUTCDay() + 6) % 7;
    startDate.setUTCDate(startDate.getUTCDate() - startDayOfWeek);

    const monthNames = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

    let gridHtml = '';
    const cur = new Date(startDate);

    // Generate day cells
    while (cur <= today) {
        const y = cur.getUTCFullYear();
        const m = String(cur.getUTCMonth() + 1).padStart(2, '0');
        const d = String(cur.getUTCDate()).padStart(2, '0');
        const dateStr = `${y}-${m}-${d}`;
        const count = dailyCounts ? (dailyCounts[dateStr] || 0) : 0;

        let level = 0;
        if (count >= 4) level = 4;
        else if (count >= 3) level = 3;
        else if (count >= 2) level = 2;
        else if (count >= 1) level = 1;

        const dateFormatted = `${monthNames[cur.getUTCMonth()]} ${cur.getUTCDate()}, ${y}`;
        const label = `${count} ${count === 1 ? 'activity' : 'activities'} on ${dateFormatted}`;

        gridHtml += `<div class="heatmap-cell level-${level}" data-date="${dateStr}" data-count="${count}" title="${label}" aria-label="${label}"></div>`;
        cur.setUTCDate(cur.getUTCDate() + 1);
    }

    containerElement.innerHTML = `
        <div class="heatmap-wrapper">
            <div class="heatmap-scroll">
                <div class="heatmap-grid" role="img" aria-label="Learning activity heatmap">
                    ${gridHtml}
                </div>
            </div>
            <div class="heatmap-footer">
                <span class="text-xs text-gray-500">Past ${weeks} weeks</span>
                <div class="heatmap-legend">
                    <span class="text-xs text-gray-400">Less</span>
                    <span class="heatmap-cell level-0" title="0 activities"></span>
                    <span class="heatmap-cell level-1" title="1 activity"></span>
                    <span class="heatmap-cell level-2" title="2 activities"></span>
                    <span class="heatmap-cell level-3" title="3 activities"></span>
                    <span class="heatmap-cell level-4" title="4+ activities"></span>
                    <span class="text-xs text-gray-400">More</span>
                </div>
            </div>
        </div>
    `;
}

async function loadAndRenderStreaks(containerId, options = {}) {
    const container = document.getElementById(containerId);
    if (!container) return;

    try {
        const token = (typeof sessionToken === 'function' ? sessionToken() : '') || localStorage.getItem('token') || '';
        const headers = token ? { 'Authorization': 'Bearer ' + token } : {};

        // Send local device date as today parameter
        const now = new Date();
        const localToday = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;

        const res = await fetch(`/api/v1/me/streaks?today=${localToday}`, { headers });
        if (!res.ok) throw new Error('Failed to load streak stats');
        const stats = await res.json();

        // Update streak stat numbers if elements exist
        const currStreakEl = document.getElementById('statCurrentStreak');
        if (currStreakEl) currStreakEl.textContent = `${stats.current_streak} ${stats.current_streak === 1 ? 'day' : 'days'}`;

        const longestStreakEl = document.getElementById('statLongestStreak');
        if (longestStreakEl) longestStreakEl.textContent = `${stats.longest_streak} ${stats.longest_streak === 1 ? 'day' : 'days'}`;

        const totalActiveEl = document.getElementById('statTotalActiveDays');
        if (totalActiveEl) totalActiveEl.textContent = `${stats.total_active_days} ${stats.total_active_days === 1 ? 'day' : 'days'}`;

        const totalReflectionsEl = document.getElementById('statTotalReflections');
        if (totalReflectionsEl) totalReflectionsEl.textContent = stats.total_reflections;

        // Render the heatmap
        const heatmapEl = document.getElementById('activityHeatmap') || container.querySelector('.heatmap-target');
        if (heatmapEl) {
            const weeks = window.innerWidth < 640 ? 16 : 28;
            renderHeatmap(heatmapEl, stats.daily_counts || {}, { today: stats.today, weeks });
        }
    } catch (err) {
        console.error('Error rendering streak heatmap:', err);
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = { renderHeatmap, loadAndRenderStreaks };
}
