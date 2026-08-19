document.addEventListener('DOMContentLoaded', () => {
    const mapEl = document.getElementById('map');
    if (!mapEl || typeof L === 'undefined') return;

    // Initialize Leaflet map centered on India
    const map = L.map('map').setView([20.5937, 78.9629], 5);
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        maxZoom: 19,
        attribution: '&copy; OpenStreetMap contributors'
    }).addTo(map);

    const markers = {};
    const countEl = document.getElementById('map-vehicle-count');

    function updateMarkers(vehicles) {
        if (!Array.isArray(vehicles)) return;
        if (countEl) {
            countEl.textContent = `${vehicles.length} vehicle${vehicles.length === 1 ? '' : 's'}`;
        }

        vehicles.forEach(v => {
            const lat = v.lat || v.latitude;
            const lng = v.lng || v.longitude;
            const id = v.vehicle_id || v.id;
            const vehNum = v.vehicle_number || v.vehicle_id || 'Vehicle';
            const speed = typeof v.speed === 'number' ? Math.round(v.speed) : 0;
            const status = v.status || 'running';

            if (!lat || !lng || !id) return;

            const popupContent = `
                <div class="p-1 font-sans text-xs">
                    <div class="font-bold text-sm text-gray-900">${vehNum}</div>
                    <div class="text-gray-600 mt-0.5">Speed: <span class="font-mono font-semibold">${speed} km/h</span></div>
                    <div class="text-gray-600">Status: <span class="capitalize font-medium">${status}</span></div>
                    ${v.trip_id ? `<div class="text-blue-600 mt-1"><a href="/trips/${v.trip_id}" class="underline">View Active Trip</a></div>` : ''}
                </div>
            `;

            if (markers[id]) {
                markers[id].setLatLng([lat, lng]);
                markers[id].getPopup().setContent(popupContent);
            } else {
                markers[id] = L.marker([lat, lng])
                    .addTo(map)
                    .bindPopup(popupContent);
            }
        });
    }

    if (window.EventSource) {
        const es = new EventSource('/map/stream');
        es.addEventListener('datastar-merge-signals', (e) => {
            try {
                const parsed = JSON.parse(e.data);
                if (parsed.vehicles) {
                    updateMarkers(parsed.vehicles);
                }
            } catch (err) {
                console.error("Map SSE parse error", err);
            }
        });
        es.onerror = () => {
            // EventSource auto-reconnects
        };
    }
});
