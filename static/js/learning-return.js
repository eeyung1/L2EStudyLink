function learningReturnPath() {
    const requested = new URLSearchParams(window.location.search).get('next');
    // Accept only the public Learning Hub routes. Never redirect to an
    // absolute URL, protocol-relative URL, or a different part of the app.
    return requested && /^\/learn(?:\/[a-z0-9]+(?:-[a-z0-9]+)*)?$/.test(requested) ? requested : '/learn';
}

function preserveLearningReturn(selector, destination) {
    const link = document.querySelector(selector);
    if (link) link.href = destination + '?next=' + encodeURIComponent(learningReturnPath());
}
