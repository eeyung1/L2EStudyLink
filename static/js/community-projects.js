/**
 * L2EStudyLink - Community Projects (W28–W29)
 * Deliverables:
 * - W28: Responsive project discovery listing, scope/search filters, sanitized detail composition,
 *        safe repo URLs, meaningful empty/error states, and deep-link browser navigation.
 * - W29: Clear owner/member/requester contexts, opted-in invitations, application flow with
 *        draft protection (retaining form content across 409/retry), atomic action locks,
 *        and request inbox decision actions with confirmation modals.
 */
(() => {
    const { node, api, el, json, tell, button, state, card, badge, decide } = L2ECommunity;
    let myID = 0;
    let projects = [];
    let collaborators = [];
    let selected = 0;
    let detailVersion = 0;
    let mutating = false;

    function message(copy, error = false) {
        tell('projectMessage', copy, error);
    }

    /**
     * Render the filtered project cards into #projectList.
     * Supports search query against title and roles_needed, and scope filter (all, open, owned).
     */
    function renderProjects() {
        const searchInput = el('projectSearch');
        const scopeSelect = el('projectScope');
        const q = searchInput ? searchInput.value.toLowerCase().trim() : '';
        const scope = scopeSelect ? scopeSelect.value : 'all';

        const filtered = projects.filter(p => {
            const matchesQuery = !q || (p.title + ' ' + (p.roles_needed || '')).toLowerCase().includes(q);
            const matchesScope = scope === 'all' ||
                (scope === 'open' && p.status === 'open') ||
                (scope === 'owned' && p.owner_id === myID);
            return matchesQuery && matchesScope;
        });

        const box = el('projectList');
        box.replaceChildren();

        if (!filtered.length) {
            state(box, projects.length ? 'No projects match your filters.' : 'No available projects yet. Post one to start a team.');
            return;
        }

        for (const p of filtered) {
            const c = card(p.title);
            if (p.id === selected) {
                c.classList.add('is-selected');
            }
            c.append(
                badge(p.status === 'open' ? 'Recruiting' : 'Recruitment closed'),
                node('p', 'By ' + p.owner_name + ' · ' + (p.member_count || 0) + ' teammate(s)'),
                node('p', p.roles_needed || 'Roles open'),
                node('p', p.time_commitment || 'Flexible commitment'),
                button('View project', () => viewProject(p.id, true))
            );
            box.append(c);
        }
    }

    /**
     * Load project list from /api/v1/projects with loading and error states.
     */
    async function loadProjects() {
        const box = el('projectList');
        box.setAttribute('aria-busy', 'true');
        try {
            projects = await api('/projects');
            renderProjects();
        } catch (e) {
            state(box, e.message, loadProjects);
        } finally {
            box.setAttribute('aria-busy', 'false');
        }
    }

    /**
     * Load and manage invitation preferences (/api/v1/collaboration/preferences).
     */
    async function loadPreference() {
        const optIn = el('inviteOptIn');
        const retry = el('retryPreference');
        const status = el('preferenceState');
        optIn.disabled = true;
        retry.hidden = true;
        try {
            const p = await api('/collaboration/preferences');
            optIn.checked = !!p.open_to_invites;
            status.textContent = p.open_to_invites ? 'Open to project invitations' : 'Not accepting new invitations';
            optIn.disabled = false;
        } catch (e) {
            status.textContent = e.message;
            retry.hidden = false;
        }
    }

    /**
     * Load opted-in fellows catalog (/api/v1/collaborators).
     */
    async function loadFellows() {
        const box = el('fellowList');
        box.setAttribute('aria-busy', 'true');
        try {
            collaborators = await api('/collaborators');
            box.replaceChildren();
            if (!collaborators.length) {
                state(box, 'No fellows have opted in yet.');
                return;
            }
            const days = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
            for (const u of collaborators) {
                const c = card(u.name);
                const hours = u.availability && u.availability.length
                    ? u.availability.map(a => days[a.day_of_week] + ' ' + a.start_time + '–' + a.end_time).join(' · ')
                    : 'Not listed';
                c.append(
                    node('p', u.bio || 'No bio shared'),
                    node('p', 'Skills: ' + (u.skills || 'Not listed')),
                    node('p', 'Published tutoring hours: ' + hours)
                );
                box.append(c);
            }
        } catch (e) {
            state(box, e.message, loadFellows);
        } finally {
            box.setAttribute('aria-busy', 'false');
        }
    }

    /**
     * Load incoming/outgoing project requests and invitations (/api/v1/project-requests).
     */
    async function loadInbox() {
        const box = el('requestInbox');
        box.setAttribute('aria-busy', 'true');
        try {
            const requests = await api('/project-requests');
            box.replaceChildren();
            if (!requests.length) {
                state(box, 'No requests or invitations yet.');
                return;
            }
            for (const r of requests) {
                const c = card(r.project_title);
                const canDecide = r.status === 'pending' && (
                    r.kind === 'application' ? r.owner_id === myID : r.user_id === myID
                );
                c.append(
                    badge(r.status),
                    node('p', r.kind === 'application'
                        ? 'Application from ' + r.user_name
                        : r.user_id === myID
                            ? 'Invitation for you'
                            : 'Invitation sent to ' + r.user_name),
                    node('p', r.note || 'No note included')
                );
                if (canDecide) {
                    const actions = node('div', undefined, 'community-actions');
                    for (const decision of ['accepted', 'declined']) {
                        actions.append(button(
                            decision === 'accepted' ? 'Accept' : 'Decline',
                            () => decide(
                                decision === 'accepted' ? 'Accept this request?' : 'Decline this request?',
                                r.project_title + '. ' + (decision === 'accepted' ? 'Acceptance adds this fellow to the team.' : 'The request will be marked declined.'),
                                async () => {
                                    await api('/project-requests/' + r.id + '/status', {
                                        method: 'PUT',
                                        body: json({ status: decision })
                                    });
                                    message('Request ' + decision + '.');
                                    await Promise.all([loadInbox(), loadProjects()]);
                                    if (selected) await viewProject(selected);
                                }
                            )
                        ));
                    }
                    c.append(actions);
                }
                c.append(button('View project', () => viewProject(r.project_id, true), true));
                box.append(c);
            }
        } catch (e) {
            state(box, e.message, loadInbox);
        } finally {
            box.setAttribute('aria-busy', 'false');
        }
    }

    /**
     * Display project detail in #projectDetails.
     * Manages deep link, owner controls, membership badge, join application, and fellow invitations.
     */
    async function viewProject(id, focus = false) {
        selected = Number(id);
        const version = ++detailVersion;
        const box = el('projectDetails');
        box.hidden = false;
        box.setAttribute('aria-busy', 'true');
        state(box, 'Loading project…');

        try {
            const p = await api('/projects/' + id);
            if (version !== detailVersion) return;
            box.replaceChildren();

            const h = node('h2', p.title);
            h.id = 'detailsHeading';

            const members = p.members || [];
            const membersList = p.owner_name + ' (owner)' + members.map(m => ', ' + m.name).join('');

            box.append(
                h,
                badge(p.status === 'open' ? 'Recruiting' : 'Recruitment closed'),
                node('p', 'Owned by ' + p.owner_name),
                node('p', p.description, 'community-description'),
                node('p', 'Roles: ' + p.roles_needed),
                node('p', 'Commitment: ' + p.time_commitment),
                node('h3', 'Team'),
                node('p', membersList)
            );

            // Secure repository URL: only allow valid http: or https: URLs, rejecting dangerous protocols like javascript:
            if (p.repo_url) {
                try {
                    const url = new URL(p.repo_url);
                    if (['https:', 'http:'].includes(url.protocol)) {
                        const link = node('a', 'Project repository (opens a new tab)');
                        link.href = url.href;
                        link.target = '_blank';
                        link.rel = 'noopener noreferrer';
                        box.append(link);
                    }
                } catch {}
            }

            const owner = p.owner_id === myID;
            const member = members.some(m => m.id === myID);

            if (p.my_request_id) {
                box.append(node('p', 'Your ' + p.my_request_kind + ': ' + p.my_request_status));
            }
            if (member) {
                box.append(badge('You are on this team'));
            }

            // Join application form: for non-owner, non-member on open projects without existing requests
            if (!owner && !member && p.status === 'open' && !p.my_request_id) {
                const form = node('form', undefined, 'community-form');
                form.dataset.unsavedWatch = '';
                const label = node('label', 'How can you contribute?');
                const note = node('textarea');
                note.id = 'joinNote';
                label.htmlFor = note.id;
                note.minLength = 10;
                note.maxLength = 500;
                note.required = true;
                const err = node('p', undefined, 'l2e-feedback is-error');
                err.setAttribute('role', 'status');
                const submit = node('button', 'Request to join');
                submit.type = 'submit';
                form.append(label, note, err, submit);

                if (window.L2EUnsaved?.watch) window.L2EUnsaved.watch(form);

                form.addEventListener('submit', async e => {
                    e.preventDefault();
                    if (mutating) return;
                    mutating = true;
                    submit.disabled = true;
                    err.textContent = '';
                    try {
                        await api('/projects/' + id + '/requests', {
                            method: 'POST',
                            body: json({ kind: 'application', note: note.value.trim() })
                        });
                        if (window.L2EUnsaved?.saved) window.L2EUnsaved.saved(form);
                        message('Application sent. The owner will decide.');
                        await Promise.all([loadInbox(), viewProject(id)]);
                    } catch (e) {
                        // Draft protection: preserve note.value across failure/retry
                        err.textContent = e.message;
                    } finally {
                        mutating = false;
                        submit.disabled = false;
                    }
                });
                box.append(form);
            }

            // Closed recruitment notice for non-members
            if (!owner && !member && p.status !== 'open' && !p.my_request_id) {
                box.append(node('p', 'Recruitment is currently closed for this project.', 'l2e-muted'));
            }

            // Owner management actions: toggle recruitment status and invite opted-in fellows
            if (owner) {
                box.append(button(
                    p.status === 'open' ? 'Close recruitment' : 'Reopen recruitment',
                    () => decide(
                        p.status === 'open' ? 'Close recruitment?' : 'Reopen recruitment?',
                        p.title + '. Existing team membership stays in place.',
                        async () => {
                            await api('/projects/' + id + '/status', {
                                method: 'PUT',
                                body: json({ status: p.status === 'open' ? 'closed' : 'open' })
                            });
                            await Promise.all([loadProjects(), viewProject(id)]);
                            message('Recruitment updated.');
                        }
                    ),
                    true
                ));

                if (p.status === 'open') {
                    // Ensure collaborators are available
                    if (!collaborators.length) {
                        try { collaborators = await api('/collaborators'); } catch {}
                    }
                    const form = node('form', undefined, 'community-form');
                    form.dataset.unsavedWatch = '';
                    const label = node('label', 'Invite an opted-in fellow');
                    const select = node('select');
                    select.id = 'inviteUser';
                    select.required = true;
                    label.htmlFor = select.id;
                    const option = node('option', 'Choose a fellow');
                    option.value = '';
                    select.append(option);

                    const eligible = collaborators.filter(u => u.id !== p.owner_id && !members.some(m => m.id === u.id));
                    for (const u of eligible) {
                        const o = node('option', u.name + ' · ' + (u.skills || 'Learner'));
                        o.value = u.id;
                        select.append(o);
                    }

                    const noteLabel = node('label', 'Invitation note (optional)');
                    const note = node('textarea');
                    note.id = 'inviteNote';
                    note.maxLength = 500;
                    noteLabel.htmlFor = note.id;

                    const submit = node('button', 'Send invitation');
                    submit.type = 'submit';
                    const err = node('p', undefined, 'l2e-feedback is-error');
                    err.setAttribute('role', 'status');

                    form.append(label, select, noteLabel, note, err, submit);
                    if (window.L2EUnsaved?.watch) window.L2EUnsaved.watch(form);

                    form.addEventListener('submit', async e => {
                        e.preventDefault();
                        if (mutating) return;
                        mutating = true;
                        submit.disabled = true;
                        err.textContent = '';
                        try {
                            await api('/projects/' + id + '/requests', {
                                method: 'POST',
                                body: json({
                                    kind: 'invitation',
                                    user_id: Number(select.value),
                                    note: note.value.trim()
                                })
                            });
                            if (window.L2EUnsaved?.saved) window.L2EUnsaved.saved(form);
                            message('Invitation sent. The fellow decides whether to join.');
                            note.value = '';
                            select.value = '';
                            await loadInbox();
                        } catch (e) {
                            err.textContent = e.message;
                        } finally {
                            mutating = false;
                            submit.disabled = false;
                        }
                    });
                    box.append(form);
                }
            }

            // Deep-link navigation: update URL and push history entry when requested
            if (focus) {
                box.scrollIntoView({ block: 'start' });
                box.focus();
                const url = new URL(location.href);
                if (url.searchParams.get('project') !== String(id)) {
                    url.searchParams.set('project', id);
                    history.pushState({}, '', url);
                }
            }
        } catch (e) {
            if (version === detailVersion) state(box, e.message, () => viewProject(id, focus));
        } finally {
            if (version === detailVersion) box.setAttribute('aria-busy', 'false');
        }
    }

    // Toggle invitation preference
    el('inviteOptIn').addEventListener('change', async function() {
        const next = this.checked;
        this.disabled = true;
        try {
            await api('/collaboration/preferences', {
                method: 'PUT',
                body: json({ open_to_invites: next })
            });
            el('preferenceState').textContent = next ? 'Open to project invitations' : 'Not accepting new invitations';
            message('Invitation preference saved.');
        } catch (e) {
            this.checked = !next;
            message(e.message, true);
        } finally {
            this.disabled = false;
        }
    });

    el('retryPreference').addEventListener('click', loadPreference);
    el('projectSearch').addEventListener('input', renderProjects);
    el('projectScope').addEventListener('change', renderProjects);
    el('projectFilters').addEventListener('submit', e => e.preventDefault());

    // Post project form with draft protection and double-submit lock
    el('projectForm').addEventListener('submit', async function(e) {
        e.preventDefault();
        const submit = this.querySelector('button[type=submit]');
        if (submit.disabled) return;
        submit.disabled = true;
        tell('projectFormError', '');

        try {
            const p = await api('/projects', {
                method: 'POST',
                body: json({
                    title: el('projectTitle').value,
                    description: el('projectDescription').value,
                    roles_needed: el('projectRoles').value,
                    time_commitment: el('projectCommitment').value,
                    repo_url: el('projectRepo').value
                })
            });
            this.reset();
            if (window.L2EUnsaved?.saved) window.L2EUnsaved.saved(this);
            message('Project posted.');
            await loadProjects();
            await viewProject(p.id, true);
        } catch (e) {
            tell('projectFormError', e.message, true);
        } finally {
            submit.disabled = false;
        }
    });

    // Deep link URL restoration on page load and popstate
    function fromURL() {
        const id = Number(new URLSearchParams(location.search).get('project'));
        if (Number.isSafeInteger(id) && id > 0) {
            viewProject(id);
        } else {
            selected = 0;
            detailVersion++;
            el('projectDetails').hidden = true;
        }
    }
    addEventListener('popstate', fromURL);

    // Auto-open post project details if targeted via link
    document.querySelector('a[href="#postProject"]')?.addEventListener('click', () => {
        const details = el('postProject');
        if (details) {
            details.open = true;
            setTimeout(() => el('projectTitle')?.focus(), 50);
        }
    });

    // Initialization sequence
    async function initialize() {
        try {
            const me = await api('/me');
            myID = me.id;
        } catch (e) {
            message(e.message, true);
            state(el('projectList'), 'Account could not be loaded.', initialize);
            return;
        }
        await Promise.all([loadPreference(), loadProjects(), loadInbox(), loadFellows()]);
        fromURL();
    }
    initialize();
})();
