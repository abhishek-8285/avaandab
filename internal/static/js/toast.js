// Global toast notifications. Zero-dependency (inline styles — safe with
// compiled Tailwind, which cannot see classes created at runtime).
// Usage: FlyToast.show('Payment failed', {tone:'error', requestId:'...', actionHref:'/contact-us'})
(function () {
    var TONES = {
        info: { bg: '#1e293b', fg: '#f1f5f9', accent: '#6366f1' },
        success: { bg: '#14532d', fg: '#f0fdf4', accent: '#22c55e' },
        warn: { bg: '#713f12', fg: '#fefce8', accent: '#eab308' },
        error: { bg: '#7f1d1d', fg: '#fef2f2', accent: '#ef4444' }
    };

    var container = null;

    function ensureContainer() {
        if (container && document.body.contains(container)) return container;
        container = document.createElement('div');
        container.id = 'fly-toast-container';
        container.setAttribute('aria-live', 'polite');
        container.style.cssText =
            'position:fixed;top:16px;right:16px;z-index:2147483647;display:flex;' +
            'flex-direction:column;gap:8px;max-width:min(92vw,380px);pointer-events:none;';
        (document.body || document.documentElement).appendChild(container);
        return container;
    }

    function show(message, opts) {
        opts = opts || {};
        var tone = TONES[opts.tone] || TONES.info;
        var root = ensureContainer();

        var el = document.createElement('div');
        el.setAttribute('role', opts.tone === 'error' ? 'alert' : 'status');
        el.style.cssText =
            'pointer-events:auto;background:' + tone.bg + ';color:' + tone.fg + ';' +
            'border-left:4px solid ' + tone.accent + ';border-radius:10px;' +
            'padding:12px 14px;font:14px/1.45 system-ui,-apple-system,sans-serif;' +
            'box-shadow:0 8px 24px rgba(0,0,0,.25);opacity:0;transform:translateX(12px);' +
            'transition:opacity .18s ease,transform .18s ease;';

        var text = document.createElement('div');
        text.textContent = String(message == null ? '' : message);
        el.appendChild(text);

        if (opts.requestId) {
            var ref = document.createElement('div');
            ref.textContent = 'ref: ' + opts.requestId;
            ref.style.cssText = 'margin-top:6px;font-size:11px;opacity:.75;font-family:ui-monospace,monospace;word-break:break-all;';
            el.appendChild(ref);
        }

        if (opts.actionHref) {
            var a = document.createElement('a');
            a.href = opts.actionHref;
            a.textContent = opts.actionLabel || 'Get help';
            a.style.cssText = 'display:inline-block;margin-top:8px;font-size:13px;font-weight:600;color:' + tone.accent + ';text-decoration:underline;';
            a.addEventListener('click', dismiss);
            el.appendChild(a);
        }

        root.appendChild(el);
        requestAnimationFrame(function () {
            el.style.opacity = '1';
            el.style.transform = 'translateX(0)';
        });

        var dismissed = false;
        function dismiss() {
            if (dismissed) return;
            dismissed = true;
            el.style.opacity = '0';
            el.style.transform = 'translateX(12px)';
            setTimeout(function () { el.remove(); }, 220);
        }

        var ttl = typeof opts.duration === 'number' ? opts.duration : (opts.tone === 'error' ? 9000 : 4500);
        setTimeout(dismiss, ttl);
        el.addEventListener('click', dismiss);
        return dismiss;
    }

    window.FlyToast = { show: show };
})();
