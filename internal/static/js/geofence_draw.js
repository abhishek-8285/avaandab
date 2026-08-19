/* geofence_draw.js — Leaflet drawing for the geofence editor (Spec 02 §8).
 * Modes:
 *   circle  — click to set the centre, slider/drag for radius (metres)
 *   polygon — click to add vertices, double-click or "Close Polygon" to finish
 * Hidden inputs updated on every change: center_lat, center_lng, radius_m, polygon.
 * Edit mode renders the existing zone (from data-* attributes) as an editable shape.
 */
(function () {
  'use strict';

  var config = {
    tiles: 'https://mt1.google.com/vt/lyrs=m&x={x}&y={y}&z={z}&gl=IN',
    fallbackTiles: 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',
    maxZoom: 19
  };

  function initGeofenceDrawer(root) {
    if (!root || typeof L === 'undefined') {
      return;
    }
    var shapeInput = root.querySelector('input[name="shape"]');
    var mode = shapeInput ? shapeInput.value : 'circle';
    var mapEl = root.querySelector('#geofence-map');
    if (!mapEl) {
      return;
    }

    var centerLat = parseFloat(mapEl.dataset.centerLat || '') || 12.9716;
    var centerLng = parseFloat(mapEl.dataset.centerLng || '') || 77.5946;
    var radiusM = parseFloat(mapEl.dataset.radiusM || '') || 500;
    var polygonJSON = mapEl.dataset.polygon || '';
    var shape = mapEl.dataset.shape || '';

    var hiddenLat = root.querySelector('input[name="center_lat"]');
    var hiddenLng = root.querySelector('input[name="center_lng"]');
    var hiddenRadius = root.querySelector('input[name="radius_m"]');
    var hiddenPolygon = root.querySelector('input[name="polygon"]');

    var map = L.map(mapEl, {
      center: [centerLat, centerLng],
      zoom: 13
    });
    var google = L.tileLayer(config.tiles, { maxZoom: config.maxZoom });
    google.on('tileerror', function () {
      if (!this._osmLoaded) {
        this._osmLoaded = true;
        L.tileLayer(config.fallbackTiles, { maxZoom: 19 }).addTo(map);
      }
    });
    google.addTo(map);

    var drawLayer = L.layerGroup().addTo(map);
    var currentCircle = null;
    var currentPolygon = null;
    var vertices = [];

    function setCircle(lat, lng, radius) {
      if (currentCircle) {
        drawLayer.removeLayer(currentCircle);
      }
      currentCircle = L.circle([lat, lng], { radius: radius }).addTo(drawLayer);
      if (hiddenLat) { hiddenLat.value = lat.toFixed(6); }
      if (hiddenLng) { hiddenLng.value = lng.toFixed(6); }
      if (hiddenRadius) { hiddenRadius.value = Math.round(radius); }
      if (shapeInput) { shapeInput.value = 'circle'; }
      if (hiddenPolygon) { hiddenPolygon.value = ''; }
    }

    function renderPolygon(coords) {
      if (currentPolygon) {
        drawLayer.removeLayer(currentPolygon);
      }
      currentPolygon = L.polygon(coords).addTo(drawLayer);
    }

    function syncPolygonInput() {
      if (hiddenPolygon) {
        hiddenPolygon.value = JSON.stringify(vertices.map(function (v) { return [v[0], v[1]]; }));
      }
      if (shapeInput) { shapeInput.value = 'polygon'; }
      if (hiddenLat) { hiddenLat.value = ''; }
      if (hiddenLng) { hiddenLng.value = ''; }
      if (hiddenRadius) { hiddenRadius.value = ''; }
    }

    function addVertex(latlng) {
      vertices.push([latlng.lat, latlng.lng]);
      renderPolygon(vertices);
      syncPolygonInput();
    }

    function closePolygon() {
      if (vertices.length >= 3) {
        if (currentPolygon) {
          currentPolygon.closePopup();
        }
      }
    }

    // Edit mode: render the existing zone.
    if (shape === 'circle') {
      setCircle(centerLat, centerLng, radiusM);
      map.setView([centerLat, centerLng], 15);
    } else if (shape === 'polygon' && polygonJSON) {
      try {
        var parsed = JSON.parse(polygonJSON);
        if (Array.isArray(parsed) && parsed.length >= 3) {
          vertices = parsed;
          renderPolygon(vertices);
          syncPolygonInput();
          map.fitBounds(L.polygon(vertices).getBounds());
        }
      } catch (e) {
        // Invalid stored polygon — start blank.
      }
    }

    map.on('click', function (e) {
      if (mode === 'polygon') {
        addVertex(e.latlng);
      } else if (mode === 'circle') {
        var r = currentCircle ? currentCircle.getRadius() : radiusM;
        setCircle(e.latlng.lat, e.latlng.lng, r);
      }
    });

    map.on('dblclick', function () {
      if (mode === 'polygon') {
        closePolygon();
      }
    });

    // Circle radius slider.
    var radiusSlider = root.querySelector('#radius-slider');
    var radiusValue = root.querySelector('#radius-value');
    if (radiusSlider) {
      radiusSlider.value = Math.max(100, Math.min(radiusM, parseInt(radiusSlider.max, 10) || 5000));
      radiusSlider.addEventListener('input', function () {
        if (mode === 'circle' && currentCircle) {
          var r = parseInt(this.value, 10) || 100;
          currentCircle.setRadius(r);
          var lat = currentCircle.getLatLng().lat;
          var lng = currentCircle.getLatLng().lng;
          if (hiddenLat) { hiddenLat.value = lat.toFixed(6); }
          if (hiddenLng) { hiddenLng.value = lng.toFixed(6); }
          if (hiddenRadius) { hiddenRadius.value = r; }
          if (radiusValue) { radiusValue.textContent = r + ' m'; }
        }
      });
    }

    var modeButtons = root.querySelectorAll('[data-draw-mode]');
    for (var i = 0; i < modeButtons.length; i++) {
      modeButtons[i].addEventListener('click', function () {
        mode = this.dataset.drawMode;
        for (var j = 0; j < modeButtons.length; j++) {
          modeButtons[j].classList.toggle('is-active', modeButtons[j] === this);
        }
      });
    }

    var closeBtn = root.querySelector('#close-polygon');
    if (closeBtn) {
      closeBtn.addEventListener('click', closePolygon);
    }

    // Initial circle when in circle mode and nothing drawn yet.
    if (mode === 'circle' && !currentCircle && !currentPolygon) {
      setCircle(centerLat, centerLng, radiusM);
    }

    map.invalidateSize();
  }

  // Datastar/htmx style: run once on load and after fragment swaps.
  function initAll() {
    var roots = document.querySelectorAll('[data-geofence-drawer]');
    for (var i = 0; i < roots.length; i++) {
      initGeofenceDrawer(roots[i]);
    }
  }

  document.addEventListener('DOMContentLoaded', initAll);
  document.addEventListener('datastar:after-swap', initAll);
})();
