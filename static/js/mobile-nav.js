document.addEventListener('DOMContentLoaded', () => {
    const sidebar = document.querySelector('.sidebar');
    const toggle = sidebar?.querySelector('.mobile-menu-toggle');
    if (!toggle) return;
    const main = document.querySelector('main');
    if(location.pathname==='/admin')sidebar.querySelector('#adminNavLink')?.classList.remove('hidden');
    sidebar.querySelectorAll('.sidebar-nav a[href]').forEach(link => {
        const active = new URL(link.href).pathname === location.pathname;
        link.classList.toggle('active', active);
        if (active) link.setAttribute('aria-current','page'); else link.removeAttribute('aria-current');
    });
    const focusable = () => Array.from(sidebar.querySelectorAll('a[href], button:not([disabled])'))
        .filter(element => element.getClientRects().length > 0);
    const setOpen = (open, restoreFocus = false) => {
        sidebar.classList.toggle('menu-open', open);
        document.body.classList.toggle('mobile-menu-open', open);
        toggle.setAttribute('aria-expanded', String(open));
        toggle.setAttribute('aria-label', open ? 'Close navigation' : 'Open navigation');
        const icon=toggle.querySelector('.menu-icon'),label=toggle.querySelector('.menu-label');
        if(icon)icon.textContent=open?'✕':'☰';
        if(label)label.textContent=open?'Close':'Menu';
        if(main)main.inert=open;
        if (restoreFocus) toggle.focus();
    };
    toggle.addEventListener('click', () => setOpen(!sidebar.classList.contains('menu-open')));
    sidebar.querySelector('.sidebar-nav').addEventListener('click', event => {
        if (event.target.closest('a')) setOpen(false);
    });
    document.addEventListener('keydown', event => {
        if (!sidebar.classList.contains('menu-open')) return;
        if (event.key === 'Escape') {event.preventDefault();setOpen(false,true);}
        else if (event.key === 'Tab') {
            const items=focusable(),first=items[0],last=items[items.length-1];
            if (!first) return;
            if (event.shiftKey && (document.activeElement===first || !sidebar.contains(document.activeElement))) {event.preventDefault();last.focus();}
            else if (!event.shiftKey && (document.activeElement===last || !sidebar.contains(document.activeElement))) {event.preventDefault();first.focus();}
        }
    });
    window.addEventListener('resize', () => {
        if (window.innerWidth>860 && sidebar.classList.contains('menu-open')) {
            const focusWasOnToggle=document.activeElement===toggle;
            setOpen(false);
            if(focusWasOnToggle)sidebar.querySelector('[aria-current="page"]')?.focus();
        }
    });
});
