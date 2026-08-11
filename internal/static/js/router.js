// FlyFleet Premium SPA Router with FOUC prevention, Smooth Anchor Scrolling, and Progress Bar
(function() {
    // Create progress bar element
    const progressBar = document.createElement('div');
    progressBar.id = 'spa-progress-bar';
    progressBar.style.cssText = 'position: fixed; top: 0; left: 0; height: 3px; bg-color: rgb(99, 102, 241); background: linear-gradient(90deg, #6366f1 0%, #a855f7 100%); width: 0%; transition: width 0.2s ease, opacity 0.4s ease; z-index: 99999; opacity: 0; box-shadow: 0 0 10px rgba(99, 102, 241, 0.5); pointer-events: none;';
    document.documentElement.appendChild(progressBar);

    let progressTimer = null;
    function startProgress() {
        clearTimeout(progressTimer);
        progressBar.style.opacity = '1';
        progressBar.style.width = '0%';
        
        let width = 0;
        const tick = () => {
            if (width < 90) {
                width += (90 - width) * 0.15;
                progressBar.style.width = width + '%';
                progressTimer = setTimeout(tick, 100);
            }
        };
        tick();
    }

    function endProgress() {
        clearTimeout(progressTimer);
        progressBar.style.width = '100%';
        setTimeout(() => {
            progressBar.style.opacity = '0';
            setTimeout(() => {
                progressBar.style.width = '0%';
            }, 400);
        }, 200);
    }

    // Main page loading function
    function loadPage(url, options = {}) {
        const pushState = options.pushState !== false;
        const fetchOpts = {
            method: options.method || 'GET',
            headers: options.headers || {},
            body: options.body
        };

        startProgress();
        if (document.body) {
            document.body.style.opacity = '0.7';
            document.body.style.transition = 'opacity 0.15s ease';
        }

        fetch(url, fetchOpts)
            .then(res => res.text())
            .then(html => {
                const parser = new DOMParser();
                const doc = parser.parseFromString(html, 'text/html');

                // 1. Sync Head Elements (Stylesheets & Title)
                const newStyles = Array.from(doc.head.querySelectorAll('link[rel="stylesheet"], style'));
                const oldStyles = Array.from(document.head.querySelectorAll('link[rel="stylesheet"], style'));
                
                oldStyles.forEach(style => {
                    const href = style.getAttribute('href');
                    if (href && !newStyles.some(ns => ns.getAttribute('href') === href)) {
                        style.remove();
                    }
                });

                newStyles.forEach(style => {
                    const href = style.getAttribute('href');
                    if (href) {
                        if (!oldStyles.some(os => os.getAttribute('href') === href)) {
                            document.head.appendChild(style.cloneNode(true));
                        }
                    } else {
                        document.head.appendChild(style.cloneNode(true));
                    }
                });

                // Update Title
                document.title = doc.title;

                // 2. Render Page Content
                requestAnimationFrame(() => {
                    if (document.body && doc.body) {
                        document.body.className = doc.body.className;
                        document.body.innerHTML = doc.body.innerHTML;
                        document.body.style.opacity = '1';
                        
                        // Handle internal anchor hash scrolling if target exists
                        if (window.location.hash) {
                            const target = document.querySelector(window.location.hash);
                            if (target) {
                                target.scrollIntoView({ behavior: 'smooth' });
                            } else {
                                window.scrollTo(0, 0);
                            }
                        } else {
                            const mainContent = document.getElementById('main-content');
                            if (mainContent) {
                                mainContent.scrollTop = 0;
                            } else {
                                window.scrollTo(0, 0);
                            }
                        }

                        // Re-run script tags
                        document.body.querySelectorAll('script').forEach(oldScript => {
                            const newScript = document.createElement('script');
                            Array.from(oldScript.attributes).forEach(attr => newScript.setAttribute(attr.name, attr.value));
                            newScript.appendChild(document.createTextNode(oldScript.innerHTML));
                            oldScript.parentNode.replaceChild(newScript, oldScript);
                        });
                    }
                    endProgress();
                });

                if (pushState) {
                    history.pushState(null, '', url);
                }
            })
            .catch(err => {
                console.error('SPA navigation failed:', err);
                endProgress();
                if (document.body) document.body.style.opacity = '1';
                window.location.href = url;
            });
    }

    // Event Interceptors - Allow standard native browser navigation for full CSS & template reliability
    document.addEventListener('click', function(e) {
        const a = e.target.closest('a');
        if (!a) return;
        const href = a.getAttribute('href');
        if (!href) return;

        // Internal page anchor links (#features, #benefits, #comparison, etc.)
        if (href.startsWith('#')) {
            e.preventDefault();
            const target = document.querySelector(href);
            if (target) {
                target.scrollIntoView({ behavior: 'smooth' });
                history.pushState(null, '', href);
            }
            return;
        }
    });

    document.addEventListener('submit', function(e) {
        const form = e.target.closest('form');
        if (!form) return;
        if (form.getAttribute('action') === '/logout') return;
        const url = new URL(form.action || window.location.href, window.location.href);
        if (url.origin !== window.location.origin) return;

        // Skip forms that use Datastar AJAX action submit hooks directly
        if (form.hasAttribute('data-on-submit')) return;

        e.preventDefault();
        const method = (form.getAttribute('method') || 'GET').toUpperCase();
        const formData = new FormData(form);
        
        let fetchOpts = {
            method: method,
            headers: {}
        };

        let targetUrl = url.pathname;
        if (method === 'GET') {
            const params = new URLSearchParams(formData);
            targetUrl += '?' + params.toString();
        } else {
            fetchOpts.body = new URLSearchParams(formData);
            fetchOpts.headers['Content-Type'] = 'application/x-www-form-urlencoded';
        }

        loadPage(targetUrl, fetchOpts);
    });

    window.addEventListener('popstate', function() {
        if (window.location.hash) {
            const target = document.querySelector(window.location.hash);
            if (target) {
                target.scrollIntoView({ behavior: 'smooth' });
                return;
            }
        }
        loadPage(window.location.pathname + window.location.search, { pushState: false });
    });
})();
