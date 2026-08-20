/**
 * Avandab Fleet & Operations - Universal 3-Way Theme Controller
 * Supports: Day (Light), Night (Dark), System (Auto)
 * Synchronizes: document.documentElement.classList, localStorage, user_theme cookie,
 * OS matchMedia prefers-color-scheme, and backend DB API (/api/v1/users/me/preferences).
 */
(function () {
    function getCookie(name) {
        var match = document.cookie.match(new RegExp('(^|;\\s*)' + name + '=([^;]*)'));
        return match ? decodeURIComponent(match[2]) : null;
    }

    function setCookie(name, val, days) {
        var d = new Date();
        d.setTime(d.getTime() + (days * 24 * 60 * 60 * 1000));
        document.cookie = name + "=" + encodeURIComponent(val) + ";path=/;max-age=" + (days * 86400) + ";SameSite=Lax";
    }

    function getCurrentThemeMode() {
        return getCookie('user_theme') || localStorage.getItem('theme_mode') || localStorage.getItem('theme') || 'system';
    }

    function isDarkModeActive(mode) {
        if (mode === 'dark') return true;
        if (mode === 'light') return false;
        return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
    }

    function applyTheme(mode) {
        var isDark = isDarkModeActive(mode);
        if (isDark) {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark');
        }

        // Update 3-way dropdown trigger icons
        document.querySelectorAll('[data-current-theme-icon]').forEach(function (icon) {
            if (mode === 'light') icon.textContent = 'light_mode';
            else if (mode === 'dark') icon.textContent = 'dark_mode';
            else icon.textContent = 'brightness_auto';
        });

        // Update legacy/single button icons
        document.querySelectorAll('[data-theme-icon]').forEach(function (icon) {
            icon.textContent = isDark ? 'light_mode' : 'dark_mode';
        });

        // Update 3-way dropdown checkmarks
        document.querySelectorAll('[data-theme-check]').forEach(function (check) {
            if (check.getAttribute('data-theme-check') === mode) {
                check.classList.remove('hidden');
            } else {
                check.classList.add('hidden');
            }
        });

        // Update aria attributes
        var toggleBtns = document.querySelectorAll('#theme-toggle, [data-theme-toggle]');
        toggleBtns.forEach(function (btn) {
            btn.setAttribute('aria-pressed', isDark ? 'true' : 'false');
        });
    }

    function setThemeMode(mode) {
        if (mode !== 'light' && mode !== 'dark' && mode !== 'system') {
            mode = 'system';
        }

        try {
            localStorage.setItem('theme_mode', mode);
            localStorage.setItem('theme', mode);
            setCookie('user_theme', mode, 30);
        } catch (e) {}

        applyTheme(mode);

        // Close any open theme dropdowns
        document.querySelectorAll('#theme-dropdown, [data-theme-dropdown]').forEach(function (dd) {
            dd.classList.add('hidden');
        });

        // Synchronize with database when authenticated (FlyFleet Rule 1)
        try {
            fetch('/api/v1/users/me/preferences', {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ theme: mode })
            }).catch(function () {});
        } catch (e) {}
    }

    // Expose global controller
    window.AvandabTheme = {
        getMode: getCurrentThemeMode,
        setMode: setThemeMode,
        apply: applyTheme,
        toggleNext: function () {
            var current = getCurrentThemeMode();
            // Cycle: light -> dark -> system -> light
            if (current === 'light') setThemeMode('dark');
            else if (current === 'dark') setThemeMode('system');
            else setThemeMode('light');
        }
    };

    function initThemeEvents() {
        var mode = getCurrentThemeMode();
        applyTheme(mode);

        // 1. Setup 3-way dropdowns
        var dropdownBtns = document.querySelectorAll('#theme-menu-btn, [data-theme-menu-btn]');
        dropdownBtns.forEach(function (btn) {
            btn.addEventListener('click', function (e) {
                e.stopPropagation();
                var wrapper = btn.closest('#theme-menu-wrapper, [data-theme-wrapper]') || btn.parentElement;
                var dropdown = wrapper ? wrapper.querySelector('#theme-dropdown, [data-theme-dropdown]') : null;
                if (dropdown) {
                    var isHidden = dropdown.classList.toggle('hidden');
                    btn.setAttribute('aria-expanded', isHidden ? 'false' : 'true');
                }
            });
        });

        // Close dropdown when clicking outside
        document.addEventListener('click', function (e) {
            document.querySelectorAll('#theme-dropdown, [data-theme-dropdown]').forEach(function (dd) {
                var wrapper = dd.closest('#theme-menu-wrapper, [data-theme-wrapper]') || dd.parentElement;
                if (wrapper && !wrapper.contains(e.target)) {
                    dd.classList.add('hidden');
                    var btn = wrapper.querySelector('#theme-menu-btn, [data-theme-menu-btn]');
                    if (btn) btn.setAttribute('aria-expanded', 'false');
                }
            });
        });

        // 2. Setup theme choices inside dropdowns
        document.querySelectorAll('[data-theme-choice]').forEach(function (btn) {
            btn.addEventListener('click', function (e) {
                e.stopPropagation();
                var choice = btn.getAttribute('data-theme-choice');
                setThemeMode(choice);
            });
        });

        // 3. Setup single toggle buttons (e.g., in public header or mobile bars)
        document.querySelectorAll('#theme-toggle, [data-theme-toggle]').forEach(function (btn) {
            // Only attach if it's not a dropdown trigger
            if (!btn.hasAttribute('data-theme-menu-btn') && btn.id !== 'theme-menu-btn') {
                btn.addEventListener('click', function (e) {
                    e.preventDefault();
                    window.AvandabTheme.toggleNext();
                });
            }
        });

        // 4. Live OS theme change listener
        try {
            var mq = window.matchMedia('(prefers-color-scheme: dark)');
            var onOSChange = function () {
                if (getCurrentThemeMode() === 'system') {
                    applyTheme('system');
                }
            };
            if (mq.addEventListener) {
                mq.addEventListener('change', onOSChange);
            } else if (mq.addListener) {
                mq.addListener(onOSChange);
            }
        } catch (e) {}
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initThemeEvents);
    } else {
        initThemeEvents();
    }
})();
