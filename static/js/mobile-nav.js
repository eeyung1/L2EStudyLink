document.addEventListener('DOMContentLoaded', () => {
    const sidebar = document.querySelector('.sidebar');
    const toggle = sidebar && sidebar.querySelector('.mobile-menu-toggle');
    if (!toggle) return;
    const focusable = () => Array.from(sidebar.querySelectorAll('a[href], button:not([disabled])'))
        .filter(element => element.getClientRects().length > 0);

    const setOpen = (open, restoreFocus = false) => {
        sidebar.classList.toggle('menu-open', open);
        document.body.classList.toggle('mobile-menu-open', open);
        toggle.setAttribute('aria-expanded', String(open));
        toggle.setAttribute('aria-label', open ? 'Close navigation' : 'Open navigation');
        toggle.querySelector('.menu-icon').textContent = open ? '✕' : '☰';
        toggle.querySelector('.menu-label').textContent = open ? 'Close' : 'Menu';
        if (restoreFocus) toggle.focus();
    };

    toggle.addEventListener('click', () => setOpen(!sidebar.classList.contains('menu-open')));
    sidebar.querySelector('.sidebar-nav').addEventListener('click', event => {
        if (event.target.closest('a')) setOpen(false);
    });
    document.addEventListener('keydown', event => {
        if (!sidebar.classList.contains('menu-open')) return;
        if (event.key === 'Escape') {
            setOpen(false, true);
        } else if (event.key === 'Tab') {
            const items = focusable();
            if (!items.length) return;
            const first = items[0];
            const last = items[items.length - 1];
            if (event.shiftKey && (document.activeElement === first || !sidebar.contains(document.activeElement))) {
                event.preventDefault();
                last.focus();
            } else if (!event.shiftKey && (document.activeElement === last || !sidebar.contains(document.activeElement))) {
                event.preventDefault();
                first.focus();
            }
        }
    });
    window.addEventListener('resize', () => {
        if (window.innerWidth > 860 && sidebar.classList.contains('menu-open')) {
            const focusWasOnToggle = document.activeElement === toggle;
            setOpen(false);
            if (focusWasOnToggle) sidebar.querySelector('.sidebar-nav a[href]')?.focus();
        }
    });
});
