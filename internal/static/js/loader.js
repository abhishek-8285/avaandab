(function () {
  var overlayEl = null;
  var btnSpinners = new WeakMap();
  var manual = 0;
  var fetchPending = 0;
  var delayTimer = null;

  function ensureOverlay() {
    if (overlayEl) return overlayEl;
    overlayEl = document.createElement('div');
    overlayEl.id = 'global-loader';
    overlayEl.setAttribute('role', 'status');
    overlayEl.setAttribute('aria-live', 'polite');
    overlayEl.setAttribute('aria-hidden', 'true');
    overlayEl.innerHTML =
      '<div class="loader-backdrop"></div>' +
      '<div class="loader-box">' +
      '<div class="loader-spinner" aria-hidden="true"></div>' +
      '<span class="loader-text">Loading…</span>' +
      '</div>';
    overlayEl.style.display = 'none';
    (document.body || document.documentElement).appendChild(overlayEl);
    return overlayEl;
  }

  function render() {
    var active = (manual + fetchPending) > 0;
    if (!active && !overlayEl) return;
    var el = overlayEl || ensureOverlay();
    if (active) {
      el.style.display = 'flex';
      el.setAttribute('aria-hidden', 'false');
    } else {
      el.style.display = 'none';
      el.setAttribute('aria-hidden', 'true');
    }
  }

  function showSoon() {
    if (delayTimer) return;
    delayTimer = setTimeout(function () { delayTimer = null; render(); }, 200);
  }

  function fetchStart() {
    fetchPending++;
    if (fetchPending === 1) showSoon();
  }

  function fetchEnd() {
    fetchPending = Math.max(0, fetchPending - 1);
    if (fetchPending === 0) { clearTimeout(delayTimer); delayTimer = null; render(); }
  }

  var Loader = {
    show: function () { manual++; render(); },
    hide: function () { manual = Math.max(0, manual - 1); render(); },
    wrap: function (p) {
      Loader.show();
      return Promise.resolve(p).finally(function () { Loader.hide(); });
    },
    spin: function (btn, label) {
      if (!btn || btnSpinners.has(btn)) return function () {};
      var prev = btn.innerHTML;
      var prevDisabled = btn.disabled;
      btn.disabled = true;
      btn.innerHTML =
        '<span class="loader-btn-spin" aria-hidden="true"></span>' +
        (label ? '<span>' + label + '</span>' : '');
      btnSpinners.set(btn, { prev: prev, prevDisabled: prevDisabled });
      return function () { Loader.unspin(btn); };
    },
    unspin: function (btn) {
      var s = btnSpinners.get(btn);
      if (!s) return;
      btn.innerHTML = s.prev;
      btn.disabled = s.prevDisabled;
      btnSpinners.delete(btn);
    },
    bindFetch: function () {
      if (window.__loaderFetchBound) return;
      window.__loaderFetchBound = true;
      if (!window.fetch) return;
      var orig = window.fetch.bind(window);
      window.fetch = function () {
        fetchStart();
        try { return orig.apply(window, arguments).finally(fetchEnd); }
        catch (e) { fetchEnd(); throw e; }
      };
    }
  };
  window.Loader = Loader;

  function init() { Loader.bindFetch(); }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init);
  else init();
})();
