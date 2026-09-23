document.addEventListener('DOMContentLoaded', () => {
    const sidebar = document.querySelector('.sidebar');
    const toggle = sidebar && sidebar.querySelector('.mobile-menu-toggle');
    if (!toggle) return;

    const setOpen = open => {
        sidebar.classList.toggle('menu-open', open);
        document.body.classList.toggle('mobile-menu-open', open);
        toggle.setAttribute('aria-expanded', String(open));
        toggle.setAttribute('aria-label', open ? 'Close navigation' : 'Open navigation');
        toggle.querySelector('.menu-icon').textContent = open ? '✕' : '☰';
        toggle.querySelector('.menu-label').textContent = open ? 'Close' : 'Menu';
    };

    toggle.addEventListener('click', () => setOpen(!sidebar.classList.contains('menu-open')));
    sidebar.querySelector('.sidebar-nav').addEventListener('click', event => {
        if (event.target.closest('a')) setOpen(false);
    });
    document.addEventListener('keydown', event => {
        if (event.key === 'Escape' && sidebar.classList.contains('menu-open')) {
            setOpen(false);
            toggle.focus();
        }
    });
    window.addEventListener('resize', () => {
        if (window.innerWidth > 860 && sidebar.classList.contains('menu-open')) setOpen(false);
    });
});
