document.addEventListener('DOMContentLoaded', () => {
    const toggles = document.querySelectorAll('.mobile-menu-toggle');
    if (toggles.length > 1) {
        for (let i = 1; i < toggles.length; i++) {
            toggles[i].remove();
        }
    }
    const sidebar = document.querySelector('.sidebar');
    if (!sidebar) return;
    const toggle = sidebar.querySelector('.mobile-menu-toggle');
    const main = document.querySelector('main');

    // Admin link visibility on admin routes
    if (location.pathname === '/admin') {
        sidebar.querySelector('#adminNavLink')?.classList.remove('hidden');
    }

    // Active link matching
    const currentPath = location.pathname === '/masterclasses' ? '/masterclass' : location.pathname;
    sidebar.querySelectorAll('.sidebar-nav a[href]').forEach(link => {
        const active = new URL(link.href, location.origin).pathname === currentPath;
        link.classList.toggle('active', active);
        if (active) link.setAttribute('aria-current', 'page');
        else link.removeAttribute('aria-current');
    });

    // Helper to get friendly page title for tablet header
    function getPageTitle() {
        const h1 = document.querySelector('main h1, .page-title');
        if (h1 && h1.textContent.trim()) return h1.textContent.trim();
        const pathTitles = {
            '/dashboard': 'Today',
            '/timetable': 'Timetable',
            '/reflections': 'Reflections',
            '/planner': 'Weekly coach',
            '/focus': 'Focus room',
            '/search-page': 'Find study partners',
            '/my-bookings': 'Study sessions',
            '/buddies': 'Accountability buddies',
            '/masterclass': 'Masterclass',
            '/masterclasses': 'Masterclass',
            '/projects': 'Projects',
            '/learn': 'Library',
            '/skills': 'Profile & skills',
            '/portfolio': 'Learner portfolio',
            '/availability': 'Tutoring availability',
            '/admin': 'Admin'
        };
        return pathTitles[currentPath] || 'Workspace';
    }

    // Tablet header enhancement (current page title + account button)
    const heading = sidebar.querySelector('.sidebar-heading');
    if (heading && toggle) {
        if (!heading.querySelector('.workspace-current-title')) {
            const titleSpan = document.createElement('span');
            titleSpan.className = 'workspace-current-title';
            titleSpan.textContent = getPageTitle();
            const brand = heading.querySelector('.sidebar-brand');
            if (brand) brand.after(titleSpan);
            else heading.prepend(titleSpan);
        }
        if (!heading.querySelector('.tablet-account-btn')) {
            const accBtn = document.createElement('button');
            accBtn.type = 'button';
            accBtn.className = 'tablet-account-btn icon-btn';
            accBtn.setAttribute('aria-label', 'Open account menu');
            accBtn.innerHTML = '<span aria-hidden="true">👤</span><span class="sr-only">Account</span>';
            accBtn.addEventListener('click', () => {
                setOpen(true, false, accBtn);
                const accGroup = sidebar.querySelector('.workspace-nav-group.nav-group-account');
                accGroup?.scrollIntoView({ behavior: 'smooth' });
                accGroup?.querySelector('a')?.focus();
            });
            toggle.before(accBtn);
        }
    }

    // Search and quick find within drawer sheet
    const search = sidebar.querySelector('#workspaceNavSearch');
    const clear = sidebar.querySelector('#workspaceNavClear');
    const count = sidebar.querySelector('#workspaceNavCount');

    function filterPages() {
        if (!search) return;
        const query = search.value.trim().toLocaleLowerCase();
        let matches = 0;
        for (const link of sidebar.querySelectorAll('.sidebar-nav a[href]')) {
            const match = !query || link.textContent.toLocaleLowerCase().includes(query);
            link.hidden = !match;
            if (match && !link.closest('#adminNavLink.hidden')) matches++;
        }
        for (const group of sidebar.querySelectorAll('.workspace-nav-group')) {
            group.hidden = ![...group.querySelectorAll('a')].some(a => !a.hidden);
        }
        if (count) {
            count.hidden = !query;
            count.textContent = matches + ' matching page' + (matches === 1 ? '' : 's');
        }
        if (clear) {
            clear.hidden = !query;
        }
    }
    search?.addEventListener('input', filterPages);
    clear?.addEventListener('click', () => {
        search.value = '';
        filterPages();
        search.focus();
    });

    // Focusable items inside sidebar for modal containment
    const focusable = () => Array.from(sidebar.querySelectorAll('a[href]:not([hidden]), button:not([disabled]):not([hidden]), input:not([disabled]):not([hidden])'))
        .filter(element => element.getClientRects().length > 0);

    let lastOpener = toggle;
    const setOpen = (open, restoreFocus = false, opener = null) => {
        if (opener) lastOpener = opener;
        sidebar.classList.toggle('menu-open', open);
        document.body.classList.toggle('mobile-menu-open', open);
        if (toggle) {
            toggle.setAttribute('aria-expanded', String(open));
            toggle.setAttribute('aria-label', open ? 'Close navigation' : 'Open navigation');
            const icon = toggle.querySelector('.menu-icon');
            const label = toggle.querySelector('.menu-label');
            if (icon) icon.textContent = open ? '✕' : '☰';
            if (label) label.textContent = open ? 'Close' : 'Menu';
        }
        if (main) main.inert = open;
        if (restoreFocus) {
            (lastOpener || toggle)?.focus();
        }
    };

    if (toggle) {
        toggle.addEventListener('click', () => setOpen(!sidebar.classList.contains('menu-open'), false, toggle));
    }

    // Phone bottom navigation bar handling
    const bottomNav = sidebar.querySelector('.phone-bottom-nav');
    if (bottomNav) {
        const isToday = currentPath === '/dashboard';
        const todayLink = bottomNav.querySelector('[data-bottom-action="today"]');
        if (todayLink) {
            todayLink.classList.toggle('active', isToday);
            if (isToday) todayLink.setAttribute('aria-current', 'page');
            else todayLink.removeAttribute('aria-current');
        }
        const isPlan = ['/timetable', '/reflections', '/planner', '/focus'].includes(currentPath);
        bottomNav.querySelector('[data-bottom-action="plan"]')?.classList.toggle('active', isPlan);
        const isTogether = ['/search-page', '/my-bookings', '/buddies', '/masterclass', '/projects'].includes(currentPath);
        bottomNav.querySelector('[data-bottom-action="together"]')?.classList.toggle('active', isTogether);
        const isMore = ['/skills', '/portfolio', '/availability', '/admin', '/learn'].includes(currentPath);
        bottomNav.querySelector('[data-bottom-action="more"]')?.classList.toggle('active', isMore);

        bottomNav.querySelectorAll('button[data-bottom-action]').forEach(btn => {
            btn.addEventListener('click', () => {
                const action = btn.dataset.bottomAction;
                setOpen(true, false, btn);
                let targetGroup = null;
                if (action === 'plan') targetGroup = sidebar.querySelector('.nav-group-plan');
                else if (action === 'together') targetGroup = sidebar.querySelector('.nav-group-together');
                else if (action === 'more') targetGroup = sidebar.querySelector('.nav-group-account');
                if (targetGroup) {
                    targetGroup.scrollIntoView({ behavior: 'smooth', block: 'start' });
                    targetGroup.querySelector('a')?.focus();
                }
            });
        });
    }

    // Desktop dropdown menus (>=1100px)
    const dropdowns = sidebar.querySelectorAll('.nav-dropdown');
    dropdowns.forEach(dropdown => {
        const trigger = dropdown.querySelector('.nav-dropdown-trigger');
        const menu = dropdown.querySelector('.nav-dropdown-menu');
        if (!trigger || !menu) return;

        // Indicate active state on trigger if dropdown contains current page
        const hasActive = !!menu.querySelector('[aria-current="page"]');
        trigger.classList.toggle('has-active', hasActive);

        function toggleDropdown(open) {
            if (open) {
                dropdowns.forEach(other => {
                    if (other !== dropdown) {
                        other.classList.remove('is-open');
                        other.querySelector('.nav-dropdown-trigger')?.setAttribute('aria-expanded', 'false');
                    }
                });
                dropdown.classList.add('is-open');
                trigger.setAttribute('aria-expanded', 'true');
            } else {
                dropdown.classList.remove('is-open');
                trigger.setAttribute('aria-expanded', 'false');
            }
        }

        trigger.addEventListener('click', (e) => {
            e.stopPropagation();
            const isOpen = dropdown.classList.contains('is-open');
            toggleDropdown(!isOpen);
        });

        trigger.addEventListener('keydown', (e) => {
            if (e.key === 'ArrowDown' || e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                toggleDropdown(true);
                const first = menu.querySelector('a, button');
                first?.focus();
            } else if (e.key === 'ArrowUp') {
                e.preventDefault();
                toggleDropdown(true);
                const items = menu.querySelectorAll('a, button');
                items[items.length - 1]?.focus();
            } else if (e.key === 'Escape') {
                toggleDropdown(false);
                trigger.focus();
            }
        });

        menu.addEventListener('keydown', (e) => {
            const items = Array.from(menu.querySelectorAll('a, button')).filter(el => !el.closest('.hidden') && el.offsetParent !== null);
            const currentIndex = items.indexOf(document.activeElement);
            if (e.key === 'Escape') {
                e.preventDefault();
                toggleDropdown(false);
                trigger.focus();
            } else if (e.key === 'ArrowDown') {
                e.preventDefault();
                const next = (currentIndex + 1) % items.length;
                items[next]?.focus();
            } else if (e.key === 'ArrowUp') {
                e.preventDefault();
                const prev = (currentIndex - 1 + items.length) % items.length;
                items[prev]?.focus();
            } else if (e.key === 'Tab') {
                toggleDropdown(false);
            }
        });
    });

    // Close desktop dropdowns on outside click
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.nav-dropdown')) {
            dropdowns.forEach(d => {
                d.classList.remove('is-open');
                d.querySelector('.nav-dropdown-trigger')?.setAttribute('aria-expanded', 'false');
            });
        }
    });

    // Close drawer sheet when a link is clicked
    sidebar.querySelector('.sidebar-nav')?.addEventListener('click', event => {
        if (event.target.closest('a')) setOpen(false);
    });

    // Outside click closes drawer sheet on mobile/tablet
    document.addEventListener('click', event => {
        if (sidebar.classList.contains('menu-open')) {
            if (!sidebar.contains(event.target)) {
                setOpen(false);
            }
        }
    });

    // Keyboard handlers (Escape dismissal, modal focus trap)
    document.addEventListener('keydown', event => {
        if (event.key === 'Escape') {
            let handledDropdown = false;
            dropdowns.forEach(d => {
                if (d.classList.contains('is-open')) {
                    d.classList.remove('is-open');
                    const trigger = d.querySelector('.nav-dropdown-trigger');
                    trigger?.setAttribute('aria-expanded', 'false');
                    trigger?.focus();
                    handledDropdown = true;
                }
            });
            if (handledDropdown) return;
        }

        if (!sidebar.classList.contains('menu-open')) return;
        if (event.key === 'Escape') {
            event.preventDefault();
            setOpen(false, true);
        } else if (event.key === 'Tab') {
            const items = focusable();
            const first = items[0];
            const last = items[items.length - 1];
            if (!first) return;
            if (event.shiftKey && (document.activeElement === first || !sidebar.contains(document.activeElement))) {
                event.preventDefault();
                last.focus();
            } else if (!event.shiftKey && (document.activeElement === last || !sidebar.contains(document.activeElement))) {
                event.preventDefault();
                first.focus();
            }
        }
    });

    // Resize cleanup
    window.addEventListener('resize', () => {
        if (window.innerWidth >= 1100 && sidebar.classList.contains('menu-open')) {
            const focusWasOnToggle = document.activeElement === toggle;
            setOpen(false);
            if (focusWasOnToggle) sidebar.querySelector('[aria-current="page"]')?.focus();
        }
    });
});
