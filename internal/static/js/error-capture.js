// Client-side error capture: breadcrumbs + global handlers + report to backend.
// Loads after toast.js. Exposes window.ErrorCapture.
(function () {
    var MAX_CRUMBS = 25;
    var crumbs = [];
    var lastRequestId = null;
    var reportCount = 0;

    function now() { return new Date().toISOString().slice(11, 23); }

    function breadcrumb(type, detail) {
        crumbs.push({ t: now(), type: type, detail: String(detail).slice(0, 300) });
        if (crumbs.length > MAX_CRUMBS) crumbs.shift();
    }

    // ---- fetch instrumentation (records status + problem+json request ids)
    var origFetch = window.fetch;
    if (origFetch) {
        window.fetch = function () {
            var url = arguments[0];
            var opts = arguments[1] || {};
            var target = typeof url === 'string' ? url : (url && url.url) || '';
            return origFetch.apply(this, arguments).then(function (res) {
                if (!res.ok && !target.includes('/api/v1/errors/client')) {
                    breadcrumb('fetch', (opts.method || 'GET') + ' ' + target + ' \u2192 ' + res.status);
                    try {
                        res.clone().json().then(function (p) {
                            if (p && p.request_id) lastRequestId = p.request_id;
                        }).catch(function () {});
                    } catch (e) {}
                }
                return res;
            });
        };
    }

    // ---- UI click breadcrumbs
    document.addEventListener('click', function (e) {
        var el = e.target.closest('a[href],button,[role="button"],input[type="submit"]');
        if (!el) return;
        var label = (el.innerText || el.value || el.getAttribute('aria-label') || '').trim().slice(0, 80);
        breadcrumb('click', label || el.tagName);
    }, true);

    // ---- route change breadcrumbs
    var origPush = history.pushState;
    history.pushState = function () {
        breadcrumb('route', 'push ' + String(arguments[2] || ''));
        return origPush.apply(this, arguments);
    };

    // ---- global handlers
    window.addEventListener('error', function (ev) {
        report(ev.message, ev.error && ev.error.stack, true);
    });
    window.addEventListener('unhandledrejection', function (ev) {
        var r = ev.reason;
        var msg = r && r.message ? r.message : String(r);
        report(msg, r && r.stack, false);
    });

    function buildPayload(message, stack) {
        return {
            message: message,
            stack: stack || '',
            path: location.pathname + location.search,
            user_agent: navigator.userAgent,
            viewport: window.innerWidth + 'x' + window.innerHeight,
            request_id: lastRequestId || '',
            breadcrumbs: crumbs.slice()
        };
    }

    function report(message, stack, isErrorEvent) {
        if (reportCount >= 5) return; // burst guard
        reportCount++;
        var payload = buildPayload(message, stack);
        breadcrumb(isErrorEvent ? 'error' : 'rejection', message);

        try {
            var orig = window.fetch;
            (orig || fetch)('/api/v1/errors/client', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
                keepalive: true,
                credentials: 'same-origin'
            }).catch(function () {});
        } catch (e) {}

        if (window.FlyToast) {
            FlyToast.show(
                'Something went wrong on this page. Our team has been notified automatically.',
                {
                    tone: 'error',
                    requestId: lastRequestId || undefined,
                    actionHref: '/contact-us?ref=' + encodeURIComponent(lastRequestId || '') +
                        '&about=' + encodeURIComponent(String(message).slice(0, 120)),
                    actionLabel: 'Report details',
                    duration: 9000
                }
            );
        }
    }

    window.ErrorCapture = {
        breadcrumb: breadcrumb,
        getRequestId: function () { return lastRequestId; },
        report: function (message, stack) { report(message, stack, false); }
    };
})();
